package runtime

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/pathpattern"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

var defaultBackendConf = &config.Backend{
	ConnectTimeout:   "10s",
	RequestBodyLimit: "64MiB",
	TTFBTimeout:      "60s",
	Timeout:          "300s",
}

var errorMissingBackend = fmt.Errorf("no backend attribute reference or block")

// NewServerConfiguration sets http handler specific defaults and validates the given gateway configuration.
// Wire up all endpoints and maps them within the returned Server.
func NewServerConfiguration(conf *config.Gateway, httpConf *HTTPConfig, log *logrus.Entry) Server {
	if len(conf.Server) == 0 {
		log.Fatal("Missing server definitions")
	}

	// (arg && env) > conf
	defaultPort := conf.Settings.DefaultPort
	if httpConf.ListenPort != defaultPort {
		defaultPort = httpConf.ListenPort
	}

	type backendDefinition struct {
		conf    *config.Backend
		handler http.Handler
	}

	backends := make(map[string]backendDefinition)
	api := make(map[*config.Endpoint]http.Handler)

	if conf.Definitions != nil {
		for _, beConf := range conf.Definitions.Backend {
			if _, ok := backends[beConf.Name]; ok {
				log.Fatalf("backend name must be unique: '%s'", beConf.Name)
			}

			if beConf.Origin == "" {
				log.Fatalf("backend %q: origin attribute is required", beConf.Name)
			}

			beConf, _ = defaultBackendConf.Merge(beConf)
			proxyOptions, err := handler.NewProxyOptions(beConf, nil, []hcl.Body{beConf.Options})
			if err != nil {
				log.Fatal(err)
			}

			proxy, err := handler.NewProxy(proxyOptions, log, conf.Context)
			if err != nil {
				log.Fatal(err)
			}
			backends[beConf.Name] = backendDefinition{
				conf:    beConf,
				handler: proxy,
			}
		}
	}

	accessControls := configureAccessControls(conf)

	server := make(Server, 0)

	for _, srvConf := range conf.Server {
		muxOptions, err := NewMuxOptions(srvConf)
		if err != nil {
			log.Fatal(err)
		}

		var spaHandler http.Handler
		if srvConf.Spa != nil {
			spaHandler, err = handler.NewSpa(srvConf.Spa.BootstrapFile)
			if err != nil {
				log.Fatal(err)
			}
			for _, spaPath := range srvConf.Spa.Paths {
				for _, p := range getPathsFromHosts(defaultPort, srvConf.Hosts,
					utils.JoinPath(srvConf.BasePath, srvConf.Spa.BasePath, spaPath)) {
					muxOptions.SPARoutes[p] = spaHandler
				}
			}
		}

		if muxOptions.FileHandler != nil { // TODO: protected handler uses template from child handler
			muxOptions.FileHandler = configureProtectedHandler(accessControls, errors.DefaultHTML,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl), muxOptions.FileHandler)
		}

		if spaHandler != nil {
			spaHandler = configureProtectedHandler(accessControls, errors.DefaultHTML,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Spa.AccessControl, srvConf.Spa.DisableAccessControl), spaHandler)
		}

		if srvConf.API == nil {
			if err = configureHandlers(defaultPort, srvConf, muxOptions, server); err != nil {
				log.Fatal(err)
			}
			continue
		}

		// map backends to endpoint
		endpoints := make(map[string]bool)
		for _, endpoint := range srvConf.API.Endpoint {
			pattern := utils.JoinPath(srvConf.BasePath, srvConf.API.BasePath, endpoint.Pattern)
			if err != nil {
				log.Fatal(err)
			}

			if endpoints[pattern] {
				log.Fatal("Duplicate endpoint: ", pattern)
			}
			endpoints[pattern] = true

			// setACHandlerFn individual wrap for access_control configuration per endpoint
			setACHandlerFn := func(protectedBackend backendDefinition) {
				protectedHandler := protectedBackend.handler

				// prefer endpoint 'path' definition over 'backend.Path'
				if endpoint.Path != "" {
					beConf, remainCtx := protectedBackend.conf.Merge(&config.Backend{Path: endpoint.Path})

					corsOptions, err := handler.NewCORSOptions(srvConf.API.CORS)
					if err != nil {
						log.Fatal(err)
					}

					proxyOptions, err := handler.NewProxyOptions(beConf, corsOptions, remainCtx)
					if err != nil {
						log.Fatal(err)
					}

					proxy, err := handler.NewProxy(proxyOptions, log, conf.Context)
					if err != nil {
						log.Fatal(err)
					}
					protectedHandler = proxy
				}

				api[endpoint] = configureProtectedHandler(accessControls, muxOptions.APIErrTpl,
					config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl).
						Merge(config.NewAccessControl(srvConf.API.AccessControl, srvConf.API.DisableAccessControl)),
					config.NewAccessControl(endpoint.AccessControl, endpoint.DisableAccessControl),
					protectedHandler)
			}

			// lookup for backend reference, prefer endpoint definition over api one
			if endpoint.Backend != "" {
				if _, ok := backends[endpoint.Backend]; !ok {
					log.Fatalf("backend %q is not defined", endpoint.Backend)
				}
				setACHandlerFn(backends[endpoint.Backend])
				continue
			}

			// otherwise try to parse an inline block and fallback for api reference or inline block
			inlineBackend, inlineConf, err := newInlineBackend(conf.Context, endpoint.InlineDefinition, srvConf.API.CORS, log)
			if err == errorMissingBackend {
				if srvConf.API.Backend != "" {
					if _, ok := backends[srvConf.API.Backend]; !ok {
						log.Fatalf("backend %q is not defined", srvConf.API.Backend)
					}
					setACHandlerFn(backends[srvConf.API.Backend])
					continue
				}
				inlineBackend, inlineConf, err = newInlineBackend(conf.Context, srvConf.API.InlineDefinition, srvConf.API.CORS, log)
				if err != nil {
					log.Fatal(err)
				}

				if inlineConf.Name == "" && inlineConf.Origin == "" {
					log.Fatal("api inline backend requires an origin attribute")
				}

			} else if err != nil {
				if err == handler.OriginRequiredError && inlineConf.Name == "" || err != handler.OriginRequiredError {
					log.Fatalf("Range: %s: %v", endpoint.InlineDefinition.MissingItemRange().String(), err) // TODO diags error
				}
			}

			if inlineConf.Name != "" { // inline backends have no label, assume a reference and override settings
				if _, ok := backends[inlineConf.Name]; !ok {
					log.Fatalf("override backend %q is not defined", inlineConf.Name)
				}

				beConf, remainCtx := backends[inlineConf.Name].conf.Merge(inlineConf)

				corsOptions, err := handler.NewCORSOptions(srvConf.API.CORS)
				if err != nil {
					log.Fatal(err)
				}

				proxyOptions, err := handler.NewProxyOptions(beConf, corsOptions, remainCtx)
				if err != nil {
					log.Fatal(err)
				}

				proxy, err := handler.NewProxy(proxyOptions, log, conf.Context)
				if err != nil {
					log.Fatal(err)
				}
				inlineBackend = proxy
			}

			setACHandlerFn(backendDefinition{conf: inlineConf, handler: inlineBackend})

			for _, hostPath := range getPathsFromHosts(defaultPort, srvConf.Hosts, pattern) {
				muxOptions.EndpointRoutes[hostPath] = api[endpoint]
			}
		}

		if err = configureHandlers(defaultPort, srvConf, muxOptions, server); err != nil {
			log.Fatal(err)
		}
	}
	return server
}

func configureHandlers(configuredPort int, server *config.Server, mux *MuxOptions, srvMux Server) error {
	hosts := server.Hosts
	if len(hosts) == 0 {
		hosts = []string{fmt.Sprintf("*:%d", configuredPort)}
	}

	for _, hp := range hosts {
		port := Port(strconv.Itoa(configuredPort))
		if strings.IndexByte(hp, ':') > 0 {
			_, p, err := net.SplitHostPort(hp)
			if err != nil {
				return err
			}
			port = Port(p)
		}
		srvMux[port] = &ServerMux{Server: server, Mux: mux}
	}
	return nil
}

func configureAccessControls(conf *config.Gateway) ac.Map {
	accessControls := make(ac.Map)

	if conf.Definitions != nil {
		for _, ba := range conf.Definitions.BasicAuth {
			name, err := validateACName(accessControls, ba.Name, "basic_auth")
			if err != nil {
				panic(err)
			}

			basicAuth, err := ac.NewBasicAuth(name, ba.User, ba.Pass, ba.File, ba.Realm)
			if err != nil {
				panic(err)
			}

			accessControls[name] = basicAuth
		}

		for _, jwt := range conf.Definitions.JWT {
			name, err := validateACName(accessControls, jwt.Name, "jwt")
			if err != nil {
				panic(err)
			}

			var jwtSource ac.Source
			var jwtKey string
			if jwt.Cookie != "" {
				jwtSource = ac.Cookie
				jwtKey = jwt.Cookie
			} else if jwt.Header != "" {
				jwtSource = ac.Header
				jwtKey = jwt.Header
			}
			var key []byte
			if jwt.KeyFile != "" {
				wd, _ := os.Getwd()
				content, err := ioutil.ReadFile(path.Join(wd, jwt.KeyFile))
				if err != nil {
					panic(err)
				}
				key = content
			} else if jwt.Key != "" {
				key = []byte(jwt.Key)
			}

			var claims ac.Claims
			if jwt.Claims != nil {
				c, diags := seetie.ExpToMap(conf.Context, jwt.Claims)
				if diags.HasErrors() {
					panic(diags.Error())
				}
				claims = c
			}
			j, err := ac.NewJWT(jwt.SignatureAlgorithm, name, claims, jwt.ClaimsRequired, jwtSource, jwtKey, key)
			if err != nil {
				panic(fmt.Sprintf("loading jwt %q definition failed: %s", name, err))
			}

			accessControls[name] = j
		}
	}

	return accessControls
}

func validateACName(accessControls ac.Map, name, acType string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return name, fmt.Errorf("Missing a non-empty label for %q", acType)
	}

	if _, ok := accessControls[name]; ok {
		return name, fmt.Errorf("Label %q already exists in the ACL", name)
	}

	return name, nil
}

func configureProtectedHandler(m ac.Map, errTpl *errors.Template, parentAC, handlerAC config.AccessControl, h http.Handler) http.Handler {
	var acList ac.List
	for _, acName := range parentAC.
		Merge(handlerAC).List() {
		m.MustExist(acName)
		acList = append(acList, m[acName])
	}
	if len(acList) > 0 {
		return handler.NewAccessControl(h, errTpl, acList...)
	}
	return h
}

func newInlineBackend(evalCtx *hcl.EvalContext, inlineDef hcl.Body, cors *config.CORS, log *logrus.Entry) (http.Handler, *config.Backend, error) {
	content, _, diags := inlineDef.PartialContent(config.Definitions{}.Schema(true))
	// ignore diag errors here, would fail anyway with our retry
	if content == nil || len(content.Blocks) == 0 {
		// no inline conf, retry for override definitions with label
		content, _, diags = inlineDef.PartialContent(config.Definitions{}.Schema(false))
		if diags.HasErrors() {
			return nil, nil, diags
		}

		if content == nil || len(content.Blocks) == 0 {
			return nil, nil, errorMissingBackend
		}
	}

	beConf := &config.Backend{}
	diags = gohcl.DecodeBody(content.Blocks[0].Body, evalCtx, beConf)
	if diags.HasErrors() {
		return nil, nil, diags
	}
	if len(content.Blocks[0].Labels) > 0 {
		beConf.Name = content.Blocks[0].Labels[0]
	}

	beConf, _ = defaultBackendConf.Merge(beConf)

	corsOptions, err := handler.NewCORSOptions(cors)
	if err != nil {
		return nil, nil, err
	}

	proxyOptions, err := handler.NewProxyOptions(beConf, corsOptions, []hcl.Body{beConf.Options})
	if err != nil {
		return nil, nil, err
	}

	proxy, err := handler.NewProxy(proxyOptions, log, evalCtx)
	return proxy, beConf, err
}

func getPathsFromHosts(defaultPort int, hosts []string, path string) []string {
	var list []string
	port := strconv.Itoa(defaultPort)
	for _, host := range hosts {
		if host != "" && host[0] == ':' {
			continue
		}

		if strings.IndexByte(host, ':') < 0 {
			host = host + ":" + port
		}

		if host != "" && host[0] == '*' {
			host = ""
		}

		list = append(list, utils.JoinPath(pathpattern.PathFromHost(host, false), "/", path))
	}
	if len(list) == 0 {
		list = []string{path}
	}
	return list
}

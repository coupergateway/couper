package runtime

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
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
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

var DefaultBackendConf = &config.Backend{
	ConnectTimeout:   "10s",
	RequestBodyLimit: "64MiB",
	TTFBTimeout:      "60s",
	Timeout:          "300s",
}

type Port int

func (p Port) String() string {
	return strconv.Itoa(int(p))
}

type ServerConfiguration map[Port]*MuxOptions

type hosts map[string]bool
type ports map[Port]hosts

type HandlerKind uint8

const (
	KindAPI HandlerKind = iota
	KindEndpoint
	KindFiles
	KindSPA
)

type endpointMap map[*config.Endpoint]*config.API

// NewServerConfiguration sets http handler specific defaults and validates the given gateway configuration.
// Wire up all endpoints and maps them within the returned Server.
func NewServerConfiguration(conf *config.Couper, log *logrus.Entry) (ServerConfiguration, error) {
	defaultPort := conf.Settings.DefaultPort

	// confCtx is created to evaluate request / response related configuration errors on start.
	noopReq := httptest.NewRequest(http.MethodGet, "https://couper.io", nil)
	noopResp := httptest.NewRecorder().Result()
	noopResp.Request = noopReq
	confCtx := eval.NewHTTPContext(conf.Context, 0, noopReq, noopReq, noopResp)

	validPortMap, hostsMap, err := validatePortHosts(conf, defaultPort)
	if err != nil {
		return nil, err
	}

	accessControls, err := configureAccessControls(conf, confCtx)
	if err != nil {
		return nil, err
	}

	serverConfiguration := make(ServerConfiguration)
	if len(validPortMap) == 0 {
		serverConfiguration[Port(defaultPort)] = NewMuxOptions(hostsMap)
	} else {
		for p := range validPortMap {
			serverConfiguration[p] = NewMuxOptions(hostsMap)
		}
	}

	endpointHandler := make(map[*config.Endpoint]http.Handler)

	for _, srvConf := range conf.Servers {
		serverOptions, err := server.NewServerOptions(srvConf)
		if err != nil {
			return nil, err
		}

		var spaHandler http.Handler
		if srvConf.Spa != nil {
			spaHandler, err = handler.NewSpa(srvConf.Spa.BootstrapFile, serverOptions)
			if err != nil {
				return nil, err
			}

			spaHandler = configureProtectedHandler(accessControls, serverOptions.ServerErrTpl,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Spa.AccessControl, srvConf.Spa.DisableAccessControl), spaHandler)

			for _, spaPath := range srvConf.Spa.Paths {
				err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, path.Join(serverOptions.SPABasePath, spaPath), spaHandler, KindSPA)
				if err != nil {
					return nil, err
				}
			}
		}

		if srvConf.Files != nil {
			fileHandler, err := handler.NewFile(serverOptions.FileBasePath, srvConf.Files.DocumentRoot, serverOptions)
			if err != nil {
				return nil, err
			}

			protectedFileHandler := configureProtectedHandler(accessControls, serverOptions.FileErrTpl,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl), fileHandler)

			err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, serverOptions.FileBasePath, protectedFileHandler, KindFiles)
			if err != nil {
				return nil, err
			}
		}

		endpointsPatterns := make(map[string]bool)

		for endpoint, parentAPI := range newEndpointMap(srvConf) {
			var basePath string
			var cors *config.CORS
			var errTpl *errors.Template

			if parentAPI != nil {
				basePath = serverOptions.APIBasePath[parentAPI]
				cors = parentAPI.CORS
				errTpl = serverOptions.APIErrTpl[parentAPI]
			} else {
				basePath = serverOptions.SrvBasePath
				errTpl = serverOptions.ServerErrTpl
			}

			pattern := utils.JoinPath(basePath, endpoint.Pattern)
			unique, cleanPattern := isUnique(endpointsPatterns, pattern)
			if !unique {
				return nil, fmt.Errorf("%s: duplicate endpoint: '%s'", endpoint.HCLBody().MissingItemRange().String(), pattern)
			}
			endpointsPatterns[cleanPattern] = true

			// setACHandlerFn individual wrap for access_control configuration per endpoint
			setACHandlerFn := func(protectedHandler http.Handler) {
				accessControl := config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl)

				if parentAPI != nil {
					accessControl = accessControl.Merge(config.NewAccessControl(parentAPI.AccessControl, parentAPI.DisableAccessControl))
				}

				endpointHandler[endpoint] = configureProtectedHandler(accessControls, errTpl, accessControl,
					config.NewAccessControl(endpoint.AccessControl, endpoint.DisableAccessControl),
					protectedHandler)
			}

			backendConf := *DefaultBackendConf
			if diags := gohcl.DecodeBody(endpoint.Remain, confCtx, &backendConf); diags.HasErrors() {
				return nil, diags
			}

			kind := KindEndpoint
			if parentAPI != nil {
				kind = KindAPI
			}
			backend, err := newProxy(confCtx, &backendConf, cors, log, serverOptions,
				conf.Settings.NoProxyFromEnv, errTpl, kind)
			if err != nil {
				return nil, err
			}

			setACHandlerFn(backend)

			err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, pattern, endpointHandler[endpoint], KindAPI)
			if err != nil {
				return nil, err
			}
		}
	}

	return serverConfiguration, nil
}

func newProxy(
	ctx *hcl.EvalContext, beConf *config.Backend, corsOpts *config.CORS, log *logrus.Entry,
	srvOpts *server.Options, noProxyFromEnv bool, errTpl *errors.Template, epType HandlerKind) (http.Handler, error) {
	corsOptions, err := handler.NewCORSOptions(corsOpts)
	if err != nil {
		return nil, err
	}

	var kind string
	switch epType {
	case KindAPI:
		kind = "api"
	case KindEndpoint:
		kind = "endpoint"
	}

	proxyOptions, err := handler.NewProxyOptions(beConf, corsOptions, noProxyFromEnv, errTpl, kind)
	if err != nil {
		return nil, err
	}

	return handler.NewProxy(proxyOptions, log, srvOpts, ctx)
}

func splitWildcardHostPort(host string, configuredPort int) (string, Port, error) {
	if !strings.Contains(host, ":") {
		return host, Port(configuredPort), nil
	}

	ho := host
	po := configuredPort
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return "", -1, err
	}
	ho = h
	if p != "" && p != "*" {
		if !rePortCheck.MatchString(p) {
			return "", -1, fmt.Errorf("invalid port given: %s", p)
		}
		po, err = strconv.Atoi(p)
		if err != nil {
			return "", -1, err
		}
	}

	return ho, Port(po), nil
}

func configureAccessControls(conf *config.Couper, confCtx *hcl.EvalContext) (ac.Map, error) {
	accessControls := make(ac.Map)

	if conf.Definitions != nil {
		for _, ba := range conf.Definitions.BasicAuth {
			name, err := validateACName(accessControls, ba.Name, "basic_auth")
			if err != nil {
				return nil, err
			}

			basicAuth, err := ac.NewBasicAuth(name, ba.User, ba.Pass, ba.File, ba.Realm)
			if err != nil {
				return nil, err
			}

			accessControls[name] = basicAuth
		}

		for _, jwt := range conf.Definitions.JWT {
			name, err := validateACName(accessControls, jwt.Name, "jwt")
			if err != nil {
				return nil, err
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
				wd, err := os.Getwd()
				if err != nil {
					return nil, err
				}
				content, err := ioutil.ReadFile(path.Join(wd, jwt.KeyFile))
				if err != nil {
					return nil, err
				}
				key = content
			} else if jwt.Key != "" {
				key = []byte(jwt.Key)
			}

			var claims ac.Claims
			if jwt.Claims != nil {
				c, diags := seetie.ExpToMap(confCtx, jwt.Claims)
				if diags.HasErrors() {
					return nil, diags
				}
				claims = c
			}
			j, err := ac.NewJWT(jwt.SignatureAlgorithm, name, claims, jwt.ClaimsRequired, jwtSource, jwtKey, key)
			if err != nil {
				return nil, fmt.Errorf("loading jwt %q definition failed: %s", name, err)
			}

			accessControls[name] = j
		}
	}

	return accessControls, nil
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

func setRoutesFromHosts(srvConf ServerConfiguration, defaultPort int, hosts []string, path string, handler http.Handler, kind HandlerKind) error {
	hostList := hosts
	if len(hostList) == 0 {
		hostList = []string{"*"}
	}

	for _, h := range hostList {
		joinedPath := utils.JoinPath("/", path)
		host, listenPort, err := splitWildcardHostPort(h, defaultPort)
		if err != nil {
			return err
		}

		if host != "*" {
			joinedPath = utils.JoinPath(
				pathpattern.PathFromHost(
					net.JoinHostPort(host, listenPort.String()), false), "/", path)
		}

		var routes map[string]http.Handler

		switch kind {
		case KindAPI:
			fallthrough
		case KindEndpoint:
			routes = srvConf[listenPort].EndpointRoutes
		case KindFiles:
			routes = srvConf[listenPort].FileRoutes
		case KindSPA:
			routes = srvConf[listenPort].SPARoutes
		default:
			return fmt.Errorf("unknown route kind")
		}

		if _, exist := routes[joinedPath]; exist {
			return fmt.Errorf("duplicate route found on port %q: %q", listenPort.String(), path)
		}
		routes[joinedPath] = handler
	}
	return nil
}

func newEndpointMap(srvConf *config.Server) endpointMap {
	endpoints := make(endpointMap)

	for _, api := range srvConf.APIs {
		for _, endpoint := range api.Endpoints {
			endpoints[endpoint] = api
		}
	}

	for _, endpoint := range srvConf.Endpoints {
		endpoints[endpoint] = nil
	}

	return endpoints
}

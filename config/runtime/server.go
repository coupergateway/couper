package runtime

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
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

var (
	errorMissingBackend = fmt.Errorf("no backend attribute reference or block")
	errorMissingServer  = fmt.Errorf("missing server definitions")
)

var (
	// reValidFormat validates the format only, validating for a valid host or port is out of scope.
	reValidFormat  = regexp.MustCompile(`^([a-z0-9.-]+|\*)(:\*|:\d{1,5})?$`)
	reCleanPattern = regexp.MustCompile(`{([^}]+)}`)
	rePortCheck    = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})$`)
)

type backendDefinition struct {
	conf    *config.Backend
	handler http.Handler
}

type Port int

func (p Port) String() string {
	return strconv.Itoa(int(p))
}

type ServerConfiguration struct {
	PortOptions map[Port]*MuxOptions
}

type hosts map[string]bool
type ports map[Port]hosts

type HandlerKind uint8

const (
	KindAPI HandlerKind = iota
	KindEndpoint
	KindFiles
	KindSPA
)

type endpointList map[*config.Endpoint]HandlerKind

// NewServerConfiguration sets http handler specific defaults and validates the given gateway configuration.
// Wire up all endpoints and maps them within the returned Server.
func NewServerConfiguration(conf *config.Gateway, httpConf *HTTPConfig, log *logrus.Entry) (*ServerConfiguration, error) {
	if len(conf.Server) == 0 {
		return nil, errorMissingServer
	}

	// (arg && env) > conf
	defaultPort := conf.Settings.DefaultPort
	if httpConf.ListenPort != defaultPort {
		defaultPort = httpConf.ListenPort
	}

	validPortMap, hostsMap, err := validatePortHosts(conf, defaultPort)
	if err != nil {
		return nil, err
	}

	backends, err := newBackendsFromDefinitions(conf, log)
	if err != nil {
		return nil, err
	}

	accessControls, err := configureAccessControls(conf)
	if err != nil {
		return nil, err
	}

	serverConfiguration := &ServerConfiguration{PortOptions: map[Port]*MuxOptions{
		Port(defaultPort): NewMuxOptions(hostsMap)},
	}
	for p := range validPortMap {
		serverConfiguration.PortOptions[p] = NewMuxOptions(hostsMap)
	}

	api := make(map[*config.Endpoint]http.Handler)

	for _, srvConf := range conf.Server {
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
		endpointsList := getEndpointsList(srvConf)

		for endpoint, epType := range endpointsList {
			var basePath string
			var cors *config.CORS
			var errTpl *errors.Template

			switch epType {
			case KindAPI:
				basePath = serverOptions.APIBasePath
				cors = srvConf.API.CORS
				errTpl = serverOptions.APIErrTpl
			case KindEndpoint:
				basePath = serverOptions.SrvBasePath
				errTpl = serverOptions.ServerErrTpl
			}

			pattern := utils.JoinPath(basePath, endpoint.Pattern)

			unique, cleanPattern := isUnique(endpointsPatterns, pattern)
			if !unique {
				return nil, fmt.Errorf("duplicate endpoint: %q", pattern)
			}
			endpointsPatterns[cleanPattern] = true

			// setACHandlerFn individual wrap for access_control configuration per endpoint
			setACHandlerFn := func(protectedBackend backendDefinition) {
				protectedHandler := protectedBackend.handler

				// prefer endpoint 'path' definition over 'backend.Path'
				if endpoint.Path != "" {
					beConf, remainCtx := protectedBackend.conf.Merge(&config.Backend{Path: endpoint.Path})
					protectedHandler = newProxy(conf.Context, beConf, cors, remainCtx, log, serverOptions, errTpl, epType)
				}

				parentAC := config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl)
				if epType == KindAPI {
					parentAC = parentAC.Merge(config.NewAccessControl(srvConf.API.AccessControl, srvConf.API.DisableAccessControl))
				}

				api[endpoint] = configureProtectedHandler(
					accessControls,
					errTpl,
					parentAC,
					config.NewAccessControl(endpoint.AccessControl, endpoint.DisableAccessControl),
					protectedHandler,
				)
			}

			// lookup for backend reference, prefer endpoint definition over api one
			if endpoint.Backend != "" {
				if _, ok := backends[endpoint.Backend]; !ok {
					return nil, fmt.Errorf("backend %q is not defined", endpoint.Backend)
				}

				// set server context for defined backends
				be := backends[endpoint.Backend]
				beConf, remain := be.conf.Merge(&config.Backend{Options: endpoint.InlineDefinition})
				refBackend := newProxy(conf.Context, be.conf, cors, remain, log, serverOptions, errTpl, epType)

				setACHandlerFn(backendDefinition{
					conf:    beConf,
					handler: refBackend,
				})
				err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, pattern, api[endpoint], epType)
				if err != nil {
					return nil, err
				}
				continue
			}

			// otherwise try to parse an inline block and fallback for api reference or inline block
			inlineBackend, inlineConf, err := newInlineBackend(conf.Context, backends, endpoint.InlineDefinition, cors, log, serverOptions)
			if err == errorMissingBackend && epType == KindAPI {
				if srvConf.API.Backend != "" {
					if _, ok := backends[srvConf.API.Backend]; !ok {
						return nil, fmt.Errorf("backend %q is not defined", srvConf.API.Backend)
					}
					setACHandlerFn(backends[srvConf.API.Backend])
					err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, pattern, api[endpoint], epType)
					if err != nil {
						return nil, err
					}
					continue
				}
				inlineBackend, inlineConf, err = newInlineBackend(conf.Context, backends, srvConf.API.InlineDefinition, cors, log, serverOptions)
				if err != nil {
					return nil, err
				}

				if inlineConf.Name == "" && inlineConf.Origin == "" {
					return nil, fmt.Errorf("api inline backend requires an origin attribute: %q", pattern)
				}
			} else if err != nil { // TODO hcl.diagnostics error
				return nil, fmt.Errorf("range: %s: %v", endpoint.InlineDefinition.MissingItemRange().String(), err)
			}

			setACHandlerFn(backendDefinition{conf: inlineConf, handler: inlineBackend})
			err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, pattern, api[endpoint], epType)
			if err != nil {
				return nil, err
			}
		}

	}
	return serverConfiguration, nil
}

func newProxy(
	ctx *hcl.EvalContext, beConf *config.Backend, corsOpts *config.CORS,
	remainCtx []hcl.Body, log *logrus.Entry, srvOpts *server.Options,
	errTpl *errors.Template, epType HandlerKind,
) http.Handler {
	corsOptions, err := handler.NewCORSOptions(corsOpts)
	if err != nil {
		log.Fatal(err)
	}

	var kind string
	switch epType {
	case KindAPI:
		kind = "api"
	case KindEndpoint:
		kind = "endpoint"
	}

	proxyOptions, err := handler.NewProxyOptions(beConf, corsOptions, remainCtx, errTpl, kind)
	if err != nil {
		log.Fatal(err)
	}

	proxy, err := handler.NewProxy(proxyOptions, log, srvOpts, ctx)
	if err != nil {
		log.Fatal(err)
	}
	return proxy
}

func newBackendsFromDefinitions(conf *config.Gateway, log *logrus.Entry) (map[string]backendDefinition, error) {
	backends := make(map[string]backendDefinition)

	if conf.Definitions == nil {
		return backends, nil
	}

	for _, beConf := range conf.Definitions.Backend {
		if _, ok := backends[beConf.Name]; ok {
			return nil, fmt.Errorf("backend name must be unique: %q", beConf.Name)
		}

		if beConf.Origin == "" {
			return nil, fmt.Errorf("backend %q: origin attribute is required", beConf.Name)
		}

		beConf, _ = defaultBackendConf.Merge(beConf)

		// TODO: Select the right KIND
		srvOpts, _ := server.NewServerOptions(&config.Server{})
		backends[beConf.Name] = backendDefinition{
			conf:    beConf,
			handler: newProxy(conf.Context, beConf, nil, []hcl.Body{beConf.Options}, log, srvOpts, srvOpts.APIErrTpl, KindAPI),
		}
	}
	return backends, nil
}

// validatePortHosts ensures expected host:port formats and unique hosts per port.
// Host options:
//	"*:<port>"					listen for all hosts on given port
//	"*:<port(configuredPort)>	given port equals configured default port, listen for all hosts
//	"*"							equals to "*:configuredPort"
//	"host:*"					equals to "host:configuredPort"
//	"host"						listen on configured default port for given host
func validatePortHosts(conf *config.Gateway, configuredPort int) (ports, hosts, error) {
	portMap := make(ports)
	hostMap := make(hosts)
	isHostsMandatory := len(conf.Server) > 1

	for _, srv := range conf.Server {
		if isHostsMandatory && len(srv.Hosts) == 0 {
			return nil, nil, fmt.Errorf("hosts attribute is mandatory for multiple servers: %q", srv.Name)
		}

		srvPortMap := make(ports)
		for _, host := range srv.Hosts {
			if !reValidFormat.MatchString(host) {
				return nil, nil, fmt.Errorf("host format is invalid: %q", host)
			}

			ho, po, err := splitWildcardHostPort(host, configuredPort)
			if err != nil {
				return nil, nil, err
			}

			if _, ok := srvPortMap[po]; !ok {
				srvPortMap[po] = make(hosts)
			}

			srvPortMap[po][ho] = true

			hostMap[fmt.Sprintf("%s:%d", ho, po)] = true
		}

		// srvPortMap contains all unique host port combinations for
		// the current server and should not exist multiple times.
		for po, ho := range srvPortMap {
			if _, ok := portMap[po]; !ok {
				portMap[po] = make(hosts)
			}

			for h := range ho {
				if _, ok := portMap[po][h]; ok {
					return nil, nil, fmt.Errorf("conflict: host %q already defined for port: %d", h, po)
				}

				portMap[po][h] = true
			}
		}
	}

	return portMap, hostMap, nil
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

func configureAccessControls(conf *config.Gateway) (ac.Map, error) {
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
				c, diags := seetie.ExpToMap(conf.Context, jwt.Claims)
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

func newInlineBackend(evalCtx *hcl.EvalContext, backends map[string]backendDefinition, inlineDef hcl.Body, cors *config.CORS, log *logrus.Entry, srvOpts *server.Options) (http.Handler, *config.Backend, error) {
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

	beConf, _ = defaultBackendConf.Merge(beConf)
	if len(content.Blocks[0].Labels) > 0 {
		beConf.Name = content.Blocks[0].Labels[0]
		if beRef, ok := backends[beConf.Name]; ok {
			beConf, _ = beRef.conf.Merge(beConf)
		} else {
			return nil, nil, fmt.Errorf("override backend %q is not defined", beConf.Name)
		}
	}

	// TODO: Select the right KIND
	proxy := newProxy(evalCtx, beConf, cors, []hcl.Body{beConf.Options}, log, srvOpts, srvOpts.APIErrTpl, KindAPI)
	return proxy, beConf, nil
}

func setRoutesFromHosts(srvConf *ServerConfiguration, confPort int, hosts []string, path string, handler http.Handler, kind HandlerKind) error {
	hostList := hosts
	if len(hostList) == 0 {
		hostList = []string{"*"}
	}

	for _, h := range hostList {
		joinedPath := utils.JoinPath("/", path)
		host, listenPort, err := splitWildcardHostPort(h, confPort)
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
			routes = srvConf.PortOptions[listenPort].EndpointRoutes
		case KindFiles:
			routes = srvConf.PortOptions[listenPort].FileRoutes
		case KindSPA:
			routes = srvConf.PortOptions[listenPort].SPARoutes
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

func isUnique(endpoints map[string]bool, pattern string) (bool, string) {
	pattern = reCleanPattern.ReplaceAllString(pattern, "{}")

	return !endpoints[pattern], pattern
}

func getEndpointsList(srvConf *config.Server) endpointList {
	endpoints := make(endpointList)

	if srvConf.API != nil {
		for _, endpoint := range srvConf.API.Endpoint {
			endpoints[endpoint] = KindAPI
		}
	}

	for _, endpoint := range srvConf.Endpoint {
		endpoints[endpoint] = KindEndpoint
	}

	return endpoints
}

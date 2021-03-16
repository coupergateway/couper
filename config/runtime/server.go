//go:generate stringer -type=HandlerKind -output=./server_string.go

package runtime

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/getkin/kin-openapi/pathpattern"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

var DefaultBackendConf = &config.Backend{
	ConnectTimeout: "10s",
	TTFBTimeout:    "60s",
	Timeout:        "300s",
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
	api HandlerKind = iota
	endpoint
	files
	spa
)

type endpointMap map[*config.Endpoint]*config.API

// NewServerConfiguration sets http handler specific defaults and validates the given gateway configuration.
// Wire up all endpoints and maps them within the returned Server.
func NewServerConfiguration(
	conf *config.Couper, log *logrus.Entry, memStore *cache.MemoryStore,
) (ServerConfiguration, error) {
	defaultPort := conf.Settings.DefaultPort

	// confCtx is created to evaluate request / response related configuration errors on start.
	noopReq := httptest.NewRequest(http.MethodGet, "https://couper.io", nil)
	noopResp := httptest.NewRecorder().Result()
	noopResp.Request = noopReq
	confCtx := conf.Context.WithClientRequest(noopReq).WithBeresps(noopResp).HCLContext()

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
		serverConfiguration[Port(defaultPort)] = NewMuxOptions(errors.DefaultHTML, hostsMap)
	} else {
		for p := range validPortMap {
			serverConfiguration[p] = NewMuxOptions(errors.DefaultHTML, hostsMap)
		}
	}

	endpointHandlers := make(map[*config.Endpoint]http.Handler)

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
				err = setRoutesFromHosts(serverConfiguration, serverOptions.ServerErrTpl, defaultPort, srvConf.Hosts, path.Join(serverOptions.SPABasePath, spaPath), spaHandler, spa)
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

			err = setRoutesFromHosts(serverConfiguration, serverOptions.ServerErrTpl, defaultPort, srvConf.Hosts, serverOptions.FileBasePath, protectedFileHandler, files)
			if err != nil {
				return nil, err
			}
		}

		endpointsPatterns := make(map[string]bool)

		for endpointConf, parentAPI := range newEndpointMap(srvConf) {
			var basePath string
			//var cors *config.CORS
			var errTpl *errors.Template

			if parentAPI != nil {
				basePath = serverOptions.APIBasePath[parentAPI]
				//cors = parentAPI.CORS
				errTpl = serverOptions.APIErrTpl[parentAPI]
			} else {
				basePath = serverOptions.SrvBasePath
				errTpl = serverOptions.ServerErrTpl
			}

			pattern := utils.JoinPath(basePath, endpointConf.Pattern)
			unique, cleanPattern := isUnique(endpointsPatterns, pattern)
			if !unique {
				return nil, fmt.Errorf("%s: duplicate endpoint: '%s'", endpointConf.HCLBody().MissingItemRange().String(), pattern)
			}
			endpointsPatterns[cleanPattern] = true

			// setACHandlerFn individual wrap for access_control configuration per endpoint
			setACHandlerFn := func(protectedHandler http.Handler) {
				accessControl := config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl)

				if parentAPI != nil {
					accessControl = accessControl.Merge(config.NewAccessControl(parentAPI.AccessControl, parentAPI.DisableAccessControl))
				}

				endpointHandlers[endpointConf] = configureProtectedHandler(accessControls, errTpl, accessControl,
					config.NewAccessControl(endpointConf.AccessControl, endpointConf.DisableAccessControl),
					protectedHandler)
			}

			var response *producer.Response
			if endpointConf.Response != nil {
				response = &producer.Response{
					Context: endpointConf.Response.Remain,
				}
			}

			var proxies producer.Proxies
			var requests producer.Requests
			//var redirect producer.Redirect

			for _, proxyConf := range endpointConf.Proxies {
				backend, berr := newBackend(confCtx, proxyConf.Backend, log, conf.Settings.NoProxyFromEnv, memStore)
				if berr != nil {
					return nil, berr
				}
				proxyHandler := handler.NewProxy(backend, proxyConf.HCLBody(), log)
				p := &producer.Proxy{
					Name:      proxyConf.Name,
					RoundTrip: proxyHandler,
				}
				proxies = append(proxies, p)
			}

			for _, requestConf := range endpointConf.Requests {
				backend, berr := newBackend(confCtx, requestConf.Backend, log, conf.Settings.NoProxyFromEnv, memStore)
				if berr != nil {
					return nil, berr
				}
				method := http.MethodGet
				if requestConf.Method != "" {
					method = requestConf.Method
				}
				requests = append(requests, &producer.Request{
					Backend: backend,
					Body:    requestConf.Body,
					Context: requestConf.Remain,
					Method:  method,
					Name:    requestConf.Name,
				})
			}

			backendConf := *DefaultBackendConf
			if diags := gohcl.DecodeBody(endpointConf.Remain, confCtx, &backendConf); diags.HasErrors() {
				return nil, diags
			}
			// TODO: redirect
			if endpointConf.Response == nil && len(proxies)+len(requests) == 0 { // && redirect == nil
				r := endpointConf.Remain.MissingItemRange()
				m := fmt.Sprintf("configuration error: endpoint %q requires at least one proxy, request, response or redirect block", endpointConf.Pattern)
				return nil, hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  m,
					Subject:  &r,
				}}
			}

			kind := endpoint
			if parentAPI != nil {
				kind = api
			}

			bodyLimit, err := parseBodyLimit(endpointConf.RequestBodyLimit)
			if err != nil {
				r := endpointConf.Remain.MissingItemRange()
				return nil, hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "parsing endpoint request body limit: " + endpointConf.Pattern,
					Subject:  &r,
				}}
			}

			// TODO: determine req/beresp.body access in this context (all including backend) or for now:
			bufferOpts := eval.MustBuffer(endpointConf.Remain)
			if len(proxies)+len(requests) > 1 { // also buffer with more possible results
				bufferOpts |= eval.BufferResponse
			}

			epOpts := &handler.EndpointOptions{
				Context:        endpointConf.Remain,
				Error:          errTpl,
				LogHandlerKind: kind.String(),
				LogPattern:     endpointConf.Pattern,
				ReqBodyLimit:   bodyLimit,
				ReqBufferOpts:  bufferOpts,
				ServerOpts:     serverOptions,
			}
			epHandler := handler.NewEndpoint(epOpts, log, proxies, requests, response)
			setACHandlerFn(epHandler)

			err = setRoutesFromHosts(serverConfiguration, serverOptions.ServerErrTpl, defaultPort, srvConf.Hosts, pattern, endpointHandlers[endpointConf], kind)
			if err != nil {
				return nil, err
			}
		}
	}

	return serverConfiguration, nil
}

func newBackend(
	evalCtx *hcl.EvalContext, backendCtx hcl.Body, log *logrus.Entry,
	ignoreProxyEnv bool, memStore *cache.MemoryStore) (http.RoundTripper, error) {
	beConf := *DefaultBackendConf
	if diags := gohcl.DecodeBody(backendCtx, evalCtx, &beConf); diags.HasErrors() {
		return nil, diags
	}

	if beConf.Name == "" {
		name, err := getBackendName(evalCtx, backendCtx)
		if err != nil {
			return nil, err
		}
		beConf.Name = name
	}

	tc := &transport.Config{
		BackendName:            beConf.Name,
		DisableCertValidation:  beConf.DisableCertValidation,
		DisableConnectionReuse: beConf.DisableConnectionReuse,
		HTTP2:                  beConf.HTTP2,
		NoProxyFromEnv:         ignoreProxyEnv,
		Proxy:                  beConf.Proxy,
		MaxConnections:         beConf.MaxConnections,
	}

	if err := parseDuration(beConf.ConnectTimeout, &tc.ConnectTimeout); err != nil {
		return nil, err
	}

	if err := parseDuration(beConf.TTFBTimeout, &tc.TTFBTimeout); err != nil {
		return nil, err
	}

	if err := parseDuration(beConf.Timeout, &tc.Timeout); err != nil {
		return nil, err
	}

	openAPIopts, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	if err != nil {
		return nil, err
	}

	options := &transport.BackendOptions{
		BasicAuth:  beConf.BasicAuth,
		OpenAPI:    openAPIopts,
		PathPrefix: beConf.PathPrefix,
	}
	backend := transport.NewBackend(backendCtx, tc, options, log)

	// TODO: partialContent with merged backendCtx fails !!!
	oauthContent, _, _ := backendCtx.PartialContent(config.OAuthBlockSchema)
	if oauthContent == nil {
		return backend, nil
	}

	if blocks := oauthContent.Blocks.OfType("oauth2"); len(blocks) > 0 {
		beConf.OAuth2 = &config.OAuth2{}

		if diags := gohcl.DecodeBody(blocks[0].Body, evalCtx, beConf.OAuth2); diags.HasErrors() {
			return nil, diags
		}

		innerContent, _, diags := beConf.OAuth2.Remain.PartialContent(beConf.OAuth2.Schema(true))
		if diags.HasErrors() {
			return nil, diags
		}
		innerBackend := innerContent.Blocks.OfType("backend")[0] // backend block is set by configload
		authBackend, authErr := newBackend(evalCtx, innerBackend.Body, log, ignoreProxyEnv, memStore)
		if authErr != nil {
			return nil, authErr
		}

		return transport.NewOAuth2(beConf.OAuth2, memStore, authBackend, backend)
	}

	return backend, nil
}

func getBackendName(evalCtx *hcl.EvalContext, backendCtx hcl.Body) (string, error) {
	content, _, _ := backendCtx.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{
		{Name: "name"}},
	})
	if content != nil && len(content.Attributes) > 0 {

		if n, exist := content.Attributes["name"]; exist {
			v, d := n.Expr.Value(evalCtx)
			if d.HasErrors() {
				return "", d
			}
			return v.AsString(), nil
		}
	}
	return "", nil
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

			accessControls[name] = ac.ValidateFunc(basicAuth.Validate)
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
				p, err := filepath.Abs(jwt.KeyFile)
				if err != nil {
					return nil, err
				}
				content, err := ioutil.ReadFile(p)
				if err != nil {
					return nil, err
				}
				key = content
			} else if jwt.Key != "" {
				key = []byte(jwt.Key)
			}

			var claims map[string]interface{}
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

			accessControls[name] = ac.ValidateFunc(j.Validate)
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

func setRoutesFromHosts(srvConf ServerConfiguration, srvErrHandler *errors.Template, defaultPort int, hosts []string, path string, handler http.Handler, kind HandlerKind) error {
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

		srvConf[listenPort].ErrorTpl = srvErrHandler

		var routes map[string]http.Handler

		switch kind {
		case api:
			fallthrough
		case endpoint:
			routes = srvConf[listenPort].EndpointRoutes
		case files:
			routes = srvConf[listenPort].FileRoutes
		case spa:
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

// parseDuration sets the target value if the given duration string is not empty.
func parseDuration(src string, target *time.Duration) error {
	d, err := time.ParseDuration(src)
	if src != "" && err != nil {
		return err
	}
	*target = d
	return nil
}

func parseBodyLimit(limit string) (int64, error) {
	const defaultReqBodyLimit = "64MiB"
	requestBodyLimit := defaultReqBodyLimit
	if limit != "" {
		requestBodyLimit = limit
	}
	return units.FromHumanSize(requestBodyLimit)
}

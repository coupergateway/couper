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
	hac "github.com/avenga/couper/handler/ac"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

const (
	api HandlerKind = iota
	endpoint
	files
	spa
)

var DefaultBackendConf = &config.Backend{
	ConnectTimeout: "10s",
	TTFBTimeout:    "60s",
	Timeout:        "300s",
}

type (
	Port                int
	Hosts               map[string]*MuxOptions
	Ports               map[Port]Hosts
	ServerConfiguration Ports
	HandlerKind         uint8
	endpointMap         map[*config.Endpoint]*config.API
	endpointHandler     map[*config.Endpoint]http.Handler
)

func (p Port) String() string {
	return strconv.Itoa(int(p))
}

func GetHostPort(hostPort string) (string, int, error) {
	var host string
	var port int

	h, p, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", -1, err
	}

	host = strings.TrimRight(h, ".")

	if p == "" || p == "*" {
		port = -1
	} else {
		port, err = strconv.Atoi(p)
		if err != nil {
			return "", -1, err
		}
	}

	return host, port, nil
}

// NewServerConfiguration sets http handler specific defaults and validates the given gateway configuration.
// Wire up all endpoints and maps them within the returned Server.
func NewServerConfiguration(
	conf *config.Couper, log *logrus.Entry, memStore *cache.MemoryStore,
) (ServerConfiguration, error) {
	// confCtx is created to evaluate request / response related configuration errors on start.
	noopReq := httptest.NewRequest(http.MethodGet, "https://couper.io", nil)
	noopResp := httptest.NewRecorder().Result()
	noopResp.Request = noopReq
	confCtx := conf.Context.WithClientRequest(noopReq).WithBeresps(noopResp).HCLContext()

	accessControls, err := configureAccessControls(conf, confCtx)
	if err != nil {
		return nil, err
	}

	var (
		serverConfiguration ServerConfiguration = make(ServerConfiguration)
		defaultPort         int                 = conf.Settings.DefaultPort
		endpointHandlers    endpointHandler     = make(endpointHandler)
		isHostsMandatory    bool                = len(conf.Servers) > 1
	)

	for _, srvConf := range conf.Servers {
		serverOptions, err := server.NewServerOptions(srvConf)
		if err != nil {
			return nil, err
		}

		if err := validateHosts(srvConf.Name, srvConf.Hosts, isHostsMandatory); err != nil {
			return nil, err
		}

		portsHosts, err := getPortsHostsList(srvConf.Hosts, defaultPort)
		if err != nil {
			return nil, err
		}

		for port, hosts := range portsHosts {
			for host, muxOpts := range hosts {
				if serverConfiguration[port] == nil {
					serverConfiguration[port] = make(Hosts)
				}

				if _, ok := serverConfiguration[port][host]; ok {
					return nil, fmt.Errorf("conflict: host %q already defined for port: %d", host, port)
				}

				serverConfiguration[port][host] = muxOpts
				serverConfiguration[port][host].ServerName = serverOptions.ServerName
				serverConfiguration[port][host].APIErrorTpls = serverOptions.APIErrTpl
				serverConfiguration[port][host].ServerErrorTpl = serverOptions.ServerErrTpl
				serverConfiguration[port][host].FilesErrorTpl = serverOptions.FileErrTpl
				serverConfiguration[port][host].APIBasePaths = serverOptions.APIBasePath
				serverConfiguration[port][host].FilesBasePath = serverOptions.FileBasePath
				serverConfiguration[port][host].SPABasePath = serverOptions.SPABasePath
			}
		}

		var spaHandler http.Handler
		if srvConf.Spa != nil {
			spaHandler, err = handler.NewSpa(srvConf.Spa.BootstrapFile, serverOptions)
			if err != nil {
				return nil, err
			}

			var h http.Handler

			corsOptions, err := middleware.NewCORSOptions(
				getCORS(srvConf.CORS, srvConf.Spa.CORS),
			)
			if err != nil {
				return nil, err
			} else if corsOptions != nil {
				h = middleware.NewCORSHandler(corsOptions, spaHandler)
			} else {
				h = spaHandler
			}

			spaHandler = configureProtectedHandler(accessControls, serverOptions.ServerErrTpl,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Spa.AccessControl, srvConf.Spa.DisableAccessControl), h)

			for _, spaPath := range srvConf.Spa.Paths {
				err = setRoutesFromHosts(serverConfiguration, portsHosts, path.Join(serverOptions.SPABasePath, spaPath), spaHandler, spa)
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

			var h http.Handler

			corsOptions, err := middleware.NewCORSOptions(
				getCORS(srvConf.CORS, srvConf.Files.CORS),
			)
			if err != nil {
				return nil, err
			} else if corsOptions != nil {
				h = middleware.NewCORSHandler(corsOptions, fileHandler)
			} else {
				h = fileHandler
			}

			protectedFileHandler := configureProtectedHandler(accessControls, serverOptions.FileErrTpl,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl), h)

			err = setRoutesFromHosts(serverConfiguration, portsHosts, serverOptions.FileBasePath, protectedFileHandler, files)
			if err != nil {
				return nil, err
			}
		}

		endpointsPatterns := make(map[string]bool)

		for endpointConf, parentAPI := range newEndpointMap(srvConf) {
			var basePath string
			var corsOptions *middleware.CORSOptions
			var errTpl *errors.Template

			if endpointConf.ErrorFile != "" {
				errTpl, err = errors.NewTemplateFromFile(endpointConf.ErrorFile)
				if err != nil {
					return nil, err
				}
			} else if parentAPI != nil {
				errTpl = serverOptions.APIErrTpl[parentAPI]
			} else {
				errTpl = serverOptions.ServerErrTpl
			}
			if parentAPI != nil {
				basePath = serverOptions.APIBasePath[parentAPI]

				cors, err := middleware.NewCORSOptions(
					getCORS(srvConf.CORS, parentAPI.CORS),
				)
				if err != nil {
					return nil, err
				}
				corsOptions = cors
			} else {
				basePath = serverOptions.SrvBasePath

				cors, err := middleware.NewCORSOptions(
					getCORS(nil, srvConf.CORS),
				)
				if err != nil {
					return nil, err
				}
				corsOptions = cors
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

				requests = append(requests, &producer.Request{
					Backend: backend,
					Context: requestConf.Remain,
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

			var h http.Handler
			if corsOptions != nil {
				h = middleware.NewCORSHandler(corsOptions, epHandler)
			} else {
				h = epHandler
			}

			setACHandlerFn(h)

			err = setRoutesFromHosts(serverConfiguration, portsHosts, pattern, endpointHandlers[endpointConf], kind)
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
		OpenAPI: openAPIopts,
	}
	backend := transport.NewBackend(backendCtx, tc, options, log)

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

func getCORS(parent, curr *config.CORS) *config.CORS {
	if curr == nil {
		return parent
	}

	if curr.Disable {
		return nil
	}

	return curr
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

		for _, saml := range conf.Definitions.SAML {
			name, err := validateACName(accessControls, saml.Name, "saml")
			if err != nil {
				return nil, err
			}

			s, err := ac.NewSAML2ACS(saml.IdpMetadataFile, name, saml.SpAcsUrl, saml.SpEntityId, saml.ArrayAttributes)
			if err != nil {
				return nil, fmt.Errorf("loading saml %q definition failed: %s", name, err)
			}

			accessControls[name] = ac.ValidateFunc(s.Validate)
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
		return hac.NewAccessControl(h, errTpl, acList...)
	}
	return h
}

func setRoutesFromHosts(
	srvConf ServerConfiguration, portsHosts Ports,
	path string, handler http.Handler, kind HandlerKind,
) error {
	path = utils.JoinPath("/", path)

	for port, hosts := range portsHosts {
		check := make(map[string]struct{})

		for host := range hosts {
			var routes map[string]http.Handler

			switch kind {
			case api:
				fallthrough
			case endpoint:
				routes = srvConf[port][host].EndpointRoutes
			case files:
				routes = srvConf[port][host].FileRoutes
			case spa:
				routes = srvConf[port][host].SPARoutes
			default:
				return fmt.Errorf("unknown route kind")
			}

			key := fmt.Sprintf("%d:%s:%s\n", port, host, path)
			if _, exist := check[key]; exist {
				return fmt.Errorf("duplicate route found on port %q: %q", port, path)
			}

			routes[path] = handler
			check[key] = struct{}{}
		}
	}

	return nil
}

func getPortsHostsList(hosts []string, defaultPort int) (Ports, error) {
	if len(hosts) == 0 {
		hosts = append(hosts, fmt.Sprintf("*:%d", defaultPort))
	}

	portsHosts := make(Ports)

	for _, hp := range hosts {
		if !strings.Contains(hp, ":") {
			hp += fmt.Sprintf(":%d", defaultPort)
		}

		host, port, err := GetHostPort(hp)
		if err != nil {
			return nil, err
		} else if port == -1 {
			port = defaultPort
		}

		if portsHosts[Port(port)] == nil {
			portsHosts[Port(port)] = make(Hosts)
		}

		portsHosts[Port(port)][host] = NewMuxOptions()
	}

	return portsHosts, nil
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

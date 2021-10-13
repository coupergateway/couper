//go:generate stringer -type=HandlerKind -output=./server_string.go

package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
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
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/oauth2"
	"github.com/avenga/couper/oauth2/oidc"
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
func NewServerConfiguration(conf *config.Couper, log *logrus.Entry, memStore *cache.MemoryStore) (ServerConfiguration, error) {
	// confCtx is created to evaluate request / response related configuration errors on start.
	noopReq, _ := http.NewRequest(http.MethodGet, "https://couper.io", nil)
	noopResp := httptest.NewRecorder().Result()
	noopResp.Request = noopReq
	evalContext := conf.Context.Value(request.ContextType).(*eval.Context)
	confCtx := evalContext.WithClientRequest(noopReq).WithBeresps(noopResp).HCLContext()

	oidcConfigs, ocErr := configureOidcConfigs(conf, confCtx, log, memStore)
	if ocErr != nil {
		return nil, ocErr
	}
	evalContext.WithOidcConfig(oidcConfigs)

	accessControls, acErr := configureAccessControls(conf, confCtx, log, memStore, oidcConfigs)
	if acErr != nil {
		return nil, acErr
	}

	var (
		serverConfiguration = make(ServerConfiguration)
		defaultPort         = conf.Settings.DefaultPort
		endpointHandlers    = make(endpointHandler)
		isHostsMandatory    = len(conf.Servers) > 1
	)

	for _, srvConf := range conf.Servers {
		serverOptions, err := server.NewServerOptions(srvConf, log)
		if err != nil {
			return nil, err
		}

		if err = validateHosts(srvConf.Name, srvConf.Hosts, isHostsMandatory); err != nil {
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
				serverConfiguration[port][host].ServerOptions = serverOptions
			}
		}

		var spaHandler http.Handler
		if srvConf.Spa != nil {
			spaHandler, err = handler.NewSpa(srvConf.Spa.BootstrapFile, serverOptions, []hcl.Body{srvConf.Spa.Remain, srvConf.Remain})
			if err != nil {
				return nil, err
			}

			corsOptions, cerr := middleware.NewCORSOptions(whichCORS(srvConf, srvConf.Spa))
			if cerr != nil {
				return nil, cerr
			}
			h := middleware.NewCORSHandler(corsOptions, spaHandler)

			spaHandler, err = configureProtectedHandler(accessControls, confCtx,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Spa.AccessControl, srvConf.Spa.DisableAccessControl),
				&protectedOptions{
					epOpts:       &handler.EndpointOptions{Error: serverOptions.ServerErrTpl},
					handler:      h,
					memStore:     memStore,
					proxyFromEnv: conf.Settings.NoProxyFromEnv,
					srvOpts:      serverOptions,
				}, nil, log)

			if err != nil {
				return nil, err
			}

			for _, spaPath := range srvConf.Spa.Paths {
				err = setRoutesFromHosts(serverConfiguration, portsHosts, path.Join(serverOptions.SPABasePath, spaPath), spaHandler, spa)
				if err != nil {
					return nil, err
				}
			}
		}

		if srvConf.Files != nil {
			fileHandler, ferr := handler.NewFile(srvConf.Files.DocumentRoot, serverOptions, []hcl.Body{srvConf.Files.Remain, srvConf.Remain})
			if ferr != nil {
				return nil, ferr
			}

			corsOptions, cerr := middleware.NewCORSOptions(whichCORS(srvConf, srvConf.Files))
			if cerr != nil {
				return nil, cerr
			}

			h := middleware.NewCORSHandler(corsOptions, fileHandler)

			protectedFileHandler, err := configureProtectedHandler(accessControls, confCtx,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl),
				&protectedOptions{
					epOpts:       &handler.EndpointOptions{Error: serverOptions.FilesErrTpl},
					handler:      h,
					memStore:     memStore,
					proxyFromEnv: conf.Settings.NoProxyFromEnv,
					srvOpts:      serverOptions,
				}, nil, log)

			if err != nil {
				return nil, err
			}

			err = setRoutesFromHosts(serverConfiguration, portsHosts, serverOptions.FilesBasePath, protectedFileHandler, files)
			if err != nil {
				return nil, err
			}
		}

		endpointPatterns := make(map[string]bool)
		endpointsMap, err := newEndpointMap(srvConf, serverOptions)
		if err != nil {
			return nil, err
		}

		for endpointConf, parentAPI := range endpointsMap {
			if endpointConf.Pattern == "" { // could happen for internally registered endpoints
				return nil, fmt.Errorf("endpoint path pattern required")
			}

			basePath := serverOptions.SrvBasePath
			if parentAPI != nil {
				basePath = serverOptions.APIBasePaths[parentAPI]
			}

			pattern := utils.JoinPath(basePath, endpointConf.Pattern)
			unique, cleanPattern := isUnique(endpointPatterns, pattern)
			if !unique {
				return nil, fmt.Errorf("%s: duplicate endpoint: '%s'", endpointConf.HCLBody().MissingItemRange().String(), pattern)
			}
			endpointPatterns[cleanPattern] = true

			corsOptions, err := middleware.NewCORSOptions(whichCORS(srvConf, parentAPI))
			if err != nil {
				return nil, err
			}
			epOpts, err := newEndpointOptions(
				confCtx, endpointConf, parentAPI, serverOptions,
				log, conf.Settings.NoProxyFromEnv, memStore,
			)
			if err != nil {
				return nil, err
			}

			modifier := []hcl.Body{srvConf.Remain}

			kind := endpoint
			if parentAPI != nil {
				kind = api

				modifier = []hcl.Body{parentAPI.Remain, srvConf.Remain}
			}
			epOpts.LogHandlerKind = kind.String()

			epHandler := handler.NewEndpoint(epOpts, log, modifier)
			protectedHandler := middleware.NewCORSHandler(corsOptions, epHandler)

			accessControl := newAC(srvConf, parentAPI)
			if parentAPI != nil && parentAPI.CatchAllEndpoint == endpointConf {
				protectedHandler = epOpts.Error.ServeError(errors.RouteNotFound)
			}
			scopeMaps := []map[string]string{}
			if parentAPI != nil {
				apiScopeMap, err := seetie.ValueToScopeMap(parentAPI.Scope)
				if err != nil {
					return nil, err
				}
				scopeMaps = append(scopeMaps, apiScopeMap)
			}
			endpointScopeMap, err := seetie.ValueToScopeMap(endpointConf.Scope)
			if err != nil {
				return nil, err
			}
			scopeMaps = append(scopeMaps, endpointScopeMap)
			scopeControl := ac.NewScopeControl(scopeMaps)
			endpointHandlers[endpointConf], err = configureProtectedHandler(accessControls, confCtx, accessControl,
				config.NewAccessControl(endpointConf.AccessControl, endpointConf.DisableAccessControl),
				&protectedOptions{
					epOpts:       epOpts,
					handler:      protectedHandler,
					memStore:     memStore,
					proxyFromEnv: conf.Settings.NoProxyFromEnv,
					srvOpts:      serverOptions,
				}, scopeControl, log)
			if err != nil {
				return nil, err
			}

			err = setRoutesFromHosts(serverConfiguration, portsHosts, pattern, endpointHandlers[endpointConf], kind)
			if err != nil {
				return nil, err
			}
		}
	}

	return serverConfiguration, nil
}

func newBackend(evalCtx *hcl.EvalContext, backendCtx hcl.Body, log *logrus.Entry,
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
		beConf.OAuth2 = &config.OAuth2ReqAuth{}

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

		// Set default value
		if beConf.OAuth2.Retries == nil {
			var one uint8 = 1
			beConf.OAuth2.Retries = &one
		}

		oauth2Client, err := oauth2.NewOAuth2CC(beConf.OAuth2, authBackend)
		if err != nil {
			return nil, err
		}

		return transport.NewOAuth2ReqAuth(beConf.OAuth2, memStore, oauth2Client, backend)
	}

	return backend, nil
}

func getBackendName(evalCtx *hcl.EvalContext, backendCtx hcl.Body) (string, error) {
	content, _, _ := backendCtx.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{
		{Name: "name"}},
	})
	if content != nil && len(content.Attributes) > 0 {

		if n, exist := content.Attributes["name"]; exist {
			v, err := eval.Value(evalCtx, n.Expr)
			if err != nil {
				return "", err
			}
			return v.AsString(), nil
		}
	}
	return "", nil
}

func whichCORS(parent *config.Server, this interface{}) *config.CORS {
	val := reflect.ValueOf(this)
	if val.IsZero() {
		return parent.CORS
	}

	corsValue := val.Elem().FieldByName("CORS")
	corsData, ok := corsValue.Interface().(*config.CORS)
	if !ok || corsData == nil {
		return parent.CORS
	}

	if corsData.Disable {
		return nil
	}

	return corsData
}

func configureOidcConfigs(conf *config.Couper, confCtx *hcl.EvalContext, log *logrus.Entry, memStore *cache.MemoryStore) (oidc.Configs, error) {
	oidcConfigs := make(oidc.Configs)
	if conf.Definitions != nil {
		for _, oidcConf := range conf.Definitions.OIDC {
			confErr := errors.Configuration.Label(oidcConf.Name)
			backend, err := newBackend(confCtx, oidcConf.Backend, log, conf.Settings.NoProxyFromEnv, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			oidcConfig, err := oidc.NewConfig(oidcConf, backend, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			oidcConfigs[oidcConf.Name] = oidcConfig
		}
	}

	return oidcConfigs, nil
}

func configureAccessControls(conf *config.Couper, confCtx *hcl.EvalContext, log *logrus.Entry,
	memStore *cache.MemoryStore, oidcConfigs oidc.Configs) (ACDefinitions, error) {

	accessControls := make(ACDefinitions)

	if conf.Definitions != nil {
		for _, baConf := range conf.Definitions.BasicAuth {
			confErr := errors.Configuration.Label(baConf.Name)
			basicAuth, err := ac.NewBasicAuth(baConf.Name, baConf.User, baConf.Pass, baConf.File)
			if err != nil {
				return nil, confErr.With(err)
			}

			if err = accessControls.Add(baConf.Name, basicAuth, baConf.ErrorHandler); err != nil {
				return nil, confErr.With(err)
			}
		}

		for _, jwtConf := range conf.Definitions.JWT {
			confErr := errors.Configuration.Label(jwtConf.Name)

			var jwt *ac.JWT
			if jwtConf.JWKsURL != "" {
				noProxy := conf.Settings.NoProxyFromEnv
				jwks, err := configureJWKS(jwtConf, conf, confCtx, log, noProxy, memStore)
				if err != nil {
					return nil, confErr.With(err)
				}

				jwt, err = ac.NewJWTFromJWKS(&ac.JWTOptions{
					Claims:         jwtConf.Claims,
					ClaimsRequired: jwtConf.ClaimsRequired,
					Name:           jwtConf.Name,
					RoleClaim:      jwtConf.RoleClaim,
					RoleMap:        jwtConf.RoleMap,
					ScopeClaim:     jwtConf.ScopeClaim,
					Source:         ac.NewJWTSource(jwtConf.Cookie, jwtConf.Header),
					JWKS:           jwks,
				})
				if err != nil {
					return nil, confErr.With(err)
				}
			} else {
				key, err := reader.ReadFromAttrFile("jwt key", jwtConf.Key, jwtConf.KeyFile)
				if err != nil {
					return nil, confErr.With(err)
				}

				jwt, err = ac.NewJWT(&ac.JWTOptions{
					Algorithm:      jwtConf.SignatureAlgorithm,
					Claims:         jwtConf.Claims,
					ClaimsRequired: jwtConf.ClaimsRequired,
					Key:            key,
					Name:           jwtConf.Name,
					RoleClaim:      jwtConf.RoleClaim,
					RoleMap:        jwtConf.RoleMap,
					ScopeClaim:     jwtConf.ScopeClaim,
					Source:         ac.NewJWTSource(jwtConf.Cookie, jwtConf.Header),
				})

				if err != nil {
					return nil, confErr.With(err)
				}
			}

			if err := accessControls.Add(jwtConf.Name, jwt, jwtConf.ErrorHandler); err != nil {
				return nil, confErr.With(err)
			}
		}

		for _, saml := range conf.Definitions.SAML {
			confErr := errors.Configuration.Label(saml.Name)
			metadata, err := reader.ReadFromFile("saml2 idp_metadata_file", saml.IdpMetadataFile)
			if err != nil {
				return nil, confErr.With(err)
			}

			s, err := ac.NewSAML2ACS(metadata, saml.Name, saml.SpAcsUrl, saml.SpEntityId, saml.ArrayAttributes)
			if err != nil {
				return nil, confErr.With(err)
			}

			if err = accessControls.Add(saml.Name, s, saml.ErrorHandler); err != nil {
				return nil, confErr.With(err)
			}
		}

		for _, oauth2Conf := range conf.Definitions.OAuth2AC {
			confErr := errors.Configuration.Label(oauth2Conf.Name)
			backend, err := newBackend(confCtx, oauth2Conf.Backend, log, conf.Settings.NoProxyFromEnv, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			oauth2Client, err := oauth2.NewOAuth2AC(oauth2Conf, oauth2Conf, backend)
			if err != nil {
				return nil, confErr.With(err)
			}

			oa, err := ac.NewOAuth2Callback(oauth2Client)
			if err != nil {
				return nil, confErr.With(err)
			}

			if err = accessControls.Add(oauth2Conf.Name, oa, oauth2Conf.ErrorHandler); err != nil {
				return nil, confErr.With(err)
			}
		}

		for _, oidcConf := range conf.Definitions.OIDC {
			confErr := errors.Configuration.Label(oidcConf.Name)
			oidcConfig := oidcConfigs[oidcConf.Name]
			oidcClient, err := oauth2.NewOidc(oidcConfig)
			if err != nil {
				return nil, confErr.With(err)
			}

			if oidcConfig.VerifierMethod != "" &&
				oidcConfig.VerifierMethod != config.CcmS256 &&
				oidcConfig.VerifierMethod != "nonce" {
				return nil, errors.Configuration.
					Label(oidcConf.Name).
					Messagef("verifier_method %s not supported", oidcConfig.VerifierMethod)
			}

			oa, err := ac.NewOAuth2Callback(oidcClient)
			if err != nil {
				return nil, confErr.With(err)
			}

			if err = accessControls.Add(oidcConf.Name, oa, oidcConf.ErrorHandler); err != nil {
				return nil, confErr.With(err)
			}
		}
	}

	return accessControls, nil
}

func configureJWKS(jwtConf *config.JWT, conf *config.Couper, confContext *hcl.EvalContext, log *logrus.Entry, ignoreProxyEnv bool, memStore *cache.MemoryStore) (*ac.JWKS, error) {
	var backend http.RoundTripper

	if jwtConf.JWKSBackendBody != nil {
		b, err := newBackend(confContext, jwtConf.JWKSBackendBody, log, ignoreProxyEnv, memStore)
		if err != nil {
			return nil, err
		}
		backend = b
	}

	evalContext := conf.Context.Value(request.ContextType).(context.Context)
	jwks, err := ac.NewJWKS(jwtConf.JWKsURL, jwtConf.JWKsTTL, backend, evalContext)
	if err != nil {
		return nil, err
	}

	return jwks, nil
}

type protectedOptions struct {
	epOpts       *handler.EndpointOptions
	handler      http.Handler
	proxyFromEnv bool
	memStore     *cache.MemoryStore
	srvOpts      *server.Options
}

func configureProtectedHandler(m ACDefinitions, ctx *hcl.EvalContext, parentAC, handlerAC config.AccessControl,
	opts *protectedOptions, scopeControl *ac.ScopeControl, log *logrus.Entry) (http.Handler, error) {
	var list ac.List
	for _, acName := range parentAC.Merge(handlerAC).List() {
		if e := m.MustExist(acName); e != nil {
			return nil, e
		}
		eh, err := newErrorHandler(ctx, opts, log, m, acName)
		if err != nil {
			return nil, err
		}
		list = append(
			list,
			ac.NewItem(acName, m[acName].Control, eh),
		)
	}
	if scopeControl != nil {
		// TODO properly create error handler
		list = append(list, ac.NewItem("scope", scopeControl, handler.NewErrorHandler(nil, opts.epOpts.Error)))
	}

	if len(list) > 0 {
		return handler.NewAccessControl(opts.handler, list), nil
	}
	return opts.handler, nil
}

func newErrorHandler(ctx *hcl.EvalContext, opts *protectedOptions, log *logrus.Entry,
	defs ACDefinitions, references ...string) (http.Handler, error) {
	kindsHandler := map[string]http.Handler{}
	for _, ref := range references {
		for _, h := range defs[ref].ErrorHandler {
			for _, k := range h.Kinds {
				if _, exist := kindsHandler[k]; exist {
					log.Fatal("error type handler exists already: " + k)
				}

				contextBody := h.HCLBody()

				epConf := &config.Endpoint{
					Remain:    contextBody,
					Proxies:   h.Proxies,
					ErrorFile: h.ErrorFile,
					Requests:  h.Requests,
					Response:  h.Response,
				}

				emptyBody := hcl.EmptyBody()
				if epConf.Response == nil { // Set dummy resp to skip related requirement checks, allowed for error_handler.
					epConf.Response = &config.Response{Remain: emptyBody}
				}

				epOpts, err := newEndpointOptions(ctx, epConf, nil, opts.srvOpts, log, opts.proxyFromEnv, opts.memStore)
				if err != nil {
					return nil, err
				}
				if epOpts.Error == nil || h.ErrorFile == "" {
					epOpts.Error = opts.epOpts.Error
				}

				epOpts.Error = epOpts.Error.WithContextFunc(func(rw http.ResponseWriter, r *http.Request) {
					beresp := &http.Response{Header: rw.Header()}
					_ = eval.ApplyResponseContext(r.Context(), contextBody, beresp)
				})

				if epOpts.Response != nil && reflect.DeepEqual(epOpts.Response.Context, emptyBody) {
					epOpts.Response = nil
				}

				epOpts.LogHandlerKind = "error_" + k
				kindsHandler[k] = handler.NewEndpoint(epOpts, log, nil)
			}
		}
	}
	return handler.NewErrorHandler(kindsHandler, opts.epOpts.Error), nil
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

func newAC(srvConf *config.Server, api *config.API) config.AccessControl {
	accessControl := config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl)

	if api != nil {
		accessControl = accessControl.Merge(config.NewAccessControl(api.AccessControl, api.DisableAccessControl))
	}

	return accessControl
}

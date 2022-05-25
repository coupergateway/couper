//go:generate stringer -type=HandlerKind -output=./server_string.go

package runtime

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload/collect"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/middleware"
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
	evalContext := conf.Context.Value(request.ContextType).(*eval.Context) // usually environment vars
	confCtx := evalContext.HCLContext()

	oidcConfigs, ocErr := configureOidcConfigs(conf, confCtx, log, memStore)
	if ocErr != nil {
		return nil, ocErr
	}
	conf.Context = evalContext.WithOidcConfig(oidcConfigs)

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

	// Populate defined backends first...
	if conf.Definitions != nil {
		for _, backend := range conf.Definitions.Backend {
			_, err := NewBackend(confCtx, backend.HCLBody(), log, conf.Settings, memStore)
			if err != nil {
				return nil, err
			}
		}
	}

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

		serverBodies := bodiesWithACBodies(conf.Definitions, srvConf.AccessControl, srvConf.DisableAccessControl)
		serverBodies = append(serverBodies, srvConf.Remain)

		var spaHandler http.Handler
		for _, spaConf := range srvConf.SPAs {
			spaHandler, err = handler.NewSpa(spaConf.BootstrapFile, serverOptions, []hcl.Body{spaConf.Remain, srvConf.Remain})
			if err != nil {
				return nil, err
			}

			epOpts := &handler.EndpointOptions{ErrorTemplate: serverOptions.ServerErrTpl}
			notAllowedMethodsHandler := epOpts.ErrorTemplate.WithError(errors.MethodNotAllowed)
			allowedMethodsHandler := middleware.NewAllowedMethodsHandler(nil, middleware.DefaultFileSpaAllowedMethods, spaHandler, notAllowedMethodsHandler)
			spaHandler = allowedMethodsHandler

			spaHandler, err = configureProtectedHandler(accessControls, confCtx,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(spaConf.AccessControl, spaConf.DisableAccessControl),
				&protectedOptions{
					epOpts:   epOpts,
					handler:  spaHandler,
					memStore: memStore,
					settings: conf.Settings,
					srvOpts:  serverOptions,
				}, log)
			if err != nil {
				return nil, err
			}

			corsOptions, cerr := middleware.NewCORSOptions(whichCORS(srvConf, spaConf), allowedMethodsHandler.MethodAllowed)
			if cerr != nil {
				return nil, cerr
			}

			spaHandler = middleware.NewCORSHandler(corsOptions, spaHandler)

			spaBodies := bodiesWithACBodies(conf.Definitions, spaConf.AccessControl, spaConf.DisableAccessControl)
			spaHandler = middleware.NewCustomLogsHandler(
				append(serverBodies, append(spaBodies, spaConf.Remain)...), spaHandler, "",
			)

			for _, spaPath := range spaConf.Paths {
				err = setRoutesFromHosts(serverConfiguration, portsHosts,
					path.Join(serverOptions.SrvBasePath, spaConf.BasePath, spaPath), spaHandler, spa)
				if err != nil {
					sbody := spaConf.HCLBody().(*hclsyntax.Body)
					return nil, hcl.Diagnostics{&hcl.Diagnostic{
						Subject: &sbody.Attributes["paths"].SrcRange,
						Summary: err.Error(),
					}}
				}
			}
		}

		if srvConf.Files != nil {
			var fileHandler http.Handler
			fileHandler, err = handler.NewFile(srvConf.Files.DocumentRoot, serverOptions, []hcl.Body{srvConf.Files.Remain, srvConf.Remain})
			if err != nil {
				return nil, err
			}

			epOpts := &handler.EndpointOptions{ErrorTemplate: serverOptions.FilesErrTpl}
			notAllowedMethodsHandler := epOpts.ErrorTemplate.WithError(errors.MethodNotAllowed)
			allowedMethodsHandler := middleware.NewAllowedMethodsHandler(nil, middleware.DefaultFileSpaAllowedMethods, fileHandler, notAllowedMethodsHandler)
			fileHandler = allowedMethodsHandler

			fileHandler, err = configureProtectedHandler(accessControls, confCtx,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl),
				&protectedOptions{
					epOpts:   epOpts,
					handler:  fileHandler,
					memStore: memStore,
					settings: conf.Settings,
					srvOpts:  serverOptions,
				}, log)
			if err != nil {
				return nil, err
			}

			corsOptions, cerr := middleware.NewCORSOptions(whichCORS(srvConf, srvConf.Files), allowedMethodsHandler.MethodAllowed)
			if cerr != nil {
				return nil, cerr
			}

			fileHandler = middleware.NewCORSHandler(corsOptions, fileHandler)

			fileBodies := bodiesWithACBodies(conf.Definitions, srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl)
			fileHandler = middleware.NewCustomLogsHandler(
				append(serverBodies, append(fileBodies, srvConf.Files.Remain)...), fileHandler, "",
			)

			err = setRoutesFromHosts(serverConfiguration, portsHosts, serverOptions.FilesBasePath, fileHandler, files)
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

			epOpts, err := newEndpointOptions(confCtx, endpointConf, parentAPI, serverOptions,
				log, conf.Settings, memStore)
			if err != nil {
				return nil, err
			}

			// Evaluate access-control related buffer options.
			acBodies := bodiesWithACBodies(conf.Definitions,
				newAC(srvConf, parentAPI).
					Merge(config.
						NewAccessControl(endpointConf.AccessControl, endpointConf.DisableAccessControl)).List(), nil)
			epOpts.BufferOpts |= eval.MustBuffer(acBodies...)

			errorHandlerDefinitions := ACDefinitions{ // misuse of definitions obj for now
				"endpoint": &AccessControl{ErrorHandler: endpointConf.ErrorHandler},
			}

			modifier := []hcl.Body{srvConf.Remain}

			kind := endpoint
			if parentAPI != nil {
				kind = api

				modifier = []hcl.Body{parentAPI.Remain, srvConf.Remain}

				errorHandlerDefinitions["api"] = &AccessControl{ErrorHandler: parentAPI.ErrorHandler}
			}
			epOpts.LogHandlerKind = kind.String()

			var epHandler, protectedHandler http.Handler
			if parentAPI != nil && parentAPI.CatchAllEndpoint == endpointConf {
				protectedHandler = epOpts.ErrorTemplate.WithError(errors.RouteNotFound)
			} else {
				epErrorHandler, err := newErrorHandler(confCtx, &protectedOptions{
					epOpts:   epOpts,
					memStore: memStore,
					settings: conf.Settings,
					srvOpts:  serverOptions,
				}, log, errorHandlerDefinitions, "api", "endpoint") // sequence of ref is important: api, endpoint (endpoint error_handler overrides api error_handler)
				if err != nil {
					return nil, err
				}
				if epErrorHandler != nil {
					epOpts.ErrorHandler = epErrorHandler
				}
				epHandler = handler.NewEndpoint(epOpts, log, modifier)

				requiredPermissionExpr := endpointConf.RequiredPermission
				if requiredPermissionExpr == nil && parentAPI != nil {
					// if required permission in endpoint {} not defined, try required permission in parent api {}
					requiredPermissionExpr = parentAPI.RequiredPermission
				}
				if requiredPermissionExpr == nil {
					protectedHandler = epHandler
				} else {
					permissionsControl := ac.NewPermissionsControl(requiredPermissionExpr)
					permissionsErrorHandler, err := newErrorHandler(confCtx, &protectedOptions{
						epOpts:   epOpts,
						memStore: memStore,
						settings: conf.Settings,
						srvOpts:  serverOptions,
					}, log, errorHandlerDefinitions, "api", "endpoint") // sequence of ref is important: api, endpoint (endpoint error_handler overrides api error_handler)
					if err != nil {
						return nil, err
					}

					protectedHandler = middleware.NewErrorHandler(permissionsControl.Validate, permissionsErrorHandler)(epHandler)
				}
			}

			accessControl := newAC(srvConf, parentAPI)

			allowedMethods := endpointConf.AllowedMethods
			if allowedMethods == nil && parentAPI != nil {
				// if allowed_methods in endpoint {} not defined, try allowed_methods in parent api {}
				allowedMethods = parentAPI.AllowedMethods
			}
			notAllowedMethodsHandler := epOpts.ErrorTemplate.WithError(errors.MethodNotAllowed)
			allowedMethodsHandler := middleware.NewAllowedMethodsHandler(allowedMethods, middleware.DefaultEndpointAllowedMethods, protectedHandler, notAllowedMethodsHandler)
			protectedHandler = allowedMethodsHandler

			epHandler, err = configureProtectedHandler(accessControls, confCtx, accessControl,
				config.NewAccessControl(endpointConf.AccessControl, endpointConf.DisableAccessControl),
				&protectedOptions{
					epOpts:   epOpts,
					handler:  protectedHandler,
					memStore: memStore,
					settings: conf.Settings,
					srvOpts:  serverOptions,
				}, log)
			if err != nil {
				return nil, err
			}

			corsOptions, err := middleware.NewCORSOptions(whichCORS(srvConf, parentAPI), allowedMethodsHandler.MethodAllowed)
			if err != nil {
				return nil, err
			}

			epHandler = middleware.NewCORSHandler(corsOptions, epHandler)

			bodies := serverBodies
			if parentAPI != nil {
				apiBodies := bodiesWithACBodies(conf.Definitions, parentAPI.AccessControl, parentAPI.DisableAccessControl)
				bodies = append(bodies, append(apiBodies, parentAPI.Remain)...)
			}
			bodies = append(bodies, bodiesWithACBodies(conf.Definitions, endpointConf.AccessControl, endpointConf.DisableAccessControl)...)
			epHandler = middleware.NewCustomLogsHandler(
				append(bodies, endpointConf.Remain), epHandler, epOpts.LogHandlerKind,
			)

			endpointHandlers[endpointConf] = epHandler
			err = setRoutesFromHosts(serverConfiguration, portsHosts, pattern, endpointHandlers[endpointConf], kind)
			if err != nil {
				return nil, err
			}
		}
	}

	return serverConfiguration, nil
}

func bodiesWithACBodies(defs *config.Definitions, ac, dac []string) []hcl.Body {
	var bodies []hcl.Body

	allAccessControls := collect.ErrorHandlerSetters(defs)

	for _, ehs := range allAccessControls {
		acConf, ok := ehs.(config.Body)
		if !ok {
			continue
		}

		t := reflect.ValueOf(acConf)
		elem := t

		if t.Kind() == reflect.Ptr {
			elem = t.Elem()
		}

		nameValue := elem.FieldByName("Name")
		if !nameValue.CanInterface() {
			continue
		}

		for _, name := range config.NewAccessControl(ac, dac).List() {
			if value, vk := nameValue.Interface().(string); vk && value == name {
				bodies = append(bodies, acConf.HCLBody())
			}
		}
	}

	return bodies
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
			backends := map[string]http.RoundTripper{}
			for k, backendBody := range oidcConf.Backends {
				var err error
				backends[k], err = NewBackend(confCtx, backendBody, log, conf.Settings, memStore)
				if err != nil {
					return nil, confErr.With(err)
				}
			}

			oidcConfig, err := oidc.NewConfig(oidcConf, backends)
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

			jwt, err := newJWT(jwtConf, conf, confCtx, log, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			if err = accessControls.Add(jwtConf.Name, jwt, jwtConf.ErrorHandler); err != nil {
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
			backend, err := NewBackend(confCtx, oauth2Conf.Backend, log, conf.Settings, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			oauth2Client, err := oauth2.NewAuthCodeClient(oauth2Conf, oauth2Conf, backend)
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
			oidcClient, err := oauth2.NewOidcClient(oidcConfig)
			if err != nil {
				return nil, confErr.With(err)
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

func newJWT(jwtConf *config.JWT, conf *config.Couper, confCtx *hcl.EvalContext,
	log *logrus.Entry, memStore *cache.MemoryStore) (*ac.JWT, error) {
	jwtOptions := &ac.JWTOptions{
		Claims:                jwtConf.Claims,
		ClaimsRequired:        jwtConf.ClaimsRequired,
		DisablePrivateCaching: jwtConf.DisablePrivateCaching,
		Name:                  jwtConf.Name,
		RolesClaim:            jwtConf.RolesClaim,
		RolesMap:              jwtConf.RolesMap,
		PermissionsClaim:      jwtConf.PermissionsClaim,
		PermissionsMap:        jwtConf.PermissionsMap,
		Source:                ac.NewJWTSource(jwtConf.Cookie, jwtConf.Header, jwtConf.TokenValue),
	}
	var (
		jwt *ac.JWT
		err error
	)
	if jwtConf.JWKsURL != "" {
		jwks, jerr := configureJWKS(jwtConf, confCtx, log, conf, memStore)
		if jerr != nil {
			return nil, jerr
		}

		jwtOptions.JWKS = jwks
		jwt, err = ac.NewJWTFromJWKS(jwtOptions)
	} else {
		key, kerr := reader.ReadFromAttrFile("jwt key", jwtConf.Key, jwtConf.KeyFile)
		if kerr != nil {
			return nil, kerr
		}

		jwtOptions.Algorithm = jwtConf.SignatureAlgorithm
		jwtOptions.Key = key
		jwt, err = ac.NewJWT(jwtOptions)
	}
	if err != nil {
		return nil, err
	}

	return jwt, nil
}

func configureJWKS(jwtConf *config.JWT, confContext *hcl.EvalContext, log *logrus.Entry, conf *config.Couper, memStore *cache.MemoryStore) (*jwk.JWKS, error) {
	var backend http.RoundTripper

	if jwtConf.Backends != nil {
		backendBody, ok := jwtConf.Backends["backend"]
		if ok {
			b, err := NewBackend(confContext, backendBody, log, conf.Settings, memStore)
			if err != nil {
				return nil, err
			}
			backend = b
		}
	}

	return jwk.NewJWKS(jwtConf.JWKsURL, jwtConf.JWKsTTL, jwtConf.JWKsMaxStale, backend)
}

type protectedOptions struct {
	epOpts   *handler.EndpointOptions
	handler  http.Handler
	memStore *cache.MemoryStore
	settings *config.Settings
	srvOpts  *server.Options
}

func configureProtectedHandler(m ACDefinitions, ctx *hcl.EvalContext, parentAC, handlerAC config.AccessControl,
	opts *protectedOptions, log *logrus.Entry) (http.Handler, error) {
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

	if len(list) > 0 {
		return handler.NewAccessControl(opts.handler, list), nil
	}
	return opts.handler, nil
}

func setRoutesFromHosts(
	srvConf ServerConfiguration, portsHosts Ports,
	path string, handler http.Handler, kind HandlerKind,
) error {
	path = utils.JoinPath("/", path)

	for port, hosts := range portsHosts {
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

			if _, exist := routes[path]; exist {
				return fmt.Errorf("duplicate route found on port %d: %s", port, path)
			}

			routes[path] = handler
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

//go:generate stringer -type=HandlerKind -output=./server_string.go

package runtime

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/accesscontrol/jwk"
	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload/collect"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/definitions"
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
	conf.Context = evalContext.
		WithMemStore(memStore).
		WithOidcConfig(oidcConfigs)

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
			_, err := NewBackend(confCtx, backend.HCLBody(), log, conf, memStore)
			if err != nil {
				return nil, err
			}
		}

		jobs := make(definitions.Jobs, 0)
		for _, job := range conf.Definitions.Job {
			serverOptions := &server.Options{
				ServerErrTpl: errors.DefaultJSON,
			}

			endpointOptions, err := NewEndpointOptions(confCtx, job.Endpoint, nil, serverOptions, log, conf, memStore)
			if err != nil {
				if diags, ok := err.(hcl.Diagnostics); ok {
					derr := diags[0]
					derr.Summary = strings.Replace(derr.Summary, "endpoint:", "beta_job:", 1)
					if strings.Contains(derr.Summary, "requires at least") {
						derr.Summary = strings.Join(append([]string{},
							strings.SplitAfter(derr.Summary, `" `)[0], "requires at least one request block"), "")
					}
					return nil, derr
				}
				return nil, err
			}

			endpointOptions.IsJob = true
			epHandler := handler.NewEndpoint(endpointOptions, log, nil)

			j := definitions.NewJob(job, epHandler, conf.Settings)
			jobs = append(jobs, j)
		}

		// do not start go-routine on config check (-watch)
		if _, exist := conf.Context.Value(request.ConfigDryRun).(bool); !exist {
			jobs.Run(conf.Context, log)
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
		var bootstrapFiles []string
		spaMountPathSeen := make(map[string]struct{})
		for _, spaConf := range srvConf.SPAs {
			spaHandler, err = handler.NewSpa(evalContext.HCLContext(), spaConf, serverOptions, []hcl.Body{spaConf.Remain, srvConf.Remain})
			if err != nil {
				return nil, err
			}

			for _, mountPath := range spaConf.Paths {
				mp := strings.Replace(mountPath, "**", "", 1)
				dir := filepath.Dir(spaConf.BootstrapFile)
				if !strings.HasSuffix(dir, mp) {
					dir = filepath.Join(dir, mp)
				}
				bfp := filepath.Join(dir, filepath.Base(spaConf.BootstrapFile))
				if _, seen := spaMountPathSeen[bfp]; !seen {
					bootstrapFiles = append(bootstrapFiles, bfp)
					spaMountPathSeen[bfp] = struct{}{}
				}
			}

			epOpts := &handler.EndpointOptions{ErrorTemplate: serverOptions.ServerErrTpl}
			notAllowedMethodsHandler := epOpts.ErrorTemplate.WithError(errors.MethodNotAllowed)
			allowedMethodsHandler := middleware.NewAllowedMethodsHandler(nil, middleware.DefaultFileSpaAllowedMethods, spaHandler, notAllowedMethodsHandler)
			spaHandler = allowedMethodsHandler

			spaHandler, err = configureProtectedHandler(accessControls, conf, confCtx,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(spaConf.AccessControl, spaConf.DisableAccessControl),
				&protectedOptions{
					epOpts:   epOpts,
					handler:  spaHandler,
					memStore: memStore,
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

			for _, p := range spaConf.Paths {
				spaPath := path.Join(serverOptions.SrvBasePath, spaConf.BasePath, p)
				err = setRoutesFromHosts(serverConfiguration, portsHosts, spaPath, spaHandler, spa)
				if err != nil {
					sbody := spaConf.HCLBody()
					return nil, hcl.Diagnostics{&hcl.Diagnostic{
						Subject: &sbody.Attributes["paths"].SrcRange,
						Summary: err.Error(),
					}}
				}
			}
		}

		var fileHandler http.Handler
		for i, filesConf := range srvConf.Files {
			fileHandler, err = handler.NewFile(
				filesConf.DocumentRoot,
				serverOptions.FilesBasePaths[i],
				handler.NewPreferSpaFn(bootstrapFiles, filesConf.DocumentRoot),
				serverOptions.FilesErrTpls[i],
				serverOptions,
				[]hcl.Body{filesConf.Remain, srvConf.Remain},
			)
			if err != nil {
				return nil, err
			}

			epOpts := &handler.EndpointOptions{ErrorTemplate: serverOptions.FilesErrTpls[i]}
			notAllowedMethodsHandler := epOpts.ErrorTemplate.WithError(errors.MethodNotAllowed)
			allowedMethodsHandler := middleware.NewAllowedMethodsHandler(nil, middleware.DefaultFileSpaAllowedMethods, fileHandler, notAllowedMethodsHandler)
			fileHandler = allowedMethodsHandler

			fileHandler, err = configureProtectedHandler(accessControls, conf, confCtx,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(filesConf.AccessControl, filesConf.DisableAccessControl),
				&protectedOptions{
					epOpts:   epOpts,
					handler:  fileHandler,
					memStore: memStore,
					srvOpts:  serverOptions,
				}, log)
			if err != nil {
				return nil, err
			}

			corsOptions, cerr := middleware.NewCORSOptions(whichCORS(srvConf, filesConf), allowedMethodsHandler.MethodAllowed)
			if cerr != nil {
				return nil, cerr
			}

			fileHandler = middleware.NewCORSHandler(corsOptions, fileHandler)

			fileBodies := bodiesWithACBodies(conf.Definitions, filesConf.AccessControl, filesConf.DisableAccessControl)
			fileHandler = middleware.NewCustomLogsHandler(
				append(serverBodies, append(fileBodies, filesConf.Remain)...), fileHandler, "",
			)

			err = setRoutesFromHosts(serverConfiguration, portsHosts, serverOptions.FilesBasePaths[i], fileHandler, files)
			if err != nil {
				return nil, err
			}
		}

		endpointsMap, err := newEndpointMap(srvConf, serverOptions)
		if err != nil {
			return nil, err
		}

		for endpointConf, parentAPI := range endpointsMap {
			if endpointConf.Pattern == "" { // could happen for internally registered endpoints
				return nil, fmt.Errorf("endpoint path pattern required")
			}

			epOpts, err := NewEndpointOptions(confCtx, endpointConf, parentAPI, serverOptions,
				log, conf, memStore)
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
				epErrorHandler, ehBufferOption, err := newErrorHandler(confCtx, conf, &protectedOptions{
					epOpts:   epOpts,
					memStore: memStore,
					srvOpts:  serverOptions,
				}, log, errorHandlerDefinitions, "api", "endpoint") // sequence of ref is important: api, endpoint (endpoint error_handler overrides api error_handler)
				if err != nil {
					return nil, err
				}
				if epErrorHandler != nil {
					epOpts.ErrorHandler = epErrorHandler
					epOpts.BufferOpts |= ehBufferOption
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
					permissionsErrorHandler, _, err := newErrorHandler(confCtx, conf, &protectedOptions{
						epOpts:   epOpts,
						memStore: memStore,
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

			epHandler, err = configureProtectedHandler(accessControls, conf, confCtx, accessControl,
				config.NewAccessControl(endpointConf.AccessControl, endpointConf.DisableAccessControl),
				&protectedOptions{
					epOpts:   epOpts,
					handler:  protectedHandler,
					memStore: memStore,
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

			basePath := serverOptions.SrvBasePath
			if parentAPI != nil {
				basePath = serverOptions.APIBasePaths[parentAPI]
			}

			pattern := utils.JoinOpenAPIPath(basePath, endpointConf.Pattern)

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
				backends[k], err = NewBackend(confCtx, backendBody, log, conf, memStore)
				if err != nil {
					return nil, confErr.With(err)
				}
			}

			oidcConfig, err := oidc.NewConfig(conf.Context, oidcConf, backends)
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

			accessControls.Add(baConf.Name, basicAuth, baConf.ErrorHandler)
		}

		for _, jwtConf := range conf.Definitions.JWT {
			confErr := errors.Configuration.Label(jwtConf.Name)

			jwt, err := newJWT(jwtConf, conf, confCtx, log, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			accessControls.Add(jwtConf.Name, jwt, jwtConf.ErrorHandler)
		}

		for _, saml := range conf.Definitions.SAML {
			confErr := errors.Configuration.Label(saml.Name)
			s, err := ac.NewSAML2ACS(saml.MetadataBytes, saml.Name, saml.SpAcsURL, saml.SpEntityID, saml.ArrayAttributes)
			if err != nil {
				return nil, confErr.With(err)
			}

			accessControls.Add(saml.Name, s, saml.ErrorHandler)
		}

		for _, oauth2Conf := range conf.Definitions.OAuth2AC {
			confErr := errors.Configuration.Label(oauth2Conf.Name)
			backend, err := NewBackend(confCtx, oauth2Conf.Backend, log, conf, memStore)
			if err != nil {
				return nil, confErr.With(err)
			}

			oauth2Client, err := oauth2.NewAuthCodeClient(confCtx, oauth2Conf, oauth2Conf, backend)
			if err != nil {
				return nil, confErr.With(err)
			}

			oa := ac.NewOAuth2Callback(oauth2Client, oauth2Conf.Name)

			accessControls.Add(oauth2Conf.Name, oa, oauth2Conf.ErrorHandler)
		}

		for _, oidcConf := range conf.Definitions.OIDC {
			confErr := errors.Configuration.Label(oidcConf.Name)
			oidcConfig := oidcConfigs[oidcConf.Name]
			oidcClient, err := oauth2.NewOidcClient(confCtx, oidcConfig)
			if err != nil {
				return nil, confErr.With(err)
			}

			oa := ac.NewOAuth2Callback(oidcClient, oidcConf.Name)

			accessControls.Add(oidcConf.Name, oa, oidcConf.ErrorHandler)
		}
	}

	return accessControls, nil
}

func newJWT(jwtConf *config.JWT, conf *config.Couper, confCtx *hcl.EvalContext,
	log *logrus.Entry, memStore *cache.MemoryStore) (*ac.JWT, error) {
	var (
		jwt                      *ac.JWT
		err                      error
		rolesMap, permissionsMap map[string][]string
	)
	rolesMap, err = reader.ReadFromAttrFileJSONObjectOptional("jwt roles map", jwtConf.RolesMap, jwtConf.RolesMapFile)
	if err != nil {
		return nil, err
	}
	permissionsMap, err = reader.ReadFromAttrFileJSONObjectOptional("jwt permissions map", jwtConf.PermissionsMap, jwtConf.PermissionsMapFile)
	if err != nil {
		return nil, err
	}
	jwtOptions := &ac.JWTOptions{
		Claims:                jwtConf.Claims,
		ClaimsRequired:        jwtConf.ClaimsRequired,
		DisablePrivateCaching: jwtConf.DisablePrivateCaching,
		Name:                  jwtConf.Name,
		RolesClaim:            jwtConf.RolesClaim,
		RolesMap:              rolesMap,
		PermissionsClaim:      jwtConf.PermissionsClaim,
		PermissionsMap:        permissionsMap,
		Source:                ac.NewJWTSource(jwtConf.Cookie, jwtConf.Header, jwtConf.TokenValue),
	}
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
	backend, err := NewBackend(confContext, jwtConf.Backend, log, conf, memStore)
	if err != nil {
		return nil, err
	}

	return jwk.NewJWKS(conf.Context, jwtConf.JWKsURL, jwtConf.JWKsTTL, jwtConf.JWKsMaxStale, backend)
}

type protectedOptions struct {
	epOpts   *handler.EndpointOptions
	handler  http.Handler
	memStore *cache.MemoryStore
	srvOpts  *server.Options
}

func configureProtectedHandler(m ACDefinitions, conf *config.Couper, ctx *hcl.EvalContext, parentAC, handlerAC config.AccessControl,
	opts *protectedOptions, log *logrus.Entry) (http.Handler, error) {
	var list ac.List
	for _, acName := range parentAC.Merge(handlerAC).List() {
		eh, _, err := newErrorHandler(ctx, conf, opts, log, m, acName)
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

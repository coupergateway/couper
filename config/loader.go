package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"

	ac "go.avenga.cloud/couper/gateway/access_control"
	"go.avenga.cloud/couper/gateway/handler"
)

var errorMissingBackend = errors.New("no backend attribute reference or block")

func LoadFile(filename string, log *logrus.Entry) *Gateway {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}
	return LoadBytes(src, log)
}

func LoadBytes(src []byte, log *logrus.Entry) *Gateway {
	config := &Gateway{}
	evalContext := handler.NewEvalContext(decodeEnvironmentRefs(src))
	// filename must match .hcl ending for further []byte processing
	if err := hclsimple.Decode("loadBytes.hcl", src, evalContext, config); err != nil {
		log.Fatalf("Failed to load configuration bytes: %s", err)
	}
	return Load(config, log, evalContext)
}

func Load(config *Gateway, log *logrus.Entry, evalCtx *hcl.EvalContext) *Gateway {
	type backendDefinition struct {
		conf    *Backend
		handler http.Handler
	}
	backends := make(map[string]backendDefinition)

	if config.Definitions != nil {
		for _, beConf := range config.Definitions.Backend {
			if _, ok := backends[beConf.Name]; ok {
				log.Fatalf("backend name must be unique: '%s'", beConf.Name)
			}
			if beConf.Timeout == "" {
				beConf.Timeout = backendDefaultTimeout
			}
			if beConf.ConnectTimeout == "" {
				beConf.ConnectTimeout = backendDefaultConnectTimeout
			}
			t, ct := parseBackendTimings(beConf.Timeout, beConf.ConnectTimeout)
			backends[beConf.Name] = backendDefinition{
				conf: beConf,
				handler: handler.NewProxy(&handler.ProxyOptions{
					ConnectTimeout: ct,
					Context:        beConf.Options,
					Hostname:       beConf.Hostname,
					Origin:         beConf.Origin,
					Path:           beConf.Path,
					Timeout:        t,
				}, log, evalCtx),
			}
		}
	}

	accessControls := configureAccessControls(config)

	for idx, server := range config.Server {
		configureDomains(server)
		configureBasePathes(server)

		if server.API == nil {
			continue
		}

		server.API.PathHandler = make(PathHandler)

		// map backends to endpoint
		endpoints := make(map[string]bool)
		for e, endpoint := range server.API.Endpoint {
			config.Server[idx].API.Endpoint[e].Server = server // assign parent
			if endpoints[endpoint.Pattern] {
				log.Fatal("Duplicate endpoint: ", endpoint.Pattern)
			}
			endpoints[endpoint.Pattern] = true

			var acList ac.List
			ac := AccessControl{server.AccessControl, server.DisableAccessControl}
			for _, acName := range ac.
				Merge(AccessControl{server.API.AccessControl, server.API.DisableAccessControl}).
				Merge(AccessControl{endpoint.AccessControl, endpoint.DisableAccessControl}).
				List() {
				if _, ok := accessControls[acName]; !ok {
					log.Fatalf("access control %q is not defined", acName)
				}
				acList = append(acList, accessControls[acName])
			}

			// setACHandlerFn individual wrap for access_control configuration per endpoint
			setACHandlerFn := func(protectedBackend backendDefinition) {
				protectedHandler := protectedBackend.handler

				// prefer endpoint 'path' definition over 'backend.Path'
				if endpoint.Path != "" {
					beConf := protectedBackend.conf.Merge(&Backend{Path: endpoint.Path})
					t, ct := parseBackendTimings(beConf.Timeout, beConf.ConnectTimeout)
					protectedHandler = handler.NewProxy(&handler.ProxyOptions{
						ConnectTimeout: ct,
						Context:        beConf.Options,
						Hostname:       beConf.Hostname,
						Origin:         beConf.Origin,
						Path:           beConf.Path,
						Timeout:        t,
					}, log, evalCtx)
				}

				if len(acList) > 0 {
					server.API.PathHandler[endpoint] = handler.NewAccessControl(protectedHandler, acList...)
					return
				}
				server.API.PathHandler[endpoint] = protectedHandler
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
			inlineBackend, inlineConf, err := newInlineBackend(evalCtx, endpoint.InlineDefinition, log)
			if err == errorMissingBackend {
				if server.API.Backend != "" {
					if _, ok := backends[server.API.Backend]; !ok {
						log.Fatalf("backend %q is not defined", server.API.Backend)
					}
					setACHandlerFn(backends[server.API.Backend])
					continue
				}
				inlineBackend, inlineConf, err = newInlineBackend(evalCtx, server.API.InlineDefinition, log)
				if err != nil {
					log.Fatal(err)
				}
			}

			if inlineConf.Name != "" { // inline backends have no label, assume a reference and override settings
				if _, ok := backends[inlineConf.Name]; !ok {
					log.Fatalf("override backend %q is not defined", inlineConf.Name)
				}
				beConf := backends[inlineConf.Name].conf.Merge(inlineConf)
				t, ct := parseBackendTimings(beConf.Timeout, beConf.ConnectTimeout)
				inlineBackend = handler.NewProxy(&handler.ProxyOptions{
					ConnectTimeout: ct,
					Context:        beConf.Options,
					Hostname:       beConf.Hostname,
					Origin:         beConf.Origin,
					Path:           beConf.Path,
					Timeout:        t,
				}, log, evalCtx)
			}

			setACHandlerFn(backendDefinition{conf: inlineConf, handler: inlineBackend})
		}
	}

	return config
}

func configureBasePathes(server *Server) {
	if server.BasePath == "" {
		server.BasePath = "/"
	}
	if !strings.HasSuffix(server.BasePath, "/") {
		server.BasePath = server.BasePath + "/"
	}
	if server.Files != nil {
		server.Files.BasePath = path.Join(server.BasePath, server.Files.BasePath)
		if !strings.HasSuffix(server.Files.BasePath, "/") {
			server.Files.BasePath = server.Files.BasePath + "/"
		}
	}
	if server.Spa != nil {
		server.Spa.BasePath = path.Join(server.BasePath, server.Spa.BasePath) + "/"
		if !strings.HasSuffix(server.Spa.BasePath, "/") {
			server.Spa.BasePath = server.Spa.BasePath + "/"
		}
	}
	if server.API != nil {
		server.API.BasePath = path.Join(server.BasePath, server.API.BasePath) + "/"
		if !strings.HasSuffix(server.API.BasePath, "/") {
			server.API.BasePath = server.API.BasePath + "/"
		}
	}
}

// configureDomains is a fallback configuration which ensures
// the request multiplexer is working properly.
func configureDomains(server *Server) {
	if len(server.Domains) > 0 {
		return
	}

	server.Domains = []string{"localhost", "127.0.0.1", "0.0.0.0", "::1"}
}

func configureAccessControls(conf *Gateway) ac.Map {
	accessControls := make(ac.Map)
	if conf.Definitions != nil {
		for _, jwt := range conf.Definitions.JWT {
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
			claims := ac.Claims{
				Audience: jwt.Claims.Audience,
				Issuer:   jwt.Claims.Issuer,
			}
			j, err := ac.NewJWT(jwt.SignatureAlgorithm, claims, jwtSource, jwtKey, key)
			if err != nil {
				panic(fmt.Sprintf("loading jwt %q definition failed: %s", jwt.Name, err))
			}
			accessControls[jwt.Name] = j
		}
	}
	return accessControls
}

func newInlineBackend(evalCtx *hcl.EvalContext, inlineDef hcl.Body, log *logrus.Entry) (http.Handler, *Backend, error) {
	content, leftOver, diags := inlineDef.PartialContent(Definitions{}.Schema(true))
	// ignore diag errors here, would fail anyway with our retry
	if content == nil || len(content.Blocks) == 0 {
		// no inline conf, retry for override definitions with label
		content, leftOver, diags = inlineDef.PartialContent(Definitions{}.Schema(false))
		if diags.HasErrors() {
			return nil, nil, diags
		}

		if content == nil || len(content.Blocks) == 0 {
			return nil, nil, errorMissingBackend
		}
	}

	beConf := &Backend{}
	diags = gohcl.DecodeBody(content.Blocks[0].Body, evalCtx, beConf)
	if diags.HasErrors() {
		return nil, nil, diags
	}
	if len(content.Blocks[0].Labels) > 0 {
		beConf.Name = content.Blocks[0].Labels[0]
	}

	beConf.Options = leftOver
	t, ct := parseBackendTimings(beConf.Timeout, beConf.ConnectTimeout)
	return handler.NewProxy(&handler.ProxyOptions{
		ConnectTimeout: ct,
		Context:        leftOver,
		Hostname:       beConf.Hostname,
		Origin:         beConf.Origin,
		Path:           beConf.Path,
		Timeout:        t,
	}, log, evalCtx), beConf, nil
}

func parseBackendTimings(total, connect string) (time.Duration, time.Duration) {
	t := total
	c := connect
	if t == "" {
		t = backendDefaultTimeout
	}
	if c == "" {
		c = backendDefaultConnectTimeout
	}
	totalD, err := time.ParseDuration(t)
	if err != nil {
		panic(err)
	}
	connectD, err := time.ParseDuration(c)
	if err != nil {
		panic(err)
	}
	return totalD, connectD
}

func decodeEnvironmentRefs(src []byte) []string {
	tokens, diags := hclsyntax.LexConfig(src, "tmp.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		panic(diags)
	}
	needle := []byte("env")
	var keys []string
	for i, token := range tokens {
		if token.Type == hclsyntax.TokenIdent &&
			bytes.Equal(token.Bytes, needle) &&
			i+2 < len(tokens) {
			value := string(tokens[i+2].Bytes)
			if sort.SearchStrings(keys, value) == len(keys) {
				keys = append(keys, value)
			}
		}
	}
	return keys
}

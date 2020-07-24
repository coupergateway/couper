package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/sirupsen/logrus"

	ac "go.avenga.cloud/couper/gateway/access_control"
	"go.avenga.cloud/couper/gateway/handler"
)

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
	accessControls := configureAccessControls(config)

	backends, err := configureBackends(config, log, evalCtx)
	if err != nil {
		log.Fatal(err)
	}

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

			if endpoint.Backend != "" {
				if _, ok := backends[endpoint.Backend]; !ok {
					log.Fatalf("backend %q is not defined", endpoint.Backend)
				}
				if len(acList) > 0 {
					server.API.PathHandler[endpoint] = handler.NewAccessControl(backends[endpoint.Backend], acList...)
					continue
				}
				server.API.PathHandler[endpoint] = backends[endpoint.Backend]
				continue
			}

			backendErr := fmt.Errorf("expected backend attribute reference or block for endpoint: %s", endpoint)

			// endpoint.Options usecase is optional blocks, so we do not need to edit and set leftOver content back to endpoint.
			content, leftOver, diags := endpoint.Options.PartialContent(Definitions{}.Schema(true))
			if diags.HasErrors() {
				log.Fatal(diags.Error())
			}

			if content == nil || len(content.Blocks) == 0 {
				log.Fatal(backendErr)
			}

			beConf, err := newInlineBackend(evalCtx, content)
			if err != nil {
				log.Fatal(err)
			}
			inlineBackend := handler.NewProxy(beConf.Origin, beConf.Hostname, beConf.Path, log, evalCtx, leftOver)

			if len(acList) > 0 {
				server.API.PathHandler[endpoint] = handler.NewAccessControl(inlineBackend, acList...)
				continue
			}
			server.API.PathHandler[endpoint] = inlineBackend
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
	// TODO: ipv6
	server.Domains = []string{"localhost", "127.0.0.1", "0.0.0.0"}
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

func configureBackends(conf *Gateway, log *logrus.Entry, evalCtx *hcl.EvalContext) (map[string]http.Handler, error) {
	backends := make(map[string]http.Handler)
	if conf.Definitions == nil {
		return backends, nil
	}

	for _, be := range conf.Definitions.Backend {
		if _, ok := backends[be.Name]; ok {
			return nil, fmt.Errorf("backend name must be unique: '%s'", be.Name)
		}
		backends[be.Name] = handler.NewProxy(be.Origin, be.Hostname, be.Path, log, evalCtx, be.Options)
	}

	return backends, nil
}

func newInlineBackend(evalCtx *hcl.EvalContext, content *hcl.BodyContent) (*Backend, error) {
	backendConf := &Backend{}
	diags := gohcl.DecodeBody(content.Blocks[0].Body, evalCtx, backendConf)
	if diags.HasErrors() {
		return backendConf, diags
	}
	return backendConf, nil
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

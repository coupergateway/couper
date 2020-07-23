package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/sirupsen/logrus"

	ac "go.avenga.cloud/couper/gateway/access_control"
	"go.avenga.cloud/couper/gateway/handler"
)

var typeMap = map[string]func(*logrus.Entry, hcl.Body) http.Handler{
	"proxy": handler.NewProxy(),
}

func LoadFile(name string, log *logrus.Entry) *Gateway {
	config := &Gateway{}
	err := hclsimple.DecodeFile(name, nil, config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}
	return Load(config, log)
}

func LoadBytes(src []byte, log *logrus.Entry) *Gateway {
	config := &Gateway{}
	// filename must match .hcl ending for further []byte processing
	if err := hclsimple.Decode("loadBytes.hcl", src, nil, config); err != nil {
		log.Fatalf("Failed to load configuration bytes: %s", err)
	}
	return Load(config, log)
}

func Load(config *Gateway, log *logrus.Entry) *Gateway {
	accessControls := configureAccessControls(config)

	backends := make(map[string]http.Handler)

	for idx, server := range config.Server {
		configureDomains(server)

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

		if server.API == nil {
			continue
		}

		// create backends
		for _, be := range server.API.Backend {
			if isKeyword(be.Name) {
				log.Fatalf("be name not allowed, reserved keyword: '%s'", be.Name)
			}
			if _, ok := backends[be.Name]; ok {
				log.Fatalf("be name must be unique: '%s'", be.Name)
			}
			backends[be.Name] = newBackend(be.Kind, be.Options, log)
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

			content, leftOver, diags := endpoint.Options.PartialContent(server.API.Schema(true))
			if diags.HasErrors() {
				log.Fatal(diags.Error())
			}
			endpoint.Options = leftOver

			if content == nil || len(content.Blocks) == 0 {
				log.Fatalf("expected backend attribute reference or block for endpoint: %s", endpoint)
			}
			kind := content.Blocks[0].Labels[0]
			inlineBackend := newBackend(kind, content.Blocks[0].Body, log)
			if len(acList) > 0 {
				server.API.PathHandler[endpoint] = handler.NewAccessControl(inlineBackend, acList...)
				continue
			}
			server.API.PathHandler[endpoint] = inlineBackend
		}
	}

	return config
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

func newBackend(kind string, options hcl.Body, log *logrus.Entry) http.Handler {
	if !isKeyword(kind) {
		log.Fatalf("Invalid backend: %s", kind)
	}
	b := typeMap[strings.ToLower(kind)](log, options)

	return b
}

func isKeyword(other string) bool {
	_, yes := typeMap[other]
	return yes
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

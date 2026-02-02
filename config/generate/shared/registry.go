package shared

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/coupergateway/couper/config"
)

// ConfigRegistry contains all config struct types that should be processed
var ConfigRegistry = []interface{}{
	&config.API{},
	&config.Backend{},
	&config.BackendTLS{},
	&config.BasicAuth{},
	&config.CORS{},
	&config.Defaults{},
	&config.Definitions{},
	&config.Endpoint{},
	&config.ErrorHandler{},
	&config.Files{},
	&config.Health{},
	&config.Introspection{},
	&config.JWTSigningProfile{},
	&config.JWT{},
	&config.Job{},
	&config.OAuth2AC{},
	&config.OAuth2ReqAuth{},
	&config.OIDC{},
	&config.OpenAPI{},
	&config.Proxy{},
	&config.RateLimit{},
	&config.RateLimiter{},
	&config.Request{},
	&config.Response{},
	&config.SAML{},
	&config.Server{},
	&config.ClientCertificate{},
	&config.ServerCertificate{},
	&config.ServerTLS{},
	&config.Settings{},
	&config.Spa{},
	&config.TokenRequest{},
	&config.Websockets{},
}

var filenameRegex = regexp.MustCompile(`(URL|JWT|OpenAPI|[a-z0-9]+)`)

// BlockNamesMap provides mappings from internal type names to HCL block names
// Used by docs generator to match documentation file names
var BlockNamesMap = map[string]string{
	"oauth2_ac":       "beta_oauth2",
	"oauth2_req_auth": "oauth2",
}

// VSCodeBlockNamesMap provides mappings for VS Code schema (HCL block names)
// Some blocks like tls are the same block used in different contexts
var VSCodeBlockNamesMap = map[string]string{
	"oauth2_ac":       "beta_oauth2",
	"oauth2_req_auth": "oauth2",
	"backend_tls":     "tls",
	"server_tls":      "tls",
}

// GetBlockName returns the HCL block name for a config struct type
func GetBlockName(impl interface{}) string {
	t := reflect.TypeOf(impl)
	name := t.String()
	name = strings.TrimPrefix(name, "*config.")
	blockName := strings.ToLower(strings.Trim(filenameRegex.ReplaceAllString(name, "${1}_"), "_"))

	if renamed, exists := BlockNamesMap[blockName]; exists {
		return renamed
	}
	return blockName
}

// GetTypeName returns the Go type name without package prefix
func GetTypeName(impl interface{}) string {
	t := reflect.TypeOf(impl)
	name := t.String()
	return strings.TrimPrefix(name, "*config.")
}

// ConfigStructInfo holds information about a config struct
type ConfigStructInfo struct {
	Impl      interface{}
	BlockName string
	TypeName  string
	Type      reflect.Type
}

// GetAllConfigStructs returns information about all registered config structs
func GetAllConfigStructs() []ConfigStructInfo {
	return getAllConfigStructsWithMap(BlockNamesMap)
}

// GetAllConfigStructsForVSCode returns config structs with VS Code-specific block name mappings
func GetAllConfigStructsForVSCode() []ConfigStructInfo {
	return getAllConfigStructsWithMap(VSCodeBlockNamesMap)
}

func getAllConfigStructsWithMap(nameMap map[string]string) []ConfigStructInfo {
	var result []ConfigStructInfo
	for _, impl := range ConfigRegistry {
		t := reflect.TypeOf(impl).Elem()
		result = append(result, ConfigStructInfo{
			Impl:      impl,
			BlockName: getBlockNameWithMap(impl, nameMap),
			TypeName:  GetTypeName(impl),
			Type:      t,
		})
	}
	return result
}

func getBlockNameWithMap(impl interface{}, nameMap map[string]string) string {
	t := reflect.TypeOf(impl)
	name := t.String()
	name = strings.TrimPrefix(name, "*config.")
	blockName := strings.ToLower(strings.Trim(filenameRegex.ReplaceAllString(name, "${1}_"), "_"))

	if renamed, exists := nameMap[blockName]; exists {
		return renamed
	}
	return blockName
}

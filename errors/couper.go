package errors

import (
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

const Wildcard = "*"

var (
	AccessControl     = &Error{synopsis: "access control error", kinds: []string{"access_control"}, httpStatus: http.StatusForbidden}
	Backend           = &Error{synopsis: "backend error", httpStatus: http.StatusBadGateway}
	BackendTimeout    = &Error{synopsis: "backend timeout error", httpStatus: http.StatusGatewayTimeout}
	BackendValidation = &Error{synopsis: "backend validation error", httpStatus: http.StatusBadRequest}
	ClientRequest     = &Error{synopsis: "client request error", httpStatus: http.StatusBadRequest}
	Evaluation        = &Error{synopsis: "expression evaluation error", kinds: []string{"evaluation"}}
	Configuration     = &Error{synopsis: "configuration error", kinds: []string{"configuration"}, httpStatus: http.StatusInternalServerError}
	Proxy             = &Error{synopsis: "proxy error", httpStatus: http.StatusBadGateway}
	Request           = &Error{synopsis: "request error", httpStatus: http.StatusBadGateway}
	RouteNotFound     = &Error{synopsis: "route not found error", httpStatus: http.StatusNotFound}
	Server            = &Error{synopsis: "internal server error", httpStatus: http.StatusInternalServerError}
	ServerShutdown    = &Error{synopsis: "server shutdown error", httpStatus: http.StatusInternalServerError}
	ServerTimeout     = &Error{synopsis: "server timeout error", httpStatus: http.StatusGatewayTimeout}
)

func TypeToSnake(t interface{}) string {
	typeStr := reflect.TypeOf(t).String()
	if strings.Contains(typeStr, ".") { // package name removal
		typeStr = strings.Split(typeStr, ".")[1]
	}
	var result []rune
	var previous rune
	for i, r := range typeStr {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(previous) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
		previous = r
	}

	return string(result)
}

func SnakeToCamel(str string) string {
	var result []rune
	if len(str) == 0 {
		return str
	}
	result = append(result, unicode.ToUpper(rune(str[0])))
	var upperNext bool
	for _, r := range str[1:] {
		if r == '_' {
			upperNext = true
			continue
		}
		if upperNext {
			result = append(result, unicode.ToUpper(r))
		} else {
			result = append(result, unicode.ToLower(r))
		}
		upperNext = false
	}
	return string(result)
}

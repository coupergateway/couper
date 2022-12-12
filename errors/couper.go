package errors

import (
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

const Wildcard = "*"

var (
	AccessControl    = &Error{synopsis: "access control error", kinds: []string{"access_control"}, httpStatus: http.StatusForbidden}
	Backend          = &Error{synopsis: "backend error", Contexts: []string{"api", "endpoint"}, kinds: []string{"backend"}, httpStatus: http.StatusBadGateway}
	ClientRequest    = &Error{synopsis: "client request error", httpStatus: http.StatusBadRequest}
	Endpoint         = &Error{synopsis: "endpoint error", Contexts: []string{"endpoint"}, kinds: []string{"endpoint"}, httpStatus: http.StatusBadGateway}
	Evaluation       = &Error{synopsis: "expression evaluation error", kinds: []string{"evaluation"}, httpStatus: http.StatusInternalServerError}
	Configuration    = &Error{synopsis: "configuration error", kinds: []string{"configuration"}, httpStatus: http.StatusInternalServerError}
	MethodNotAllowed = &Error{synopsis: "method not allowed error", httpStatus: http.StatusMethodNotAllowed}
	Proxy            = &Error{synopsis: "proxy error", httpStatus: http.StatusBadGateway}
	Request          = &Error{synopsis: "request error", httpStatus: http.StatusBadGateway}
	RouteNotFound    = &Error{synopsis: "route not found error", httpStatus: http.StatusNotFound}
	Server           = &Error{synopsis: "internal server error", httpStatus: http.StatusInternalServerError}
	ServerShutdown   = &Error{synopsis: "server shutdown error", httpStatus: http.StatusInternalServerError}
)

func TypeToSnake(t interface{}) string {
	typeStr := reflect.TypeOf(t).String()
	if strings.Contains(typeStr, ".") { // package name removal
		typeStr = strings.Split(typeStr, ".")[1]
	}
	return TypeToSnakeString(typeStr)
}

func TypeToSnakeString(typeStr string) string {
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

package middleware

import "net/http"

type Next func(http.Handler) http.Handler

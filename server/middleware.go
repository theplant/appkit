package server

import (
	"net/http"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/contexts/trace"
	"github.com/theplant/appkit/log"
)

func DefaultMiddleware(logger log.Logger) func(http.Handler) http.Handler {
	return Compose(
		// Recovery should come before logReq to set the status code to 500
		Recovery,
		LogRequest,
		log.WithLogger(logger),
		trace.WithRequestTrace,
		contexts.WithHTTPStatus,
	)
}

// Middleware represents the form of HTTP middleware constructors.
type Middleware func(http.Handler) http.Handler

// Compose provides a convenient way to chain the HTTP
// middleware functions.
//
// In short, it transforms
//
// `Middleware3(Middleware2(Middleware1(HttpHandler)))`
//
// to
//
// `Compose(Middleware1, Middleware2, Middleware3)(HttpHandler)`
//
// More details: https://github.com/theplant/hsm2-backend/pull/258#discussion_r70732260
func Compose(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for _, m := range middlewares {
			h = m(h)
		}
		return h
	}
}

// idMiddleware is middleware that has no effect, useful for optional
// middleware, instead of returning a custom function every time.
func IdMiddleware(handler http.Handler) http.Handler {
	return handler
}

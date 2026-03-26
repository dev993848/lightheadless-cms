package middleware

import "net/http"

// Chain applies a series of middleware to a handler, wrapping them in order
// so that the first middleware listed is outermost (executed first).
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// Apply in reverse so the first middleware is outermost.
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

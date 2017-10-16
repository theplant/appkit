package server

import (
	"context"
	"net/http"
)

type key int

const headerKey key = iota

func WithHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		newCtx := context.WithValue(ctx, headerKey, w.Header())
		h.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func ForceHeader(ctx context.Context) (h http.Header) {
	h = ctx.Value(headerKey).(http.Header)
	if h == nil {
		panic("no header in context, please setup prottp.WithHeader middleware")
	}
	return
}

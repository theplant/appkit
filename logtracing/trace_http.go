package logtracing

import (
	"fmt"
	"net/http"
)

func HTTPClientKVs(req *http.Request) []interface{} {
	return []interface{}{
		"span.type", "http",
		"span.role", "client",
		"http.url", req.URL.String(),
		"http.method", req.Method,
	}
}

func TraceHTTPRequest(do func(*http.Request) (*http.Response, error), baseName string, req *http.Request) (resp *http.Response, err error) {
	ctx, _ := StartSpan(req.Context(), httpClientRequestName(baseName, req))
	defer func() { EndSpan(ctx, err) }()
	AppendSpanKVs(ctx,
		HTTPClientKVs(req)...,
	)
	resp, err = do(req)
	if err == nil {
		AppendSpanKVs(ctx,
			"http.status", resp.Status,
		)
	}
	return resp, err
}

func httpClientRequestName(base string, req *http.Request) string {
	return fmt.Sprintf("%s.call(%s)", base, req.URL.Path)
}

type HTTPTransport struct {
	BaseName     string
	RoundTripper http.RoundTripper
}

func (tr *HTTPTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return TraceHTTPRequest(tr.RoundTripper.RoundTrip, tr.BaseName, req)
}

// TODO: add a middleware to trace http requests

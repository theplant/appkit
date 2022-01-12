package logtracing

import (
	"fmt"
	"net/http"
)

type HTTPTransport struct {
	BaseName     string
	RoundTripper http.RoundTripper
}

func (tr *HTTPTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return TraceHTTPRequest(tr.RoundTripper.RoundTrip, tr.BaseName, req)
}

func TraceHTTPRequest(do func(*http.Request) (*http.Response, error), baseName string, req *http.Request) (resp *http.Response, err error) {
	ctx, _ := StartSpan(req.Context(), httpRequestName(baseName, req))
	defer func() { EndSpan(ctx, err) }()
	AppendKVs(
		ctx,
		"span.type", "http",
		"span.role", "client",
		"http.url", req.URL.String(),
		"http.method", req.Method,
	)
	resp, err = do(req)
	if err == nil {
		AppendKVs(
			ctx,
			"http.status", resp.Status,
		)
	}
	return resp, err
}

func httpRequestName(base string, req *http.Request) string {
	return fmt.Sprintf("%s.call(%s)", base, req.URL.Path)
}

package trace

import (
	"fmt"
	"net/http"
)

type TracedTransport struct {
	BaseName     string
	RoundTripper http.RoundTripper
}

func (tr *TracedTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return TraceHttpRequest(tr.RoundTripper.RoundTrip, tr.BaseName, req)
}

func httpRequestName(base string, req *http.Request) string {
	return fmt.Sprintf("%s.call(%s)", base, req.URL.Path)
}

func TraceHttpRequest(do func(*http.Request) (*http.Response, error), baseName string, req *http.Request) (resp *http.Response, err error) {
	ctx, _ := StartSpan(req.Context(), httpRequestName(baseName, req))
	defer func() { EndSpan(ctx, err) }()
	// AppendKVs(
	// 	ctx,
	// 	"span.type", "http",
	// 	"span.role", "client",
	// 	"http.url", req.URL.String(),
	// 	"http.method", req.Method,
	// )
	// resp, err = do(req)
	// if err == nil {
	// 	AppendKVs(
	// 		ctx,
	// 		"http.status", resp.Status,
	// 	)
	// }
	return resp, err
}

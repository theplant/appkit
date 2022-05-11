package logtracing

import (
	"fmt"
	"net"
	"net/http"
	"strings"
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
	req = req.WithContext(ctx)
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

func HTTPServerKVs(req *http.Request) []interface{} {
	return []interface{}{
		"span.type", "http",
		"span.role", "server",
		"http.method", req.Method,
		"http.path", req.URL.Path,
		"http.query_string", req.URL.RawQuery,
		"http.user_agent", req.UserAgent(),
		"http.client_ip", clientIP(req),
	}
}

func clientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	if index := strings.IndexByte(clientIP, ','); index >= 0 {
		clientIP = clientIP[0:index]
	}
	clientIP = strings.TrimSpace(clientIP)
	if len(clientIP) > 0 {
		return clientIP
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

func HTTPServerResponseKVs(status string) []interface{} {
	return []interface{}{
		"http.status", status,
	}
}

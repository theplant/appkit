// Package routes is a commen place to put all applicatioin routes.
// In order to easy setup routes for application and testing.
package server

import (
	"crypto/md5"
	"fmt"
	"hash"
	golog "log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/log"
)

func NewServer(logger log.Logger, addr string, handler http.Handler) *http.Server {
	server := http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: golog.New(log.LogWriter(logger.Error()), "", golog.Llongfile),
	}
	return &server
}

func DefaultMiddleware(logger log.Logger) func(http.Handler) http.Handler {
	return Compose(
		// Recovery should come before logReq to set the status code to 500
		Recovery,
		logReq,
		contexts.WithLogger(logger),
		contexts.TraceRequest,
		contexts.Status,
	)
}

// Lifted from gin-gonic/gin/context.go
func ClientIP(r *http.Request) string {
	clientIP := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	if len(clientIP) > 0 {
		return clientIP
	}
	clientIP = r.Header.Get("X-Forwarded-For")
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

// Will absorb panics in earlier Middleware. Times the request and logs the result. FIXME split the timing out into a separate Middleware
var logReq = func(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.RequestURI

		ctx := r.Context()

		logger, ok := contexts.Logger(ctx)

		if !ok {
			h.ServeHTTP(rw, r)
			return
		}

		l := logger.With(
			"context", "http",
			"path", path,
			"method", r.Method,
			"client_ip", ClientIP(r),
		)

		l.Debug().Log("msg", fmt.Sprintf("%s %s", r.Method, path))

		defer func() {
			duration := int64(time.Since(start) / time.Microsecond)

			status, _ := contexts.ResponseStatus(r.Context())

			l = l.With(
				"request_us", duration,
				"status", status,
				//					"response_size", rw.Size(),
			)

			msg := fmt.Sprintf("%s %s -> %03d %s", r.Method, path, status, http.StatusText(status))

			// Will absorb panics in earlier middleware
			if err := recover(); err != nil {
				stack := stack(7)
				httprequest, _ := httputil.DumpRequest(r, false)
				l = l.With(
					"err", err,
					"request", string(httprequest),
					"stack", string(stack),
				)
				msg = fmt.Sprintf("%s (panic: %v)", msg, err)
			}

			if status > 500 {
				l.Error().Log("msg", msg)
			} else if status == 500 {
				l.Crit().Log("msg", msg)
			} else if status >= 400 {
				l.Warn().Log("msg", msg)
			} else {
				l.Info().Log("msg", msg)
			}
		}()

		h.ServeHTTP(rw, r)

	})
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

//////////////////////////////////////////////////
// eTagWriter will buffer 200 OK responses to GETs, and calculate an
// ETag as md5(body). If this same ETag is passed by the client in
// `If-None-Match`, the API will return 304 Not Modified with an empty
// body.
type eTagWriter struct {
	http.ResponseWriter
	request *http.Request
	code    int
	hash    hash.Hash
	data    []byte
}

func newETagWriter(w http.ResponseWriter, r *http.Request) *eTagWriter {
	return &eTagWriter{
		ResponseWriter: w,
		request:        r,
		code:           http.StatusOK,
		hash:           md5.New(),
		data:           []byte{},
	}
}

func (w *eTagWriter) Write(data []byte) (int, error) {
	w.data = append(w.data, data...)
	w.hash.Write(data)
	return len(data), nil
}

func (w *eTagWriter) eTag() string {
	return fmt.Sprintf("\"%x\"", w.hash.Sum(nil))
}

func (w *eTagWriter) end() {
	wr := w.ResponseWriter
	_, tagged := w.Header()["ETag"]
	if !tagged && w.code == http.StatusOK {
		// Set the ETag HTTP header in Response
		respTag := w.eTag()
		w.Header().Set("ETag", respTag)

		// ... Get the ETag from the request
		reqTag := w.request.Header.Get("If-None-Match")

		// ... Ignore/strip weak validator mark (`W/<ETag>`)
		reqTag = strings.TrimPrefix(reqTag, "W/")

		// ... Compare client's ETag to request's ETag
		if len(reqTag) > 0 && reqTag == respTag {
			// ... and 304 if they match!
			wr.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// ... Otherwise, send the buffered data.
	wr.WriteHeader(w.code)
	data := w.data
	for len(data) > 0 {
		c, err := wr.Write(data)
		if err != nil {
			panic(err)
		} else if c == 0 {
			panic(fmt.Errorf("didn't write anything, got %d bytes left in my hands", len(data)))
		}
		data = data[c:]
	}
}

// Buffer the HTTP status, we can't write it until we have the complete response
func (w *eTagWriter) WriteHeader(code int) {
	w.code = code
}

func ETag(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			wr := newETagWriter(w, r)
			h.ServeHTTP(wr, r)
			wr.end()
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

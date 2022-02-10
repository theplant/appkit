package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/logtracing"
)

// Will absorb panics in earlier Middleware. Times the request and logs the result.
func LogRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ctx, span := logtracing.StartSpan(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		r = r.WithContext(ctx)
		span.AppendKVs(
			logtracing.HTTPServerKVs(r)...,
		)

		// NOTE for compatibility
		span.AppendKVs(
			"context", "http",
			"path", r.RequestURI,
			"method", r.Method,
			"client_ip", clientIP(r),
		)

		defer func() {
			status, _ := contexts.HTTPStatus(r.Context())
			span.AppendKVs(
				"http.status", status,
			)

			// NOTE for compatibility
			span.End()
			span.AppendKVs(
				"request_us", span.Duration().Microseconds(),
				"status", status,
				"user_agent", r.UserAgent(),
			)

			// Will absorb panics in earlier middleware
			if err := recover(); err != nil {
				span.RecordPanic(err)

				// NOTE for compacibility
				stack := stack(7)
				span.AppendKVs(
					"err", err,
					"stack", string(stack),
				)
			}

			logtracing.LogSpan(r.Context(), span)
		}()

		h.ServeHTTP(rw, r)

	})
}

// Adapted from gin-gonic/gin/context.go and gin-gonic/gin/recovery.go

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

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

// stack returns a nicely formated stack frame, skipping skip frames
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

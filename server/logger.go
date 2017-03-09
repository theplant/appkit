package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
	"time"

	"github.com/theplant/appkit/contexts"
	"github.com/theplant/appkit/log"
)

// Will absorb panics in earlier Middleware. Times the request and logs the result. FIXME split the timing out into a separate Middleware
func LogRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.RequestURI

		ctx := r.Context()

		logger, ok := log.FromContext(ctx)

		if !ok {
			h.ServeHTTP(rw, r)
			return
		}

		l := logger.With(
			"context", "http",
			"path", path,
			"method", r.Method,
			"client_ip", clientIP(r),
		)

		l.Debug().Log("msg", fmt.Sprintf("%s %s", r.Method, path))

		defer func() {
			duration := int64(time.Since(start) / time.Microsecond)

			status, _ := contexts.HTTPStatus(r.Context())

			l = l.With(
				"request_us", duration,
				"status", status,
				//					"response_size", rw.Size(),
				"user_agent", r.UserAgent(),
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

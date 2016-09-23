package server

import (
	"crypto/md5"
	"fmt"
	"hash"
	"net/http"
	"strings"
)

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

// ETag is http.Handler that will, for `GET` requests:
//
// 1. Calculate ETag as md5(body)
// 2. Add ETag HTTP header to response
// 3. If client sends `If-None-Match` header with matching ETag,
//    discard body and respond with `304 Not Modified` on any `200 OK`
//    responses
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

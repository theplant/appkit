package errornotifier_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theplant/appkit/errornotifier"
	au "github.com/theplant/appkit/errornotifier/utils"
)

var errHandlerException = errors.New("panic on handler")

func TestRecoverMiddleware(t *testing.T) {
	bufferNotifier := &au.BufferNotifier{}

	server := newRecoverTestServer(bufferNotifier)
	defer func() {
		server.Close()
	}()

	if len(bufferNotifier.Notices) != 0 {
		t.Fatalf("Notices must be empty.")
	}

	_, err := http.Get(server.URL + "/recover")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bufferNotifier.Notices) != 1 {
		t.Fatalf("Unexpected notices length, got %d.", len(bufferNotifier.Notices))
	}

	if bufferNotifier.Notices[0].Error != errHandlerException {
		t.Fatalf("Got unexpected error: %v ", bufferNotifier.Notices[0].Error)
	}
}

// newRecoverTestServer prepares a test HTTP server that has the Recover
// middleware configured at `/recover`
func newRecoverTestServer(n errornotifier.Notifier) *httptest.Server {

	// Catch handler panics so that test can continue.
	clearPanics := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recover()
			}()

			h.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/recover", func(w http.ResponseWriter, r *http.Request) {
		panic(errHandlerException)
	})

	// Recover middleware is executed first.
	server := httptest.NewServer(clearPanics(errornotifier.Recover(n)(mux)))

	return server
}

func ExampleNotifyOnPanic() {
	bufferNotifier := &au.BufferNotifier{}

	// return nil => Do nothing
	err := errornotifier.NotifyOnPanic(bufferNotifier, nil, func() {
		fmt.Println("do nothing")
	})
	fmt.Printf("%v %d\n", err, len(bufferNotifier.Notices))

	// panic in func => return panic error
	err = errornotifier.NotifyOnPanic(bufferNotifier, nil, func() {
		panicErr := "panic"
		fmt.Println(panicErr)
		panic(panicErr)
	})
	fmt.Printf("%v %d\n", err, len(bufferNotifier.Notices))

	// Output:
	// do nothing
	// <nil> 0
	// panic
	// panic 1
}

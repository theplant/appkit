package sessions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGorillaContextMemoryleak(t *testing.T) {
	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		t.Fatal(err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, val := "key", "val"

		err = Put(r.Context(), key, val)
		if err != nil {
			t.Errorf("cannot operate session, err: %v", err)
		}

		// Call WithContext between two session calls.
		// To test if the previous value set by first request point is missing by the shallow copy of WithContext
		const testCtxKey sessionContextKey = 10
		ctx := context.WithValue(r.Context(), testCtxKey, "tmpVal")
		r = r.WithContext(ctx)

		if value, _ := Get(r.Context(), key); value != val {
			t.Errorf("session value changed in one request lifetime")
		}
	})

	respWriter := httptest.NewRecorder()

	conf := &Config{
		Name: "test",
		Key:  "6bude5uOm9eZV280BjP6f6a5bEj7fg2PWl6GysY68CmXfOv8NFZ9O6ZIpbllQPtr",
	}

	handler := WithSession(conf)

	handler(testHandler).ServeHTTP(respWriter, req)
}

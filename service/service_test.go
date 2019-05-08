package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// ADDR -> PORT -> 9800 fallback
func TestServerPort(t *testing.T) {
	port := "19999"

	os.Setenv("PORT", port)
	defer func() { os.Unsetenv("PORT") }()

	testServer(t, port)
}

func TestServerAddr(t *testing.T) {
	port := "20001"

	os.Setenv("ADDR", ":"+port)
	defer func() { os.Unsetenv("ADDR") }()

	testServer(t, port)
}

func TestServerDefaultPort(t *testing.T) {
	testServer(t, "9800")

}

func testServer(t *testing.T, port string) {

	ch := make(chan struct{})

	go ListenAndServe(func(_ context.Context, mux *http.ServeMux) error {
		mux.HandleFunc("/", func(_ http.ResponseWriter, _ *http.Request) {
			go func() { ch <- struct{}{} }()
		})
		return nil
	})

	<-time.After(100 * time.Millisecond)

	_, err := http.Get("http://localhost:" + port)

	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:

	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for request")
	}
}

func TestBasicAuth(t *testing.T) {
	os.Setenv("BASICAUTH_USERNAME", "username")
	defer func() { os.Unsetenv("BASICAUTH_USERNAME") }()

	os.Setenv("BASICAUTH_PASSWORD", "password")
	defer func() { os.Unsetenv("BASICAUTH_PASSWORD") }()

	os.Setenv("BASICAUTH_USERAGENTWHITELISTREGEXP", "non-default-user-agent")
	defer func() { os.Unsetenv("BASICAUTH_USERAGENTWHITELISTREGEXP") }()

	os.Setenv("BASICAUTH_PATHWHITELISTREGEXP", "/whitelisted-path")
	defer func() { os.Unsetenv("BASICAUTH_PATHWHITELISTREGEXP") }()

	ctx, c := serviceContext()
	defer c.Close()

	m, c2, err := middleware(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()

	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))

	t.Run("http auth with username/password", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.SetBasicAuth("username", "password")

		w := httptest.ResponseRecorder{}

		h.ServeHTTP(&w, r)

		if w.Code != 204 {
			t.Fatalf("unexpected status code, wanted 204, got %d", w.Code)
		}
	})

	cases := []struct {
		userAgent string
		path      string
		expected  int
	}{
		{userAgent: "non-default-user-agent", path: "/", expected: 204},
		{path: "/whitelisted-path", expected: 204},
		{userAgent: "non-default-user-agent", path: "/whitelisted-path", expected: 204},
		{path: "/", expected: 401},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("http auth with %+v", c), func(t *testing.T) {
			r := httptest.NewRequest("GET", c.path, nil)
			if c.userAgent != "" {
				r.Header.Set("User-Agent", c.userAgent)
			}

			w := httptest.ResponseRecorder{}

			h.ServeHTTP(&w, r)

			if w.Code != c.expected {
				t.Fatalf("unexpected status code, wanted %d, got %d", c.expected, w.Code)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	// Loose testing of CORS configuration
	os.Setenv("CORS_RawAllowedOrigins", "cors1.example.com,cors2.example.com")
	defer func() { os.Unsetenv("CORS_RawAllowedOrigins") }()

	os.Setenv("CORS_AllowCredentials", "true")
	defer func() { os.Unsetenv("CORS_AllowCredentials") }()

	ctx, c := serviceContext()
	defer c.Close()

	m, c2, err := middleware(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Close()

	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))

	cases := []struct {
		origin   string
		expected string
	}{
		{origin: "cors1.example.com", expected: "cors1.example.com"},
		{origin: "cors2.example.com", expected: "cors2.example.com"},
		{origin: "not-cors", expected: ""},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("request with %+v", c), func(t *testing.T) {
			r := httptest.NewRequest("OPTIONS", "/", nil)
			if c.origin != "" {
				r.Header.Set("Origin", c.origin)
			}
			r.Header.Set("Access-Control-Request-Method", "GET")

			w := httptest.ResponseRecorder{}

			h.ServeHTTP(&w, r)

			if w.Header().Get("Access-Control-Allow-Origin") != c.expected {
				t.Fatalf("unexpected response, wanted %q, got %q", c.origin, w.Header().Get("Access-Control-Allow-Origin"))

			}
		})
	}
}

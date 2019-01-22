package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func wrapper(label string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("-> %s\n", label)
			h.ServeHTTP(w, r)
			fmt.Printf("<- %s\n", label)
		})
	}
}

func ExampleCompositionOrder() {

	handler := Compose(
		wrapper("top"),
		wrapper("middle"),
		wrapper("bottom"),
	)

	s := httptest.NewServer(handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("handler")
		w.WriteHeader(204)
	})))

	req, err := http.NewRequest("POST", s.URL, nil)
	if err != nil {
		panic(err)
	}

	exec(s, req)

	// Output:
	// -> bottom
	// -> middle
	// -> top
	// handler
	// <- top
	// <- middle
	// <- bottom
	// 204
}

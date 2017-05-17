package sessions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestGorillaContextMemoryleak(t *testing.T) {
	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		t.Fatal(err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var originSessionOptions *sessions.Options

		if st := GetSessionStore(w, r); st != nil {
			originSessionOptions = st.session.Options
		} else {
			t.Errorf("session store not generated and get set in the context")
		}

		ctx := context.WithValue(r.Context(), "tmpKey", "tmpVal")
		r = r.WithContext(ctx)

		if st := GetSessionStore(w, r); st == nil {
			t.Errorf("session store isn't set in the context")
		} else if originSessionOptions != st.session.Options {
			t.Errorf("session changed in one request lifetime")
		}
	})

	respWriter := httptest.NewRecorder()

	conf := &SessionConfig{
		Name: "test",
		Key:  "6bude5uOm9eZV280BjP6f6a5bEj7fg2PWl6GysY68CmXfOv8NFZ9O6ZIpbllQPtr",
	}
	handler := GenerateSessionStore(conf)

	handler(testHandler).ServeHTTP(respWriter, req)
}

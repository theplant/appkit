package sessions

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

var sessionStore *sessions.CookieStore

type session struct {
	w      http.ResponseWriter
	r      *http.Request
	config *Config
}

// Config the necessary configs for session
type Config struct {
	Name   string
	Key    string
	MaxAge int
	Secure bool
}

func (s session) setupSession() {
	if s.config.Name == "" {
		panic("session name must be present")
	}

	sessionStoreKey, err := base64.StdEncoding.DecodeString(s.config.Key)
	if err != nil {
		panic(err)
	}

	sessionStore = sessions.NewCookieStore(sessionStoreKey)

	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = s.config.Secure
	if s.config.MaxAge != 0 {
		sessionStore.MaxAge(s.config.MaxAge)
	}
}

type sessionContextKey int

const sessionCtxKey sessionContextKey = iota

// WithSession WithSession middleware generate session store for the whole request lifetime.
// later session operations should call `GetSession` to get the generated session store
func WithSession(conf *Config) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			se := newSession(w, r, conf)
			se.setupSession()
			// Record session storer in context
			ctx := context.WithValue(r.Context(), sessionCtxKey, se)

			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}
func newSession(w http.ResponseWriter, r *http.Request, config *Config) *session {
	return &session{w, r, config}
}

func getSession(ctx context.Context) *session {
	if se, ok := ctx.Value(sessionCtxKey).(*session); ok {
		return se
	}

	panic("Cannot get session in the request, please make sure the WithSession middleware has been executed before calling this function.")
}

// Get Get value of the given key in the session.
func Get(ctx context.Context, key string) (string, error) {
	s := getSession(ctx)
	session, err := sessionStore.Get(s.r, s.config.Name)
	if err != nil {
		return "", err
	}

	strInf, ok := session.Values[key]
	if !ok {
		return "", fmt.Errorf("Cannot find value for: %s", key)
	}

	str, ok := strInf.(string)
	if !ok {
		return "", fmt.Errorf("The value is not a string: %+v", strInf)
	}

	return str, nil
}

// Put put key-value map into the session
func Put(ctx context.Context, key, value string) error {
	s := getSession(ctx)
	session, err := sessionStore.Get(s.r, s.config.Name)
	if err != nil {
		return err
	}

	session.Values[key] = value
	session.Save(s.r, s.w)
	return nil
}

// Del delete value from the session by given key
func Del(ctx context.Context, key string) error {
	s := getSession(ctx)
	session, err := sessionStore.Get(s.r, s.config.Name)
	if err != nil {
		return err
	}

	delete(session.Values, key)
	session.Save(s.r, s.w)

	return nil
}

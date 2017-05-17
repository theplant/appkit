package sessions

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/theplant/appkit/errornotifier"
	"github.com/theplant/appkit/log"
)

var sessionStore *sessions.CookieStore

type session struct {
	w http.ResponseWriter
	r *http.Request
}

// SessionConfig the necessary configs for session
type SessionConfig struct {
	Name   string
	Key    string
	MaxAge int
	Secure bool
}

var config *SessionConfig
var notifier errornotifier.Notifier

func setupSession() {
	if config.Name == "" {
		panic("session name must be present")
	}

	sessionStoreKey, err := base64.StdEncoding.DecodeString(config.Key)
	if err != nil {
		panic(err)
	}

	sessionStore = sessions.NewCookieStore(sessionStoreKey)

	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = config.Secure
	if config.MaxAge != 0 {
		sessionStore.MaxAge(config.MaxAge)
	}
}

type sessionContextKey int

const sessionCtxKey sessionContextKey = 0

// GenerateSession GenerateSession middleware generate session store for the whole request lifetime.
// later session operations should call `GetSession` to get the generated session store
func GenerateSession(conf *SessionConfig, logger log.Logger) func(http.Handler) http.Handler {
	config = conf
	notifier = errornotifier.NewLogNotifier(logger)

	setupSession()

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			se := newSession(w, r)
			// Record session storer in context
			ctx := context.WithValue(r.Context(), sessionCtxKey, se)

			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		})
	}
}
func newSession(w http.ResponseWriter, r *http.Request) *session {
	return &session{w, r}
}

// GetSession GetSession fetch the session generated by GenerateSession
func GetSession(w http.ResponseWriter, r *http.Request) *session {
	if se, ok := r.Context().Value(sessionCtxKey).(*session); ok {
		return se
	}

	panic("Cannot get session in the request, please make sure the GenerateSession middleware has been executed before calling this function.")
}

func (s session) Get(key string) (string, bool) {
	session, err := sessionStore.Get(s.r, config.Name)
	if err != nil {
		notifier.Notify(err, nil)
		return "", false
	}

	strInf, ok := session.Values[key]
	if !ok {
		return "", false
	}

	str, ok := strInf.(string)
	if !ok {
		return "", false
	}

	return str, true
}

func (s session) Put(key, value string) {
	session, err := sessionStore.Get(s.r, config.Name)
	if err != nil {
		notifier.Notify(err, nil)
		return
	}

	session.Values[key] = value
	session.Save(s.r, s.w)
}

func (s session) Del(key string) {
	session, err := sessionStore.Get(s.r, config.Name)
	if err != nil {
		notifier.Notify(err, nil)
		return
	}

	delete(session.Values, key)
	session.Save(s.r, s.w)
}

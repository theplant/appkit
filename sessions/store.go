package sessions

import "github.com/gorilla/sessions"

// CookieStoreConfig is a general cookie storage configuration. It
// extends the `gorilla/sessions.Options` and compatible to work with
// `jinzhu/configor.Load` to be convenient to:
// * Set the least required fields
// * Enable the *Secure* and *HttpOnly* by default
// * Set default value for `Path` and `MaxAge`
//
// The `gorilla/sessions.Options`:
// https://github.com/gorilla/sessions/blob/7910f5bb5ac86ab08f97d8bda39b476fc117b684/sessions.go#L19-L34
type CookieStoreConfig struct {
	Name       string `required:"true"`
	Key        string `required:"true"`
	Domain     string
	Path       string `default:"/"`
	MaxAge     int    `default:"2592000"` // 2592000 = 30 * 24 * 60 * 60
	NoHTTPOnly bool
	NoSecure   bool
}

// NewCookieStore initializes a `gorilla/sessions.CookieStore` by
// `CookieStoreConfig`.
//
// The `gorilla/sessions.CookieStore`:
// https://github.com/gorilla/sessions/blob/7910f5bb5ac86ab08f97d8bda39b476fc117b684/store.go#L66-L70
func NewCookieStore(config CookieStoreConfig) *sessions.CookieStore {
	cs := sessions.NewCookieStore([]byte(config.Key))

	cs.Options = &sessions.Options{
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   config.MaxAge,
		Secure:   !config.NoSecure,
		HttpOnly: !config.NoHTTPOnly,
	}

	cs.MaxAge(cs.Options.MaxAge)

	return cs
}

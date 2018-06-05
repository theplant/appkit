package sessions_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
	"github.com/theplant/appkit/sessions"
)

func ExampleConfig_Default() {
	os.Setenv("TEST_APPKIT_SESSIONS_COOKIESTORE_Name", "COOKIESTORE_NAME")
	os.Setenv("TEST_APPKIT_SESSIONS_COOKIESTORE_Key", "COOKIESTORE_KEY")

	config, err := loadConfig()

	if err != nil {
		panic(errors.Wrap(err, "failed to load cookie store config"))
	}

	fmt.Printf("Name: %q\n", config.Name)
	fmt.Printf("Key: %q\n", config.Key)
	fmt.Printf("Path: %q\n", config.Path)
	fmt.Printf("Domain: %q\n", config.Domain)
	fmt.Printf("MaxAge: %d\n", config.MaxAge)
	fmt.Printf("NoHTTPOnly: %t\n", config.NoHTTPOnly)
	fmt.Printf("NoSecure: %t\n", config.NoSecure)

	// Output:
	// Name: "COOKIESTORE_NAME"
	// Key: "COOKIESTORE_KEY"
	// Path: "/"
	// Domain: ""
	// MaxAge: 2592000
	// NoHTTPOnly: false
	// NoSecure: false
}

func ExampleConfig_MissingName() {
	os.Unsetenv("TEST_APPKIT_SESSIONS_COOKIESTORE_Name")
	os.Setenv("TEST_APPKIT_SESSIONS_COOKIESTORE_Key", "secret")

	_, err := loadConfig()

	fmt.Println(err)

	// Output:
	// Name is required, but blank
}

func ExampleConfig_MissingKey() {
	os.Setenv("TEST_APPKIT_SESSIONS_COOKIESTORE_Name", "_cookiestore")
	os.Unsetenv("TEST_APPKIT_SESSIONS_COOKIESTORE_Key")

	_, err := loadConfig()

	fmt.Println(err)

	// Output:
	// Key is required, but blank
}

func loadConfig() (config sessions.Config, err error) {
	err = configor.New(&configor.Config{ENVPrefix: "TEST_APPKIT_SESSIONS_COOKIESTORE"}).Load(&config)

	return
}

func TestNewCookieStore(t *testing.T) {
	write := func(config sessions.Config) (cookie *http.Cookie, err error) {
		req, err := http.NewRequest("GET", "/", nil)

		if err != nil {
			return nil, errors.Wrapf(err, "can not initialize a request")
		}

		cookieStore := sessions.NewCookieStore(config)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := cookieStore.New(r, config.Name)

			if err != nil {
				panic(errors.Wrap(err, "can not initialize the session"))
			}

			if err := session.Save(r, w); err != nil {
				panic(errors.Wrap(err, "can not save the session"))
			}
		})

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		return recorder.Result().Cookies()[0], nil
	}

	runCase := func(config sessions.Config) func(t *testing.T) {
		return func(t *testing.T) {
			cookie, err := write(config)

			if err != nil {
				t.Fatalf("can not use cookieStore with config %+v: %v", config, err)
			}

			if got, want := cookie.Name, config.Name; got != want {
				t.Errorf("got Name = %q, but want %q", got, want)
			}
			if got, want := cookie.Path, config.Path; got != want {
				t.Errorf("got Path = %q, but want %q", got, want)
			}
			if got, want := cookie.Domain, config.Domain; got != want {
				t.Errorf("got Domain = %q, but want %q", got, want)
			}
			if got, want := cookie.MaxAge, config.MaxAge; got != want {
				t.Errorf("got MaxAge = %d, but want %d", got, want)
			}
			if got, want := cookie.HttpOnly, !config.NoHTTPOnly; got != want {
				t.Errorf("got HttpOnly = %t, but want %t", got, want)
			}
			if got, want := cookie.Secure, !config.NoSecure; got != want {
				t.Errorf("got Secure = %t, but want %t", got, want)
			}
		}
	}

	type C = sessions.Config

	t.Run("Default", runCase(C{Name: "cookiestore_name", Key: "secret"}))
	t.Run("Customized", runCase(C{Name: "name", Key: "secret", Path: "/cookie-path", Domain: "hello.local", MaxAge: 60, NoHTTPOnly: true, NoSecure: true}))
}

package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/cors"
	"github.com/theplant/appkit/log"
)

// CrossSiteConfig is configuration for cross-site request protection:
// - CSRF for writes
// - CORS for reads
type CrossSiteConfig struct {
	// RawAllowedOrigins is comma-separated list of hosts (with
	// `https://` prefix) that are allowed to make requests to the
	// server. Used to reject requests for CSRF, and to control
	// browser behaviour with CORS (deny access to response body).
	RawAllowedOrigins string `required:"true"`

	// AllowCredentials configures whether CORS requests are allowed to send "credentials":
	//
	// > Servers can also notify clients whether "credentials"
	// > (including Cookies and HTTP Authentication data) should be sent
	// > with requests
	//
	// (From https://en.wikipedia.org/wiki/Cross-origin_resource_sharing)
	AllowCredentials bool `required:"true"`

	// CSRFRequiredHeader will reject requests that do *not* have
	// this header set. The value of the header is ignored. This is an
	// additional layer of CSRF protection:
	//
	// 1. Without this header, requests will be rejected.
	//
	// 2. If JS on browser tries to include this header, it will
	//    trigger CORS policy validation by the browser.
	//
	// 3. Browser will make a CORS OPTIONS request, and if the origin
	//    isn't in the list of allowed origins, the browser will abort
	//    without making a real request.
	//
	// 4. If the origin *is* in the list of allowed origins, the
	//    browser will proceed with the real request.
	//
	CSRFRequiredHeader string `required:"true" default:"X-Csrf"`
}

// SecureMiddleware is middleware to (currently) enforce CORS and CSRF
// protection on requests to this service. OWASP CSRF
// recommendation[1] is:
//
// > General Recommendations For Automated CSRF Defense
// >
// > We recommend two separate checks as your standard CSRF defense that does not require user intervention. [...]
// >
// > 1. Check standard headers to verify the request is same origin
// > 2. AND Check CSRF token
//
// [1]: https://www.owasp.org/index.php/Cross-Site_Request_Forgery_(CSRF)_Prevention_Cheat_Sheet
func SecureMiddleware(logger log.Logger, cs CrossSiteConfig) Middleware {
	logger = logger.With("context", "appkit/server.SecureMiddleware")

	allowedOrigins := strings.Split(cs.RawAllowedOrigins, ",")
	for i, allowedOrigin := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigin)
	}

	return Compose(
		verifyHeader(logger, cs.CSRFRequiredHeader),
		verifyOrigin(allowedOrigins, logger),
		corsPolicy(allowedOrigins, cs, logger),
	)
}

// failCrossSiteRequest is a simple helper to fail the request in a
// consistent manner.
func failCrossSiteRequest(w http.ResponseWriter) {
	http.Error(w, http.StatusText(400), 400)
}

// verifyHeader ensures that the request contains a header with the
// same name as `csrfHeader`. If the header is missing, the request is
// terminated with `400 Bad Request`. This prevents drive-by form
// submission by requiring that requests are sent with a custom HTTP
// header. Custom headers cannot be set via plain HTML forms, and
// require Javascript. And cross-site Javascript will refuse to set
// this header due to the CORS policy.
//
// Note: are there any browsers that implement CORS but do *not* send `Origin` headers?
func verifyHeader(l log.Logger, csrfHeader string) Middleware {
	csrfHeader = http.CanonicalHeaderKey(csrfHeader)

	l = l.With("during", "appkit/server.verifyHeader")

	if csrfHeader == "" {
		// Info (not Warn) because this can be set to "" for
		// non-API-based apps (ie. ones that render HTML forms by
		// themselves)
		l.Info().Log("msg", "no CSRF header set, disabling header verification")
		return IdMiddleware
	}

	l.Info().Log(
		"msg", fmt.Sprintf(
			"Requests with no %v header will be rejected", csrfHeader),
		"csrf_header", csrfHeader,
	)

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			_, ok := req.Header[csrfHeader]
			if !ok {
				log.ForceContext(req.Context()).
					Warn().
					Log(
						"msg", "Request missing csrf header",
						"during", "appkit/server.verifyHeader",
					)

				failCrossSiteRequest(res)
				return
			}

			handler.ServeHTTP(res, req)
		})
	}
}

// verifyOrigin checks that the request's origin or referrer is in the
// list of allowed origins. This is mitigation against CSRF "confused
// deputy" attacks where a browser that is "authorised" on our site is
// tricked by another site into making requests.
func verifyOrigin(allowed []string, l log.Logger) Middleware {
	l = l.With("during", "appkit/server.verifyOrigin")

	if len(allowed) == 0 {
		l.Warn().Log("msg", "no allowed origins, disabling origin/referrer verification")
		return IdMiddleware
	}

	l.Info().Log(
		"msg", fmt.Sprintf("Cross-site requests allowed from origins: %v", allowed),
		"allowed_origins", strings.Join(allowed, ","),
	)

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			origin, ok := req.Header["Origin"]
			referrer := req.Header["Referer"]

			logger := log.ForceContext(req.Context()).With("during", "appkit/server.verifyOrigin")

			if !ok {
				logger.Warn().Log(
					"msg", "No origin header, falling back to referrer",
					"referrer", strings.Join(referrer, ","),
				)

				origin = []string{}

				for _, r := range referrer {
					u, err := url.Parse(r)
					if err == nil && u != nil && u.Host != "" && u.Scheme != "" {
						ref := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
						origin = append(origin, ref)
					}
				}
			}

			if len(origin) == 0 {
				logger.Warn().Log("msg", "No origin or referrer for request")
			}

			for _, a := range allowed {
				for _, o := range origin {
					if a == o {
						handler.ServeHTTP(res, req)
						return
					}
				}
			}

			logger.Error().Log("msg", "CSRF failure: origin/referrer does not match target origin",
				"allowed_origins", strings.Join(allowed, ","),
				"origin", strings.Join(origin, ","),
				"referrer", strings.Join(referrer, ","),
			)

			failCrossSiteRequest(res)
		})
	}
}

// corsPolicy will use github.com/rs/cors to define a CORS policy for
// the system, based on the CrossSiteConfig
func corsPolicy(allowedOrigins []string, cs CrossSiteConfig, l log.Logger) Middleware {
	l.Info().Log(
		"msg", fmt.Sprintf("CORS: allowed at origins: %v, allowed with credentials: %v, allowed CSRF header %v", allowedOrigins, cs.AllowCredentials, cs.CSRFRequiredHeader),
		"during", "appkit/server.corsPolicy",
		"allowed_origins", strings.Join(allowedOrigins, ","),
		"allow_credentials", cs.AllowCredentials,
		"allowed_headers", cs.CSRFRequiredHeader,
	)

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: cs.AllowCredentials,
		AllowedHeaders:   []string{cs.CSRFRequiredHeader},
	})

	return c.Handler
}

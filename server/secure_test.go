package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/theplant/appkit/log"
)

//////////////////////////////////////////////////
// CSRF Required Header Verification

func ExampleCSRFHeader_Blank() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "", // Explicitly blank
	}

	s, req := setup(cfg)

	req.Header.Add("origin", "http://example.com")

	exec(s, req)

	// Output: executed handler
	// 200
}

func ExampleCSRFHeader_Set() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "X-Csrf", // Same as default
	}

	s, req := setup(cfg)

	req.Header.Add("origin", "http://example.com")
	req.Header.Add("X-Csrf", ".") // Value is ignored

	exec(s, req)

	// Output: executed handler
	// 200
}

func ExampleCSRFHeader_Missing() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "X-Csrf", // Same as default
	}

	s, req := setup(cfg)

	req.Header.Add("origin", "http://example.com")

	exec(s, req)

	// Output: level=warn msg="Request missing csrf header" during=appkit/server.verifyHeader
	// 400
}

func ExampleCSRFHeader_CaseSensitivity() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "X-Csrf", // Same as default
	}

	s, req := setup(cfg)

	req.Header.Add("origin", "http://example.com")
	req.Header.Add("X-CSRF", ".") // Header is all uppercase

	exec(s, req)

	// Output: executed handler
	// 200
}

//////////////////////////////////////////////////
// Origin Verification

func ExampleOrigin() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com",
	}

	s, req := setup(cfg)

	req.Header.Add("origin", "http://example.com")

	exec(s, req)

	// Output: executed handler
	// 200
}

func ExampleOrigin_MultipleAllowedOrigins() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com,http://2.example.com",
	}

	s, req := setup(cfg)

	req.Header.Add("origin", "http://example.com")

	exec(s, req)

	req.Header.Add("origin", "http://2.example.com")

	exec(s, req)

	// Output: executed handler
	// 200
	// executed handler
	// 200
}

func ExampleOrigin_ReferrerFallback() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com",
	}

	s, req := setup(cfg)

	req.Header.Add("referer", "http://example.com/a/referer")

	exec(s, req)

	// Output: level=warn during=appkit/server.verifyOrigin msg="No origin header, falling back to referrer" referrer=http://example.com/a/referer
	// executed handler
	// 200
}

func ExampleOrigin_MissingOriginAndReferrer() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com,http://2.example.com",
	}

	s, req := setup(cfg)

	exec(s, req)

	// Output: level=warn during=appkit/server.verifyOrigin msg="No origin header, falling back to referrer" referrer=
	// level=warn during=appkit/server.verifyOrigin msg="No origin or referrer for request"
	// level=error during=appkit/server.verifyOrigin msg="CSRF failure: origin/referrer does not match target origin" allowed_origins=http://example.com,http://2.example.com origin= referrer=
	// 400
}

func ExampleOrigin_InvalidReferrerFallback() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com",
	}

	s, req := setup(cfg)

	req.Header.Add("referer", "not a valid referer url")

	exec(s, req)

	// Output: level=warn during=appkit/server.verifyOrigin msg="No origin header, falling back to referrer" referrer="not a valid referer url"
	// level=warn during=appkit/server.verifyOrigin msg="No origin or referrer for request"
	// level=error during=appkit/server.verifyOrigin msg="CSRF failure: origin/referrer does not match target origin" allowed_origins=http://example.com origin= referrer="not a valid referer url"
	// 400
}

func ExampleOrigin_OverlappingReferrerFallback() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com",
	}

	s, req := setup(cfg)

	req.Header.Add("referer", "http://example.com.evil.com/path")

	exec(s, req)

	// Output: level=warn during=appkit/server.verifyOrigin msg="No origin header, falling back to referrer" referrer=http://example.com.evil.com/path
	//level=error during=appkit/server.verifyOrigin msg="CSRF failure: origin/referrer does not match target origin" allowed_origins=http://example.com origin=http://example.com.evil.com referrer=http://example.com.evil.com/path
	// 400
}

func ExampleCors() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com",
	}

	s, req := setup(cfg)

	req.Method = "OPTIONS"

	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("origin", "http://example.com")

	printCORSHeaders(exec(s, req))

	// Output: 200
	// Vary: [Origin Access-Control-Request-Method Access-Control-Request-Headers]
	// Access-Control-Allow-Origin: [http://example.com]
	// Access-Control-Allow-Methods: [POST]
	// Access-Control-Allow-Headers: []
	// Access-Control-Allow-Credentials: []
	// Access-Control-Max_age: []
}

func ExampleCors_MultipleOrigins() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins: "http://example.com,http://2.example.com",
	}

	s, req := setup(cfg)

	req.Method = "OPTIONS"

	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("origin", "http://example.com")

	printCORSHeaders(exec(s, req))

	s, req = setup(cfg)

	req.Method = "OPTIONS"

	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("origin", "http://2.example.com")

	printCORSHeaders(exec(s, req))

	// Output: 200
	// Vary: [Origin Access-Control-Request-Method Access-Control-Request-Headers]
	// Access-Control-Allow-Origin: [http://example.com]
	// Access-Control-Allow-Methods: [POST]
	// Access-Control-Allow-Headers: []
	// Access-Control-Allow-Credentials: []
	// Access-Control-Max_age: []
	// 200
	// Vary: [Origin Access-Control-Request-Method Access-Control-Request-Headers]
	// Access-Control-Allow-Origin: [http://2.example.com]
	// Access-Control-Allow-Methods: [POST]
	// Access-Control-Allow-Headers: []
	// Access-Control-Allow-Credentials: []
	// Access-Control-Max_age: []
}

func ExampleCors_NoOrigin() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "",
	}

	s, req := setup(cfg)

	req.Method = "OPTIONS"

	req.Header.Add("Access-Control-Request-Method", "POST")

	printCORSHeaders(exec(s, req))

	// Output: 200
	// Vary: [Origin Access-Control-Request-Method Access-Control-Request-Headers]
	// Access-Control-Allow-Origin: []
	// Access-Control-Allow-Methods: []
	// Access-Control-Allow-Headers: []
	// Access-Control-Allow-Credentials: []
	// Access-Control-Max_age: []
}

func ExampleCors_DisallowedOrigin() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "",
	}

	s, req := setup(cfg)

	req.Method = "OPTIONS"

	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("origin", "http://evil.com")

	printCORSHeaders(exec(s, req))

	// Output: 200
	// Vary: [Origin Access-Control-Request-Method Access-Control-Request-Headers]
	// Access-Control-Allow-Origin: []
	// Access-Control-Allow-Methods: []
	// Access-Control-Allow-Headers: []
	// Access-Control-Allow-Credentials: []
	// Access-Control-Max_age: []
}

func ExampleCors_CSRFHeaderRequest() {
	cfg := CrossSiteConfig{
		RawAllowedOrigins:  "http://example.com",
		CSRFRequiredHeader: "X-Csrf",
	}

	s, req := setup(cfg)

	req.Method = "OPTIONS"

	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("Access-Control-Request-Headers", "X-Csrf")
	req.Header.Add("origin", "http://example.com")

	printCORSHeaders(exec(s, req))

	// Output: 200
	// Vary: [Origin Access-Control-Request-Method Access-Control-Request-Headers]
	// Access-Control-Allow-Origin: [http://example.com]
	// Access-Control-Allow-Methods: [POST]
	// Access-Control-Allow-Headers: [X-Csrf]
	// Access-Control-Allow-Credentials: []
	// Access-Control-Max_age: []
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("executed handler")
}

func handler(cfg CrossSiteConfig) http.Handler {
	// Use no-op logger to ignore configuration logging. Request
	// logging will use NewTestLogger.
	l := log.NewNopLogger()

	middleware := Compose(
		SecureMiddleware(l, cfg),
		log.WithLogger(log.NewTestLogger()),
	)

	return middleware(http.HandlerFunc(testHandler))
}

func setup(cfg CrossSiteConfig) (*httptest.Server, *http.Request) {

	s := httptest.NewServer(handler(cfg))

	req, err := http.NewRequest("POST", s.URL, nil)
	if err != nil {
		panic(err)
	}

	return s, req
}

func exec(s *httptest.Server, req *http.Request) *http.Response {
	resp, err := s.Client().Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.StatusCode)

	return resp
}

func printCORSHeaders(resp *http.Response) {
	headers := []string{
		"Vary",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Credentials",
		"Access-Control-Max_age",
	}

	for _, h := range headers {
		fmt.Printf("%s: %+v\n", h, resp.Header[h])
	}
}

/*
Package cors sets the appropriate CORS(https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
response headers, and lets you customize. Following customizations are allowed:
  - provide a list of allowed domains
  - provide a list of headers
  - set the max-age of CORS headers

The list of allowed methods are
*/
package cors

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/naughtygopher/webgo/v7"
)

func TestCORSEmptyconfig(t *testing.T) {
	port := "9696"
	routes := getRoutes()
	routes = append(routes, AddOptionsHandlers(nil)...)
	router, err := setup(port, routes)
	if err != nil {
		t.Error(err.Error())
		return
	}
	router.Use(CORS(&Config{TimeoutSecs: 50}))
	router.SetupMiddleware()

	url := fmt.Sprintf("http://localhost:%s/hello", port)
	w := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		url,
		nil,
	)

	router.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Body)
	str := string(body)
	if str != "hello" {
		t.Errorf(
			"Expected body '%s', got '%s'",
			"hello",
			str,
		)
	}

	if w.Header().Get(headerMethods) != defaultAllowMethods {
		t.Errorf(
			"Expected header %s to be '%s', got '%s'",
			headerMethods,
			defaultAllowMethods,
			w.Header().Get(headerMethods),
		)
	}
	if w.Header().Get(headerCreds) != "true" {
		t.Errorf(
			"Expected header %s to be 'true', got '%s'",
			headerCreds,
			w.Header().Get(headerCreds),
		)
	}
	if w.Header().Get(headerAccessControlAge) != "50" {
		t.Errorf(
			"Expected '%s' to be '50', got '%s'",
			headerAccessControlAge,
			w.Header().Get(headerAccessControlAge),
		)
	}

	if w.Header().Get(headerAllowHeaders) != allowHeaders {
		t.Errorf(
			"Expected '%s' to be '%s', got '%s'",
			headerAllowHeaders,
			allowHeaders,
			w.Header().Get(headerAllowHeaders),
		)
	}

	// check OPTIONS method
	w = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodOptions,
		url,
		nil,
	)
	router.ServeHTTP(w, req)
	body, _ = io.ReadAll(w.Body)
	str = string(body)
	if str != "" {
		t.Errorf(
			"Expected empty body, got '%s'",
			str,
		)
	}

}

func TestCORSWithConfig(t *testing.T) {
	port := "9696"
	routes := AddOptionsHandlers(getRoutes())
	router, err := setup(port, routes)
	if err != nil {
		t.Error(err.Error())
		return
	}

	cfg := &Config{
		Routes:         routes,
		AllowedOrigins: []string{"example.com", fmt.Sprintf("localhost:%s", port)},
		AllowedHeaders: []string{"x-custom"},
	}
	router.Use(CORS(cfg))

	baseAPI := fmt.Sprintf("http://localhost:%s", port)
	url := fmt.Sprintf("%s/hello", baseAPI)
	w := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		url,
		nil,
	)

	router.SetupMiddleware()
	router.ServeHTTP(w, req)

	if w.Header().Get(headerMethods) != "GET,OPTIONS" {
		t.Errorf(
			"Expected value for %s header is 'GET', got '%s'",
			headerMethods,
			w.Header().Get(headerMethods),
		)
	}

	want := strings.Join(cfg.AllowedHeaders, ",") + ","
	if w.Header().Get(headerAllowHeaders) != want {
		t.Errorf(
			"Expected value for %s header is '%s', got '%s'",
			headerAllowHeaders,
			want,
			w.Header().Get(headerAllowHeaders),
		)
	}

	// test OPTIONS request
	w = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodOptions,
		url,
		nil,
	)

	req.Header.Set("Origin", "helloworld.com")
	router.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Body)
	str := string(body)
	if str != "" {
		t.Errorf(
			"Expected empty body, got '%s'",
			str,
		)
	}
	// since origin is set as "helloworld.com",  which is not in the allowed list of origins
	// CORS headers should NOT be set
	if w.Header().Get(headerOrigin) != "" {
		t.Errorf(
			"Expected empty value for header '%s', got '%s'",
			headerOrigin,
			w.Header().Get(headerOrigin),
		)
	}
	if w.Header().Get(headerAccessControlAge) != "" {
		t.Errorf(
			"Expected empty value for header '%s', got '%s'",
			headerAccessControlAge,
			w.Header().Get(headerAccessControlAge),
		)
	}
	if w.Header().Get(headerCreds) != "" {
		t.Errorf(
			"Expected empty value for header '%s', got '%s'",
			headerCreds,
			w.Header().Get(headerCreds),
		)
	}
	if w.Header().Get(headerMethods) != "" {
		t.Errorf(
			"Expected empty value for header '%s', got '%s'",
			headerMethods,
			w.Header().Get(headerMethods),
		)
	}
	if w.Header().Get(headerAllowHeaders) != "" {
		t.Errorf(
			"Expected empty value for header '%s', got '%s'",
			headerAllowHeaders,
			w.Header().Get(headerAllowHeaders),
		)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`hello`))
}

func getRoutes() []*webgo.Route {
	return []*webgo.Route{
		{
			Name:     "hello",
			Pattern:  "/hello",
			Method:   http.MethodGet,
			Handlers: []http.HandlerFunc{handler},
		},
	}
}
func setup(port string, routes []*webgo.Route) (*webgo.Router, error) {
	cfg := &webgo.Config{
		Port:            "9696",
		ReadTimeout:     time.Second * 1,
		WriteTimeout:    time.Second * 1,
		ShutdownTimeout: time.Second * 10,
		CertFile:        "tests/ssl/server.crt",
		KeyFile:         "tests/ssl/server.key",
	}
	router := webgo.NewRouter(cfg, routes...)

	return router, nil
}

package webgo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterConvenienceMethods(t *testing.T) {
	t.Parallel()

	methods := []struct {
		name   string
		method string
		add    func(*Router)
	}{
		{"get", http.MethodGet, func(r *Router) { r.Get("/get", []http.HandlerFunc{methodHandler(http.MethodGet)}) }},
		{"head", http.MethodHead, func(r *Router) { r.Head("/head", []http.HandlerFunc{methodHandler(http.MethodHead)}) }},
		{"post", http.MethodPost, func(r *Router) { r.Post("/post", []http.HandlerFunc{methodHandler(http.MethodPost)}) }},
		{"put", http.MethodPut, func(r *Router) { r.Put("/put", []http.HandlerFunc{methodHandler(http.MethodPut)}) }},
		{"patch", http.MethodPatch, func(r *Router) { r.Patch("/patch", []http.HandlerFunc{methodHandler(http.MethodPatch)}) }},
		{"delete", http.MethodDelete, func(r *Router) { r.Delete("/delete", []http.HandlerFunc{methodHandler(http.MethodDelete)}) }},
		{"options", http.MethodOptions, func(r *Router) { r.Options("/options", []http.HandlerFunc{methodHandler(http.MethodOptions)}) }},
	}

	for _, tc := range methods {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			router := NewRouter(&Config{})
			tc.add(router)

			path := "/" + tc.name
			req := httptest.NewRequest(tc.method, path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
			body, err := io.ReadAll(rec.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != tc.method {
				t.Fatalf("expected body %q, got %q", tc.method, string(body))
			}
		})
	}
}

func TestRouterConvenienceMethodOptions(t *testing.T) {
	t.Parallel()

	router := NewRouter(&Config{})
	router.Get(
		"/named/:id",
		[]http.HandlerFunc{paramHandler},
		RouteName("get-by-id"),
		TrailingSlash(true),
	)

	req := httptest.NewRequest(http.MethodGet, "/named/42/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "42" {
		t.Fatalf("expected param value 42, got %q", string(body))
	}
}

func methodHandler(method string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Send(w, "text/plain", method, http.StatusOK)
	}
}

func paramHandler(w http.ResponseWriter, r *http.Request) {
	ctx := Context(r)
	Send(w, "text/plain", ctx.URIParams["id"], http.StatusOK)
}

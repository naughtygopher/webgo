/*
Package webgo is a lightweight framework for building web apps. It has a multiplexer,
middleware plugging mechanism & context management of its own. The primary goal
of webgo is to get out of the developer's way as much as possible. i.e. it does
not enforce you to build your app in any particular pattern, instead just helps you
get all the trivial things done faster and easier.

e.g.
1. Getting named URI parameters.
2. Multiplexer for regex matching of URI and such.
3. Inject special app level configurations or any such objects to the request context as required.
*/
package webgo

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResponseStatus(t *testing.T) {
	t.Parallel()
	w := newCRW(httptest.NewRecorder(), http.StatusOK)
	SendError(w, nil, http.StatusNotFound)
	if http.StatusNotFound != ResponseStatus(w) {
		t.Errorf(
			"Expected status '%d', got '%d'",
			http.StatusNotFound,
			ResponseStatus(w),
		)
	}

	// ideally we should get 200 from ResponseStatus; but it can get accurate status code only
	// when `customresponsewriter` is used
	rw := httptest.NewRecorder()
	SendError(rw, nil, http.StatusNotFound)
	if http.StatusOK != ResponseStatus(rw) {
		t.Errorf(
			"Expected status '%d', got '%d'",
			http.StatusOK,
			ResponseStatus(rw),
		)
	}
}

func TestStart(t *testing.T) {
	t.Parallel()
	router, _ := setup(t, "9696")
	go router.Start()
	time.Sleep(time.Second * 2)
	err := router.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
}
func TestStartHTTPS(t *testing.T) {
	t.Parallel()
	router, _ := setup(t, "8443")
	go router.StartHTTPS()
	time.Sleep(time.Second * 2)
	err := router.ShutdownHTTPS()
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()
	err := errors.New("hello world, failed")
	router, _ := setup(t, "7878")
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, r)

	SetError(r, err)
	gotErr := GetError(r)

	if !errors.Is(err, gotErr) {
		t.Fatalf("expected err %v, got %v", err, gotErr)
	}
}

func BenchmarkRouter(b *testing.B) {
	GlobalLoggerConfig(nil, nil, LogCfgDisableDebug, LogCfgDisableInfo, LogCfgDisableWarn)
	t := &testing.T{}
	router, err := setup(t, "1595")
	if err != nil {
		b.Error(err)
		return
	}
	w := httptest.NewRecorder()
	base, _ := http.NewRequest(http.MethodGet, "/a/b/-/c/~/d/./e", nil)

	b.ReportAllocs()
	for b.Loop() {
		// ServeHTTP overwrites the request with a copy holding the webgo context,
		// so the request is restored on every iteration to avoid nesting contexts
		r := *base
		router.ServeHTTP(w, &r)
		if w.Result().StatusCode != http.StatusOK {
			b.Error("expected status 200, got", w.Result().StatusCode)
			return
		}
	}
}

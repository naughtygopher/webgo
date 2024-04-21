package webgo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRouter_ServeHTTP(t *testing.T) {
	t.Parallel()
	port := "9696"
	router, err := setup(t, port)
	if err != nil {
		t.Error(err.Error())
		return
	}
	m := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		w.Header().Add("middleware", "true")
		next(w, r)
	}
	router.Use(m)
	router.UseOnSpecialHandlers(m)
	router.SetupMiddleware()

	list := testTable()

	baseAPI := fmt.Sprintf("http://localhost:%s", port)

	for _, l := range list {
		url := baseAPI
		if l.Path != "" {
			switch l.TestType {
			case "checkpath",
				"checkpathnotrailingslash",
				"chaining",
				"notfound",
				"chaining-nofallthrough":
				{
					url = strings.Join([]string{url, l.Path}, "")
				}
			case "checkparams", "widlcardwithouttrailingslash":
				{
					for idx, key := range l.ParamKeys {
						// in case of wildcard params, they have to be replaced first for proper URL construction
						l.Path = strings.Replace(l.Path, ":"+key+"*", l.Params[idx], 1)
						l.Path = strings.Replace(l.Path, ":"+key, l.Params[idx], 1)
					}
					url = strings.Join([]string{url, l.Path}, "")
				}
			}
		}

		respRec := httptest.NewRecorder()
		req := httptest.NewRequest(
			l.Method,
			url,
			l.Body,
		)
		router.ServeHTTP(respRec, req)

		switch l.TestType {
		case "checkpath", "checkpathnotrailingslash":
			{
				err = checkPath(req, respRec)
			}
		case "widlcardwithouttrailingslash":
			{
				err = checkPathWildCard(req, respRec)
			}
		case "chaining":
			{
				err = checkChaining(req, respRec)
			}
		case "checkparams":
			{
				err = checkParams(req, respRec, l.ParamKeys, l.Params)
			}
		case "notimplemented":
			{
				err = checkNotImplemented(req, respRec)
			}
		case "notfound":
			{
				err = checkNotFound(req, respRec)
			}
		}

		if err != nil && !l.WantErr {
			t.Errorf(
				"'%s' (%s '%s'): %s",
				l.Name,
				l.Method,
				url,
				err.Error(),
			)
			if l.Err != nil {
				if !errors.Is(err, l.Err) {
					t.Errorf(
						"expected error '%s', got %s",
						l.Err.Error(),
						err.Error(),
					)
				}
			}
		} else if err == nil && l.WantErr {
			t.Errorf(
				"'%s' (%s '%s') expected error, but received nil",
				l.Name,
				l.Method,
				url,
			)
		}

		err = checkMiddleware(req, respRec)
		if err != nil {
			t.Error(err.Error())
		}
	}
}

func setup(t *testing.T, port string) (*Router, error) {
	t.Helper()
	cfg := &Config{
		Port:            port,
		ReadTimeout:     time.Second * 1,
		WriteTimeout:    time.Second * 1,
		ShutdownTimeout: time.Second * 10,
		CertFile:        "tests/ssl/server.crt",
		KeyFile:         "tests/ssl/server.key",
	}
	router := NewRouter(cfg, getRoutes(t)...)
	return router, nil
}

func getRoutes(t *testing.T) []*Route {
	t.Helper()

	list := testTable()
	rr := make([]*Route, 0, len(list))
	for _, l := range list {
		switch l.TestType {
		case "checkpath", "checkparams", "checkparamswildcard":
			{
				rr = append(rr,
					&Route{
						Name:                    l.Name,
						Method:                  l.Method,
						Pattern:                 l.Path,
						TrailingSlash:           true,
						FallThroughPostResponse: false,
						Handlers:                []http.HandlerFunc{successHandler},
					},
				)
			}
		case "checkpathnotrailingslash", "widlcardwithouttrailingslash":
			{
				rr = append(rr,
					&Route{
						Name:                    l.Name,
						Method:                  l.Method,
						Pattern:                 l.Path,
						TrailingSlash:           false,
						FallThroughPostResponse: false,
						Handlers:                []http.HandlerFunc{successHandler},
					},
				)

			}

		case "chaining":
			{
				rr = append(
					rr,
					&Route{
						Name:                    l.Name,
						Method:                  l.Method,
						Pattern:                 l.Path,
						TrailingSlash:           false,
						FallThroughPostResponse: false,
						Handlers:                []http.HandlerFunc{chainHandler, successHandler},
					},
				)
			}
		case "chaining-nofallthrough":
			{
				{
					rr = append(
						rr,
						&Route{
							Name:                    l.Name,
							Method:                  l.Method,
							Pattern:                 l.Path,
							TrailingSlash:           false,
							FallThroughPostResponse: false,
							Handlers:                []http.HandlerFunc{chainHandler, chainNoFallthroughHandler, successHandler},
						},
					)
				}
			}
		}
	}
	return rr
}

func chainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("chained", "true")
}

func chainNoFallthroughHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("chained", "true")
	_, _ = w.Write([]byte(`yay, blocked!`))
}

func successHandler(w http.ResponseWriter, r *http.Request) {
	wctx := Context(r)
	params := wctx.Params()
	R200(
		w,
		map[string]interface{}{
			"path":   r.URL.Path,
			"params": params,
		},
	)
}

func checkPath(req *http.Request, resp *httptest.ResponseRecorder) error {
	want := req.URL.EscapedPath()
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response, '%s'", err.Error())
	}

	body := struct {
		Data struct {
			Path   string
			Params map[string]string
		}
	}{}
	err = json.Unmarshal(rbody, &body)
	if err != nil {
		return fmt.Errorf(
			"json decode failed '%s', got response: '%s'",
			err.Error(),
			string(rbody),
		)
	}

	if want != body.Data.Path {
		return fmt.Errorf("wanted URI path '%s', got '%s'", want, body.Data.Path)
	}

	return nil
}

func checkPathWildCard(req *http.Request, resp *httptest.ResponseRecorder) error {
	want := req.URL.EscapedPath()
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response, '%s'", err.Error())
	}

	body := struct {
		Data struct {
			Path   string
			Params map[string]string
		}
	}{}
	err = json.Unmarshal(rbody, &body)
	if err != nil {
		return fmt.Errorf("json decode failed '%s', got response: '%s'", err.Error(), string(rbody))
	}

	if want != body.Data.Path {
		return fmt.Errorf("wanted URI path '%s', got '%s'", want, body.Data.Path)
	}

	if len(body.Data.Params) != 1 {
		return fmt.Errorf("expected no.of params: %d, got %d. response: '%s'", 1, len(body.Data.Params), string(rbody))
	}

	wantWildcardParamValue := ""
	parts := strings.Split(want, "/")[2:]
	wantWildcardParamValue = strings.Join(parts, "/")
	if body.Data.Params["a"] != wantWildcardParamValue {
		return fmt.Errorf(
			"wildcard value\nexpected: %s\ngot: %s",
			wantWildcardParamValue,
			body.Data.Params["a"],
		)
	}

	return nil
}

func checkParams(req *http.Request, resp *httptest.ResponseRecorder, keys []string, expected []string) error {
	rbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response, '%s'", err.Error())
	}

	body := struct {
		Data struct {
			Params map[string]string
		}
	}{}
	err = json.Unmarshal(rbody, &body)
	if err != nil {
		return fmt.Errorf("json decode failed '%s', for response '%s'", err.Error(), string(rbody))
	}

	for idx, key := range keys {
		want := expected[idx]
		if body.Data.Params[key] != want {
			return fmt.Errorf(
				"expected value for '%s' is '%s', got '%s'",
				key,
				want,
				body.Data.Params[key],
			)
		}
	}

	return nil
}

func checkNotImplemented(req *http.Request, resp *httptest.ResponseRecorder) error {
	if resp.Result().StatusCode != http.StatusNotImplemented {
		return fmt.Errorf(
			"expected code %d, got %d",
			http.StatusNotImplemented,
			resp.Code,
		)
	}
	return nil
}

func checkNotFound(req *http.Request, resp *httptest.ResponseRecorder) error {
	if resp.Result().StatusCode != http.StatusNotFound {
		return fmt.Errorf(
			"expected code %d, got %d",
			http.StatusNotFound,
			resp.Code,
		)
	}
	return nil
}

func checkChaining(req *http.Request, resp *httptest.ResponseRecorder) error {
	if resp.Header().Get("chained") != "true" {
		return fmt.Errorf(
			"Expected header value for 'chained', to be 'true', got '%s'",
			resp.Header().Get("chained"),
		)
	}
	return nil
}

func checkMiddleware(req *http.Request, resp *httptest.ResponseRecorder) error {
	if resp.Header().Get("middleware") != "true" {
		return fmt.Errorf(
			"Expected header value for 'middleware', to be 'true', got '%s'",
			resp.Header().Get("middleware"),
		)
	}
	return nil
}

func testTable() []struct {
	Name      string
	TestType  string
	Path      string
	Method    string
	Want      interface{}
	WantErr   bool
	Err       error
	ParamKeys []string
	Params    []string
	Body      io.Reader
} {
	return []struct {
		Name      string
		TestType  string
		Path      string
		Method    string
		Want      interface{}
		WantErr   bool
		Err       error
		ParamKeys []string
		Params    []string
		Body      io.Reader
	}{
		{
			Name:     "Check root path without params",
			TestType: "checkpath",
			Path:     "/",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check root path without params - duplicate",
			TestType: "checkpath",
			Path:     "/",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - 1",
			TestType: "checkpath",
			Path:     "/a",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - 2",
			TestType: "checkpath",
			Path:     "/a/b",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - 3",
			TestType: "checkpath",
			Path:     "/a/b/-/c",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - 4",
			TestType: "checkpath",
			Path:     "/a/b/-/c/~/d",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - 5",
			TestType: "checkpath",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - 5",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e/notrail",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - OPTION",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodOptions,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - HEAD",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodHead,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - POST",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodPost,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - PUT",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodPut,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - PATCH",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodPatch,
			WantErr:  false,
		},
		{
			Name:     "Check nested path without params - DELETE",
			TestType: "checkpathnotrailingslash",
			Path:     "/a/b/-/c/~/d/./e",
			Method:   http.MethodDelete,
			WantErr:  false,
		},
		{
			Name:      "Check with params - 1",
			TestType:  "checkparams",
			Path:      "/params/:a",
			Method:    http.MethodGet,
			ParamKeys: []string{"a"},
			Params:    []string{"hello"},
			WantErr:   false,
		},
		{
			Name:      "Check with params - 2",
			TestType:  "checkparams",
			Path:      "/params/:a/:b",
			Method:    http.MethodGet,
			ParamKeys: []string{"a", "b"},
			Params:    []string{"hello", "world"},
			WantErr:   false,
		},
		{
			Name:      "Check with wildcard",
			TestType:  "checkparams",
			Path:      "/wildcard/:a*",
			Method:    http.MethodGet,
			ParamKeys: []string{"a"},
			Params:    []string{"w1/hello/world/hi/there"},
			WantErr:   false,
		},
		{
			Name:      "Check with wildcard - 2",
			TestType:  "checkparams",
			Path:      "/wildcard2/:a*",
			Method:    http.MethodGet,
			ParamKeys: []string{"a"},
			Params:    []string{"w2/hello/world/hi/there/-/~/./again"},
			WantErr:   false,
		},
		{
			Name:      "Check with wildcard - 3",
			TestType:  "widlcardwithouttrailingslash",
			Path:      "/wildcard3/:a*",
			Method:    http.MethodGet,
			ParamKeys: []string{"a"},
			Params:    []string{"w3/hello/world/hi/there/-/~/./again/"},
			WantErr:   true,
		},
		{
			Name:      "Check with wildcard - 4",
			TestType:  "widlcardwithouttrailingslash",
			Path:      "/wildcard3/:a*",
			Method:    http.MethodGet,
			ParamKeys: []string{"a"},
			Params:    []string{"w4/hello/world/hi/there/-/~/./again"},
			WantErr:   false,
		},
		{
			Name:     "Check not implemented",
			TestType: "notimplemented",
			Path:     "/notimplemented",
			Method:   "HELLO",
			WantErr:  false,
		},
		{
			Name:     "Check not found",
			TestType: "notfound",
			Path:     "/notfound",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check chaining",
			TestType: "chaining",
			Path:     "/chained",
			Method:   http.MethodGet,
			WantErr:  false,
		},
		{
			Name:     "Check chaining",
			TestType: "chaining-nofallthrough",
			Path:     "/chained/nofallthrough",
			Method:   http.MethodGet,
			WantErr:  false,
		},
	}
}

type testLogger struct {
	out bytes.Buffer
}

func (tl *testLogger) Debug(data ...interface{}) {
	tl.out.Write([]byte(fmt.Sprint(data...)))
}
func (tl *testLogger) Info(data ...interface{}) {
	tl.out.Write([]byte(fmt.Sprint(data...)))
}
func (tl *testLogger) Warn(data ...interface{}) {
	tl.out.Write([]byte(fmt.Sprint(data...)))
}
func (tl *testLogger) Error(data ...interface{}) {
	tl.out.Write([]byte(fmt.Sprint(data...)))
}
func (tl *testLogger) Fatal(data ...interface{}) {
	tl.out.Write([]byte(fmt.Sprint(data...)))
}

func Test_httpHandlers(t *testing.T) {
	// t.Parallel()
	tl := &testLogger{
		out: bytes.Buffer{},
	}
	LOGHANDLER = tl

	// test invalid method
	httpHandlers(
		[]*Route{
			{
				Name:    "invalid method",
				Pattern: "/hello/world",
				Method:  "HELLO",
			},
		})
	got := tl.out.String()
	want := "Unsupported HTTP method provided. Method: 'HELLO'"
	if got != want {
		t.Errorf(
			"Expected the error to end with '%s', got '%s'",
			want,
			got,
		)
	}
	tl.out.Reset()

	// test empty handlers
	httpHandlers(
		[]*Route{
			{
				Name:    "empty handlers",
				Pattern: "/hello/world",
				Method:  http.MethodGet,
			},
		})
	str := tl.out.String()
	want = "provided for the route '/hello/world', method 'GET'"
	got = str[len(str)-len(want):]
	if got != want {
		t.Errorf(
			"Expected the error to end with '%s', got '%s'",
			want,
			got,
		)
	}
	tl.out.Reset()
}

func TestWildcardMadness(t *testing.T) {
	port := "9696"
	t.Helper()
	cfg := &Config{
		Port:            port,
		ReadTimeout:     time.Second * 1,
		WriteTimeout:    time.Second * 1,
		ShutdownTimeout: time.Second * 10,
		CertFile:        "tests/ssl/server.crt",
		KeyFile:         "tests/ssl/server.key",
	}
	router := NewRouter(cfg, []*Route{
		{
			Name:          "wildcard madness",
			Pattern:       "/hello/:w*/world/:p1/:w2*/hi/there",
			Handlers:      []http.HandlerFunc{successHandler},
			Method:        http.MethodGet,
			TrailingSlash: true,
		},
	}...)

	baseAPI := fmt.Sprintf("http://localhost:%s", port)
	url := fmt.Sprintf(
		"%s%s",
		baseAPI,
		"/hello/a/b/c/-d/~/e/world/fgh/i/j/k~/l-/hi/there/",
	)

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	respRec := httptest.NewRecorder()
	router.ServeHTTP(respRec, req)

	rbody, err := io.ReadAll(respRec.Body)
	if err != nil {
		t.Error(err)
	}
	if respRec.Code != http.StatusOK {
		t.Errorf("expected status code: %d, got: %d. response: '%s'", http.StatusOK, respRec.Code, string(rbody))
	}

	url = fmt.Sprintf(
		"%s%s",
		baseAPI,
		"/hello/a/b/c/-d/~/e/world/fgh/i/j/k~/l-/hi/there",
	)

	req, _ = http.NewRequest(http.MethodGet, url, nil)
	respRec = httptest.NewRecorder()
	router.ServeHTTP(respRec, req)

	if respRec.Code != http.StatusOK {
		t.Errorf("expected status code: %d, got: %d", http.StatusOK, respRec.Code)
	}

	err = checkParams(
		req,
		respRec,
		[]string{"w", "p1", "w2"},
		[]string{
			"a/b/c/-d/~/e",
			"fgh",
			"i/j/k~/l-",
		})
	if err != nil {
		t.Error(err)
	}
}

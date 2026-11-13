package webgo

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestRouteGroupsPathPrefix(t *testing.T) {
	t.Parallel()
	routes := []Route{
		{
			Name:     "r1",
			Pattern:  "/a",
			Method:   http.MethodGet,
			Handlers: []http.HandlerFunc{dummyHandler},
		},
		{
			Name:     "r2",
			Pattern:  "/b/:c",
			Method:   http.MethodGet,
			Handlers: []http.HandlerFunc{dummyHandler},
		},
		{
			Name:     "r3",
			Pattern:  "/:w*",
			Method:   http.MethodGet,
			Handlers: []http.HandlerFunc{dummyHandler},
		},
	}

	const prefix = "/v7.0.0"
	expectedSkipMiddleware := true
	rg := NewRouteGroup("/v7.0.0", expectedSkipMiddleware, routes...)

	list := rg.Routes()
	for idx := range list {
		route := list[idx]
		originalRoute := routes[idx]
		expectedPattern := fmt.Sprintf("%s%s", prefix, originalRoute.Pattern)
		if route.Pattern != expectedPattern {
			t.Errorf("Expected pattern %q, got %q", expectedPattern, route.Pattern)
		}
		if route.skipMiddleware != expectedSkipMiddleware {
			t.Errorf("Expected skip %v, got %v", expectedSkipMiddleware, route.skipMiddleware)
		}
	}
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {}

func BenchmarkMatchWithWildcard(b *testing.B) {
	route := Route{
		Name:                    "widlcard",
		Method:                  http.MethodGet,
		TrailingSlash:           true,
		FallThroughPostResponse: true,
		Pattern:                 "/:w*/static1/:myvar/:w2*",
		Handlers:                []http.HandlerFunc{dummyHandler},
	}

	uri := "/hello/world/how/are/you/static1/hello2/world2/how2/are2/you2/static2"
	err := route.init()
	if err != nil {
		b.Error(err)
		return
	}

	for b.Loop() {
		ok, _ := route.matchPath(uri)
		if !ok {
			b.Errorf("Expected match, got no match")
			break
		}
	}
}

func TestMatchWithWildcard(t *testing.T) {
	route := Route{
		Name:                    "widlcard",
		Method:                  http.MethodGet,
		TrailingSlash:           true,
		FallThroughPostResponse: true,
		Pattern:                 "/:w*/static1/:myvar/:w2*/static2",
		Handlers:                []http.HandlerFunc{dummyHandler},
	}
	err := route.init()
	if err != nil {
		t.Error(err)
		return
	}

	uri := "/hello/world/how/are/you/static1/hello2/world2/how2/are2/you2/static2"
	wantParams := map[string]string{
		"w":     "hello/world/how/are/you",
		"myvar": "hello2",
		"w2":    "world2/how2/are2/you2",
	}
	matched, params := route.matchPath(uri)
	if !matched {
		t.Errorf("Expected match, got no match")
		return
	}
	if !reflect.DeepEqual(params, wantParams) {
		t.Errorf("Expected params %v, got %v", wantParams, params)
		return
	}

	t.Run("no match", func(t *testing.T) {
		route := Route{
			Name:                    "widlcard",
			Method:                  http.MethodGet,
			TrailingSlash:           true,
			FallThroughPostResponse: true,
			Pattern:                 "/:w*/static1/:myvar/:w2*/static2",
			Handlers:                []http.HandlerFunc{dummyHandler},
		}
		err := route.init()
		if err != nil {
			t.Error(err)
			return
		}

		uri := "/hello/world/how/are/you/static2/hello2/world2/how2/are2/you2/static2"
		matched, params := route.matchPath(uri)
		if matched {
			t.Errorf("Expected no match, got match")
			return
		}
		if params != nil {
			t.Errorf("Expected params %v, got %v", nil, params)
			return
		}
	})
	t.Run("match with more params", func(t *testing.T) {
		route := Route{
			Name:                    "widlcard",
			Method:                  http.MethodGet,
			TrailingSlash:           true,
			FallThroughPostResponse: true,
			Pattern:                 "/:w*/static1/:myvar/:w2*/static2/:myvar2/:w3*/static3",
			Handlers:                []http.HandlerFunc{dummyHandler},
		}
		err := route.init()
		if err != nil {
			t.Error(err)
			return
		}

		uri := "/hello/world/how/are/you/static1/hello2/world2/how2/are2/you2/static2/hello3/world3/how3/are3/you3/static3"
		wantParams := map[string]string{
			"w":      "hello/world/how/are/you",
			"myvar":  "hello2",
			"w2":     "world2/how2/are2/you2",
			"myvar2": "hello3",
			"w3":     "world3/how3/are3/you3",
		}
		matched, params := route.matchPath(uri)
		if !matched {
			t.Errorf("Expected match, got no match")
			return
		}
		if !reflect.DeepEqual(params, wantParams) {
			t.Errorf("Expected params %v, got %v", wantParams, params)
			return
		}
	})
	t.Run("match - end with wildcard", func(t *testing.T) {
		route := Route{
			Name:                    "widlcard",
			Method:                  http.MethodGet,
			TrailingSlash:           true,
			FallThroughPostResponse: true,
			Pattern:                 "/:w*/static1/:myvar/:w2*",
			Handlers:                []http.HandlerFunc{dummyHandler},
		}
		err := route.init()
		if err != nil {
			t.Error(err)
			return
		}

		uri := "/hello/world/how/are/you/static1/hello2/world2/how2/are2/you2/static2"
		wantParams := map[string]string{
			"w":     "hello/world/how/are/you",
			"myvar": "hello2",
			"w2":    "world2/how2/are2/you2/static2",
		}
		matched, params := route.matchPath(uri)
		if !matched {
			t.Errorf("Expected match, got no match")
			return
		}
		if !reflect.DeepEqual(params, wantParams) {
			t.Errorf("Expected params %v, got %v", wantParams, params)
			return
		}
	})

	t.Run("root URI, no match", func(t *testing.T) {
		route := Route{
			Name:                    "",
			Method:                  http.MethodGet,
			TrailingSlash:           true,
			FallThroughPostResponse: true,
			Pattern:                 "/-/health",
			Handlers:                []http.HandlerFunc{dummyHandler},
		}
		err := route.init()
		if err != nil {
			t.Error(err)
			return
		}
		matched, _ := route.matchPath("/")
		if matched {
			t.Errorf("Expected no match, got match")
			return
		}
	})
	t.Run("root URI, should match", func(t *testing.T) {
		route := Route{
			Name:                    "",
			Method:                  http.MethodGet,
			TrailingSlash:           true,
			FallThroughPostResponse: true,
			Pattern:                 "/",
			Handlers:                []http.HandlerFunc{dummyHandler},
		}
		err := route.init()
		if err != nil {
			t.Error(err)
			return
		}
		matched, _ := route.matchPath("/")
		if !matched {
			t.Errorf("Expected match, got no match")
			return
		}
	})
}

func TestMatchPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		pattern       string
		trailingSlash bool
		uri           string
		want          bool
	}{
		{"exact", "/a/b", false, "/a/b", true},
		{"exact with trailing slash enabled", "/a/b", true, "/a/b", true},
		{"extra slash, trailing slash enabled", "/a/b", true, "/a/b/", true},
		{"extra slash, trailing slash disabled", "/a/b", false, "/a/b/", false},
		{"different path", "/a/b", true, "/a/c", false},
		{"longer path", "/a/b", true, "/a/bc", false},
		{"two extra characters", "/a/b", true, "/a/b/c", false},
		{"root, trailing slash disabled", "/", false, "/", false},
		{"root, trailing slash enabled", "/", true, "/", true},
		{"empty uri, trailing slash disabled", "/a", false, "", false},
		{"empty uri, trailing slash enabled", "/a", true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			route := Route{
				Method:        http.MethodGet,
				Pattern:       tt.pattern,
				TrailingSlash: tt.trailingSlash,
				Handlers:      []http.HandlerFunc{dummyHandler},
			}
			if err := route.init(); err != nil {
				t.Fatal(err)
			}

			got, _ := route.matchPath(tt.uri)
			if got != tt.want {
				t.Errorf("matchPath(%q) = %v, want %v", tt.uri, got, tt.want)
			}
		})
	}
}

// matchWithWildcardRef is the v7.3.0 implementation of Route.matchWithWildcard, kept as
// the reference to prove the allocation free rewrite did not change the matching rules.
func matchWithWildcardRef(r *Route, requestURI string) (bool, map[string]string) {
	if len(r.fragments) == 0 {
		return false, nil
	}

	params := make(map[string]string, r.paramsCount)
	uriFragments := strings.Split(requestURI, "/")[1:]
	fragmentsLastIdx := len(r.fragments) - 1
	fragmentIdx := 0
	uriParameter := make([]string, 0, len(uriFragments))

	for idx, fragment := range uriFragments {
		if fragment == "" {
			break
		}

		if fragmentIdx > fragmentsLastIdx {
			return false, nil
		}

		currentFragment := r.fragments[fragmentIdx]
		if !currentFragment.isVariable && currentFragment.fragment != fragment {
			return false, nil
		}

		uriParameter = append(uriParameter, fragment)
		if currentFragment.isVariable {
			params[currentFragment.fragment] = strings.Join(uriParameter, "/")
		}

		if !currentFragment.hasWildcard {
			uriParameter = make([]string, 0, len(uriFragments)-idx)
			fragmentIdx++
			continue
		}

		nextIdx := fragmentIdx + 1
		if nextIdx > fragmentsLastIdx {
			continue
		}
		nextPart := r.fragments[nextIdx]

		if !nextPart.isVariable && nextPart.fragment == fragment {
			params[currentFragment.fragment] = strings.Join(uriParameter[:len(uriParameter)-1], "/")
			uriParameter = make([]string, 0, len(uriFragments)-idx)
			fragmentIdx += 2
		}
	}

	if len(params) != r.paramsCount {
		return false, nil
	}

	return true, params
}

func TestMatchWithWildcardStaticPrefix(tt *testing.T) {
	tt.Parallel()

	prefixes := map[string]string{
		"/users/:id":               "/users/",
		"/users/:id/posts/:pid":    "/users/",
		"/:id":                     "/",
		"/:w*/static1/:myvar/:w2*": "/",
		"/a/b/c/:id":               "/a/b/c/",
		"/a/b/:w*":                 "/a/b/",
		"/static":                  "",
		"/files/:path*/download":   "/files/",
	}
	for pattern, want := range prefixes {
		route := Route{Pattern: pattern}
		route.parseURIWithParams()
		if route.staticPrefix != want {
			tt.Errorf("pattern %q: expected prefix %q, got %q", pattern, want, route.staticPrefix)
		}
	}
}

func TestMatchWithWildcardMatchesReference(tt *testing.T) {
	tt.Parallel()

	patterns := []string{
		"/:id",
		"/:id/",
		"/users/:id",
		"/users/:id/",
		"/users/:id/posts/:pid",
		"/users/:id/posts/:pid/",
		"/a/b/c/:id",
		"/files/:path*",
		"/files/:path*/download",
		"/files/:path*/:name",
		"/:w*/static1/:myvar/:w2*",
		"/:w*/static1",
		"/:w*",
		"/a/:w*/b/:x*/c",
		"/:a/:a",
		"/x:y/:z",
		"/a*/:b",
		"/a//:id",
	}
	uris := []string{
		"", "/", "//", "/users", "/users/", "/users//", "/users/1", "/users/1/", "/users/1/2",
		"/users/1/posts", "/users/1/posts/", "/users/1/posts/2", "/users/1/posts/2/", "/users/1/posts/2/3",
		"/user/1", "/Users/1", "/a/b/c", "/a/b/c/1", "/a/b/c/1/", "/a/b/d/1", "/a//1", "/a//", "/a///1",
		"/files", "/files/", "/files/x", "/files/x/y/z", "/files/x/y/z/", "/files/x/download",
		"/files/x/y/download", "/files/download", "/files/download/download", "/files/x/y/z/download/n",
		"/hello/world/static1/v1/a/b/c", "/hello/static1/v1/a", "/static1/static1/static1/static1",
		"/static1", "/static1/", "/a/static1/b", "/q/static1/v/w", "/a/b/x/c/y/b/z/c", "/a/b/c/b/c",
		"/foo/bar", "/foo/bar/", "/1/1", "/ab", "/xy/1", "/x:y/1",
	}

	for _, pattern := range patterns {
		route := Route{Pattern: pattern}
		route.parseURIWithParams()
		for _, uri := range uris {
			wantOK, wantParams := matchWithWildcardRef(&route, uri)
			gotOK, gotParams := route.matchWithWildcard(uri)
			if wantOK != gotOK {
				tt.Errorf("pattern %q, uri %q: expected match %v, got %v", pattern, uri, wantOK, gotOK)
				continue
			}
			if !wantOK {
				if gotParams != nil {
					tt.Errorf("pattern %q, uri %q: expected nil params on miss, got %v", pattern, uri, gotParams)
				}
				continue
			}
			if !maps.Equal(wantParams, gotParams) {
				tt.Errorf("pattern %q, uri %q: expected params %v, got %v", pattern, uri, wantParams, gotParams)
			}
		}
	}
}

func TestMatchWithWildcardMissDoesNotAllocate(tt *testing.T) {
	route := Route{Pattern: "/users/:id/posts/:pid"}
	route.parseURIWithParams()

	uris := []string{
		"/does/not/exist",
		"/users/1/comments/2",
		"/users/1/posts",
	}
	for _, uri := range uris {
		allocs := testing.AllocsPerRun(100, func() {
			if ok, _ := route.matchWithWildcard(uri); ok {
				tt.Fatalf("expected %q to miss", uri)
			}
		})
		if allocs != 0 {
			tt.Errorf("uri %q: expected 0 allocs on miss, got %v", uri, allocs)
		}
	}
}

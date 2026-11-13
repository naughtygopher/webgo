package webgo

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Route defines a route for each API
type Route struct {
	// Name is unique identifier for the route
	Name string
	// Method is the HTTP request method/type
	Method string
	// Pattern is the URI pattern to match
	Pattern string
	// TrailingSlash if set to true, the URI will be matched with or without
	// a trailing slash. IMPORTANT: It does not redirect.
	TrailingSlash bool

	// FallThroughPostResponse if enabled will execute all the handlers even if a response was already sent to the client
	FallThroughPostResponse bool

	// Handlers is a slice of http.HandlerFunc which can be middlewares or anything else. Though only 1 of them will be allowed to respond to client.
	// subsequent writes from the following handlers will be ignored
	Handlers []http.HandlerFunc

	hasWildcard bool
	fragments   []uriFragment
	paramsCount int

	// skipMiddleware if true, middleware added using `router` will not be applied to this Route.
	// This is used only when a Route is set using the RouteGroup, which can have its own set of middleware
	skipMiddleware bool
	// middlewareList is used at the last stage, i.e. right before starting the server
	middlewarelist []Middleware

	initialized bool

	serve http.HandlerFunc
}
type uriFragment struct {
	isVariable  bool
	hasWildcard bool
	// fragment will be the key name, if it's a variable/named URI parameter
	fragment string
}

func (r *Route) parseURIWithParams() {
	// if there are no URI params, then there's no need to set route parts
	if !strings.Contains(r.Pattern, ":") {
		return
	}

	fragments := strings.Split(r.Pattern, "/")
	if len(fragments) == 1 {
		return
	}

	rFragments := make([]uriFragment, 0, len(fragments))
	for _, fragment := range fragments[1:] {
		hasParam := false
		hasWildcard := false

		if strings.Contains(fragment, ":") {
			hasParam = true
			r.paramsCount++
		}
		if strings.Contains(fragment, "*") {
			r.hasWildcard = true
			hasWildcard = true
		}

		key := strings.ReplaceAll(fragment, ":", "")
		key = strings.ReplaceAll(key, "*", "")
		rFragments = append(
			rFragments,
			uriFragment{
				isVariable:  hasParam,
				hasWildcard: hasWildcard,
				fragment:    key,
			})
	}
	r.fragments = rFragments
}

func (r *Route) setupMiddleware(reverse bool) {
	if reverse {
		for i := range r.middlewarelist {
			m := r.middlewarelist[i]
			srv := r.serve
			r.serve = func(rw http.ResponseWriter, req *http.Request) {
				m(rw, req, srv)
			}
		}
	} else {
		for _, m := range slices.Backward(r.middlewarelist) {
			srv := r.serve
			r.serve = func(rw http.ResponseWriter, req *http.Request) {
				m(rw, req, srv)
			}
		}
	}
	// clear middlewarelist since it's already setup for the route
	r.middlewarelist = nil
}

// init does all the initializations required for the route
func (r *Route) init() error {
	if r.initialized {
		return nil
	}
	r.initialized = true

	r.parseURIWithParams()
	r.serve = defaultRouteServe(r)
	return nil
}

// matchPath matches the requestURI with the URI pattern of the route
func (r *Route) matchPath(requestURI string) (bool, map[string]string) {
	if r.TrailingSlash {
		// requestURI is either pattern or pattern + "/"
		plen := len(r.Pattern)
		if requestURI == r.Pattern ||
			(len(requestURI) == plen+1 && requestURI[plen] == '/' && requestURI[:plen] == r.Pattern) {
			return true, nil
		}
	} else {
		if requestURI != "" && requestURI[len(requestURI)-1] == '/' {
			return false, nil
		}
		if requestURI == r.Pattern {
			return true, nil
		}
	}

	return r.matchWithWildcard(requestURI)
}

func (r *Route) matchWithWildcard(requestURI string) (bool, map[string]string) {
	// if r.fragments is empty, it means there are no variables in the URI pattern
	// hence no point checking
	if len(r.fragments) == 0 {
		return false, nil
	}

	params := make(map[string]string, r.paramsCount)
	uriFragments := strings.Split(requestURI, "/")[1:]
	fragmentsLastIdx := len(r.fragments) - 1
	fragmentIdx := 0
	uriParameter := make([]string, 0, len(uriFragments))

	for idx, fragment := range uriFragments {
		// if part is empty, it means it's end of URI with trailing slash
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

		// if the URI has more fragments/params after wildcard,
		// the immediately following part after wildcard cannot be a variable or another wildcard.
		if !nextPart.isVariable && nextPart.fragment == fragment {
			// remove the last added 'part' from parameters, as it's part of the static URI
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

func (r *Route) use(mm ...Middleware) {
	if r.middlewarelist == nil {
		r.middlewarelist = make([]Middleware, 0, len(mm))
	}
	r.middlewarelist = append(r.middlewarelist, mm...)
}

func routeServeChainedHandlers(r *Route) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {

		crw, ok := rw.(*customResponseWriter)
		if !ok {
			crw = newCRW(rw, http.StatusOK)
		}

		for _, handler := range r.Handlers {
			if crw.written && !r.FallThroughPostResponse {
				break
			}
			handler(crw, req)
		}
	}
}

func defaultRouteServe(r *Route) http.HandlerFunc {
	if len(r.Handlers) > 1 {
		return routeServeChainedHandlers(r)
	}

	// when there is only 1 handler, custom response writer is not required to check if response
	// is already written or fallthrough is enabled
	return r.Handlers[0]
}

type RouteGroup struct {
	routes []*Route
	// skipRouterMiddleware if set to true, middleware applied to the router will not be applied
	// to this route group.
	skipRouterMiddleware bool
	// PathPrefix is the URI prefix for all routes in this group
	PathPrefix string
}

func (rg *RouteGroup) Add(rr ...Route) {
	for idx := range rr {
		route := rr[idx]
		route.skipMiddleware = rg.skipRouterMiddleware
		route.Pattern = fmt.Sprintf("%s%s", rg.PathPrefix, route.Pattern)
		rg.routes = append(rg.routes, &route)
	}
}

func (rg *RouteGroup) Use(mm ...Middleware) {
	for _, route := range rg.routes {
		route.use(mm...)
	}
}

func (rg *RouteGroup) Routes() []*Route {
	return rg.routes
}

func NewRouteGroup(pathPrefix string, skipRouterMiddleware bool, rr ...Route) *RouteGroup {
	rg := RouteGroup{
		PathPrefix:           pathPrefix,
		skipRouterMiddleware: skipRouterMiddleware,
	}
	rg.Add(rr...)
	return &rg
}

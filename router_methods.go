package webgo

import "net/http"

// RouteOption configures optional fields on a Route when using convenience methods.
type RouteOption func(*Route)

// RouteName sets the unique name of the route.
func RouteName(name string) RouteOption {
	return func(r *Route) {
		r.Name = name
	}
}

// TrailingSlash sets whether the URI is matched with or without a trailing slash.
func TrailingSlash(trailing bool) RouteOption {
	return func(r *Route) {
		r.TrailingSlash = trailing
	}
}

// FallThroughPostResponse sets whether handlers run after a response is sent.
func FallThroughPostResponse(enabled bool) RouteOption {
	return func(r *Route) {
		r.FallThroughPostResponse = enabled
	}
}

func (rtr *Router) registerRoute(method, pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	route := &Route{
		Method:   method,
		Pattern:  pattern,
		Handlers: handlers,
	}
	for _, opt := range opts {
		opt(route)
	}
	rtr.Add(route)
}

// Get registers a GET route on the router.
func (rtr *Router) Get(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodGet, pattern, handlers, opts...)
}

// Head registers a HEAD route on the router.
func (rtr *Router) Head(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodHead, pattern, handlers, opts...)
}

// Post registers a POST route on the router.
func (rtr *Router) Post(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodPost, pattern, handlers, opts...)
}

// Put registers a PUT route on the router.
func (rtr *Router) Put(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodPut, pattern, handlers, opts...)
}

// Patch registers a PATCH route on the router.
func (rtr *Router) Patch(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodPatch, pattern, handlers, opts...)
}

// Delete registers a DELETE route on the router.
func (rtr *Router) Delete(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodDelete, pattern, handlers, opts...)
}

// Options registers an OPTIONS route on the router.
func (rtr *Router) Options(pattern string, handlers []http.HandlerFunc, opts ...RouteOption) {
	rtr.registerRoute(http.MethodOptions, pattern, handlers, opts...)
}

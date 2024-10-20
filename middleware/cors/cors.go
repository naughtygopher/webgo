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
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/naughtygopher/webgo/v7"
)

const (
	headerOrigin           = "Access-Control-Allow-Origin"
	headerMethods          = "Access-Control-Allow-Methods"
	headerCreds            = "Access-Control-Allow-Credentials"
	headerAllowHeaders     = "Access-Control-Allow-Headers"
	headerReqHeaders       = "Access-Control-Request-Headers"
	headerAccessControlAge = "Access-Control-Max-Age"
	allowHeaders           = "Accept,Content-Type,Content-Length,Accept-Encoding,Access-Control-Request-Headers,"
)

var (
	defaultAllowMethods = "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS"
)

func allowedDomains() []string {
	// The domains mentioned here are default
	domains := []string{"*"}
	return domains
}

func getReqOrigin(r *http.Request) string {
	return r.Header.Get("Origin")
}

func allowedOriginsRegex(allowedOrigins ...string) []regexp.Regexp {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	} else {
		// If "*" is one of the allowed domains, i.e. all domains, then rest of the values are ignored
		for _, val := range allowedOrigins {
			val = strings.TrimSpace(val)

			if val == "*" {
				allowedOrigins = []string{"*"}
				break
			}
		}
	}

	allowedOriginRegex := make([]regexp.Regexp, 0, len(allowedOrigins))
	for _, ao := range allowedOrigins {
		parts := strings.Split(ao, ":")
		str := strings.TrimSpace(parts[0])
		if str == "" {
			continue
		}

		if str == "*" {
			allowedOriginRegex = append(
				allowedOriginRegex,
				*(regexp.MustCompile(".+")),
			)
			break
		}

		regStr := fmt.Sprintf(`^(http)?(https)?(:\/\/)?(.+\.)?%s(:[0-9]+)?$`, str)

		allowedOriginRegex = append(
			allowedOriginRegex,
			// Allow any port number of the specified domain
			*(regexp.MustCompile(regStr)),
		)
	}

	return allowedOriginRegex
}

func allowedMethods(routes []*webgo.Route) string {
	if len(routes) == 0 {
		return defaultAllowMethods
	}

	methods := make([]string, 0, len(routes))
	for _, r := range routes {
		found := false
		for _, m := range methods {
			if m == r.Method {
				found = true
				break
			}
		}
		if found {
			continue
		}
		methods = append(methods, r.Method)
	}
	sort.Strings(methods)
	return strings.Join(methods, ",")
}

// Config holds all the configurations which is available for customizing this middleware
type Config struct {
	TimeoutSecs    int
	Routes         []*webgo.Route
	AllowedOrigins []string
	AllowedHeaders []string
}

func allowedHeaders(headers []string) string {
	if len(headers) == 0 {
		return allowHeaders
	}

	allowedHeaders := strings.Join(headers, ",")
	if allowedHeaders[len(allowedHeaders)-1] != ',' {
		allowedHeaders += ","
	}
	return allowedHeaders
}

func allowOrigin(reqOrigin string, allowedOriginRegex []regexp.Regexp) bool {

	for _, o := range allowedOriginRegex {
		// Set appropriate response headers required for CORS
		if o.MatchString(reqOrigin) || reqOrigin == "" {
			return true
		}
	}
	return false
}

// Middleware can be used as well, it lets the user use this middleware without webgo
func Middleware(allowedOriginRegex []regexp.Regexp, corsTimeout, allowedMethods, allowedHeaders string) webgo.Middleware {
	return func(rw http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		reqOrigin := getReqOrigin(req)
		allowed := allowOrigin(reqOrigin, allowedOriginRegex)

		if !allowed {
			// If CORS failed, no respective headers are set. But the execution is allowed to continue
			// Earlier this middleware blocked access altogether, which was considered an added
			// security measure despite it being outside the scope of this middelware. Though, such
			// restrictions create unnecessary complexities during inter-app communication.
			next(rw, req)
			return
		}

		// Set appropriate response headers required for CORS
		rw.Header().Set(headerOrigin, reqOrigin)
		rw.Header().Set(headerAccessControlAge, corsTimeout)
		rw.Header().Set(headerCreds, "true")
		rw.Header().Set(headerMethods, allowedMethods)
		rw.Header().Set(headerAllowHeaders, allowedHeaders+req.Header.Get(headerReqHeaders))

		if req.Method == http.MethodOptions {
			webgo.SendHeader(rw, http.StatusOK)
			return
		}

		next(rw, req)
	}
}

// AddOptionsHandlers appends OPTIONS handler for all the routes
// The response body would be empty for all the new handlers added
func AddOptionsHandlers(routes []*webgo.Route) []*webgo.Route {
	dummyHandler := func(w http.ResponseWriter, r *http.Request) {}
	if len(routes) == 0 {
		return []*webgo.Route{
			{
				Name:          "cors",
				Pattern:       "/:w*",
				Method:        http.MethodOptions,
				TrailingSlash: true,
				Handlers:      []http.HandlerFunc{dummyHandler},
			},
		}
	}

	list := make([]*webgo.Route, 0, len(routes))
	list = append(list, routes...)

	for _, r := range routes {
		list = append(list, &webgo.Route{
			Name:          fmt.Sprintf("%s-CORS", r.Name),
			Method:        http.MethodOptions,
			Pattern:       r.Pattern,
			TrailingSlash: true,
			Handlers:      []http.HandlerFunc{dummyHandler},
		})
	}

	return list
}

// CORS is a single CORS middleware which can be applied to the whole app at once
func CORS(cfg *Config) webgo.Middleware {
	if cfg == nil {
		cfg = new(Config)
		// 30 minutes
		cfg.TimeoutSecs = 30 * 60
	}

	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = allowedDomains()
	}

	allowedOriginRegex := allowedOriginsRegex(allowedOrigins...)
	allowedmethods := allowedMethods(cfg.Routes)
	allowedHeaders := allowedHeaders(cfg.AllowedHeaders)
	corsTimeout := fmt.Sprintf("%d", cfg.TimeoutSecs)

	return Middleware(
		allowedOriginRegex,
		corsTimeout,
		allowedmethods,
		allowedHeaders,
	)
}

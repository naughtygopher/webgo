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
	"context"
	"crypto/tls"
	"net/http"
)

var supportedHTTPMethods = []string{
	http.MethodOptions,
	http.MethodHead,
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

// ctxkey is a custom string type to store the WebGo context inside HTTP request context
type ctxkey string

const wgoCtxKey = ctxkey("webgocontext")

// ContextPayload is the WebgoContext. A new instance of ContextPayload is injected inside every request's context object
type ContextPayload struct {
	Route     *Route
	Err       error
	URIParams map[string]string
}

// Params returns the URI parameters of the respective route
func (cp *ContextPayload) Params() map[string]string {
	return cp.URIParams
}

func (cp *ContextPayload) reset() {
	cp.Route = nil
	cp.Err = nil
}

// SetError sets the err within the context
func (cp *ContextPayload) SetError(err error) {
	cp.Err = err
}

// Error returns the error set within the context
func (cp *ContextPayload) Error() error {
	return cp.Err
}

// Context returns the ContextPayload injected inside the HTTP request context
func Context(r *http.Request) *ContextPayload {
	return r.Context().Value(wgoCtxKey).(*ContextPayload)
}

// SetError is a helper function to set the error in webgo context
func SetError(r *http.Request, err error) {
	ctx := Context(r)
	ctx.SetError(err)
}

// GetError is a helper function to get the error from webgo context
func GetError(r *http.Request) error {
	return Context(r).Error()
}

// ResponseStatus returns the response status code. It works only if the http.ResponseWriter
// is not wrapped in another response writer before calling ResponseStatus
func ResponseStatus(rw http.ResponseWriter) int {
	crw, ok := rw.(*customResponseWriter)
	if !ok {
		return http.StatusOK
	}
	return crw.statusCode
}
func (router *Router) setupServer() {
	cfg := router.config

	router.srvMu.Lock()
	defer router.srvMu.Unlock()

	router.httpsServer = &http.Server{
		Addr:              "",
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		},
	}
	router.httpServer = &http.Server{
		Addr:              "",
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	router.SetupMiddleware()
}

// SetupMiddleware initializes all the middleware added using "Use".
// This function need not be called explicitly, if using router.Start()
// or router.StartHTTPS(). Instead if the router is being passed to an external server
// then the SetupMiddleware function should be called
func (router *Router) SetupMiddleware() {
	// load middleware for all routes
	for _, routes := range router.allHandlers {
		for _, route := range routes {
			route.setupMiddleware(router.config.ReverseMiddleware)
		}
	}
}

// StartHTTPS starts the server with HTTPS enabled
func (router *Router) StartHTTPS() {
	cfg := router.config
	if cfg.CertFile == "" {
		LOGHANDLER.Fatal("No certificate provided for HTTPS")
	}

	if cfg.KeyFile == "" {
		LOGHANDLER.Fatal("No key file provided for HTTPS")
	}

	router.setupServer()

	host := cfg.Host
	if len(cfg.HTTPSPort) > 0 {
		host += ":" + cfg.HTTPSPort
	}

	router.srvMu.Lock()
	srv := router.httpsServer
	srv.Addr = host
	router.srvMu.Unlock()

	LOGHANDLER.Info("HTTPS server, listening on", host)
	err := srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	if err != nil && err != http.ErrServerClosed {
		LOGHANDLER.Error("HTTPS server exited with error:", err.Error())
	}
}

// Start starts the HTTP server with the appropriate configurations
func (router *Router) Start() {
	router.setupServer()

	cfg := router.config
	host := cfg.Host
	if len(cfg.Port) > 0 {
		host += ":" + cfg.Port
	}

	router.srvMu.Lock()
	srv := router.httpServer
	srv.Addr = host
	router.srvMu.Unlock()

	LOGHANDLER.Info("HTTP server, listening on", host)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		LOGHANDLER.Error("HTTP server exited with error:", err.Error())
	}
}

// Shutdown gracefully shuts down HTTP server
func (router *Router) Shutdown() error {
	router.srvMu.Lock()
	srv := router.httpServer
	router.srvMu.Unlock()
	if srv == nil {
		return nil
	}
	timer := router.config.ShutdownTimeout

	ctx, cancel := context.WithTimeout(context.TODO(), timer)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		LOGHANDLER.Error(err)
	}
	return err
}

// ShutdownHTTPS gracefully shuts down HTTPS server
func (router *Router) ShutdownHTTPS() error {
	router.srvMu.Lock()
	srv := router.httpsServer
	router.srvMu.Unlock()
	if srv == nil {
		return nil
	}
	timer := router.config.ShutdownTimeout

	ctx, cancel := context.WithTimeout(context.TODO(), timer)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil && err != http.ErrServerClosed {
		LOGHANDLER.Error(err)
	}
	return err
}

// OriginalResponseWriter returns the Go response writer stored within the webgo custom response
// writer
func OriginalResponseWriter(rw http.ResponseWriter) http.ResponseWriter {
	crw, ok := rw.(*customResponseWriter)
	if !ok {
		return nil
	}

	return crw.ResponseWriter
}

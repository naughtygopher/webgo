<p align="center"><img src="https://user-images.githubusercontent.com/1092882/60883564-20142380-a268-11e9-988a-d98fb639adc6.png" alt="webgo gopher" width="256px"/></p>

[![](https://github.com/naughtygopher/webgo/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/naughtygopher/webgo/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/naughtygopher/webgo/v7.svg)](https://pkg.go.dev/github.com/naughtygopher/webgo/v7)
[![Coverage Status](https://coveralls.io/repos/github/naughtygopher/webgo/badge.svg?branch=master)](https://coveralls.io/github/naughtygopher/webgo?branch=master)
[![](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#web-frameworks)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/naughtygopher/webgo/blob/master/LICENSE)

# WebGo v7

WebGo is a minimalistic router for [Go](https://golang.org) to build web applications (server side) with no 3rd party dependencies. WebGo will always be Go standard library compliant; with the HTTP handlers having the same signature as [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc).

### Contents

1. [Router](https://github.com/naughtygopher/webgo#router)
2. [Handler chaining](https://github.com/naughtygopher/webgo#handler-chaining)
3. [Middleware](https://github.com/naughtygopher/webgo#middleware)
4. [Error handling](https://github.com/naughtygopher/webgo#error-handling)
5. [Helper functions](https://github.com/naughtygopher/webgo#helper-functions)
6. [HTTPS ready](https://github.com/naughtygopher/webgo#https-ready)
7. [Server timeouts](https://github.com/naughtygopher/webgo#server-timeouts)
8. [Graceful shutdown](https://github.com/naughtygopher/webgo#graceful-shutdown)
9. [Logging](https://github.com/naughtygopher/webgo#logging)
10. [Server-Sent Events](https://github.com/naughtygopher/webgo#server-sent-events)
11. [Usage](https://github.com/naughtygopher/webgo#usage)

## Router

Webgo has a simplistic, linear path matching router and supports defining [URI](https://developer.mozilla.org/en-US/docs/Glossary/URI)s with the following patterns

1. `/api/users` - URI with no dynamic values
2. `/api/users/:userID`
   - URI with a named parameter, `userID`
   - If TrailingSlash is set to true, it will accept the URI ending with a '/' (it does not redirect), refer to the [sample](https://github.com/naughtygopher/webgo/blob/master/cmd/main.go)
3. `/api/users/:misc*`
   - Named URI parameter `misc`, with a wildcard suffix '\*'
   - This matches everything after `/api/users`. e.g. `/api/users/a/b/c/d`

When there are multiple handlers matching the same URI, only the first occurring handler will handle the request.
Refer to the [sample](https://github.com/naughtygopher/webgo/blob/master/cmd/main.go) to see how routes are configured. You can access named parameters of the URI using the `Context` function.

Note: webgo Context is **not** available inside the special handlers (not found & method not implemented). Calling `webgo.Context(r)` there panics

```golang
func helloWorld(w http.ResponseWriter, r *http.Request) {
	// WebGo context
	wctx := webgo.Context(r)
	// URI paramaters, map[string]string
	params := wctx.Params()
	// route, the webgo.Route which is executing this request
	route := wctx.Route
	webgo.R200(
		w,
		fmt.Sprintf(
			"Route name: '%s', params: '%s'",
			route.Name,
			params,
		),
	)
}
```

## Handler chaining

Handler chaining lets you execute multiple handlers for a given route. Execution of a chain can be configured to run even after a handler has written a response to the HTTP request, if you set `FallThroughPostResponse` to `true` (refer to the `api` route in the [sample](https://github.com/naughtygopher/webgo/blob/master/cmd/main.go)).

## Middleware

WebGo [middleware](https://pkg.go.dev/github.com/naughtygopher/webgo/v7#Middleware) lets you wrap all the routes with a middleware unlike handler chaining. The router exposes a method [Use](https://pkg.go.dev/github.com/naughtygopher/webgo/v7#Router.Use) && [UseOnSpecialHandlers](https://pkg.go.dev/github.com/naughtygopher/webgo/v7#Router.UseOnSpecialHandlers) to add a Middleware to the router.

NotFound && NotImplemented are considered `Special` handlers. `webgo.Context(r)` within special handlers panics, as there is no webgo context.

Any number of middleware can be added to the router. By default, middleware are executed in the order they were added, i.e. in case of the following code

```golang
func main() {
	router.Use(accesslog.AccessLog, cors.CORS(nil))
	router.Use(<more middleware>)
}
```

**_AccessLog_** would be executed first, followed by **_CORS_**, followed by the middleware added after. If `ReverseMiddleware` is set to `true` in `Config`, the order is reversed (last added executes first).

`Use` applies only to the routes which are already added to the router, so call it after all the routes are added. If you pass the router to your own `http.Server` instead of using `router.Start()` or `router.StartHTTPS()`, call `router.SetupMiddleware()` once before serving, otherwise the middleware are not applied.

## Error handling

Webgo context has 2 methods, `SetError` & `Error` (also available as the helpers `webgo.SetError(r, err)` & `webgo.GetError(r)`), to set & get an error within a request context. It enables Webgo to implement a single middleware where you can handle errors returned within an HTTP handler. Refer to `errLogger` in [main.go](https://github.com/naughtygopher/webgo/blob/master/cmd/main.go) & `ErrorSetterHandler` in [handlers.go](https://github.com/naughtygopher/webgo/blob/master/cmd/handlers.go) of the sample for an example.

## Helper functions

WebGo provides a few helper functions. When using `SendResponse` (or the other `Rxxx` responder functions, e.g. `R200`), the response is wrapped in WebGo's response struct and is serialized as JSON. `Send` is the exception, it writes the data as is, with the content type you provide.

```json
{
  "data": "<any valid JSON payload>",
  "status": "<HTTP status code, of type integer>"
}
```

When using `SendError` (or the error responders, e.g. `R400`, `R500`), the response is wrapped in WebGo's error response struct and is serialized as JSON.

```json
{
  "errors": "<any valid JSON payload>",
  "status": "<HTTP status code, of type integer>"
}
```

## HTTPS ready

HTTPS server can be started easily, by providing the key & cert file. You can also have both HTTP & HTTPS servers running side by side.

Start HTTPS server

```golang
cfg := &webgo.Config{
	Port: "80",
	HTTPSPort: "443",
	CertFile: "/path/to/certfile",
	KeyFile: "/path/to/keyfile",
}
router := webgo.NewRouter(cfg, routes()...)
router.StartHTTPS()
```

Starting both HTTP & HTTPS server

```golang
cfg := &webgo.Config{
	Port: "80",
	HTTPSPort: "443",
	CertFile: "/path/to/certfile",
	KeyFile: "/path/to/keyfile",
}

router := webgo.NewRouter(cfg, routes()...)
go router.StartHTTPS()
router.Start()
```

## Server timeouts

`ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` & `IdleTimeout` of `Config` are passed on to the underlying `http.Server`. None of them are enabled unless set. It is recommended to set `ReadHeaderTimeout` (or `ReadTimeout`) when the server is exposed to the internet, to protect against slow clients holding connections open.

```golang
cfg := &webgo.Config{
	Port:              "8080",
	ReadHeaderTimeout: 5 * time.Second,
	IdleTimeout:       120 * time.Second,
}
```

## Graceful shutdown

Graceful shutdown lets you shutdown the server without affecting any live connections/clients connected to the server. Any new connection request after initiating a shutdown would be ignored.

Sample code to show how to use shutdown

```golang
func main() {
	osSig := make(chan os.Signal, 5)

	cfg := &webgo.Config{
		Host:            "",
		Port:            "8080",
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
	}
	router := webgo.NewRouter(cfg, routes()...)

	signal.Notify(osSig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-osSig
		// Initiate HTTP server shutdown
		err := router.Shutdown()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println("shutdown complete")
			os.Exit(0)
		}

		// If you have HTTPS server running, you can use the following code
		// err := router.ShutdownHTTPS()
		// if err != nil {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// } else {
		// 	fmt.Println("shutdown complete")
		// 	os.Exit(0)
		// }
	}()

	router.Start()
}
```

## Logging

WebGo exposes a singleton & global scoped logger variable [LOGHANDLER](https://pkg.go.dev/github.com/naughtygopher/webgo/v7#LOGHANDLER) with which you can plug in your custom logger by implementing the [Logger](https://pkg.go.dev/github.com/naughtygopher/webgo/v7#Logger) interface.

### Configuring the default Logger

The default logger uses Go standard library's `log.Logger` with `os.Stdout` (for debug and info logs) & `os.Stderr` (for warning, error, fatal) as default io.Writers. You can set the io.Writer as well as disable specific types of logs using the `GlobalLoggerConfig(stdout, stderr, cfgs...)` function. It is not safe for concurrent use, so call it once, at startup.

## Server-Sent Events

[MDN has a very good documentation of what SSE (Server-Sent Events)](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) are. The SSE extension (`github.com/naughtygopher/webgo/v7/extensions/sse`) works with any `net/http` server, it is not limited to webgo. The sample app provided shows how to use it with webgo.

## Usage

A fully functional sample is provided [here](https://github.com/naughtygopher/webgo/blob/master/cmd/main.go).

### Benchmark

1. [the-benchmarker](https://github.com/the-benchmarker/web-frameworks)
2. [go-web-framework-benchmark](https://github.com/smallnest/go-web-framework-benchmark)

### Contributing

Refer [here](https://github.com/naughtygopher/webgo/blob/master/CONTRIBUTING.md) to find out details about making a contribution

### Credits

Thanks to all the [contributors](https://github.com/naughtygopher/webgo/graphs/contributors)

## The gopher

The gopher used here was created using [Gopherize.me](https://gopherize.me/). WebGo stays out of developers' way, so sitback and enjoy a cup of coffee.

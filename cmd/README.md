# Webgo Sample

### Server Sent Events

![sse-demo](https://user-images.githubusercontent.com/1092882/158047065-447eb868-1efd-4a8d-b748-7caee2b3fcfd.png)

This picture shows the sample SSE implementation provided with this application. In the sample app, the server is
sending timestamp every second, to all the clients.

**Important**: _[SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events)
is a live connection between server & client. So a short WriteTimeout duration in webgo.Config will
keep dropping the connection. If you have any middleware which is setting deadlines or timeouts on the
request.Context, will also effect these connections._

## How to run

If you have Go installed on your computer, open the terminal and:

```bash
$ cd $GOPATH/src
$ git clone https://github.com/naughtygopher/webgo.git
$ cd webgo/cmd
$ go run *.go

Info 2023/02/05 08:51:26 HTTP server, listening on :8080
Info 2023/02/05 08:51:26 HTTPS server, listening on :9595
```

Or if you have [Docker](https://www.docker.com/), open the terminal and:

```bash
$ git clone https://github.com/naughtygopher/webgo.git
$ cd webgo
$ docker run \
-p 8080:8080 \
-p 9595:9595 \
-v ${PWD}:/go/src/github.com/naughtygopher/webgo/ \
-w /go/src/github.com/naughtygopher/webgo/cmd \
--rm -ti golang:latest go run *.go

Info 2023/02/05 08:51:26 HTTP server, listening on :8080
Info 2023/02/05 08:51:26 HTTPS server, listening on :9595
```

You can try the following API calls with the sample app. It also uses all the features provided by webgo

1. `http://localhost:8080/`
   - Loads an HTML page
2. `http://localhost:8080/matchall/`
   - Route with wildcard parameter configured
   - All URIs which begin with `/matchall` will be matched because it has a wildcard variable
   - e.g.
     - http://localhost:8080/matchall/hello
     - http://localhost:8080/matchall/hello/world
     - http://localhost:8080/matchall/hello/world/user
3. `http://localhost:8080/api/<param>`
   - Route with a named 'param' configured
   - It will match all requests which match `/api/<single parameter>`
   - e.g.
     - http://localhost:8080/api/hello
     - http://localhost:8080/api/world
4. `http://localhost:8080/error-setter`
   - Route which sets an error and sets response status 500
5. `http://localhost:8080/v7.0.0/api/<param>`
   - Route with a named 'param' configured
   - It will match all requests which match `/v7.0.0/api/<single parameter>`
   - e.g.
     - http://localhost:8080/v7.0.0/api/hello
     - http://localhost:8080/v7.0.0/api/world

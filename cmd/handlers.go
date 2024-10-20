package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/naughtygopher/webgo/v7"
	"github.com/naughtygopher/webgo/v7/extensions/sse"
)

// StaticFilesHandler is used to serve static files
func StaticFilesHandler(rw http.ResponseWriter, r *http.Request) {
	wctx := webgo.Context(r)
	// '..' is replaced to prevent directory traversal which could go out of static directory
	path := strings.ReplaceAll(wctx.Params()["w"], "..", "-")
	path = strings.ReplaceAll(path, "~", "-")

	rw.Header().Set("Last-Modified", lastModified)
	http.ServeFile(rw, r, fmt.Sprintf("./static/%s", path))
}

func OriginalResponseWriterHandler(w http.ResponseWriter, r *http.Request) {
	rw := webgo.OriginalResponseWriter(w)
	if rw == nil {
		webgo.Send(w, "text/html", "got nil", http.StatusPreconditionFailed)
		return
	}

	webgo.Send(w, "text/html", "success", http.StatusOK)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fs, err := os.OpenFile("./static/index.html", os.O_RDONLY, 0600)
	if err != nil {
		webgo.SendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	info, err := fs.Stat()
	if err != nil {
		webgo.SendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]byte, info.Size())
	_, err = fs.Read(out)
	if err != nil {
		webgo.SendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pushHomepage(r, w)

	_, _ = w.Write(out)

}

func pushCSS(pusher http.Pusher, r *http.Request, path string) {
	cssOpts := &http.PushOptions{
		Header: http.Header{
			"Accept-Encoding": r.Header["Accept-Encoding"],
			"Content-Type":    []string{"text/css; charset=UTF-8"},
		},
	}
	err := pusher.Push(path, cssOpts)
	if err != nil {
		webgo.LOGHANDLER.Error(err)
	}
}

func pushJS(pusher http.Pusher, r *http.Request, path string) {
	cssOpts := &http.PushOptions{
		Header: http.Header{
			"Accept-Encoding": r.Header["Accept-Encoding"],
			"Content-Type":    []string{"application/javascript"},
		},
	}
	err := pusher.Push(path, cssOpts)
	if err != nil {
		webgo.LOGHANDLER.Error(err)
	}
}

func pushHomepage(r *http.Request, w http.ResponseWriter) {
	pusher, ok := w.(http.Pusher)
	if !ok {
		return
	}

	cp, _ := r.Cookie("pusher")
	if cp != nil {
		return
	}

	cookie := &http.Cookie{
		Name:   "pusher",
		Value:  "css,js",
		MaxAge: 300,
	}
	http.SetCookie(w, cookie)
	pushCSS(pusher, r, "/static/css/main.css")
	pushCSS(pusher, r, "/static/css/normalize.css")
	pushJS(pusher, r, "/static/js/main.js")
	pushJS(pusher, r, "/static/js/sse.js")
}

func SSEHandler(sse *sse.SSE) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := webgo.Context(r).Params()
		r.Header.Set(sse.ClientIDHeader, params["clientID"])

		err := sse.Handler(w, r)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Println("errorLogger:", err.Error())
			return
		}
	}
}

func ErrorSetterHandler(w http.ResponseWriter, r *http.Request) {
	err := errors.New("oh no, server error")
	webgo.SetError(r, err)

	webgo.R500(w, err.Error())
}

func ParamHandler(w http.ResponseWriter, r *http.Request) {
	// WebGo context
	wctx := webgo.Context(r)
	// URI parameters, map[string]string
	params := wctx.Params()
	// route, the webgo.Route which is executing this request
	route := wctx.Route
	webgo.R200(
		w,
		map[string]interface{}{
			"route_name":    route.Name,
			"route_pattern": route.Pattern,
			"params":        params,
			"chained":       r.Header.Get("chained"),
		},
	)
}

func InvalidJSONHandler(w http.ResponseWriter, r *http.Request) {
	webgo.R200(w, make(chan int))
}

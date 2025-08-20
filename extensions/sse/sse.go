// Package sse implements Server-Sent Events(SSE)
// This extension is compliant with any net/http implementation, and is not limited to WebGo.
package sse

import (
	"context"
	"net/http"
)

type SSE struct {
	// ClientIDHeader is the HTTP request header in which the client ID is set. Default is `sse-clientid`
	ClientIDHeader string
	// UnsupportedMessage is used to send the error response to client if the
	// server doesn't support SSE
	UnsupportedMessage func(http.ResponseWriter, *http.Request) error

	// OnCreateClient is a hook, for when a client is added to the active clients. count is the number
	// of active clients after adding the latest client
	OnCreateClient func(ctx context.Context, client *Client, count int)

	// OnRemoveClient is a hook, for when a client is removed from the active clients. count is the number
	// of active clients after removing a client
	OnRemoveClient func(ctx context.Context, clientID string, count int)

	// OnSend is a hook, which is called *after* a message is sent to a client
	OnSend func(ctx context.Context, client *Client, err error)
	// BeforeSend is a hook, which is called before starting to listen for messages to send
	BeforeSend func(ctx context.Context, client *Client)

	Clients ClientManager
}

// Handler returns an error rather than being directly used as an http.HandlerFunc,
// to let the user handle error. e.g. if the error has to be logged
func (sse *SSE) Handler(w http.ResponseWriter, r *http.Request) error {
	flusher, hasFlusher := w.(http.Flusher)
	if !hasFlusher {
		return sse.UnsupportedMessage(w, r)
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ctx := r.Context()
	clientID := r.Header.Get(sse.ClientIDHeader)
	client := sse.NewClient(ctx, w, clientID)
	defer sse.RemoveClient(ctx, clientID)

	sse.BeforeSend(ctx, client)
	for {
		select {

		case payload, ok := <-client.Msg:
			if !ok {
				return nil
			}

			_, err := w.Write(payload.Bytes())
			sse.OnSend(ctx, client, err)
			if err != nil {
				return err
			}

		case <-ctx.Done():
			{
				err := ctx.Err()
				sse.OnSend(ctx, client, err)
				return err
			}
		}

		flusher.Flush()
	}
}

// HandlerFunc is a convenience function which can be directly used with net/http implementations.
// Important: You cannot handle any error returned by the Handler
func (sse *SSE) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	_ = sse.Handler(w, r)
}

// Broadcast sends the message to all active clients
func (sse *SSE) Broadcast(msg Message) {
	sse.Clients.Range(func(cli *Client) {
		cli.Msg <- &msg
	})
}

func (sse *SSE) NewClient(ctx context.Context, w http.ResponseWriter, clientID string) *Client {
	cli, count := sse.Clients.New(ctx, w, clientID)
	sse.OnCreateClient(ctx, cli, count)
	return cli
}

func (sse *SSE) ActiveClients() int {
	return sse.Clients.Active()
}

func (sse *SSE) RemoveClient(ctx context.Context, clientID string) {
	sse.OnRemoveClient(
		ctx,
		clientID,
		sse.Clients.Remove(clientID),
	)
}

func (sse *SSE) Client(id string) *Client {
	return sse.Clients.Client(id)
}

func DefaultCreateHook(ctx context.Context, client *Client, count int)  {}
func DefaultRemoveHook(ctx context.Context, clientID string, count int) {}
func DefaultOnSend(ctx context.Context, client *Client, err error)      {}
func DefaultBeforeSend(ctx context.Context, client *Client)             {}

func New() *SSE {
	s := &SSE{
		ClientIDHeader: "sse-clientid",
		Clients:        NewClientManager(),

		UnsupportedMessage: DefaultUnsupportedMessageHandler,

		OnRemoveClient: DefaultRemoveHook,
		OnCreateClient: DefaultCreateHook,
		OnSend:         DefaultOnSend,
		BeforeSend:     DefaultBeforeSend,
	}

	return s
}

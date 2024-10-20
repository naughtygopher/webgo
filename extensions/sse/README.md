# Server-Sent Events

This extension provides support for [Server-Sent](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) Events for any net/http compliant http server.
It provides the following hooks for customizing the workflows:

1. `OnCreateClient func(ctx context.Context, client *Client, count int)`
2. `OnRemoveClient func(ctx context.Context, clientID string, count int)`
3. `OnSend func(ctx context.Context, client *Client, err error)`
4. `BeforeSend func(ctx context.Context, client *Client)`

```golang
import (
    "github.com/naughtygopher/webgo/extensions/sse"
)
func main() {
    sseService := sse.New()
    // broadcast to all active clients
    sseService.Broadcast(Message{
        Data:  "Hello world",
        Retry: time.MilliSecond,
	})

	// You can replace the ClientManager with your custom implementation, and override the default one
	// sseService.Clients = <your custom client manager>

    // send message to an individual client
    clientID := "cli123"
    cli := sseService.Client(clientID)
    if cli != nil {
        cli.Message <- &Message{Data: fmt.Sprintf("Hello %s",clientID), Retry: time.MilliSecond }
    }
}
```

## Client Manager

Client manager is an interface which is required for SSE to function, since this is an interface it's easier for you to replace if required. The default implementation is a simple one using mutex. If you have a custom implementation which is faster/better, you can easily swap out the default one.

```golang
type ClientManager interface {
	// New should return a new client, and the total number of active clients after adding this new one
	New(ctx context.Context, w http.ResponseWriter, clientID string) (*Client, int)
	// Range should iterate through all the active clients
	Range(func(*Client))
	// Remove should remove the active client given a clientID, and close the connection
	Remove(clientID string) int
	// Active returns the number of active clients
	Active() int
	// Clients returns a list of all active clients
	Clients() []*Client
	// Client returns *Client if clientID is active
	Client(clientID string) *Client
}
```

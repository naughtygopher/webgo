package sse

import (
	"context"
	"net/http"
	"sync"
)

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

type Client struct {
	ID             string
	Msg            chan *Message
	ResponseWriter http.ResponseWriter
	Ctx            context.Context
}

type Clients struct {
	clients   map[string]*Client
	locker    sync.Mutex
	MsgBuffer int
}

func (cs *Clients) New(ctx context.Context, w http.ResponseWriter, clientID string) (*Client, int) {
	mchan := make(chan *Message, cs.MsgBuffer)
	cli := &Client{
		ID:             clientID,
		Msg:            mchan,
		ResponseWriter: w,
		Ctx:            ctx,
	}

	cs.locker.Lock()
	defer cs.locker.Unlock()

	cs.clients[clientID] = cli

	return cli, len(cs.clients)
}

func (cs *Clients) Range(f func(cli *Client)) {
	cs.locker.Lock()
	copied := make([]*Client, 0, len(cs.clients))
	for clientID := range cs.clients {
		copied = append(copied, cs.clients[clientID])
	}
	cs.locker.Unlock()

	for i := range copied {
		f(copied[i])
	}
}

func (cs *Clients) Remove(clientID string) int {
	cs.locker.Lock()
	defer cs.locker.Unlock()

	delete(cs.clients, clientID)
	count := len(cs.clients)

	return count
}

func (cs *Clients) Active() int {
	cs.locker.Lock()
	defer cs.locker.Unlock()

	count := len(cs.clients)
	return count

}

// MessageChannels returns a slice of message channels of all clients
// which you can then use to send message concurrently
func (cs *Clients) Clients() []*Client {
	idx := 0
	cs.locker.Lock()
	defer cs.locker.Unlock()

	list := make([]*Client, len(cs.clients))
	for clientID := range cs.clients {
		cli := cs.clients[clientID]
		list[idx] = cli
		idx++
	}

	return list
}

func (cs *Clients) Client(clientID string) *Client {
	cs.locker.Lock()
	defer cs.locker.Unlock()
	cli := cs.clients[clientID]

	return cli
}

func NewClientManager() ClientManager {
	return &Clients{
		clients: make(map[string]*Client),
		locker:  sync.Mutex{},
	}
}

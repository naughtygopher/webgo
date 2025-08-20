package sse

import (
	"context"
	"net/http"
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

type eventType int

const (
	eTypeNewClient eventType = iota
	eTypeClientList
	eTypeRemoveClient
	eTypeActiveClientCount
	eTypeClient
)

func (et eventType) String() string {
	switch et {
	case eTypeNewClient:
		return "new_client"
	case eTypeClientList:
		return "client_list"
	case eTypeRemoveClient:
		return "remove_client"
	case eTypeActiveClientCount:
		return "active_client_count"
	}
	return "unknown"
}

type event struct {
	Type     eventType
	ClientID string
	Client   *Client
	Response chan *eventResponse
}
type eventResponse struct {
	Clients          []*Client
	RemainingClients int
	Client           *Client
}
type Clients struct {
	clients   map[string]*Client
	MsgBuffer int
	events    chan<- event
}

func (cs *Clients) listener(events <-chan event) {
	for ev := range events {
		switch ev.Type {
		case eTypeNewClient:
			cs.clients[ev.Client.ID] = ev.Client

		case eTypeClientList:
			copied := make([]*Client, 0, len(cs.clients))
			for clientID := range cs.clients {
				copied = append(copied, cs.clients[clientID])
			}
			ev.Response <- &eventResponse{
				Clients: copied,
			}

		case eTypeRemoveClient:
			cli := cs.clients[ev.ClientID]
			if cli == nil {
				ev.Response <- nil
				continue
			}

			// Ctx.Done() is needed to close its streaming handler
			cli.Ctx.Done()
			delete(cs.clients, ev.ClientID)
			ev.Response <- nil

		case eTypeClient:
			ev.Response <- &eventResponse{
				Client: cs.clients[ev.ClientID],
			}
		}
	}
}

func (cs *Clients) New(ctx context.Context, w http.ResponseWriter, clientID string) (*Client, int) {
	mchan := make(chan *Message, cs.MsgBuffer)
	cli := &Client{
		ID:             clientID,
		Msg:            mchan,
		ResponseWriter: w,
		Ctx:            ctx,
	}

	cs.events <- event{
		Type:   eTypeNewClient,
		Client: cli,
	}

	return cli, len(cs.clients)
}

func (cs *Clients) Range(f func(cli *Client)) {
	rch := make(chan *eventResponse)
	cs.events <- event{
		Type:     eTypeClientList,
		Response: rch,
	}

	response := <-rch
	for i := range response.Clients {
		f(response.Clients[i])
	}

}

func (cs *Clients) Remove(clientID string) int {
	rch := make(chan *eventResponse)
	cs.events <- event{
		Type:     eTypeRemoveClient,
		ClientID: clientID,
		Response: rch,
	}

	<-rch

	return len(cs.clients)
}

func (cs *Clients) Active() int {
	return len(cs.clients)

}

// MessageChannels returns a slice of message channels of all clients
// which you can then use to send message concurrently
func (cs *Clients) Clients() []*Client {
	rch := make(chan *eventResponse)
	cs.events <- event{
		Type:     eTypeClientList,
		Response: rch,
	}

	response := <-rch
	return response.Clients
}

func (cs *Clients) Client(clientID string) *Client {
	rch := make(chan *eventResponse)
	cs.events <- event{
		Type:     eTypeClientList,
		Response: rch,
	}
	cli := <-rch
	return cli.Client
}

func NewClientManager() ClientManager {
	const buffer = 10
	events := make(chan event, buffer)
	cli := &Clients{
		clients:   make(map[string]*Client),
		events:    events,
		MsgBuffer: buffer,
	}
	go cli.listener(events)
	return cli
}

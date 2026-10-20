package sse

import (
	"context"
	"testing"
	"time"
)

func testClientManager(t *testing.T, clients ...*Client) *Clients {
	t.Helper()
	events := make(chan event)
	manager := &Clients{
		clients: make(map[string]*Client),
		events:  events,
	}
	for _, client := range clients {
		manager.clients[client.ID] = client
	}
	done := make(chan struct{})
	go func() {
		manager.listener(events)
		close(done)
	}()
	t.Cleanup(func() {
		close(events)
		<-done
	})
	return manager
}

func TestClientLookup(t *testing.T) {
	first := &Client{ID: "first"}
	second := &Client{ID: "second"}
	emptyID := &Client{ID: ""}
	manager := testClientManager(t, first, second, emptyID)
	service := &SSE{Clients: manager}

	for _, lookup := range []struct {
		name string
		get  func(string) *Client
	}{
		{name: "manager", get: manager.Client},
		{name: "service", get: service.Client},
	} {
		t.Run(lookup.name, func(t *testing.T) {
			for _, tc := range []struct {
				id   string
				want *Client
			}{
				{id: "first", want: first},
				{id: "second", want: second},
				{id: "", want: emptyID},
				{id: "missing"},
				{id: "first", want: first},
			} {
				t.Run(tc.id, func(t *testing.T) {
					if got := lookup.get(tc.id); got != tc.want {
						t.Fatalf("Client(%q) = %p, want %p", tc.id, got, tc.want)
					}
				})
			}
		})
	}

	if got := len(manager.Clients()); got != 3 {
		t.Fatalf("lookups left %d clients, want 3", got)
	}
}

func TestClientLookupEmptyManager(t *testing.T) {
	manager := testClientManager(t)
	for _, id := range []string{"missing", ""} {
		if got := manager.Client(id); got != nil {
			t.Fatalf("Client(%q) = %p, want nil", id, got)
		}
	}
}

func TestClientLookupAfterRemoval(t *testing.T) {
	first := &Client{ID: "first", Ctx: context.Background()}
	second := &Client{ID: "second", Ctx: context.Background()}
	manager := testClientManager(t, first, second)
	if got := manager.Client(first.ID); got != first {
		t.Fatalf("Client(%q) = %p, want %p", first.ID, got, first)
	}
	if got := manager.Remove(first.ID); got != 1 {
		t.Fatalf("Remove(%q) left %d clients, want 1", first.ID, got)
	}
	if got := manager.Client(first.ID); got != nil {
		t.Fatalf("removed Client(%q) = %p, want nil", first.ID, got)
	}
	if got := manager.Client(second.ID); got != second {
		t.Fatalf("Client(%q) = %p, want %p", second.ID, got, second)
	}
}

func TestClientLookupDirectedMessage(t *testing.T) {
	first := &Client{ID: "first", Msg: make(chan *Message, 1)}
	second := &Client{ID: "second", Msg: make(chan *Message, 1)}
	manager := testClientManager(t, first, second)
	service := &SSE{Clients: manager}
	client := service.Client(second.ID)
	if client == nil {
		t.Fatalf("Client(%q) = nil", second.ID)
	}
	message := &Message{Data: "hello"}
	client.Msg <- message
	select {
	case got := <-second.Msg:
		if got != message {
			t.Fatalf("received message %p, want %p", got, message)
		}
	case <-time.After(time.Second):
		t.Fatal("selected client received no message")
	}
	select {
	case <-first.Msg:
		t.Fatal("unselected client received a message")
	default:
	}
}

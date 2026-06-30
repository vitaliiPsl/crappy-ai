package ask

import (
	"context"
	"testing"
	"time"
)

func TestAwaitResolvesByID(t *testing.T) {
	p := NewBroker()

	emitted := make(chan Request, 1)

	answered := make(chan Response, 1)
	go func() {
		resp, err := p.Await(context.Background(), Request{ID: "r1", Title: "go?"}, func(r Request) {
			emitted <- r
		})
		if err != nil {
			t.Errorf("Await: %v", err)
		}

		answered <- resp
	}()

	select {
	case r := <-emitted:
		if r.ID != "r1" {
			t.Fatalf("emitted request id = %q, want r1", r.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("emit was not called")
	}

	if !p.Resolve(Response{RequestID: "r1", OptionID: "allow"}) {
		t.Fatal("Resolve reported no waiter")
	}

	select {
	case resp := <-answered:
		if resp.OptionID != "allow" {
			t.Fatalf("Await returned %q, want allow", resp.OptionID)
		}
	case <-time.After(time.Second):
		t.Fatal("Await did not return after Resolve")
	}
}

func TestAwaitContextCancel(t *testing.T) {
	p := NewBroker()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := p.Await(ctx, Request{ID: "r1"}, nil); err == nil {
		t.Fatal("Await with cancelled context should return an error")
	}

	if p.Resolve(Response{RequestID: "r1"}) {
		t.Fatal("Resolve found a waiter that should have been forgotten")
	}
}

func TestResolveNoWaiter(t *testing.T) {
	p := NewBroker()

	if p.Resolve(Response{RequestID: "missing"}) {
		t.Fatal("Resolve should report no waiter for an unknown id")
	}
}

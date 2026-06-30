package ask

import (
	"context"
	"sync"
)

type Request struct {
	ID      string
	Title   string
	Detail  string
	Options []Option
}

type Option struct {
	ID    string
	Label string
}

type Response struct {
	RequestID string
	OptionID  string
}

type Asker interface {
	Ask(ctx context.Context, req Request) (Response, error)
}

type Broker struct {
	mu      sync.Mutex
	waiting map[string]chan Response
}

func NewBroker() *Broker {
	return &Broker{
		waiting: make(map[string]chan Response),
	}
}

func (b *Broker) Await(ctx context.Context, req Request, emit func(Request)) (Response, error) {
	ch := make(chan Response, 1)

	b.mu.Lock()
	b.waiting[req.ID] = ch
	b.mu.Unlock()

	defer b.forget(req.ID)

	if emit != nil {
		emit(req)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return Response{}, ctx.Err()
	}
}

func (b *Broker) Resolve(resp Response) bool {
	b.mu.Lock()

	ch, ok := b.waiting[resp.RequestID]
	if ok {
		delete(b.waiting, resp.RequestID)
	}
	b.mu.Unlock()

	if !ok {
		return false
	}

	ch <- resp

	return true
}

func (b *Broker) forget(id string) {
	b.mu.Lock()
	delete(b.waiting, id)
	b.mu.Unlock()
}

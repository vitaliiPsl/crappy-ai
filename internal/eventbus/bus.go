package eventbus

import "sync"

const defaultBuffer = 64

type Option func(*options)

type options struct {
	buffer int
}

func WithBuffer(n int) Option {
	return func(o *options) {
		o.buffer = n
	}
}

type Bus[E any] struct {
	buffer int

	mu     sync.Mutex
	subs   map[*Subscription[E]]struct{}
	closed bool
}

func New[E any](opts ...Option) *Bus[E] {
	cfg := options{buffer: defaultBuffer}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Bus[E]{
		buffer: cfg.buffer,
		subs:   make(map[*Subscription[E]]struct{}),
	}
}

func (b *Bus[E]) Subscribe() *Subscription[E] {
	sub := &Subscription[E]{
		bus:  b,
		ch:   make(chan E, b.buffer),
		done: make(chan struct{}),
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		sub.finish()

		return sub
	}

	b.subs[sub] = struct{}{}

	return sub
}

func (b *Bus[E]) Publish(ev E) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return
	}

	snapshot := make([]*Subscription[E], 0, len(b.subs))
	for sub := range b.subs {
		snapshot = append(snapshot, sub)
	}
	b.mu.Unlock()

	for _, sub := range snapshot {
		select {
		case sub.ch <- ev:
		case <-sub.done:
		}
	}
}

func (b *Bus[E]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	for sub := range b.subs {
		sub.finish()
	}

	b.subs = nil
}

func (b *Bus[E]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.subs)
}

func (b *Bus[E]) remove(sub *Subscription[E]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subs, sub)
}

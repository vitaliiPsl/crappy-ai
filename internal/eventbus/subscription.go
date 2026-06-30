package eventbus

import "sync"

type Subscription[E any] struct {
	bus  *Bus[E]
	ch   chan E
	done chan struct{}
	once sync.Once
}

func (s *Subscription[E]) Events() <-chan E {
	return s.ch
}

func (s *Subscription[E]) Done() <-chan struct{} {
	return s.done
}

func (s *Subscription[E]) Close() {
	s.bus.remove(s)
	s.finish()
}

func (s *Subscription[E]) finish() {
	s.once.Do(func() { close(s.done) })
}

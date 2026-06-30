package eventbus

import (
	"testing"
	"time"
)

func recv(t *testing.T, ch <-chan int) int {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")

		return 0
	}
}

func TestPublishReachesAllSubscribers(t *testing.T) {
	b := New[int]()

	a := b.Subscribe()
	c := b.Subscribe()

	b.Publish(42)

	if got := recv(t, a.Events()); got != 42 {
		t.Fatalf("a got %d, want 42", got)
	}

	if got := recv(t, c.Events()); got != 42 {
		t.Fatalf("c got %d, want 42", got)
	}

	if b.Len() != 2 {
		t.Fatalf("Len = %d, want 2", b.Len())
	}
}

func TestCloseSubscriptionStopsDelivery(t *testing.T) {
	b := New[int]()

	sub := b.Subscribe()
	sub.Close()

	if b.Len() != 0 {
		t.Fatalf("Len = %d, want 0 after close", b.Len())
	}

	select {
	case <-sub.Done():
	default:
		t.Fatal("Done should be closed after Close")
	}

	// Publishing must not block on the closed subscriber, and nothing arrives.
	b.Publish(1)

	select {
	case v := <-sub.Events():
		t.Fatalf("closed subscription received %d", v)
	default:
	}
}

func TestCloseIsIdempotent(_ *testing.T) {
	b := New[int]()
	sub := b.Subscribe()

	sub.Close()
	sub.Close() // must not panic (double close of done)
}

func TestBusCloseEndsSubscriptions(t *testing.T) {
	b := New[int]()
	sub := b.Subscribe()

	b.Close()

	select {
	case <-sub.Done():
	default:
		t.Fatal("Bus.Close should end the subscription's Done")
	}

	// A subscription's own Close after Bus.Close must still be safe.
	sub.Close()

	// Subscribe after Close yields an already-finished subscription.
	after := b.Subscribe()
	select {
	case <-after.Done():
	default:
		t.Fatal("Subscribe after Close should be already done")
	}

	b.Publish(1) // no-op, must not panic
}

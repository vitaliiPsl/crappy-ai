package background

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestManagerStartWaitSuccess(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	release := make(chan struct{})

	started, err := manager.Start("worker", func(context.Context) (string, error) {
		<-release

		return "done", nil
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if started.ID == "" || started.Tool != "worker" || started.Status != StatusRunning {
		t.Fatalf("started = %+v, want running worker job with ID", started)
	}

	close(release)

	done, err := manager.Wait(context.Background(), started.ID)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if done.Status != StatusSucceeded || done.Output != "done" || done.CompletedAt == nil {
		t.Fatalf("done = %+v, want succeeded with output and completion time", done)
	}
}

func TestManagerStartFailure(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	wantErr := errors.New("boom")

	started, err := manager.Start("worker", func(context.Context) (string, error) {
		return "", wantErr
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	done, err := manager.Wait(context.Background(), started.ID)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if done.Status != StatusFailed || done.Error != "boom" {
		t.Fatalf("done = %+v, want failed boom", done)
	}
}

func TestManagerCancelRunningJob(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	ctxDone := make(chan struct{})

	started, err := manager.Start("worker", func(ctx context.Context) (string, error) {
		<-ctx.Done()
		close(ctxDone)

		return "", ctx.Err()
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	canceled, err := manager.Cancel(started.ID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if canceled.Status != StatusCanceled {
		t.Fatalf("canceled status = %q, want canceled", canceled.Status)
	}

	<-ctxDone

	done, err := manager.Wait(context.Background(), started.ID)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if done.Status != StatusCanceled {
		t.Fatalf("done status = %q, want canceled", done.Status)
	}
}

func TestManagerStartConcurrentIDsAreUnique(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	const count = 64

	var wg sync.WaitGroup

	ids := make(chan string, count)
	for range count {
		wg.Add(1)

		go func() {
			defer wg.Done()

			started, err := manager.Start("worker", func(context.Context) (string, error) {
				return "ok", nil
			})
			if err != nil {
				t.Errorf("Start: %v", err)

				return
			}

			ids <- started.ID
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[string]bool)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate job id %q", id)
		}

		seen[id] = true
	}

	if len(seen) != count {
		t.Fatalf("ids len = %d, want %d", len(seen), count)
	}
}

func TestManagerListNewestFirst(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	first, err := manager.Start("first", func(context.Context) (string, error) { return "first", nil })
	if err != nil {
		t.Fatalf("Start first: %v", err)
	}

	second, err := manager.Start("second", func(context.Context) (string, error) { return "second", nil })
	if err != nil {
		t.Fatalf("Start second: %v", err)
	}

	jobs, err := manager.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("jobs len = %d, want 2", len(jobs))
	}

	if jobs[0].ID != second.ID || jobs[1].ID != first.ID {
		t.Fatalf("jobs order = [%s %s], want newest first [%s %s]", jobs[0].ID, jobs[1].ID, second.ID, first.ID)
	}
}

func TestManagerWaitMissing(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	_, err := manager.Wait(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Wait error = %v, want ErrNotFound", err)
	}
}

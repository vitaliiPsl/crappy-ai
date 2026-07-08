package background

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestManagerStartWaitSuccess(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	release := make(chan struct{})

	jobs := manager.ForSession("session-1")

	started, err := jobs.Start("worker", func(context.Context) (kit.ToolOutput, error) {
		<-release

		return kit.NewToolOutput(kit.NewTextContent("done")), nil
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if started.ID == "" || started.SessionID != "session-1" || started.Tool != "worker" || started.Status != StatusRunning {
		t.Fatalf("started = %+v, want running worker job with ID", started)
	}

	close(release)

	done, err := jobs.Wait(context.Background(), started.ID)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if done.Status != StatusSucceeded || done.Output == nil || kit.ContentsText(done.Output.Content) != "done" || done.CompletedAt == nil {
		t.Fatalf("done = %+v, want succeeded with output and completion time", done)
	}
}

func TestManagerStartFailure(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	wantErr := errors.New("boom")

	jobs := manager.ForSession("session-1")

	started, err := jobs.Start("worker", func(context.Context) (kit.ToolOutput, error) {
		return kit.ToolOutput{}, wantErr
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	done, err := jobs.Wait(context.Background(), started.ID)
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

	jobs := manager.ForSession("session-1")

	started, err := jobs.Start("worker", func(ctx context.Context) (kit.ToolOutput, error) {
		<-ctx.Done()
		close(ctxDone)

		return kit.ToolOutput{}, ctx.Err()
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	canceled, err := jobs.Cancel(started.ID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if canceled.Status != StatusCanceled {
		t.Fatalf("canceled status = %q, want canceled", canceled.Status)
	}

	<-ctxDone

	done, err := jobs.Wait(context.Background(), started.ID)
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

			started, err := manager.ForSession("session-1").Start("worker", func(context.Context) (kit.ToolOutput, error) {
				return kit.NewToolOutput(kit.NewTextContent("ok")), nil
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

	jobs := manager.ForSession("session-1")

	first, err := jobs.Start("first", func(context.Context) (kit.ToolOutput, error) {
		return kit.NewToolOutput(kit.NewTextContent("first")), nil
	})
	if err != nil {
		t.Fatalf("Start first: %v", err)
	}

	second, err := jobs.Start("second", func(context.Context) (kit.ToolOutput, error) {
		return kit.NewToolOutput(kit.NewTextContent("second")), nil
	})
	if err != nil {
		t.Fatalf("Start second: %v", err)
	}

	list, err := jobs.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("jobs len = %d, want 2", len(list))
	}

	if list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("jobs order = [%s %s], want newest first [%s %s]", list[0].ID, list[1].ID, second.ID, first.ID)
	}
}

func TestManagerWaitMissing(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	_, err := manager.ForSession("session-1").Wait(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Wait error = %v, want ErrNotFound", err)
	}
}

func TestManagerFiltersBySession(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	sessionOneJobs := manager.ForSession("session-1")
	sessionTwoJobs := manager.ForSession("session-2")
	allJobs := manager.ForSession("")

	sessionOne, err := sessionOneJobs.Start("one", func(context.Context) (kit.ToolOutput, error) {
		return kit.NewToolOutput(kit.NewTextContent("one")), nil
	})
	if err != nil {
		t.Fatalf("Start session one: %v", err)
	}

	sessionTwo, err := sessionTwoJobs.Start("two", func(context.Context) (kit.ToolOutput, error) {
		return kit.NewToolOutput(kit.NewTextContent("two")), nil
	})
	if err != nil {
		t.Fatalf("Start session two: %v", err)
	}

	jobs, err := sessionOneJobs.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(jobs) != 1 || jobs[0].ID != sessionOne.ID {
		t.Fatalf("session jobs = %+v, want only %s", jobs, sessionOne.ID)
	}

	all, err := allJobs.List()
	if err != nil {
		t.Fatalf("List all: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("all jobs len = %d, want 2", len(all))
	}

	if _, err := sessionOneJobs.Get(sessionTwo.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get other session error = %v, want ErrNotFound", err)
	}

	if _, err := sessionOneJobs.Wait(context.Background(), sessionTwo.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Wait other session error = %v, want ErrNotFound", err)
	}

	if _, err := sessionOneJobs.Cancel(sessionTwo.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Cancel other session error = %v, want ErrNotFound", err)
	}

	got, err := allJobs.Get(sessionTwo.ID)
	if err != nil {
		t.Fatalf("Get all sessions: %v", err)
	}

	if got.ID != sessionTwo.ID {
		t.Fatalf("got job = %s, want %s", got.ID, sessionTwo.ID)
	}
}

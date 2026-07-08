package background

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc

	store   Store
	mu      sync.Mutex
	next    atomic.Uint64
	running map[string]*runningJob
}

type runningJob struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewManager(ctx context.Context) *Manager {
	ctx, cancel := context.WithCancel(ctx)

	return &Manager{
		ctx:     ctx,
		cancel:  cancel,
		store:   NewMemoryStore(),
		running: make(map[string]*runningJob),
	}
}

func (m *Manager) Close() {
	m.cancel()

	m.mu.Lock()

	running := make([]*runningJob, 0, len(m.running))
	for id, job := range m.running {
		delete(m.running, id)
		running = append(running, job)
	}
	m.mu.Unlock()

	for _, job := range running {
		job.cancel()
		closeDone(job)
	}
}

func (m *Manager) Start(sessionID, toolName string, run func(context.Context) (kit.ToolOutput, error)) (Job, error) {
	if err := m.ctx.Err(); err != nil {
		return Job{}, err
	}

	ctx, cancel := context.WithCancel(m.ctx)
	running := &runningJob{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	id := fmt.Sprintf("job_%d", m.next.Add(1))

	job := Job{
		ID:        id,
		SessionID: sessionID,
		Tool:      toolName,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}

	if err := m.store.Create(job); err != nil {
		cancel()

		return Job{}, err
	}

	m.mu.Lock()
	m.running[job.ID] = running
	m.mu.Unlock()

	go func() {
		var (
			output kit.ToolOutput
			err    error
		)

		defer func() {
			if recovered := recover(); recovered != nil {
				output = kit.ToolOutput{}
				err = fmt.Errorf("job panicked: %v", recovered)
			}

			m.finish(job.ID, output, err)
		}()

		output, err = run(ctx)
	}()

	return job, nil
}

func (m *Manager) Get(sessionID, id string) (Job, error) {
	job, err := m.store.Get(id)
	if err != nil {
		return Job{}, err
	}

	if !matchesSession(job, sessionID) {
		return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	return job, nil
}

func (m *Manager) List(sessionID string) ([]Job, error) {
	jobs, err := m.store.List()
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		filtered := jobs[:0]
		for _, job := range jobs {
			if matchesSession(job, sessionID) {
				filtered = append(filtered, job)
			}
		}

		jobs = filtered
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].StartedAt.After(jobs[j].StartedAt)
	})

	return jobs, nil
}

func (m *Manager) Wait(ctx context.Context, sessionID, id string) (Job, error) {
	job, err := m.Get(sessionID, id)
	if err != nil {
		return Job{}, err
	}

	m.mu.Lock()
	running := m.running[id]
	m.mu.Unlock()

	if running == nil {
		return job, nil
	}

	select {
	case <-running.done:
		return m.Get(sessionID, id)
	case <-ctx.Done():
		return Job{}, ctx.Err()
	}
}

func (m *Manager) Cancel(sessionID, id string) (Job, error) {
	job, err := m.store.Get(id)
	if err != nil {
		return Job{}, err
	}

	if !matchesSession(job, sessionID) {
		return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	m.mu.Lock()

	running := m.running[id]
	if running != nil {
		delete(m.running, id)
	}
	m.mu.Unlock()

	if running != nil {
		running.cancel()
		closeDone(running)
	}

	if job.Status != StatusRunning {
		return job, nil
	}

	now := time.Now()
	job.Status = StatusCanceled
	job.CompletedAt = &now

	job.Error = context.Canceled.Error()
	if err := m.store.Update(job); err != nil {
		return Job{}, err
	}

	return job, nil
}

func (m *Manager) finish(id string, output kit.ToolOutput, err error) {
	m.mu.Lock()

	running := m.running[id]
	if running != nil {
		delete(m.running, id)
		closeDone(running)
	}
	m.mu.Unlock()

	job, getErr := m.store.Get(id)
	if getErr != nil || job.Status == StatusCanceled {
		return
	}

	now := time.Now()

	job.CompletedAt = &now
	if err != nil {
		job.Status = StatusFailed
		job.Error = err.Error()
	} else {
		job.Status = StatusSucceeded
		job.Output = &output
	}

	_ = m.store.Update(job)
}

func closeDone(job *runningJob) {
	select {
	case <-job.done:
	default:
		close(job.done)
	}
}

func matchesSession(job Job, sessionID string) bool {
	return sessionID == "" || job.SessionID == sessionID
}

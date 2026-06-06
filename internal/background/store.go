package background

import (
	"fmt"
	"sync"
)

type Store interface {
	Get(id string) (Job, error)
	List() ([]Job, error)
	Create(job Job) error
	Update(job Job) error
}

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]Job)}
}

func (s *MemoryStore) Get(id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	return job, nil
}

func (s *MemoryStore) List() ([]Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *MemoryStore) Create(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job already exists: %s", job.ID)
	}

	s.jobs[job.ID] = job

	return nil
}

func (s *MemoryStore) Update(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[job.ID]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, job.ID)
	}

	s.jobs[job.ID] = job

	return nil
}

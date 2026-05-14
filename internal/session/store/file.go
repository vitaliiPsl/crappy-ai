package store

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	sessionFile   = "session.json"
	eventsFile    = "events.jsonl"
	scannerBuffer = 1 << 20
)

type FileStore struct {
	mu    sync.RWMutex
	cache map[string]*session.Session
	dir   string
}

func NewFileStore(dir string) (*FileStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("sessions dir is required")
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}

	return &FileStore{
		cache: make(map[string]*session.Session),
		dir:   dir,
	}, nil
}

func (st *FileStore) Create(_ context.Context, title string) (*session.Session, error) {
	now := time.Now()

	s := &session.Session{
		ID:        uuid.NewString(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	st.mu.Lock()
	st.cache[s.ID] = s
	st.mu.Unlock()

	if err := st.writeSessionFile(s); err != nil {
		return nil, fmt.Errorf("persist new session: %w", err)
	}

	snapshot := *s

	return &snapshot, nil
}

func (st *FileStore) Get(_ context.Context, id string) (*session.Session, error) {
	st.mu.RLock()
	s, ok := st.cache[id]
	st.mu.RUnlock()

	if ok {
		snapshot := *s

		return &snapshot, nil
	}

	s, err := st.readSessionFile(id)
	if err != nil {
		return nil, err
	}

	st.mu.Lock()
	st.cache[id] = s
	st.mu.Unlock()

	snapshot := *s

	return &snapshot, nil
}

func (st *FileStore) List(_ context.Context) ([]*session.Session, error) {
	entries, err := os.ReadDir(st.dir)
	if err != nil {
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var sessions []*session.Session
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		id := e.Name()

		st.mu.RLock()
		s, cached := st.cache[id]
		st.mu.RUnlock()

		if cached {
			snapshot := *s
			sessions = append(sessions, &snapshot)

			continue
		}

		s, err := st.readSessionFile(id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping unreadable session %s: %v\n", id, err)

			continue
		}

		st.mu.Lock()
		st.cache[id] = s
		st.mu.Unlock()

		snapshot := *s
		sessions = append(sessions, &snapshot)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

func (st *FileStore) Delete(_ context.Context, id string) error {
	st.mu.Lock()
	delete(st.cache, id)
	st.mu.Unlock()

	if err := os.RemoveAll(st.sessionDir(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete session dir: %w", err)
	}

	return nil
}

func (st *FileStore) Save(_ context.Context, sess *session.Session) error {
	snapshot := *sess

	st.mu.Lock()
	st.cache[sess.ID] = &snapshot
	st.mu.Unlock()

	return st.writeSessionFile(sess)
}

func (st *FileStore) AppendEvents(_ context.Context, id string, events ...session.Event) error {
	st.mu.RLock()
	sess, ok := st.cache[id]
	st.mu.RUnlock()

	if !ok {
		s, err := st.readSessionFile(id)
		if err != nil {
			return fmt.Errorf("session %q not found: %w", id, err)
		}

		st.mu.Lock()
		st.cache[id] = s
		st.mu.Unlock()

		sess = s
	}

	dir := st.sessionDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	f, err := os.OpenFile(
		filepath.Join(dir, eventsFile),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("open events file: %w", err)
	}

	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			return fmt.Errorf("encode event: %w", err)
		}
	}

	sess.UpdatedAt = time.Now()

	return st.writeSessionFile(sess)
}

func (st *FileStore) LoadEvents(_ context.Context, id string) ([]session.Event, error) {
	path := filepath.Join(st.sessionDir(id), eventsFile)

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	defer func() { _ = f.Close() }()

	var events []session.Event

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, scannerBuffer), scannerBuffer)

	for scanner.Scan() {
		var ev session.Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}

		events = append(events, ev)
	}

	return events, scanner.Err()
}

func (st *FileStore) readSessionFile(id string) (*session.Session, error) {
	f, err := os.Open(filepath.Join(st.sessionDir(id), sessionFile))
	if err != nil {
		return nil, fmt.Errorf("open session.json: %w", err)
	}

	defer func() { _ = f.Close() }()

	var s session.Session
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode session.json: %w", err)
	}

	return &s, nil
}

func (st *FileStore) writeSessionFile(s *session.Session) error {
	dir := st.sessionDir(s.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	tmp := filepath.Join(dir, sessionFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmp, filepath.Join(dir, sessionFile))
}

func (st *FileStore) sessionDir(id string) string {
	return filepath.Join(st.dir, id)
}

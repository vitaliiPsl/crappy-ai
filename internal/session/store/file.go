package store

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const (
	sessionFile   = "session.json"
	eventsFile    = "events.jsonl"
	artifactsDir  = "artifacts"
	artifactExt   = ".json"
	scannerBuffer = 1 << 20
)

var artifactNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var _ session.ArtifactStore = (*FileStore)(nil)

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

func (st *FileStore) Create(_ context.Context, params session.CreateParams) (*session.Session, error) {
	now := time.Now()

	s := &session.Session{
		ID:        uuid.NewString(),
		ParentID:  params.ParentID,
		Title:     params.Title,
		Cwd:       params.Cwd,
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

		if !cached {
			var err error

			s, err = st.readSessionFile(id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping unreadable session %s: %v\n", id, err)

				continue
			}

			st.mu.Lock()
			st.cache[id] = s
			st.mu.Unlock()
		}

		if s.ParentID != "" {
			continue
		}

		snapshot := *s
		sessions = append(sessions, &snapshot)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

func (st *FileStore) Delete(ctx context.Context, id string) error {
	children, err := st.childIDs(id)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := st.Delete(ctx, child); err != nil {
			return err
		}
	}

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
	sess, err := st.ensureSession(id)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", id, err)
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

	return st.touch(sess)
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

func (st *FileStore) SaveArtifact(_ context.Context, id, name string, value any) error {
	path, err := st.artifactPath(id, name)
	if err != nil {
		return err
	}

	sess, err := st.ensureSession(id)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", id, err)
	}

	if err := os.MkdirAll(st.artifactsDir(id), 0o700); err != nil {
		return fmt.Errorf("create artifacts dir: %w", err)
	}

	if err := writeJSONFile(path, value); err != nil {
		return fmt.Errorf("save artifact %q: %w", name, err)
	}

	return st.touch(sess)
}

func (st *FileStore) LoadArtifact(_ context.Context, id, name string, value any) (bool, error) {
	path, err := st.artifactPath(id, name)
	if err != nil {
		return false, err
	}

	if _, err := st.ensureSession(id); err != nil {
		return false, fmt.Errorf("session %q not found: %w", id, err)
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("open artifact %q: %w", name, err)
	}

	defer func() { _ = f.Close() }()

	if err := json.NewDecoder(f).Decode(value); err != nil {
		return false, fmt.Errorf("decode artifact %q: %w", name, err)
	}

	return true, nil
}

func (st *FileStore) DeleteArtifact(_ context.Context, id, name string) error {
	path, err := st.artifactPath(id, name)
	if err != nil {
		return err
	}

	sess, err := st.ensureSession(id)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", id, err)
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("delete artifact %q: %w", name, err)
	}

	return st.touch(sess)
}

func (st *FileStore) ListArtifacts(_ context.Context, id string) ([]string, error) {
	if _, err := st.ensureSession(id); err != nil {
		return nil, fmt.Errorf("session %q not found: %w", id, err)
	}

	entries, err := os.ReadDir(st.artifactsDir(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read artifacts dir: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name, ok := artifactName(entry.Name())
		if !ok {
			continue
		}

		names = append(names, name)
	}

	sort.Strings(names)

	return names, nil
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

	return writeJSONFile(filepath.Join(dir, sessionFile), s)
}

func (st *FileStore) sessionDir(id string) string {
	return filepath.Join(st.dir, id)
}

func (st *FileStore) artifactsDir(id string) string {
	return filepath.Join(st.sessionDir(id), artifactsDir)
}

func (st *FileStore) artifactPath(id, name string) (string, error) {
	if err := validateArtifactName(name); err != nil {
		return "", err
	}

	return filepath.Join(st.artifactsDir(id), name+artifactExt), nil
}

func (st *FileStore) touch(sess *session.Session) error {
	sess.UpdatedAt = time.Now()

	return st.writeSessionFile(sess)
}

func (st *FileStore) childIDs(parentID string) ([]string, error) {
	entries, err := os.ReadDir(st.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var ids []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == parentID {
			continue
		}

		s, err := st.ensureSession(e.Name())
		if err != nil {
			continue
		}

		if s.ParentID == parentID {
			ids = append(ids, s.ID)
		}
	}

	return ids, nil
}

func (st *FileStore) ensureSession(id string) (*session.Session, error) {
	st.mu.RLock()
	sess, ok := st.cache[id]
	st.mu.RUnlock()

	if ok {
		return sess, nil
	}

	sess, err := st.readSessionFile(id)
	if err != nil {
		return nil, err
	}

	st.mu.Lock()
	st.cache[id] = sess
	st.mu.Unlock()

	return sess, nil
}

func validateArtifactName(name string) error {
	if !artifactNamePattern.MatchString(name) {
		return fmt.Errorf("invalid artifact name %q", name)
	}

	return nil
}

func artifactName(filename string) (string, bool) {
	name, ok := strings.CutSuffix(filename, artifactExt)
	if !ok || validateArtifactName(name) != nil {
		return "", false
	}

	return name, true
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

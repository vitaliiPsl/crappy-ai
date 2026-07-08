package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const testSessionID = "session-1"

type artifactStore struct {
	data    map[string][]byte
	loadErr error
	saveErr error
}

func newArtifactStore() *artifactStore {
	return &artifactStore{data: make(map[string][]byte)}
}

func (s *artifactStore) SaveArtifact(_ context.Context, _ string, name string, value any) error {
	if s.saveErr != nil {
		return s.saveErr
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	s.data[name] = data

	return nil
}

func (s *artifactStore) LoadArtifact(_ context.Context, _ string, name string, value any) (bool, error) {
	if s.loadErr != nil {
		return false, s.loadErr
	}

	data, ok := s.data[name]
	if !ok {
		return false, nil
	}

	return true, json.Unmarshal(data, value)
}

func (s *artifactStore) DeleteArtifact(context.Context, string, string) error {
	return nil
}

func (s *artifactStore) ListArtifacts(context.Context, string) ([]string, error) {
	return nil, nil
}

func TestWritePlanTool_SavesPlanArtifact(t *testing.T) {
	store := newArtifactStore()
	tool := newTool(testSessionID, store)

	out, err := tool.Execute(kit.NewRunContext(context.Background()), kit.NewToolCall("call-1", toolName, map[string]any{
		"explanation": "Need a few steps",
		"items": []any{
			map[string]any{"step": "Inspect code", "status": string(StatusCompleted)},
			map[string]any{"step": "Implement planning", "status": string(StatusInProgress)},
		},
	}))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := kit.ContentsText(out.Content); got != toolResult {
		t.Fatalf("output = %q, want %q", got, toolResult)
	}

	var got Plan

	ok, err := store.LoadArtifact(context.Background(), testSessionID, ArtifactName, &got)
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}

	if !ok {
		t.Fatal("plan artifact was not saved")
	}

	if got.Explanation != "Need a few steps" || len(got.Items) != 2 {
		t.Fatalf("plan = %+v, want saved input", got)
	}
}

func TestWritePlanTool_PropagatesSaveError(t *testing.T) {
	wantErr := errors.New("disk full")
	store := newArtifactStore()
	store.saveErr = wantErr

	tool := newTool(testSessionID, store)

	_, err := tool.Execute(kit.NewRunContext(context.Background()), kit.NewToolCall("call-1", toolName, map[string]any{
		"items": []any{
			map[string]any{"step": "One", "status": string(StatusPending)},
		},
	}))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute error = %v, want wraps %v", err, wantErr)
	}
}

func TestCurrentPlanText_FormatsSavedPlan(t *testing.T) {
	store := newArtifactStore()
	if err := store.SaveArtifact(context.Background(), testSessionID, ArtifactName, Plan{
		Explanation: "Working backend first",
		Items: []Item{
			{Step: "Create artifact store", Status: StatusCompleted},
			{Step: "Add planning extension", Status: StatusInProgress},
		},
	}); err != nil {
		t.Fatalf("SaveArtifact: %v", err)
	}

	got, err := currentPlanText(context.Background(), testSessionID, store)
	if err != nil {
		t.Fatalf("currentPlanText: %v", err)
	}

	for _, want := range []string{
		"Current plan:",
		"Working backend first",
		"- [completed] Create artifact store",
		"- [in_progress] Add planning extension",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plan text missing %q:\n%s", want, got)
		}
	}
}

func TestCurrentPlanText_NoopsWithoutPlan(t *testing.T) {
	store := newArtifactStore()

	got, err := currentPlanText(context.Background(), testSessionID, store)
	if err != nil {
		t.Fatalf("currentPlanText: %v", err)
	}

	if got != "" {
		t.Fatalf("plan text = %q, want empty", got)
	}
}

func TestCurrentPlanText_PropagatesLoadError(t *testing.T) {
	wantErr := errors.New("cannot read artifact")
	store := newArtifactStore()
	store.loadErr = wantErr

	_, err := currentPlanText(context.Background(), testSessionID, store)
	if !errors.Is(err, wantErr) {
		t.Fatalf("currentPlanText error = %v, want wraps %v", err, wantErr)
	}
}

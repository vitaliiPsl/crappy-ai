package config

import (
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	settingsmodels "github.com/vitaliiPsl/crappy-ai/internal/settings/models"
)

func TestDefaultsAllowWorkspaceReads(t *testing.T) {
	got := defaults().Permissions.Allow
	want := []model.Rule{
		{Tool: "list", Pattern: "./**"},
		{Tool: "read_file", Pattern: "./**"},
	}

	if len(got) != len(want) {
		t.Fatalf("allow rules = %+v, want %+v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("allow rule %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDefaultModelExistsInBundledModels(t *testing.T) {
	cfg := defaults()
	models := settingsmodels.DefaultModels()[cfg.Provider]

	for _, model := range models {
		if model.ID == cfg.Model {
			return
		}
	}

	t.Fatalf("default model %q for provider %q is not in bundled model metadata", cfg.Model, cfg.Provider)
}

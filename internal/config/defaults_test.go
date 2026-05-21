package config

import (
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
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

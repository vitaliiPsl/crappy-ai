package model

import (
	"slices"
	"testing"
)

func TestPermissionsAddSkipsDuplicateRules(t *testing.T) {
	var permissions Permissions

	rule := Rule{Tool: "read_file", Pattern: "./**"}

	permissions.Add(Allow, rule)
	permissions.Add(Allow, rule)

	if !slices.Equal(permissions.Allow, []Rule{rule}) {
		t.Fatalf("allow rules = %+v, want one copy of %+v", permissions.Allow, rule)
	}
}

func TestPermissionsAddKeepsSameRuleUnderDifferentDecisions(t *testing.T) {
	var permissions Permissions

	rule := Rule{Tool: "read_file", Pattern: "./**"}

	permissions.Add(Deny, rule)
	permissions.Add(Allow, rule)

	if !slices.Equal(permissions.Deny, []Rule{rule}) {
		t.Fatalf("deny rules = %+v, want %+v", permissions.Deny, []Rule{rule})
	}

	if !slices.Equal(permissions.Allow, []Rule{rule}) {
		t.Fatalf("allow rules = %+v, want %+v", permissions.Allow, []Rule{rule})
	}
}

func TestMergeSkipsDuplicateRulesPerDecision(t *testing.T) {
	read := Rule{Tool: "read_file", Pattern: "./**"}
	list := Rule{Tool: "list", Pattern: "./**"}

	got := Merge(
		Permissions{Default: Ask, Allow: []Rule{read, list}},
		Permissions{Default: Deny, Allow: []Rule{read}},
	)

	want := Permissions{
		Default: Deny,
		Allow:   []Rule{read, list},
	}

	if got.Default != want.Default {
		t.Fatalf("default = %q, want %q", got.Default, want.Default)
	}

	if !slices.Equal(got.Allow, want.Allow) {
		t.Fatalf("allow rules = %+v, want %+v", got.Allow, want.Allow)
	}
}

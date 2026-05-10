package session

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestParseCommand(t *testing.T) {
	msg, ok := parseCommand("/help sessions")
	if !ok {
		t.Fatal("parseCommand did not parse command")
	}

	if msg.Name != "help" {
		t.Fatalf("Name = %q, want help", msg.Name)
	}

	if len(msg.Args) != 1 || msg.Args[0] != "sessions" {
		t.Fatalf("Args = %v, want [sessions]", msg.Args)
	}
}

func TestParseCommand_IgnoresNormalMessages(t *testing.T) {
	if _, ok := parseCommand("hello"); ok {
		t.Fatal("parseCommand parsed normal message")
	}

	if _, ok := parseCommand("/help\nmore"); ok {
		t.Fatal("parseCommand parsed multiline message")
	}
}

func TestFooterCommandSuggestionSelection(t *testing.T) {
	footer := newFooter(newCommandRegistry(), "test-model")
	footer.input.SetValue("/he")
	footer.suggestions.Update(footer.input.Value())

	if len(footer.suggestions.matches) != 1 || footer.suggestions.matches[0].Name != "help" {
		t.Fatalf("matches = %v, want help", footer.suggestions.matches)
	}

	var cmd tea.Cmd
	var consumed bool
	footer, cmd, consumed = footer.Update(key(tea.KeyEnter))
	if !consumed {
		t.Fatal("Update enter did not consume suggestion selection")
	}

	if cmd != nil {
		t.Fatal("Update enter returned command while selecting suggestion")
	}

	if got := footer.input.Value(); got != "/help" {
		t.Fatalf("input value = %q, want /help", got)
	}

	footer, cmd, consumed = footer.Update(key(tea.KeyEnter))
	if !consumed {
		t.Fatal("Update enter did not consume command submission")
	}

	if cmd == nil {
		t.Fatal("Update enter did not return command")
	}

	raw := cmd()
	msg, ok := raw.(commandMsg)
	if !ok {
		t.Fatalf("command msg = %T, want commandMsg", raw)
	}

	if msg.Name != "help" {
		t.Fatalf("Name = %q, want help", msg.Name)
	}
}

func TestFooterUpWithoutSuggestionsFallsThrough(t *testing.T) {
	footer := newFooter(newCommandRegistry(), "test-model")

	_, _, consumed := footer.Update(key(tea.KeyUp))
	if consumed {
		t.Fatal("Update up consumed without command suggestions")
	}
}

func key(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

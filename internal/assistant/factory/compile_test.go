package factory

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestStaticContextText(t *testing.T) {
	got := staticContext([]ContextPiece{
		{Content: " static one "},
		{Content: ""},
		{Content: "static two"},
		{Resolve: func(context.Context) (string, error) {
			return "dynamic", nil
		}},
	})

	want := "static one\n\nstatic two"
	if got != want {
		t.Fatalf("staticContext = %q, want %q", got, want)
	}
}

func TestDynamicContextPieces(t *testing.T) {
	pieces := dynamicContextPieces([]ContextPiece{
		{Content: "static"},
		{Name: "first", Resolve: func(context.Context) (string, error) {
			return "dynamic one", nil
		}},
		{Name: "second", Resolve: func(context.Context) (string, error) {
			return "dynamic two", nil
		}},
	})

	if len(pieces) != 2 || pieces[0].Name != "first" || pieces[1].Name != "second" {
		t.Fatalf("dynamicContextPieces = %+v, want first and second", pieces)
	}
}

func TestResolveDynamicContextAppendsInstructions(t *testing.T) {
	hook := resolveDynamicContext([]ContextPiece{
		{Resolve: func(context.Context) (string, error) {
			return " dynamic one ", nil
		}},
		{Resolve: func(context.Context) (string, error) {
			return "", nil
		}},
		{Resolve: func(context.Context) (string, error) {
			return "dynamic two", nil
		}},
	})

	req, err := hook(kit.NewRunContext(context.Background()), kit.ModelRequest{Instructions: "existing"})
	if err != nil {
		t.Fatalf("hook: %v", err)
	}

	want := "existing\n\ndynamic one\n\ndynamic two"
	if req.Instructions != want {
		t.Fatalf("Instructions = %q, want %q", req.Instructions, want)
	}
}

func TestCompileAddsContextOptions(t *testing.T) {
	compiled, err := Compile(AgentSpec{
		Context: []ContextPiece{
			{Content: " static one "},
			{Content: ""},
			{Content: "static two"},
			{Resolve: func(context.Context) (string, error) {
				return " dynamic one ", nil
			}},
			{Resolve: func(context.Context) (string, error) {
				return "dynamic two", nil
			}},
		},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if len(compiled.Options) != 2 {
		t.Fatalf("len(Options) = %d, want static and dynamic context options", len(compiled.Options))
	}
}

func TestResolveDynamicContextReturnsResolverError(t *testing.T) {
	want := errors.New("boom")

	hook := resolveDynamicContext([]ContextPiece{
		{
			Name:   "Dynamic context",
			Source: "test",
			Resolve: func(context.Context) (string, error) {
				return "", want
			},
		},
	})

	_, err := hook(kit.NewRunContext(context.Background()), kit.ModelRequest{})
	if !errors.Is(err, want) {
		t.Fatalf("hook error = %v, want %v", err, want)
	}
}

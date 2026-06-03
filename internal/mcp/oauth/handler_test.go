package oauth

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestHandlerPassiveAuthorizeReturnsAuthorizationRequired(t *testing.T) {
	oauthHandler, err := NewPassiveHandler(HandlerConfig{
		Config: &Config{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	err = oauthHandler.Authorize(context.Background(), nil, &http.Response{Body: http.NoBody})
	if !errors.Is(err, ErrAuthorizationRequired) {
		t.Fatalf("Authorize() error = %v, want ErrAuthorizationRequired", err)
	}
}

func TestHandlerTokenSourceUsesAuthorizerInPassiveMode(t *testing.T) {
	oauthHandler, err := NewPassiveHandler(HandlerConfig{
		Config: &Config{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	source, err := oauthHandler.TokenSource(context.Background())
	if err != nil {
		t.Fatalf("TokenSource() error = %v", err)
	}

	if source != nil {
		t.Fatal("TokenSource() = non-nil before authorization, want nil")
	}
}

func TestHandlerInteractiveAuthorizationIsConfigured(t *testing.T) {
	oauthHandler, err := NewInteractiveHandler(HandlerConfig{
		Config: &Config{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	appHandler, ok := oauthHandler.(*handler)
	if !ok {
		t.Fatalf("handler = %T, want *handler", oauthHandler)
	}

	if _, ok := appHandler.authorization.(interactiveAuthorization); !ok {
		t.Fatalf("authorization = %T, want interactiveAuthorization", appHandler.authorization)
	}
}

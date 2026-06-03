package oauth

import (
	"context"
	"io"
	"net/http"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

type authorizationBehavior interface {
	Authorize(context.Context, mcpauth.OAuthHandler, *http.Request, *http.Response) error
}

type passiveAuthorization struct{}

func (passiveAuthorization) Authorize(_ context.Context, _ mcpauth.OAuthHandler, _ *http.Request, resp *http.Response) error {
	closeAuthorizationResponse(resp)

	return ErrAuthorizationRequired
}

type interactiveAuthorization struct{}

func (interactiveAuthorization) Authorize(ctx context.Context, authorizer mcpauth.OAuthHandler, req *http.Request, resp *http.Response) error {
	return authorizer.Authorize(ctx, req, resp)
}

func closeAuthorizationResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)
}

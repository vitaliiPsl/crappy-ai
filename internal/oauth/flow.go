package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"

	"golang.org/x/oauth2"
)

// CodeFlow performs an OAuth authorization-code flow with PKCE.
type CodeFlow struct {
	Config     oauth2.Config
	HTTPClient *http.Client
}

// CodeFlowOptions adds parameters to the authorization and token requests.
type CodeFlowOptions struct {
	Authorization []oauth2.AuthCodeOption
	Token         []oauth2.AuthCodeOption
}

func (f CodeFlow) Authorize(ctx context.Context, callback Callback, options CodeFlowOptions) (*oauth2.Token, error) {
	if callback == nil {
		return nil, errors.New("oauth: callback is not configured")
	}

	verifier := oauth2.GenerateVerifier()

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	authOptions := append([]oauth2.AuthCodeOption(nil), options.Authorization...)
	authOptions = append(authOptions, oauth2.S256ChallengeOption(verifier))
	authURL := f.Config.AuthCodeURL(state, authOptions...)

	code, returnedState, err := callback.Wait(ctx, authURL, f.Config.RedirectURL)
	if err != nil {
		return nil, err
	}

	if returnedState != state {
		return nil, errors.New("oauth: authorization state mismatch")
	}

	tokenOptions := append([]oauth2.AuthCodeOption(nil), options.Token...)
	tokenOptions = append(tokenOptions, oauth2.VerifierOption(verifier))

	return f.Config.Exchange(f.context(ctx), code, tokenOptions...)
}

func (f CodeFlow) TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
	return f.Config.TokenSource(f.context(ctx), token)
}

func IsInvalidGrant(err error) bool {
	var retrieveErr *oauth2.RetrieveError

	return errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant"
}

func (f CodeFlow) context(ctx context.Context) context.Context {
	if f.HTTPClient == nil {
		return ctx
	}

	return context.WithValue(ctx, oauth2.HTTPClient, f.HTTPClient)
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

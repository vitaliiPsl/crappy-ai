package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type CallbackServer struct {
	redirectURL string
	prompter    Prompter
}

type callbackResult struct {
	code  string
	state string
	err   error
}

func NewCallbackServer(redirectURL string, prompter Prompter) *CallbackServer {
	if prompter == nil {
		prompter = NewBrowserPrompter()
	}

	return &CallbackServer{
		redirectURL: redirectURL,
		prompter:    prompter,
	}
}

func (s *CallbackServer) Wait(ctx context.Context, authURL string) (string, string, error) {
	redirect, err := url.Parse(s.redirectURL)
	if err != nil {
		return "", "", err
	}

	resultCh := make(chan callbackResult, 1)
	server := &http.Server{Handler: callbackHandler(redirect.Path, resultCh)}

	listener, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		return "", "", fmt.Errorf("start oauth callback listener: %w", err)
	}

	go s.serve(server, listener, resultCh)
	defer s.shutdown(server)

	if err := s.prompter.Prompt(authURL); err != nil {
		return "", "", fmt.Errorf("prompt oauth authorization URL: %w", err)
	}

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case result := <-resultCh:
		return result.code, result.state, result.err
	}
}

func (s *CallbackServer) serve(server *http.Server, listener net.Listener, resultCh chan<- callbackResult) {
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		sendCallbackResult(resultCh, callbackResult{err: err})
	}
}

func (s *CallbackServer) shutdown(server *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)
}

func callbackHandler(path string, resultCh chan<- callbackResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != path {
			http.NotFound(w, req)

			return
		}

		query := req.URL.Query()
		if errText := query.Get("error"); errText != "" {
			sendCallbackResult(resultCh, callbackResult{err: fmt.Errorf("authorization failed: %s", errText)})

			_, _ = fmt.Fprintln(w, "Authorization failed. You can close this window.")

			return
		}

		code := query.Get("code")
		if code == "" {
			sendCallbackResult(resultCh, callbackResult{err: errors.New("authorization callback missing code")})

			_, _ = fmt.Fprintln(w, "Authorization callback is missing a code. You can close this window.")

			return
		}

		sendCallbackResult(resultCh, callbackResult{
			code:  code,
			state: query.Get("state"),
		})

		_, _ = fmt.Fprintln(w, "Authorization complete. You can close this window.")
	})
}

func sendCallbackResult(resultCh chan<- callbackResult, result callbackResult) {
	select {
	case resultCh <- result:
	default:
	}
}

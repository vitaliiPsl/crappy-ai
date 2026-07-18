package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

// BrowserCallback opens an authorization URL and captures its loopback redirect.
type BrowserCallback struct{}

type callbackResult struct {
	code  string
	state string
	err   error
}

func NewBrowserCallback() *BrowserCallback {
	return &BrowserCallback{}
}

func (c *BrowserCallback) Wait(ctx context.Context, authURL string, redirectURL string) (string, string, error) {
	redirect, err := url.Parse(redirectURL)
	if err != nil {
		return "", "", fmt.Errorf("oauth: parse redirect URL: %w", err)
	}

	resultCh := make(chan callbackResult, 1)
	server := &http.Server{Handler: callbackHandler(redirect.Path, resultCh)}

	listener, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		return "", "", fmt.Errorf("oauth: start callback listener: %w", err)
	}

	go serveCallback(server, listener, resultCh)
	defer shutdownCallback(server)

	if err := openBrowser(authURL); err != nil {
		return "", "", fmt.Errorf("oauth: open authorization URL: %w", err)
	}

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case result := <-resultCh:
		return result.code, result.state, result.err
	}
}

func serveCallback(server *http.Server, listener net.Listener, resultCh chan<- callbackResult) {
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		sendCallbackResult(resultCh, callbackResult{err: err})
	}
}

func shutdownCallback(server *http.Server) {
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

		sendCallbackResult(resultCh, callbackResult{code: code, state: query.Get("state")})

		_, _ = fmt.Fprintln(w, "Authorization complete. You can close this window.")
	})
}

func sendCallbackResult(resultCh chan<- callbackResult, result callbackResult) {
	select {
	case resultCh <- result:
	default:
	}
}

func openBrowser(authURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", authURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL)
	default:
		cmd = exec.Command("xdg-open", authURL)
	}

	return cmd.Start()
}

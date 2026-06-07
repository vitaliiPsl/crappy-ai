package background

import "context"

type sessionKey struct{}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if sessionID == "" {
		return ctx
	}

	return context.WithValue(ctx, sessionKey{}, sessionID)
}

func SessionID(ctx context.Context) string {
	sessionID, _ := ctx.Value(sessionKey{}).(string)

	return sessionID
}

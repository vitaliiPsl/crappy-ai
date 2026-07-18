package oauth

import "context"

type Callback interface {
	Wait(ctx context.Context, authURL string, redirectURL string) (code string, state string, err error)
}

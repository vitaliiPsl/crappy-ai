package oauth

import "errors"

var (
	ErrAuthRequired = errors.New("provider oauth: authentication required")
	ErrInvalidGrant = errors.New("provider oauth: invalid grant")
)

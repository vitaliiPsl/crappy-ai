package oauth

import "time"

type Credential struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type Authorization struct {
	BearerToken string
	Headers     map[string]string
}

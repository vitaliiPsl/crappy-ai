package oauth

import (
	"context"
	"time"

	"golang.org/x/oauth2"
)

type SessionStore interface {
	Load(ctx context.Context, key SessionKey) (*Session, error)
	Save(ctx context.Context, key SessionKey, session Session) error
	Delete(ctx context.Context, key SessionKey) error
}

type SessionKey struct {
	ServerName string
	ServerURL  string
}

func NewSessionKey(serverName, serverURL string) SessionKey {
	return SessionKey{
		ServerName: serverName,
		ServerURL:  serverURL,
	}
}

func (k SessionKey) ID() string {
	return k.ServerName + "|" + k.ServerURL
}

type Session struct {
	ServerURL string `json:"server_url"`
	Token     Token  `json:"token"`
}

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
}

func sessionFromToken(serverURL string, token *oauth2.Token) Session {
	session := Session{
		ServerURL: serverURL,
		Token: Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    token.TokenType,
			ExpiresAt:    token.Expiry,
		},
	}

	if scope, ok := token.Extra("scope").(string); ok {
		session.Token.Scope = scope
	}

	return session
}

func (s Session) oauthToken() *oauth2.Token {
	token := &oauth2.Token{
		AccessToken:  s.Token.AccessToken,
		RefreshToken: s.Token.RefreshToken,
		TokenType:    s.Token.TokenType,
		Expiry:       s.Token.ExpiresAt,
	}

	if s.Token.Scope != "" {
		token = token.WithExtra(map[string]any{
			"scope": s.Token.Scope,
		})
	}

	return token
}

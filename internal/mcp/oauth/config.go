package oauth

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
)

const (
	defaultCallbackHost = "127.0.0.1"
	defaultCallbackPort = 14545
	defaultCallbackPath = "/oauth/callback"
)

type Config struct {
	Enabled *bool `yaml:"enabled,omitempty"`

	ClientID        string `yaml:"client_id,omitempty"`
	ClientSecret    string `yaml:"client_secret,omitempty"`
	ClientSecretEnv string `yaml:"client_secret_env,omitempty"`

	Scopes []string `yaml:"scopes,omitempty"`

	RedirectURL  string `yaml:"redirect_url,omitempty"`
	CallbackHost string `yaml:"callback_host,omitempty"`
	CallbackPort int    `yaml:"callback_port,omitempty"`
}

func (c Config) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}

	return *c.Enabled
}

func (c Config) ResolveClientSecret() string {
	if c.ClientSecretEnv != "" {
		return os.Getenv(c.ClientSecretEnv)
	}

	return c.ClientSecret
}

func RedirectURL(cfg Config) (string, error) {
	if cfg.RedirectURL != "" {
		return validateRedirectURL(cfg.RedirectURL)
	}

	host := cfg.CallbackHost
	if host == "" {
		host = defaultCallbackHost
	}

	port := cfg.CallbackPort
	if port == 0 {
		port = defaultCallbackPort
	}

	return validateRedirectURL(fmt.Sprintf("http://%s:%d%s", host, port, defaultCallbackPath))
}

func validateRedirectURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	if u.Scheme != "http" {
		return "", errors.New("redirect_url must use http for the local callback")
	}

	if u.Hostname() == "" {
		return "", errors.New("redirect_url must include a host")
	}

	if u.Port() == "" || u.Port() == "0" {
		return "", errors.New("redirect_url must include a non-zero port")
	}

	if !isLoopbackHost(u.Hostname()) {
		return "", errors.New("redirect_url host must be localhost or a loopback address")
	}

	if u.Path == "" {
		u.Path = defaultCallbackPath
	}

	return u.String(), nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

package models

import (
	"context"
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-adk/kit"
	adkproviders "github.com/vitaliiPsl/crappy-adk/providers"

	appProviders "github.com/vitaliiPsl/crappy-ai/internal/providers"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

type Registry struct {
	settingsStore *settings.Store
	providers     *appProviders.Manager
}

func NewRegistry(settingsStore *settings.Store, providers *appProviders.Manager) *Registry {
	return &Registry{
		settingsStore: settingsStore,
		providers:     providers,
	}
}

func (r *Registry) GetProviders() []settings.ProviderSettings {
	return r.settingsStore.Get().Providers
}

func (r *Registry) GetProvider(id string) (settings.ProviderSettings, error) {
	p, ok := findProvider(r.settingsStore.Get().Providers, id)
	if !ok {
		return settings.ProviderSettings{}, fmt.Errorf("unknown provider %q", id)
	}

	return p, nil
}

func (r *Registry) Build(ctx context.Context, provider, model string) (kit.Model, error) {
	return r.buildModel(ctx, r.settingsStore.Get(), provider, model)
}

func (r *Registry) Authenticate(ctx context.Context, providerID string) error {
	if r.providers == nil {
		return fmt.Errorf("provider oauth is not configured")
	}

	provider, err := r.GetProvider(providerID)
	if err != nil {
		return err
	}

	if provider.Auth.Type != settings.ProviderAuthOAuth {
		return fmt.Errorf("provider %q does not use oauth", provider.ID)
	}

	_, err = r.providers.Authenticate(ctx, provider.ID, provider.Auth.Driver, provider.Auth.OAuth)

	return err
}

func (r *Registry) Logout(ctx context.Context, providerID string) error {
	if r.providers == nil {
		return fmt.Errorf("provider oauth is not configured")
	}

	provider, err := r.GetProvider(providerID)
	if err != nil {
		return err
	}

	if provider.Auth.Type != settings.ProviderAuthOAuth {
		return fmt.Errorf("provider %q does not use oauth", provider.ID)
	}

	return r.providers.Logout(ctx, provider.ID, provider.Auth.Driver)
}

func (r *Registry) OAuthStatus(ctx context.Context, providerID string) (provideroauth.Snapshot, error) {
	if r.providers == nil {
		return provideroauth.Snapshot{}, fmt.Errorf("provider oauth is not configured")
	}

	provider, err := r.GetProvider(providerID)
	if err != nil {
		return provideroauth.Snapshot{}, err
	}

	if provider.Auth.Type != settings.ProviderAuthOAuth {
		return provideroauth.Snapshot{}, fmt.Errorf("provider %q does not use oauth", provider.ID)
	}

	return r.providers.Status(ctx, provider.ID, provider.Auth.Driver)
}

func (r *Registry) Limits(ctx context.Context, providerID string) (provideroauth.Limits, error) {
	if r.providers == nil {
		return provideroauth.Limits{}, fmt.Errorf("provider limits are not configured")
	}

	provider, err := r.GetProvider(providerID)
	if err != nil {
		return provideroauth.Limits{}, err
	}

	if provider.Auth.Type != settings.ProviderAuthOAuth {
		return provideroauth.Limits{}, fmt.Errorf("provider %q does not use oauth", provider.ID)
	}

	return r.providers.Limits(
		ctx,
		provider.ID,
		provider.Auth.Driver,
		provider.Auth.OAuth,
	)
}

func (r *Registry) OAuthDrivers(providerID string) []string {
	_, ok := findProvider(r.settingsStore.Get().Providers, providerID)
	if !ok || r.providers == nil {
		return nil
	}

	return r.providers.OAuthDrivers()
}

func (r *Registry) buildModel(
	ctx context.Context,
	s settings.Settings,
	providerName,
	modelID string,
) (kit.Model, error) {
	if providerName == "" {
		return nil, fmt.Errorf("config: provider is not set")
	}

	if modelID == "" {
		return nil, fmt.Errorf("config: model is not set")
	}

	provider, ok := findProvider(s.Providers, providerName)
	if !ok {
		return nil, fmt.Errorf("settings: unknown provider %q", providerName)
	}

	adapter, ok := apiAdapters[provider.API]
	if !ok {
		return nil, fmt.Errorf("provider %q: unknown api %q", provider.ID, provider.API)
	}

	modelConfig := findModel(s.Models[providerName], modelID)
	opts := []adkproviders.ModelOption{
		adkproviders.WithModelConfig(modelConfig),
	}

	if provider.BaseURL != "" {
		opts = append(opts, adkproviders.WithBaseURL(provider.BaseURL))
	}

	authOpts, err := r.authOptions(ctx, provider)
	if err != nil {
		return nil, err
	}

	return adapter(modelID, append(opts, authOpts...)...)
}

func (r *Registry) authOptions(
	ctx context.Context,
	provider settings.ProviderSettings,
) ([]adkproviders.ModelOption, error) {
	switch provider.Auth.Type {
	case settings.ProviderAuthAPIKey:
		return r.apiKeyAuthOptions(provider)

	case settings.ProviderAuthOAuth:
		return r.oauthAuthOptions(ctx, provider)

	default:
		return nil, fmt.Errorf("provider %q: unsupported auth type %q", provider.ID, provider.Auth.Type)
	}
}

func (r *Registry) apiKeyAuthOptions(provider settings.ProviderSettings) ([]adkproviders.ModelOption, error) {
	apiKey := provider.Auth.APIKey
	if apiKey == "" && provider.Auth.APIKeyEnv != "" {
		apiKey = os.Getenv(provider.Auth.APIKeyEnv)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("provider %q: no API key configured", provider.ID)
	}

	return []adkproviders.ModelOption{adkproviders.WithAPIKey(apiKey)}, nil
}

func (r *Registry) oauthAuthOptions(
	ctx context.Context,
	provider settings.ProviderSettings,
) ([]adkproviders.ModelOption, error) {
	if r.providers == nil {
		return nil, fmt.Errorf("provider %q: oauth is not supported", provider.ID)
	}

	if provider.Auth.APIKey != "" || provider.Auth.APIKeyEnv != "" {
		return nil, fmt.Errorf("provider %q: API key settings cannot be used with oauth", provider.ID)
	}

	if provider.Auth.Driver == "" {
		return nil, fmt.Errorf("provider %q: oauth driver is not configured", provider.ID)
	}

	auth, err := r.providers.Resolve(ctx, provider.ID, provider.Auth.Driver, provider.Auth.OAuth)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", provider.ID, err)
	}

	return []adkproviders.ModelOption{
		adkproviders.WithBearerToken(auth.BearerToken),
		adkproviders.WithHeaders(auth.Headers),
	}, nil
}

func findProvider(providers []settings.ProviderSettings, id string) (settings.ProviderSettings, bool) {
	for _, p := range providers {
		if p.ID == id {
			return p, true
		}
	}

	return settings.ProviderSettings{}, false
}

func findModel(models []kit.ModelConfig, modelID string) kit.ModelConfig {
	for _, model := range models {
		if model.ID == modelID {
			return model
		}
	}

	return kit.ModelConfig{ID: modelID}
}

package providers

import (
	"context"
	"slices"

	adkproviders "github.com/vitaliiPsl/crappy-adk/providers"

	appoauth "github.com/vitaliiPsl/crappy-ai/internal/oauth"
	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

type OAuthDriver interface {
	provideroauth.Provider

	ID() string
	ModelOptions(auth provideroauth.Authorization) []adkproviders.ModelOption
}

type Manager struct {
	oauth   *provideroauth.Manager
	drivers map[string]OAuthDriver
}

func NewManager(store provideroauth.Store, callback appoauth.Callback, registered ...OAuthDriver) *Manager {
	drivers := make(map[string]OAuthDriver, len(registered))

	oauthProviders := make(map[string]provideroauth.Provider, len(registered))
	for _, driver := range registered {
		drivers[driver.ID()] = driver
		oauthProviders[driver.ID()] = driver
	}

	return &Manager{
		oauth:   provideroauth.NewManager(store, callback, oauthProviders),
		drivers: drivers,
	}
}

func (m *Manager) Authenticate(ctx context.Context, providerID, driverID string) (provideroauth.Authorization, error) {
	return m.oauth.Authenticate(ctx, providerID, driverID)
}

func (m *Manager) Resolve(ctx context.Context, providerID, driverID string) (provideroauth.Authorization, error) {
	return m.oauth.Resolve(ctx, providerID, driverID)
}

func (m *Manager) ModelOptions(driverID string, auth provideroauth.Authorization) []adkproviders.ModelOption {
	driver, ok := m.drivers[driverID]
	if !ok {
		return nil
	}

	return driver.ModelOptions(auth)
}

func (m *Manager) Logout(ctx context.Context, providerID, driverID string) error {
	return m.oauth.Logout(ctx, providerID, driverID)
}

func (m *Manager) Status(ctx context.Context, providerID, driverID string) (provideroauth.Snapshot, error) {
	return m.oauth.Status(ctx, providerID, driverID)
}

func (m *Manager) SupportsOAuth() bool {
	return len(m.drivers) > 0
}

func (m *Manager) OAuthDrivers() []string {
	drivers := make([]string, 0, len(m.drivers))
	for id := range m.drivers {
		drivers = append(drivers, id)
	}

	slices.Sort(drivers)

	return drivers
}

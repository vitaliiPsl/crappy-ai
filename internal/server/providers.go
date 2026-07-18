package server

import (
	"context"
	"fmt"

	provideroauth "github.com/vitaliiPsl/crappy-ai/internal/providers/oauth"
)

func (s *Server) AuthenticateProvider(ctx context.Context, providerID string) error {
	if s.modelsRegistry == nil {
		return fmt.Errorf("provider oauth is not configured")
	}

	return s.modelsRegistry.Authenticate(ctx, providerID)
}

func (s *Server) LogoutProvider(ctx context.Context, providerID string) error {
	if s.modelsRegistry == nil {
		return fmt.Errorf("provider oauth is not configured")
	}

	return s.modelsRegistry.Logout(ctx, providerID)
}

func (s *Server) ProviderOAuthStatus(ctx context.Context, providerID string) (provideroauth.Snapshot, error) {
	if s.modelsRegistry == nil {
		return provideroauth.Snapshot{}, fmt.Errorf("provider oauth is not configured")
	}

	return s.modelsRegistry.OAuthStatus(ctx, providerID)
}

func (s *Server) ProviderSupportsOAuth(providerID string) bool {
	return s.modelsRegistry != nil && s.modelsRegistry.SupportsOAuth(providerID)
}

func (s *Server) ProviderOAuthDrivers(providerID string) []string {
	if s.modelsRegistry == nil {
		return nil
	}

	return s.modelsRegistry.OAuthDrivers(providerID)
}

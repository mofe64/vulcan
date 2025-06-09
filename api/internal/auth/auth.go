package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/mofe64/vulkan/api/internal/config"
	"golang.org/x/oauth2"
)

type VulkanAuth struct {
	Provider  *oidc.Provider
	OAuth2Cfg *oauth2.Config
	Verifier  *oidc.IDTokenVerifier
}

func BuildVulkanAuth(ctx context.Context, cfg *config.VulkanConfig) (*VulkanAuth, error) {
	// "https://dex.vulkan.strawhatengineer.com/dex"
	provider, err := oidc.NewProvider(ctx, cfg.DEX_URL)
	if err != nil {
		return nil, err
	}
	oAuth2Cfg := &oauth2.Config{
		ClientID:     cfg.OIDC_CLIENT_ID,
		ClientSecret: cfg.OIDC_CLIENT_SECRET,
		Endpoint:     provider.Endpoint(), // pulls /auth & /token URLs from Dex discovery
		RedirectURL:  "https://api.vulkan.strawhatengineer.com/api/auth/callback",
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile", "offline_access"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDC_CLIENT_ID,
	})
	return &VulkanAuth{
		Provider:  provider,
		OAuth2Cfg: oAuth2Cfg,
		Verifier:  verifier,
	}, nil
}

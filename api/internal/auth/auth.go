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
	// initialize the provider using the PUBLIC issuer URL.
	// this is used for discovery and to get the auth/token endpoints.
	// this MUST match the 'issuer' in Dex's config.
	provider, err := oidc.NewProvider(ctx, cfg.OIDC_ISSUER)
	if err != nil {
		return nil, err
	}

	// configure the OAuth2 client, using the endpoints discovered from the public provider URL.
	oAuth2Cfg := &oauth2.Config{
		ClientID:     cfg.OIDC_CLIENT_ID,
		ClientSecret: cfg.OIDC_CLIENT_SECRET,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "https://api.vulkan.strawhatengineer.com/api/auth/callback",
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile", "offline_access"},
	}

	// Build the token verifier manually for high performance.
	// Create a key set that explicitly uses the INTERNAL JWKS URL for fast key fetching.
	keySet := oidc.NewRemoteKeySet(ctx, cfg.OIDCJWKSURL)

	// Create a verifier that trusts the PUBLIC issuer URL but uses the
	// key set we just configured with the INTERNAL URL.
	verifier := oidc.NewVerifier(cfg.OIDC_ISSUER, keySet, &oidc.Config{
		ClientID: cfg.OIDC_CLIENT_ID,
	})

	return &VulkanAuth{
		Provider:  provider,
		OAuth2Cfg: oAuth2Cfg,
		Verifier:  verifier,
	}, nil
}

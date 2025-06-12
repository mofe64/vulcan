package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mofe64/vulkan/api/internal/auth"
	"github.com/mofe64/vulkan/api/internal/db/repository"
	"github.com/mofe64/vulkan/api/internal/dto"
	"golang.org/x/oauth2"
)

type AuthService interface {
	ExchangeCodeForToken(ctx context.Context, code, codeVerifier string) (dto.TokenDetails, error)
	RefreshToken(ctx context.Context, refreshToken string) (dto.TokenDetails, error)
}
type authService struct {
	auth              *auth.VulkanAuth
	tokenRepo         repository.TokenRepository
	userRepo          repository.UserRepository
	OnboardingService OnboardingService
}

func NewAuthService(auth *auth.VulkanAuth, tr repository.TokenRepository, ur repository.UserRepository) AuthService {
	return &authService{
		auth:      auth,
		tokenRepo: tr,
		userRepo:  ur,
	}
}

func (s *authService) ExchangeCodeForToken(ctx context.Context, code, codeVerifier string) (dto.TokenDetails, error) {

	// exchange the code for token
	token, err := s.auth.OAuth2Cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return dto.TokenDetails{}, errors.New("token exchange failed: " + err.Error())
	}

	// pull & verify the ID token (access token is opaque, we only care about ID token)
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return dto.TokenDetails{}, errors.New("no id_token in response")
	}
	idToken, err := s.auth.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return dto.TokenDetails{}, errors.New("invalid id_token")
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return dto.TokenDetails{}, errors.New("failed to parse id_token claims: " + err.Error())
	}

	// Check if user exists in DB
	// if user does not exist, trigger onboarding process

	existingUserId, err := s.userRepo.GetUserIdByOIDCSub(ctx, claims.Sub)
	var userID uuid.UUID
	if err != nil {
		return dto.TokenDetails{}, errors.New("user lookup failed: " + err.Error())
	}
	// if user does not exist, trigger onboarding process
	if existingUserId == uuid.Nil {
		userID, err = s.OnboardingService.Onboard(ctx, claims.Sub, claims.Email)
		if err != nil {
			return dto.TokenDetails{}, errors.New("onboarding failed: " + err.Error())
		}
	} else {
		userID = existingUserId
	}

	if err := s.tokenRepo.StoreRefreshToken(ctx, userID, token.RefreshToken, time.Now().Add(30*24*time.Hour)); err != nil {
		return dto.TokenDetails{}, errors.New("refresh token store failed: " + err.Error())
	}
	// 8. Return token details
	return dto.TokenDetails{
		AccessToken:  rawIDToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    int(time.Until(idToken.Expiry).Seconds()),
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (dto.TokenDetails, error) {
	userID, err := s.tokenRepo.CheckRefreshToken(ctx, refreshToken) // validates & checks expiry
	if err != nil {
		return dto.TokenDetails{}, errors.New("invalid or expired refresh token")
	}

	newToken, err := s.auth.OAuth2Cfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	}).Token()
	if err != nil {
		return dto.TokenDetails{}, errors.New("refresh token exchange failed: " + err.Error())
	}

	rawIDToken, _ := newToken.Extra("id_token").(string)
	idToken, _ := s.auth.Verifier.Verify(ctx, rawIDToken)

	// Rotate refresh tokens (recommended)
	if newRT := newToken.RefreshToken; newRT != "" && newRT != refreshToken {
		s.tokenRepo.RotateRefreshToken(ctx, userID, refreshToken, newRT, time.Now().Add(30*24*time.Hour))
	}

	return dto.TokenDetails{
		AccessToken:  rawIDToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresIn:    int(time.Until(idToken.Expiry).Seconds()),
	}, nil
}

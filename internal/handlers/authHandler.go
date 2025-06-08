package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mofe64/vulkan/internal/auth"
	"github.com/mofe64/vulkan/internal/db/repository"
	"golang.org/x/oauth2"
)

type AuthHandler interface {
	ExchangeCodeForToken() gin.HandlerFunc
	RefreshToken() gin.HandlerFunc
}

type authHandler struct {
	auth *auth.VulkanAuth
	tr   *repository.TokenRepository
	ur   *repository.UserRepository
}

func NewAuthHandler(auth *auth.VulkanAuth, tr *repository.TokenRepository, ur *repository.UserRepository) AuthHandler {
	return &authHandler{
		auth: auth,
		tr:   tr,
		ur:   ur,
	}
}

func (h *authHandler) ExchangeCodeForToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Code         string `json:"code" binding:"required"`
			CodeVerifier string `json:"code_verifier" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		ctx := c.Request.Context()
		// 4. Exchange the code for tokens
		token, err := h.auth.OAuth2Cfg.Exchange(ctx, body.Code,
			oauth2.SetAuthURLParam("code_verifier", body.CodeVerifier))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token exchange failed"})
			return
		}

		// 5. Pull & verify the ID token (access token is opaque, we only care about ID token)
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no id_token in response"})
			return
		}
		idToken, err := h.auth.Verifier.Verify(ctx, rawIDToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid id_token"})
			return
		}

		// 6. Read Dex claims
		var claims struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
		}
		if err := idToken.Claims(&claims); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "claims parse failed"})
			return
		}

		// 7. Check if user exists in DB
		// if user does not exist, create new user,
		// create default org, and default project

		existingUserId, err := h.ur.GetUserByOIDCSub(ctx, claims.Sub)
		var userID uuid.UUID
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
			return
		}
		// if user does not exist, create new user
		if existingUserId == uuid.Nil {
			userID, err = h.ur.InsertNewUser(ctx, claims.Sub, claims.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user insert failed"})
				return
			}
			// create default org

			// create default project
		} else {
			// upsert user (creates if not exists, updates email if exists and email is different)
			userID, err = h.ur.UpsertUser(ctx, claims.Sub, claims.Email) // returns UUID
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user upsert failed"})
				return
			}
		}

		if err := h.tr.StoreRefreshToken(ctx, userID, token.RefreshToken, time.Now().Add(30*24*time.Hour)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh token store failed"})
			return
		}

		// 8. Send response (access token in JSON, refresh token as HttpOnly cookie)
		c.SetCookie("refresh_token", token.RefreshToken, 30*24*3600, "/", ".strawhatengineer.com", true, true)
		c.JSON(http.StatusOK, gin.H{
			"access_token": rawIDToken,
			"expires_in":   int(time.Until(idToken.Expiry).Seconds()),
		})
	}
}
func (h *authHandler) RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no refresh token"})
			return
		}
		ctx := c.Request.Context()
		userID, err := h.tr.CheckRefreshToken(ctx, refreshToken) // validates & checks expiry
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}

		newToken, err := h.auth.OAuth2Cfg.TokenSource(ctx, &oauth2.Token{
			RefreshToken: refreshToken,
		}).Token()
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token exchange failed"})
			return
		}

		rawIDToken, _ := newToken.Extra("id_token").(string)
		idToken, _ := h.auth.Verifier.Verify(ctx, rawIDToken)

		// Rotate refresh tokens (recommended)
		if newRT := newToken.RefreshToken; newRT != "" && newRT != refreshToken {
			h.tr.RotateRefreshToken(ctx, userID, refreshToken, newRT, time.Now().Add(30*24*time.Hour))
			c.SetCookie("refresh_token", newRT, 30*24*3600, "/", ".strawhatengineer.com", true, true)
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": rawIDToken,
			"expires_in":   int(time.Until(idToken.Expiry).Seconds()),
		})
	}
}

func RefreshHandler(c *gin.Context) {

}

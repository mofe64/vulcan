package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mofe64/vulkan/api/internal/auth"
	"github.com/mofe64/vulkan/api/internal/dto"
	"github.com/mofe64/vulkan/api/internal/service"
)

type AuthHandler interface {
	ExchangeCodeForToken() gin.HandlerFunc
	RefreshToken() gin.HandlerFunc
}

type authHandler struct {
	auth        *auth.VulkanAuth
	authService service.AuthService
}

func NewAuthHandler(auth *auth.VulkanAuth, authService service.AuthService) AuthHandler {
	return &authHandler{
		auth:        auth,
		authService: authService,
	}
}

func (h *authHandler) ExchangeCodeForToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body dto.CodeExchangeRequest
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		ctx := c.Request.Context()
		token, err := h.authService.ExchangeCodeForToken(ctx, body.Code, body.CodeVerifier)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "token exchange failed",
				"details": err.Error(),
			})
			return
		}

		// send response (access token in JSON, refresh token as HttpOnly cookie)
		c.SetCookie("refresh_token", token.RefreshToken, 30*24*3600, "/", ".strawhatengineer.com", true, true)
		c.JSON(http.StatusOK, gin.H{
			"access_token": token.AccessToken,
			"expires_in":   token.ExpiresIn,
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

		token, err := h.authService.RefreshToken(ctx, refreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "refresh token failed",
				"details": err.Error(),
			})
			return
		}

		c.SetCookie("refresh_token", token.RefreshToken, 30*24*3600, "/", ".strawhatengineer.com", true, true)
		c.JSON(http.StatusOK, gin.H{
			"access_token": token.AccessToken,
			"expires_in":   token.ExpiresIn,
		})
	}
}

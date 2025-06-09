package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mofe64/vulkan/api/internal/handlers"
)

func RegisterAuthRoutes(router *gin.Engine, authHandler handlers.AuthHandler) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/exchange", authHandler.ExchangeCodeForToken())
		authGroup.POST("/refresh", authHandler.RefreshToken())
	}
}

package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/internal/api/handlers"
	"github.com/jafarshop/b2bapi/internal/api/middleware"
)

// NewRouter creates and configures the Gin router
func NewRouter(cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(loggingMiddleware(logger))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Partner routes (require authentication)
		partnerRoutes := v1.Group("")
		partnerRoutes.Use(middleware.AuthMiddleware(repos, logger))
		partnerRoutes.Use(middleware.IdempotencyMiddleware(repos, logger))
		{
			partnerRoutes.POST("/carts/submit", handlers.HandleCartSubmit(cfg, repos, logger))
			partnerRoutes.GET("/orders/:id", handlers.HandleGetOrder(repos, logger))
		}

		// Admin routes (internal - for now using same auth, can be separated later)
		adminRoutes := v1.Group("/admin")
		adminRoutes.Use(middleware.AuthMiddleware(repos, logger))
		{
			adminRoutes.POST("/orders/:id/confirm", handlers.HandleConfirmOrder(repos, logger))
			adminRoutes.POST("/orders/:id/reject", handlers.HandleRejectOrder(repos, logger))
			adminRoutes.POST("/orders/:id/ship", handlers.HandleShipOrder(repos, logger))
			adminRoutes.GET("/orders", handlers.HandleListOrders(repos, logger))
		}
	}

	return router
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		status := c.Writer.Status()
		logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
		)
	}
}

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mofe64/vulkan/api/internal/auth"
	"github.com/mofe64/vulkan/api/internal/config"
	"github.com/mofe64/vulkan/api/internal/db"
	"github.com/mofe64/vulkan/api/internal/db/migrations"
	"github.com/mofe64/vulkan/api/internal/db/repository"
	"github.com/mofe64/vulkan/api/internal/events"
	"github.com/mofe64/vulkan/api/internal/handlers"
	"github.com/mofe64/vulkan/api/internal/k8s"
	"github.com/mofe64/vulkan/api/internal/logger"
	"github.com/mofe64/vulkan/api/internal/middleware"
	"github.com/mofe64/vulkan/api/internal/routes"
	"github.com/mofe64/vulkan/api/internal/service"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

type DependencyContainer struct {
	Database  *pgxpool.Pool
	Auth      *auth.VulkanAuth
	K8sClient client.Client
	EventBus  *events.EventBus
	log       *zap.Logger
	cfg       *config.VulkanConfig
}

func initializeDependencies() *DependencyContainer {

	log := logger.Get()
	// short-lived context for start-up ops
	startupCtx, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()

	// load config

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// connect to db
	database, err := db.Connect(startupCtx, cfg.DBURL)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}

	// run migrations
	if err := migrations.RunMigrations(startupCtx, cfg.DBURL, log); err != nil {
		log.Fatal("DB migration failed", zap.Error(err))
	}

	// build oidc (dex) auth
	auth, err := auth.BuildVulkanAuth(startupCtx, cfg)
	if err != nil {
		log.Fatal("Failed to build Vulkan auth", zap.Error(err))
	}

	//create k8s client
	k8sClient, err := k8s.New(startupCtx, cfg.InCluster)
	if err != nil {
		log.Fatal("Failed to create Kubernetes client", zap.Error(err))
	}
	// create event bus
	bus, err := events.NewEventBus(cfg.NATS_URL)
	if err != nil {
		log.Fatal("Failed to create event bus", zap.Error(err))
	}

	return &DependencyContainer{
		Database:  database,
		Auth:      auth,
		K8sClient: k8sClient,
		EventBus:  bus,
		log:       log,
		cfg:       cfg,
	}

}

func main() {
	r := gin.Default()
	// r.Use(cors.Default())
	r.Use(gin.Recovery())

	applicationDependencies := initializeDependencies()
	database := applicationDependencies.Database
	auth := applicationDependencies.Auth
	k8sClient := applicationDependencies.K8sClient
	bus := applicationDependencies.EventBus
	log := applicationDependencies.log
	cfg := applicationDependencies.cfg

	userRepository := repository.NewUserRepo(database)
	tokenRepository := repository.NewTokenRepo(database)

	// not using org service yet, but initializing it for future use
	_ = service.NewOrgService(database, k8sClient, log, bus)

	authService := service.NewAuthService(auth, tokenRepository, userRepository)
	authHandler := handlers.NewAuthHandler(auth, authService)

	// Set up ping route
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	routes.RegisterAuthRoutes(r, authHandler)

	// jwt middleware
	r.Use(middleware.RequireAuth(auth))
	// opa middleware
	r.Use(middleware.NewOPAAuth(*cfg))

	vulkanServerPort := cfg.VulkanServerPort
	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:" + vulkanServerPort,
	}

	go func() {
		log.Info("Starting vulkan server on port " + vulkanServerPort)
		if err := s.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Error on server listen", zap.Error(err))
		}
	}()

	// Create a channel to receive OS signals (e.g., interrupt or termination)
	c := make(chan os.Signal, 1)
	// Register OS signals to be captured by the channel
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	// Block until a signal is received.
	// Wait until a signal is received on the channel
	sig := <-c
	log.Info("Received signal:", zap.String("signal", sig.String()))

	// Create a new context with a 30-second timeout for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// Ensure the context is canceled when the main function exits
	defer cancel()
	// shut down event bus gracefully
	bus.Close()
	// Initiate a graceful shutdown of the server, allowing existing connections to complete
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", zap.Error(err))
	}
	log.Info("Server gracefully stopped")

}

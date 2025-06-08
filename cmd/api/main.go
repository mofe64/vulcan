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
	"github.com/mofe64/vulkan/internal/auth"
	"github.com/mofe64/vulkan/internal/config"
	"github.com/mofe64/vulkan/internal/db"
	"github.com/mofe64/vulkan/internal/db/migrations"
	"github.com/mofe64/vulkan/internal/db/repository"
	"github.com/mofe64/vulkan/internal/events"
	"github.com/mofe64/vulkan/internal/handlers"
	"github.com/mofe64/vulkan/internal/k8s"
	"github.com/mofe64/vulkan/internal/logger"
	"github.com/mofe64/vulkan/internal/middleware"
	"github.com/mofe64/vulkan/internal/server"
	"github.com/mofe64/vulkan/internal/service"

	"go.uber.org/zap"
)

func main() {
	log := logger.Get()

	// root context: cancelled on SIGINT/SIGTERM
	appCtx, appCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer appCancel() // just in case main exits without a signal

	// short-lived context for start-up I/O
	startupCtx, cancelTimeout := context.WithTimeout(appCtx, 10*time.Second)
	defer cancelTimeout()

	// load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// connect to db
	db, err := db.Connect(startupCtx, cfg.DBURL)
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

	orgSvc := service.NewOrgService(db, k8sClient, log, bus)
	userRepository := repository.NewUserRepo(db)
	tokenRepository := repository.NewTokenRepo(db)
	authHandler := handlers.NewAuthHandler(auth, tokenRepository, userRepository)

	// create the Vulkan server
	vulkanServer, err := server.NewVulkanServer(orgSvc, log)
	if err != nil {
		log.Fatal("Failed to create Vulkan server", zap.Error(err))
	}
	r := gin.Default()
	// r.Use(cors.Default())

	r.Use(gin.Recovery())

	r.POST("/api/auth/exchange", authHandler.ExchangeCodeForToken())
	r.POST("/api/auth/refresh", authHandler.RefreshToken())

	// jwt middleware
	r.Use(middleware.RequireAuth(auth))
	// opa middleware
	r.Use(middleware.NewOPAAuth(*cfg))

	server.RegisterHandlers(r, &vulkanServer)

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

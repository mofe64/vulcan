package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mofe64/vulkan/pkg/server"
)

func main() {
	//Todo: move config to dedicated config file
	vulkanServerPort := os.Getenv("VULKAN_SERVER_PORT")
	if vulkanServerPort == "" {
		vulkanServerPort = "8080"
	}
	vulkanServer := server.NewVulkanServer()
	r := gin.Default()

	r.Use(gin.Recovery())

	server.RegisterHandlers(r, &vulkanServer)

	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:" + vulkanServerPort,
	}

	go func() {
		log.Println("Starting vulkan server on port " + vulkanServerPort)
		if err := s.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("Error on server listen %v\n", err)
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
	log.Println("Received signal:", sig)

	// Create a new context with a 30-second timeout for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// Ensure the context is canceled when the main function exits
	defer cancel()

	// Initiate a graceful shutdown of the server, allowing existing connections to complete
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting ....")

}

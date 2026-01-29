package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"epub-reader/internal/web"
	"epub-reader/pkg/storage"
)

func main() {
	// Parse flags
	port := flag.Int("port", 8080, "HTTP server port")
	dbPath := flag.String("db", storage.DefaultDBPath(), "Database path")
	staticDir := flag.String("static", "static", "Static files directory")
	flag.Parse()

	// Initialize database
	store, err := storage.NewSQLiteStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	// Create server
	server, err := web.NewServer(store, *staticDir)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Try to find an available port
	var httpServer *http.Server
	var actualPort int
	maxAttempts := 10
	
	for attempt := 0; attempt < maxAttempts; attempt++ {
		tryPort := *port + attempt
		addr := fmt.Sprintf(":%d", tryPort)
		
		httpServer = &http.Server{
			Addr:         addr,
			Handler:      server,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		
		// Test if port is available
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			if attempt < maxAttempts-1 {
				log.Printf("Port %d is busy, trying %d...", tryPort, tryPort+1)
				continue
			}
			log.Fatalf("Could not find available port after %d attempts (tried %d-%d)", maxAttempts, *port, *port+maxAttempts-1)
		}
		
		// Port is available, close test listener
		listener.Close()
		actualPort = tryPort
		
		if attempt > 0 {
			log.Printf("Port %d was busy, using port %d instead", *port, actualPort)
		}
		break
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server at http://localhost:%d", actualPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

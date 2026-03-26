package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lightcms/cms/internal/config"
	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/handlers"
	"github.com/lightcms/cms/internal/services"
)

// Version and BuildTime are set at compile time via -ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// Open database.
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	// Run migrations.
	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Seed admin on first run.
	created, password, err := db.SeedAdmin(database)
	if err != nil {
		log.Fatalf("seed admin: %v", err)
	}
	if created {
		printFirstRunBanner(password)
	}

	// Ensure uploads directory exists.
	if err := os.MkdirAll(cfg.UploadPath, 0755); err != nil {
		log.Fatalf("create upload dir: %v", err)
	}

	// Start Bitrix24 worker pool (10 workers, queue size 100).
	bitrixPool := services.NewBitrixPool(database, 10, 100)

	// Build router.
	router := handlers.NewRouter(database, cfg, bitrixPool, Version, BuildTime)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown context.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in background.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("CMS %s (built %s) listening on http://0.0.0.0:%d", Version, BuildTime, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error.
	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case <-ctx.Done():
		log.Println("Shutting down server...")
	}

	// Graceful shutdown with 30-second timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	// Drain the Bitrix24 pool.
	log.Println("Draining Bitrix24 queue...")
	bitrixPool.Shutdown(15 * time.Second)

	log.Println("Shutdown complete.")
}

// printFirstRunBanner prints the initial admin credentials to stdout.
func printFirstRunBanner(password string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║          FIRST RUN — ADMIN CREDENTIALS           ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Printf( "║  Email:    %-38s ║\n", "admin@example.com")
	fmt.Printf( "║  Password: %-38s ║\n", password)
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║  Change these credentials in /admin/settings     ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
}

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

	"github.com/A7med-noureldin/wallet-ledger/internal/handler"
	"github.com/A7med-noureldin/wallet-ledger/internal/ledger"
	"github.com/A7med-noureldin/wallet-ledger/internal/storage"
)

func main() {
	db, err := storage.NewDatabase("ledger.db")
	if err != nil {
		log.Fatalf("Fatal: could not initialize database: %v\n", err)
	}
	defer db.Close()

	repo := storage.NewRepository(db)

	if err := repo.Migrate(context.Background()); err != nil {
		log.Fatalf("Fatal: could not run database migrations: %v\n", err)
	}

	service := ledger.NewService(repo)
	httpHandler := handler.New(service)

	mux := http.NewServeMux()
	httpHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Starting wallet-ledger API on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: HTTP server crashed: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server safely...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown with error: %v\n", err)
	}

	log.Println("Server exited properly. Goodbye!")
}

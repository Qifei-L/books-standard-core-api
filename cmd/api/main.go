package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Qifei-L/books-standard-core-api/internal/account"
	"github.com/Qifei-L/books-standard-core-api/internal/auth"
	"github.com/Qifei-L/books-standard-core-api/internal/bill"
	"github.com/Qifei-L/books-standard-core-api/internal/contact"
	"github.com/Qifei-L/books-standard-core-api/internal/invoice"
	"github.com/Qifei-L/books-standard-core-api/internal/journal"
	platformdb "github.com/Qifei-L/books-standard-core-api/internal/platform/db"
	"github.com/Qifei-L/books-standard-core-api/internal/report"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()
	pool, err := platformdb.Connect(ctx)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	authSvc := auth.NewService(pool)

	corsOrigins := []string{"http://localhost:4200", "http://localhost:5173"}
	if o := os.Getenv("CORS_ORIGINS"); o != "" {
		corsOrigins = []string{o}
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(auth.SecurityHeaders)
	r.Use(auth.CORSMiddleware(corsOrigins))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	authHandler := auth.NewHandler(authSvc)
	authHandler.RegisterRoutes(r)

	r.Group(func(r chi.Router) {
		r.Use(authSvc.Authenticate)

		authHandler.RegisterProtected(r)
		contact.NewHandler(contact.NewService(pool)).RegisterRoutes(r)
		account.NewHandler(account.NewService(pool)).RegisterRoutes(r)
		invoice.NewHandler(invoice.NewService(pool)).RegisterRoutes(r)
		bill.NewHandler(bill.NewService(pool)).RegisterRoutes(r)
		journal.NewHandler(journal.NewService(pool)).RegisterRoutes(r)
		report.NewHandler(report.NewService(pool)).RegisterRoutes(r)
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

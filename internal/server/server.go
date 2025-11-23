package server

import (
	"fmt"
	"net/http"
	"time"

	"pizza-must/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	*http.Server
	config *config.Config
	logger *zap.Logger
}

func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	// Create router
	router := chi.NewRouter()

	// Add basic middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &Server{
		Server: &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
			Handler:      router,
			IdleTimeout:  time.Minute,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		config: cfg,
		logger: logger,
	}

	return server
}

func (s *Server) Close() error {
	s.logger.Info("Closing server resources")
	s.logger.Sync()
	return nil
}

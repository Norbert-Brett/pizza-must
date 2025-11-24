package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"pizza-must/internal/config"
	custommiddleware "pizza-must/internal/middleware"
	"pizza-must/internal/repository"
	"pizza-must/internal/service"
	"pizza-must/internal/transport"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	*http.Server
	config *config.Config
	logger *zap.Logger
	db     *sql.DB
}

func NewServer(cfg *config.Config, logger *zap.Logger, db *sql.DB) *Server {
	// Create router
	router := chi.NewRouter()

	// Add basic middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(custommiddleware.ErrorHandlingMiddleware(logger))

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo, refreshTokenRepo, cfg.JWT.Secret)

	// Initialize handlers
	userHandler := transport.NewUserHandler(userService, logger)

	// Create auth middleware
	authMiddleware := custommiddleware.AuthMiddleware(cfg.JWT.Secret, logger)

	// Register routes
	userHandler.RegisterRoutes(router, authMiddleware)

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
		db:     db,
	}

	return server
}

func (s *Server) Close() error {
	s.logger.Info("Closing server resources")

	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Failed to close database connection", zap.Error(err))
		}
	}

	s.logger.Sync()
	return nil
}

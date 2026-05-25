// Package server builds the HTTP layer: Gin router, middleware chain, route mounts.
package server

import (
	"database/sql"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"

	"github.com/datisekai/longthu.fun/backend/internal/auth"
	"github.com/datisekai/longthu.fun/backend/internal/bankaccounts"
	"github.com/datisekai/longthu.fun/backend/internal/config"
	"github.com/datisekai/longthu.fun/backend/internal/groups"
	"github.com/datisekai/longthu.fun/backend/internal/players"
	"github.com/datisekai/longthu.fun/backend/internal/paymentintents"
	"github.com/datisekai/longthu.fun/backend/internal/public"
	"github.com/datisekai/longthu.fun/backend/internal/ratelimit"
	"github.com/datisekai/longthu.fun/backend/internal/sessions"
)

// Server bundles the wired-up router + its dependencies.
type Server struct {
	cfg    *config.Config
	db     *sql.DB
	router *gin.Engine
}

// New builds the Server. Routes are mounted; the router is ready to .Run().
func New(cfg *config.Config, db *sql.DB, gitSHA string) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestid.New())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AppBaseURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Idempotency-Key"},
		AllowCredentials: true, // required for the lt_session cookie on cross-port localhost dev
	}))

	// Public — no auth required.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": gitSHA,
		})
	})

	// Auth — Story 1.5. The /me endpoint sits inside the same group but is
	// gated by SessionMiddleware so an unauthenticated GET cleanly returns 401.
	authSvc := auth.NewService(db)
	authHandler := auth.NewHandler(authSvc, []byte(cfg.JWTSecret), cfg.AppBaseURL)

	v1Public := r.Group("/api/v1")
	v1Public.Use(ratelimit.Middleware())
	authHandler.RegisterAuthRoutes(v1Public) // register / login / logout — no middleware

	v1Auth := r.Group("/api/v1")
	v1Auth.Use(auth.SessionMiddleware([]byte(cfg.JWTSecret)))
	authHandler.RegisterMeRoute(v1Auth) // /auth/me — needs middleware

	groupsSvc := groups.NewService(db)
	groupsHandler := groups.NewHandler(groupsSvc)
	groupsHandler.RegisterRoutes(v1Auth)

	bankAccountsSvc := bankaccounts.NewService(db)
	bankAccountsHandler := bankaccounts.NewHandler(bankAccountsSvc)
	bankAccountsHandler.RegisterRoutes(v1Auth)

	playersSvc := players.NewService(db)
	playersHandler := players.NewHandler(playersSvc)
	playersHandler.RegisterRoutes(v1Auth)

	sessionsSvc := sessions.NewService(db)
	sessionsHandler := sessions.NewHandler(sessionsSvc, cfg.AppBaseURL)
	sessionsHandler.RegisterRoutes(v1Auth)

	publicHandler := public.NewHandler(db)
	publicHandler.RegisterRoutes(v1Public)

	paymentIntentsSvc := paymentintents.NewService(db)
	paymentIntentsHandler := paymentintents.NewHandler(paymentIntentsSvc)
	paymentIntentsHandler.RegisterPublicRoutes(v1Public)

	return &Server{cfg: cfg, db: db, router: r}
}

// Router exposes the underlying *gin.Engine for tests + main.
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Run starts the HTTP server on cfg.Port. Blocks.
func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Port)
}

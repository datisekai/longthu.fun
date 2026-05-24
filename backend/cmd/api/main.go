package main

import (
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/datisekai/longthu.fun/backend/internal/config"
	"github.com/datisekai/longthu.fun/backend/internal/server"
)

// gitSHA is overridden at build time:
//
//	go build -ldflags "-X main.gitSHA=$(git rev-parse --short HEAD)" ./cmd/api
var gitSHA = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	db, err := openDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("db open", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		slog.Error("db ping", "error", err)
		os.Exit(1)
	}

	srv := server.New(cfg, db, gitSHA)
	slog.Info("server starting", "port", cfg.Port, "version", gitSHA)
	if err := srv.Run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// openDB opens a MySQL connection from a DATABASE_URL. The Makefile / dev
// scripts use the `mysql://` prefix golang-migrate requires; the Go SQL
// driver expects the bare DSN, so we strip the prefix here.
func openDB(rawDSN string) (*sql.DB, error) {
	dsn := strings.TrimPrefix(rawDSN, "mysql://")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

# Longthu.fun backend — make targets for the dev loop.
#
# DATABASE_URL defaults to the docker-compose mysql service for local dev.
# Override per-call: `make migrate-up DATABASE_URL=mysql://...`.

DATABASE_URL ?= mysql://longthu:longthu@tcp(localhost:3306)/longthu?parseTime=true&multiStatements=true

# golang-migrate uses a custom DSN format with a `mysql://` prefix that includes
# `tcp(...)` rather than the Go driver's typical `mysql://user:pass@host:port/db`.
MIGRATE = $(shell go env GOPATH)/bin/migrate
MIGRATE_PATH = ./migrations
MIGRATE_DB = "$(DATABASE_URL)"

SQLC = $(shell go env GOPATH)/bin/sqlc

.PHONY: help migrate-up migrate-down migrate-status migrate-create sqlc-generate build run test vet fmt

help:
	@echo "Targets:"
	@echo "  make migrate-up               Apply all pending migrations"
	@echo "  make migrate-down             Roll back the most recent migration"
	@echo "  make migrate-status            Show current migration version"
	@echo "  make migrate-create NAME=foo   Create a new pair of migration files"
	@echo "  make sqlc-generate            Regenerate sqlc Go files"
	@echo "  make build                    go build ./..."
	@echo "  make run                      go run ./cmd/api"
	@echo "  make test                     go test ./..."
	@echo "  make vet                      go vet ./..."
	@echo "  make fmt                      gofmt -w ."

migrate-up:
	$(MIGRATE) -path $(MIGRATE_PATH) -database $(MIGRATE_DB) up

migrate-down:
	$(MIGRATE) -path $(MIGRATE_PATH) -database $(MIGRATE_DB) down 1

migrate-status:
	$(MIGRATE) -path $(MIGRATE_PATH) -database $(MIGRATE_DB) version

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "usage: make migrate-create NAME=add_foo"; exit 1; fi
	$(MIGRATE) create -ext sql -dir $(MIGRATE_PATH) -seq $(NAME)

sqlc-generate:
	$(SQLC) generate

build:
	go build ./...

run:
	go run ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# Makefile for Veritas Registry

.PHONY: help build start stop restart logs clean dev-start dev-stop test

help:
	@echo "Veritas Registry - Make commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build       Build Docker images"
	@echo "  make start       Start services"
	@echo "  make stop        Stop services"
	@echo "  make restart     Restart services"
	@echo "  make logs        View logs"
	@echo "  make clean       Clean up containers and images"
	@echo "  make dev-start   Start development environment"
	@echo "  make dev-stop    Stop development environment"
	@echo "  make test        Run tests"
	@echo ""

build:
	@echo "Building Docker images..."
	docker compose build

start:
	@echo "Starting Veritas Registry..."
	docker compose up -d
	@echo "Services started."
	@echo "  Backend: http://127.0.0.1:8080"
	@echo "  Frontend: http://localhost:3000 (HTTPS: https://localhost:3443)"

stop:
	@echo "Stopping services..."
	docker compose down

restart: stop start

logs:
	docker compose logs -f

clean:
	@echo "Cleaning up..."
	docker compose down -v
	docker system prune -f

dev-start:
	@echo "Starting development environment..."
	docker compose -f docker-compose.dev.yml up

dev-stop:
	@echo "Stopping development environment..."
	docker compose -f docker-compose.dev.yml down

test:
	@echo "Running backend tests..."
	cd backend && go test ./...
	@echo "Tests passed."

backend-build:
	@echo "Building backend..."
	cd backend && go build -o bin/server ./cmd/server

backend-run:
	@echo "Running backend..."
	cd backend && ./bin/server

frontend-install:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

frontend-build:
	@echo "Building frontend..."
	cd frontend && npm run build

frontend-dev:
	@echo "Running frontend dev server..."
	cd frontend && npm run dev

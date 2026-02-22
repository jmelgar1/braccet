.PHONY: db-up db-down migrate-auth migrate-tournament migrate-bracket migrate-community migrate-all migrate-down-auth migrate-down-tournament migrate-down-bracket migrate-down-community run-auth run-tournament run-bracket run-community run-gateway

# Database URLs for golang-migrate (local development)
AUTH_DB_URL := postgres://auth_user:auth_pass@localhost:5432/braccet_auth?sslmode=disable
TOURNAMENT_DB_URL := postgres://tournament_user:tournament_pass@localhost:5433/braccet_tournament?sslmode=disable
BRACKET_DB_URL := postgres://bracket_user:bracket_pass@localhost:5434/braccet_bracket?sslmode=disable
COMMUNITY_DB_URL := postgres://community_user:community_pass@localhost:5435/braccet_community?sslmode=disable

# Start all database containers
db-up:
	docker compose up -d
	@echo "Waiting for databases to be ready..."
	@sleep 5

# Stop all database containers
db-down:
	docker compose down

# Run services (loads .env.local for each)
run-auth:
	@cd auth && set -a && . ./.env.local && set +a && go run ./cmd/...

run-tournament:
	@cd tournament && set -a && . ./.env.local && set +a && go run ./cmd/...

run-bracket:
	@cd bracket && set -a && . ./.env.local && set +a && go run ./cmd/...

run-gateway:
	@cd gateway && set -a && . ./.env.local && set +a && go run ./cmd/...

run-community:
	@cd community && set -a && . ./.env.local && set +a && go run ./cmd/...

# Run migrations for Auth service
migrate-auth:
	migrate -path auth/migrations -database "$(AUTH_DB_URL)" up

# Run migrations for Tournament service
migrate-tournament:
	migrate -path tournament/migrations -database "$(TOURNAMENT_DB_URL)" up

# Run migrations for Bracket service
migrate-bracket:
	migrate -path bracket/migrations -database "$(BRACKET_DB_URL)" up

# Run migrations for Community service
migrate-community:
	migrate -path community/migrations -database "$(COMMUNITY_DB_URL)" up

# Run all migrations
migrate-all: migrate-auth migrate-tournament migrate-bracket migrate-community

# Rollback migrations
migrate-down-auth:
	migrate -path auth/migrations -database "$(AUTH_DB_URL)" down 1

migrate-down-tournament:
	migrate -path tournament/migrations -database "$(TOURNAMENT_DB_URL)" down 1

migrate-down-bracket:
	migrate -path bracket/migrations -database "$(BRACKET_DB_URL)" down 1

migrate-down-community:
	migrate -path community/migrations -database "$(COMMUNITY_DB_URL)" down 1

# Create new migration (usage: make create-migration SERVICE=auth NAME=add_column)
create-migration:
	migrate create -ext sql -dir $(SERVICE)/migrations -seq $(NAME)

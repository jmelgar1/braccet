.PHONY: db-up db-down migrate-auth migrate-tournament migrate-bracket migrate-all migrate-down-auth migrate-down-tournament migrate-down-bracket run-auth run-tournament run-bracket

# Database URLs for golang-migrate (local development)
AUTH_DB_URL := mysql://auth_user:auth_pass@tcp(localhost:3306)/braccet_auth
TOURNAMENT_DB_URL := mysql://tournament_user:tournament_pass@tcp(localhost:3307)/braccet_tournament
BRACKET_DB_URL := mysql://bracket_user:bracket_pass@tcp(localhost:3308)/braccet_bracket

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
	@set -a && . ./services/auth/.env.local && set +a && go run ./services/auth/cmd/...

run-tournament:
	@set -a && . ./services/tournament/.env.local && set +a && go run ./services/tournament/cmd/...

run-bracket:
	@set -a && . ./services/bracket/.env.local && set +a && go run ./services/bracket/cmd/...

# Run migrations for Auth service
migrate-auth:
	migrate -path services/auth/migrations -database "$(AUTH_DB_URL)" up

# Run migrations for Tournament service
migrate-tournament:
	migrate -path services/tournament/migrations -database "$(TOURNAMENT_DB_URL)" up

# Run migrations for Bracket service
migrate-bracket:
	migrate -path services/bracket/migrations -database "$(BRACKET_DB_URL)" up

# Run all migrations
migrate-all: migrate-auth migrate-tournament migrate-bracket

# Rollback migrations
migrate-down-auth:
	migrate -path services/auth/migrations -database "$(AUTH_DB_URL)" down 1

migrate-down-tournament:
	migrate -path services/tournament/migrations -database "$(TOURNAMENT_DB_URL)" down 1

migrate-down-bracket:
	migrate -path services/bracket/migrations -database "$(BRACKET_DB_URL)" down 1

# Create new migration (usage: make create-migration SERVICE=auth NAME=add_column)
create-migration:
	migrate create -ext sql -dir services/$(SERVICE)/migrations -seq $(NAME)

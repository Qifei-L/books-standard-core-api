run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate-up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

tidy:
	go mod tidy

.PHONY: run build migrate-up migrate-down tidy

SHELL := /bin/bash

# run the application
dev:
	go run cmd/main.go

# build the application
build:
	go build -o gin-langchain-ollama cmd/main.go

run:
	set -a && source .env && set +a && ./gin-langchain-ollama

# migration commands
migrate-up:
	./migrate.sh up

## make migration-down qtd=1
migrate-down:
	./migrate.sh down $(qtd)


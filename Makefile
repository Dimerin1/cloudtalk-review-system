.PHONY: up down build lint test

up:
	docker compose up --build

down:
	docker compose down -v

build:
	go build ./...

lint:
	golangci-lint run ./...

test:
	go test ./... -v

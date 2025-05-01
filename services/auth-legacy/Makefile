.PHONY: dev stop

dev:
	docker compose up -d
	go run ./cmd/main.go --config=$(shell pwd)/configs/file/development.yaml
stop:
	docker-compose down

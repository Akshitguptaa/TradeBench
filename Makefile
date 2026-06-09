.PHONY: up down build proto migrate lint test seed smoke

up:
	docker compose up --build -d

down:
	docker compose down -v

proto:
	bash scripts/gen-proto.sh

build:
	docker compose build

migrate:
	bash scripts/migrate.sh

test:
	cd tests && go test -v -race -count=1 ./...

smoke:
	bash scripts/smoke-test.sh

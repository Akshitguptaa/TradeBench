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
lint:
	@echo "Linting Go modules..."
	@for mod in $$(find . -name go.mod -not -path "*/\.*"); do \
		dir=$$(dirname "$$mod"); \
		echo "Linting $$dir..."; \
		(cd "$$dir" && go vet ./...); \
	done

test:
	cd tests && go test -v -race -count=1 ./...

smoke:
	bash scripts/smoke-test.sh

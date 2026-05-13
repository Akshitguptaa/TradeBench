.PHONY: dev dev-down proto build

dev:
	docker-compose -f docker-compose.yml up --build

dev-down:
	docker-compose down -v

proto:
	./scripts/gen-proto.sh

build:
	docker-compose build

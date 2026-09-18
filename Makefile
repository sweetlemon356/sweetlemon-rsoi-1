ENV_FILE ?= src/.env
-include $(ENV_FILE)
export

MIGRATE := cd src && go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1

.PHONY: db-up db-down migrate-up migrate-down run fmt test vet openapi-check check postman docker-build

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

migrate-up:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

run:
	cd src && go run ./cmd/server

fmt:
	cd src && go fmt ./...

test:
	cd src && go test ./...

vet:
	cd src && go vet ./...

openapi-check:
	cmp person-service.yaml src/internal/httpapi/openapi.yaml

check: test vet openapi-check

postman:
	npx --yes newman@6.2.2 run 'postman/[inst] Lab1.postman_collection.json' -e 'postman/[inst][local] Lab1.postman_environment.json'

docker-build:
	docker build --platform linux/amd64 -t sweetlemon-rsoi-1:local .

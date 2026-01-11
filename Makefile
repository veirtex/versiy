include .env
export POSTGRES_ADDR

MIGRATION_PATH = ./cmd/migrate/migrations

.PHONY: migrate-create
migration:
	@migrate create -seq -ext sql -dir $(MIGRATION_PATH) $(filter-out $@,$(MAKECMDGOALS))

.PHONY: migrate-up
migrate-up:
	@migrate -path=$(MIGRATION_PATH) -database="$(POSTGRES_ADDR)" up

.PHONY: migrate-down
migrate-down:
	@migrate -path=$(MIGRATION_PATH) -database="$(POSTGRES_ADDR)" down $(filter-out $@,$(MAKECMDGOALS))

BINDIR := bin

.PHONY: build install test lint clean sqlc db-migrate-up db-migrate-down

build:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/iceberg ./cmd/iceberg
	go build -o $(BINDIR)/iceberg-agent ./cmd/iceberg-agent

install: build
	mkdir -p $$HOME/.local/bin
	cp $(BINDIR)/iceberg $$HOME/.local/bin/iceberg
	cp $(BINDIR)/iceberg-agent $$HOME/.local/bin/iceberg-agent

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf $(BINDIR)

sqlc:
	sqlc generate

db-migrate-up:
	migrate -path db/migrations -database "$${DATABASE_URL}" up

db-migrate-down:
	migrate -path db/migrations -database "$${DATABASE_URL}" down 1

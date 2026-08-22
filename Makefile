BINARY=bin/server
GO=go
GOFLAGS=-v

.PHONY: build run test test-short test-cover lint clean

build:
	$(GO) build $(GOFLAGS) -o $(BINARY) ./cmd/server

run: build
	./$(BINARY)

test:
	$(GO) test $(GOFLAGS) ./...

test-short:
	$(GO) test -short ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

lint:
	$(GO) vet ./...

clean:
	rm -rf bin/ coverage.out

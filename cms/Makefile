APP     = cms
CMD     = ./cmd/server
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD   = $(shell date -u +%Y%m%d%H%M%S)
LDFLAGS = -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD)"

.PHONY: build test lint clean run run-preview

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(APP) $(CMD)

test:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f $(APP) coverage.out coverage.html

run:
	go run $(CMD)

run-preview:
	go run ./preview/

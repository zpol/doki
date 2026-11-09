APP=doki
PKG=./cmd/doki
VERSION?=0.1.0-dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-s -w \
	-X github.com/zpol/doki/internal/version.Version=$(VERSION) \
	-X github.com/zpol/doki/internal/version.Commit=$(COMMIT) \
	-X github.com/zpol/doki/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build test race vet fmt tidy run doctor clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) $(PKG)

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

run: build
	./bin/$(APP)

doctor: build
	./bin/$(APP) doctor

clean:
	rm -rf bin

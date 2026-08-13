VERSION ?= v0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/garund/garund/internal/buildinfo.Version=$(VERSION) \
           -X github.com/garund/garund/internal/buildinfo.Commit=$(COMMIT) \
           -X github.com/garund/garund/internal/buildinfo.BuildDate=$(BUILD_DATE)

.PHONY: all build build-frontend build-server build-agent install test lint dev clean release

all: build

build-frontend:
	@echo "Building Next.js static frontend..."
	@if [ ! -d "frontend/node_modules" ]; then \
		echo "Installing frontend dependencies..."; \
		(cd frontend && npm install); \
	fi
	cd frontend && NEXT_PUBLIC_API_BASE_URL="" npm run build
	@mkdir -p internal/web/dist
	rm -rf internal/web/dist/*
	cp -r frontend/out/* internal/web/dist/

build-server:
	@echo "Building Garund Control Plane CLI binary..."
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/garund ./main.go

build-agent:
	@echo "Building Garund Agent binary..."
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/garund-agent ./cmd/garund-agent/main.go

build: build-frontend build-server build-agent
	@echo "\n✓ Garund built successfully at bin/garund"
	@echo "  Run './bin/garund start' or run 'make install' to copy to ~/.local/bin/garund\n"

install: build
	@echo "Installing garund binary to ~/.local/bin..."
	@mkdir -p $(HOME)/.local/bin
	@cp bin/garund $(HOME)/.local/bin/garund
	@echo "\n✓ Garund installed successfully to $(HOME)/.local/bin/garund"
	@echo "  Run 'garund start' to launch!\n"

test:
	@echo "Running backend unit tests..."
	go test -v ./...

lint:
	@echo "Running go vet..."
	go vet ./...

dev-server:
	go run ./main.go

dev-frontend:
	cd frontend && npm run dev

dev:
	@echo "Starting Garund in development mode..."
	make -j2 dev-server dev-frontend

release: build-frontend
	@echo "Building release binaries for all target platforms..."
	@mkdir -p dist
	rm -rf dist/*
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/garund-linux-amd64 ./main.go
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/garund-linux-arm64 ./main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/garund-darwin-amd64 ./main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/garund-darwin-arm64 ./main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/garund-windows-amd64.exe ./main.go
	@cd dist && sha256sum * > SHA256SUMS
	@echo "Release build complete in dist/"

clean:
	rm -rf bin/ dist/ internal/web/dist/*

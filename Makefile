VERSION ?= v0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/garund/garund/internal/buildinfo.Version=$(VERSION) \
           -X github.com/garund/garund/internal/buildinfo.Commit=$(COMMIT) \
           -X github.com/garund/garund/internal/buildinfo.BuildDate=$(BUILD_DATE)

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

.PHONY: all build build-frontend build-server build-agent frontend backend run install uninstall test lint dev dev-server dev-frontend clean release

all: build

frontend: build-frontend

backend: build-server

run: build-server
	@./bin/garund start

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
	@echo "  Run './bin/garund start' or run 'make install' to copy to $(BINDIR)/garund\n"

install: build
	@echo "Installing garund binary to $(BINDIR)..."
	@mkdir -p $(BINDIR)
	@cp bin/garund $(BINDIR)/garund
	@chmod +x $(BINDIR)/garund
	@echo "\n✓ Garund installed successfully to $(BINDIR)/garund"
	@if echo ":$$PATH:" | grep -q ":$(BINDIR):"; then \
		echo "  Run 'garund start' to launch!\n"; \
	else \
		echo "\nNotice: $(BINDIR) is not in your current PATH."; \
		echo "  To run 'garund' directly in this session and future sessions, export it:"; \
		echo "      export PATH=\"$(BINDIR):\$$PATH\"\n"; \
		echo "  Add to your shell profile (~/.bashrc or ~/.zshrc):"; \
		echo "      echo 'export PATH=\"$(BINDIR):\$$PATH\"' >> ~/.bashrc\n"; \
	fi

uninstall:
	@echo "Uninstalling garund binary from $(BINDIR)..."
	@if [ -f "$(BINDIR)/garund" ]; then \
		rm -f "$(BINDIR)/garund"; \
		echo "✓ Removed $(BINDIR)/garund"; \
	else \
		echo "Notice: $(BINDIR)/garund not found."; \
	fi

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


.PHONY: all build test dev agent clean

all: build

build: build-server build-agent build-frontend

build-server:
	@echo "Building Garund Control Plane binary..."
	@mkdir -p bin
	go build -o bin/garund ./main.go

build-agent:
	@echo "Building Garund Agent binary..."
	@mkdir -p bin
	go build -o bin/garund-agent ./cmd/garund-agent/main.go

build-frontend:
	@echo "Building Next.js frontend..."
	cd frontend && npm run build

test:
	@echo "Running backend unit tests..."
	go test -v ./...

dev-server:
	go run ./main.go

dev-frontend:
	cd frontend && npm run dev

dev:
	@echo "Starting Garund Multi-Cluster Platform in development mode..."
	make -j2 dev-server dev-frontend

agent: build-agent
	@echo "Running local Garund Agent..."
	./bin/garund-agent

clean:
	rm -rf bin/

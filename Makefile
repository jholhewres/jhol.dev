.PHONY: dev build build-web build-go serve clean

# Development: parallel Go + Vite
dev:
	@echo "Starting dev servers..."
	@echo "Frontend: http://localhost:5173"
	@echo "Backend:  http://localhost:8123"
	@cd web && npm run dev &
	@go run ./cmd/server -dev -data ./data

# Production build
build: build-web build-go

build-web:
	cd web && npm ci && npm run build
	rm -rf cmd/server/dist
	cp -r web/dist cmd/server/dist

build-go:
	CGO_ENABLED=0 go build -o bin/jhol-dev ./cmd/server

# Build and run production server
serve: build
	./bin/jhol-dev -content ./content -data ./data

clean:
	rm -rf bin/ cmd/server/dist/ web/dist/ web/node_modules/

.PHONY: all build run gotool clean help

BINARY="event-server"
OLD_MODULE="grpc_gateway_framework"
PROTO_MODULE = github.com/arwoosa/event

all: gotool build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/${BINARY} ./cmd/event-server

# Start console (management) API server
run-console:
	@go run ./cmd/event-server console --config conf/config.yaml

# Start public API server
run-public:
	@go run ./cmd/event-server public --config conf/config.yaml

gotool:
	@echo "Running Go formatting tools..."
	@command -v gofumpt >/dev/null || { echo "Installing gofumpt..."; go install mvdan.cc/gofumpt@latest; }
	gofumpt -w .
	go vet ./...
	@echo "Code formatting completed."

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

# Testing
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-unit:
	go test ./internal/models ./internal/service -v

test-models:
	go test ./internal/models -v

test-service:
	go test ./internal/service -v

test-integration:
	go test ./internal/dao/repository -v

test-clean:
	rm -f coverage.out coverage.html

# Run tests with race detection
test-race:
	go test ./... -race -v

# Run tests with short flag (skip long-running tests)
test-short:
	go test ./... -short -v

# Run integration tests (requires Docker for testcontainers)
test-integration-testcontainer:
	go test ./internal/dao/repository -v -tags=integration

# Run unit tests only (exclude integration tests)
test-unit-only:
	go test ./internal/models ./internal/service -v

clean:
	@if [ -f ./bin/${BINARY} ]; then rm ./bin/${BINARY} ; fi

help:
	@echo "Build & Run:"
	@echo "make build - 編譯二進制檔案"
	@echo "make run-console - 啟動 console (管理) API 服務"
	@echo "make run-public - 啟動 public (公開) API 服務"
	@echo "make run - 啟動 console 服務 (向後兼容)"
	@echo "make clean - 移除二進制檔案"
	@echo "make gotool - Go tool 'fmt' and 'vet'"
	@echo "make lint - 運行 golangci-lint 檢查"
	@echo "make lint-fix - 運行 golangci-lint 並自動修復"
	@echo ""
	@echo "Services:"
	@echo "console - 內部管理 API (/console/events/*)"
	@echo "public  - 公開讀取 API (/events/*)"
	@echo ""
	@echo "Testing:"
	@echo "make test - 運行所有測試"
	@echo "make test-coverage - 運行測試並生成覆蓋率報告"
	@echo "make test-unit - 運行單元測試 (models + service)"
	@echo "make test-models - 運行 models 測試"
	@echo "make test-service - 運行 service 測試" 
	@echo "make test-integration - 運行集成測試"
	@echo "make test-race - 運行測試並檢測競態條件"
	@echo "make test-short - 運行快速測試"
	@echo "make test-clean - 清理測試生成的文件"

grpc:
	docker run --rm -v $$(pwd):/workspace -w /workspace 94peter/grpc-gateway-builder \
		protoc -I. -I /proto -I/proto/validate \
		--go_out=. \
		--go_opt=module=$(PROTO_MODULE) \
		--go-grpc_out=. \
		--go-grpc_opt=module=$(PROTO_MODULE) \
		--grpc-gateway_out=. \
		--grpc-gateway_opt=module=$(PROTO_MODULE) \
		--validate_out="lang=go,module=$(PROTO_MODULE):." \
		--openapiv2_out=docs --openapiv2_opt=logtostderr=true,json_names_for_fields=false,allow_merge=true \
		proto/*

# 	# Generate Go code for all proto files (including order for client usage)
# 	docker run --rm -v $$(pwd):/workspace -w /workspace 94peter/grpc-gateway-builder \
# 		protoc -I. -I /proto -I/proto/validate \
# 			--go_out=. --go_opt=paths=source_relative \
# 			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
# 			--validate_out="lang=go,paths=source_relative:." \
# 			--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
# 			--openapiv2_out=docs --openapiv2_opt=logtostderr=true,json_names_for_fields=false,allow_merge=true,output_format=yaml \
# 			api/event/*.proto api/*.proto
# 	# Generate gRPC-Gateway
# 	protoc -I . -I third_party/googleapis \
# 		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
# 		api/event/*.proto api/*.proto
# 	# Generate OpenAPI JSON only for event service APIs (exclude order)
# 	protoc -I . -I third_party/googleapis \
# 		--openapiv2_out=docs --openapiv2_opt=logtostderr=true,json_names_for_fields=false,allow_merge=true \
# 		api/event/*.proto api/*.proto
# 	# Generate OpenAPI YAML only for event service APIs (exclude order)
# 	protoc -I . -I third_party/googleapis \
# 		--openapiv2_out=docs --openapiv2_opt=logtostderr=true,json_names_for_fields=false,allow_merge=true,output_format=yaml \
# 		api/event/*.proto api/*.proto

docker_run:
	 docker run -p 8081:8081 -d partivo_event:1.0

# Docker Compose commands
docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

docker-compose-logs:
	docker-compose logs -f

docker-compose-build:
	docker build -t partivo_event:1.0 . && docker-compose up -d

# Individual service commands
docker-console-only:
	docker-compose up -d event-console mongodb

docker-public-only:
	docker-compose up -d event-public mongodb

docker-restart:
	docker-compose restart

mod:
	go mod edit -module ${BINARY}; \
	find . -type f -name '*.go' -exec sed -i '' "s|${OLD_MODULE}|${BINARY}|g" {} +; \
	go mod tidy

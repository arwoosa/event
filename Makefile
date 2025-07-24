.PHONY: all build run gotool clean help

BINARY="xxx"
OLD_MODULE="grpc_gateway_framework"

all: gotool build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o ./bin/${BINARY} ./cmd/server

run:
	@go run ./cmd/server -c internal/conf/config.yaml

gotool:
	go fmt ./...
	go vet ./...

clean:
	@if [ -f ./bin/${BINARY} ]; then rm ./bin/${BINARY} ; fi

help:
	@echo "make - 格式化 Go 程式碼, then go build"
	@echo "make build - go build"
	@echo "make run - go run"
	@echo "make clean - 移除二進制檔案 和 vim swap files"
	@echo "make gotool - Go tool 'fmt' and 'vet'"

grpc:
	protoc -I api \
	--go_out=api --go_opt=paths=source_relative \
	--go-grpc_out=api --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=api --grpc-gateway_opt=paths=source_relative \
	--validate_out=paths=source_relative,lang=go:api \
	api/*/*.proto api/*.proto

docker_run:
	 docker run -p 8081:8081 -d -v ./logs:/app/logs/ partivo_community:1.0

mod:
	go mod edit -module ${BINARY}; \
	find . -type f -name '*.go' -exec sed -i '' "s|${OLD_MODULE}|${BINARY}|g" {} +; \
	go mod tidy

wire:
	go run github.com/google/wire/cmd/wire ./cmd/server

clear_wire:
	rm cmd/server/.!*wire_gen.go


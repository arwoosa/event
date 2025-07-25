# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Run
- `make build` - Build the binary for Linux/amd64
- `make run` - Run the service locally with development config
- `go run ./cmd/server -c internal/conf/config.yaml` - Alternative run command

### Code Quality
- `make gotool` - Format code and run vet
- `go fmt ./...` - Format Go code
- `go vet ./...` - Run Go vet

### Protocol Buffers
- `make grpc` - Generate gRPC code from .proto files
- Generates Go files, gRPC-Gateway, and validation code

### Dependency Injection
- `make wire` - Generate Wire dependency injection code
- `make clear_wire` - Remove generated wire files

### Testing
- `go test ./...` - Run all tests (currently only in vulpes submodule)
- Tests are located in `pkg/vulpes/` subdirectories

### Module Management
- `make mod` - Update module name and references (used for project setup)
- `go mod tidy` - Clean up module dependencies

### Docker
- `make docker_run` - Run containerized service on port 8081

## Architecture Overview

This is a **Go gRPC microservice** for event management using the **Vulpes toolkit** framework. The service provides both gRPC and HTTP REST APIs through gRPC-Gateway.

### Key Architecture Patterns
- **Clean Architecture**: Separated into transport, service, repository, and data layers
- **Microservice**: Designed for horizontal scaling and service isolation
- **Dual Protocol**: gRPC for internal services, HTTP REST for web clients
- **Brand Isolation**: Multi-tenant architecture with brand-level data separation

### Core Technologies
- **Go 1.23+** with standard libraries
- **gRPC + gRPC-Gateway** for dual protocol support
- **MongoDB** for document storage with geospatial indexing
- **Vulpes Framework** for shared utilities and middleware
- **Prometheus** for metrics and monitoring
- **Viper** for configuration management

## Project Structure

```
event/
├── cmd/server/           # Application entry point
├── internal/             # Private application code
│   ├── conf/            # Configuration management
│   ├── service/         # Business logic layer
│   ├── dao/             # Data access layer
│   ├── models/          # Domain models
│   ├── dto/             # Data transfer objects
│   └── helper/          # Utility functions
├── api/                 # Protocol buffer definitions
├── pkg/vulpes/          # Shared toolkit (submodule)
└── deployments/         # Docker and deployment configs
```

### Vulpes Framework Integration
The project uses a local submodule of the Vulpes toolkit (`pkg/vulpes/`) which provides:
- **EzGRPC**: Automated gRPC server setup with interceptors
- **MongoDB utilities**: Connection management and operations
- **Structured logging**: Zap-based JSON logging
- **Prometheus metrics**: Automatic gRPC metrics collection
- **Validation**: Request/response validation utilities

## Configuration

### Config Files
- `internal/conf/config.yaml` - Main development configuration
- `internal/conf/config_docker.yaml` - Docker environment config
- Configuration is loaded via Viper with command-line flag support

### Environment Setup
The service requires:
- **MongoDB** running on localhost:27017 (configurable)
- **Go 1.23+** for development
- **Protocol Buffer compiler** for gRPC code generation

### Key Configuration Sections
- **MongoDB**: Database connection and credentials
- **Logging**: Structured JSON logging with rotation
- **Server**: Port configuration (default 8081)
- **Timezone**: Application timezone setting

## Business Domain

This service manages **Events** for a multi-brand platform with:

### Core Entities
- **Event**: Main entity with title, location, sessions, content
- **Location**: Geographic data with coordinates for spatial queries
- **Session**: Time-based event scheduling
- **Brand**: Tenant isolation boundary

### API Endpoints
- **Console API** (`/console/events/*`): Full CRUD for authenticated brand members
- **Public API** (`/events/*`): Read-only public access to published events

### Status Workflow
Events follow a state machine: `draft → published → archived`
- State transitions have business rule validations
- Published events can only be modified with order service checks

## Development Patterns

### Error Handling
- Use custom error types in `internal/dto/error.go`
- Convert business errors to appropriate gRPC status codes
- Structured error logging with context

### Database Operations
- Use repository pattern in `dao/repository/`
- MongoDB operations with proper indexing
- Geospatial queries for location-based features

### Validation
- Request validation using Vulpes validate package
- Business rule validation in service layer
- Input sanitization for security

### Headers and Authentication
The service expects these headers from API Gateway:
- `X-User-Id`, `X-User-Email`, `X-User-Name`, `X-User-Avatar`
- `X-Brand-Id` for brand isolation (needs to be added to header map)

## Important Notes

- **No direct authentication**: Relies on API Gateway for auth/authz
- **Brand isolation**: All data operations must include brand_id filtering
- **Submodule dependency**: Vulpes toolkit is included as Git submodule
- **Proto generation**: Run `make grpc` after modifying .proto files
- **Wire dependency injection**: Use `make wire` when adding new dependencies
# Event Microservice

A Go-based event management microservice built with gRPC and gRPC-Gateway, providing both console (management) and public APIs for event operations.

## Architecture

This microservice follows Clean Architecture principles with:
- **gRPC + HTTP REST APIs**: Dual protocol support via gRPC-Gateway
- **MongoDB**: Document storage with geospatial indexing
- **Brand Isolation**: Multi-tenant architecture with brand-level data separation
- **Vulpes Framework**: Shared utilities and middleware

## Environment Requirements

### System Requirements
- **Go**: 1.23+ (toolchain go1.24.1)
- **MongoDB**: 4.4+ (for geospatial queries support)
- **Protocol Buffers**: Latest version for gRPC code generation

### Development Dependencies
```bash
# Install Protocol Buffer compiler
brew install protobuf

# Install Go tools for gRPC
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

## Configuration Parameters

### Main Configuration (`conf/config.yaml`)

#### Application Settings
```yaml
name: "partivo_event"          # Service name
mode: "dev"                    # Environment: "dev" or "prod"
port: 8081                     # Service port
version: "1.0.0"               # Application version
time_zone: "Asia/Taipei"       # Timezone for the application
```

#### Logging Configuration
```yaml
log:
  level: "debug"               # Log level: debug, info, warn, error
```

#### MongoDB Configuration
```yaml
mongodb:
  host: "127.0.0.1"           # MongoDB host
  port: 27017                  # MongoDB port
  # user: ""                   # MongoDB username (optional)
  # password: ""               # MongoDB password (optional)
  db: "partivo_event"         # Database name
```

#### External Services
```yaml
external:
  order_service:
    endpoint: "192.168.1.134:8081"  # Order service gRPC endpoint
    timeout: "10s"                  # Request timeout
  # media_service:                  # Future media service config
  #   endpoint: ""
  #   timeout: "10s"
```

#### Pagination Settings
```yaml
pagination:
  default_page_size: 20        # Default page size for listings
  max_page_size: 100           # Maximum allowed page size
  default_location_radius: 1000 # Default radius for geo queries (meters)
```

### Docker Configuration (`conf/config_docker.yaml`)

Same as main configuration but with Docker-specific settings:
```yaml
mongodb:
  host: "host.docker.internal"  # Docker host networking
```

## Deployment Methods

### 1. Local Development

#### Prerequisites
```bash
# Start MongoDB
brew services start mongodb-community
# or
docker run -d -p 27017:27017 --name mongodb mongo:latest
```

#### Build and Run
```bash
# Build the service
make build

# Run Console API (management)
make run-console

# Run Public API (read-only)
make run-public

# Or run with custom config
go run ./cmd/event-server console --config conf/config.yaml
go run ./cmd/event-server public --config conf/config.yaml
```

### 2. Docker Deployment

#### Build Docker Image
```bash
# Build image
docker build -t partivo_event:1.0 .

# Run container
make docker_run
# or
docker run -p 8081:8081 -d partivo_event:1.0
```

#### Docker Compose Deployment (Recommended)

Use the provided Docker Compose setup to run both Console and Public APIs:

```bash
# Build image first
docker build -t partivo_event:1.0 .

# Start all services (Console API + Public API + MongoDB)
make docker-compose-up
# or
docker-compose up -d

# Check logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

**Service URLs:**
- Console API: http://localhost:8081
- Public API: http://localhost:8082
- MongoDB: localhost:27017

**Individual service commands:**
```bash
# Run only Console API
make docker-console-only

# Run only Public API
make docker-public-only

# Restart services
make docker-restart
```

### 3. Production Deployment

#### Environment Variables
```bash
# Override config file location
export CONFIG_FILE=/app/conf/config_production.yaml

# MongoDB connection (if using environment variables)
export MONGODB_HOST=your-mongodb-host
export MONGODB_PORT=27017
export MONGODB_DB=partivo_event_prod
```

#### Production Checklist
- [ ] Use `mode: "prod"` in configuration
- [ ] Configure proper MongoDB replica set
- [ ] Set up log rotation and monitoring
- [ ] Configure service monitoring and alerting
- [ ] Set up reverse proxy (nginx) for load balancing
- [ ] Enable HTTPS/TLS termination

## Service Startup

### Dual API Architecture

The service provides two separate APIs:

#### Console API (`make run-console`)
- **Purpose**: Internal management operations
- **Endpoints**: `/console/events/*`
- **Features**: Full CRUD operations, event state management
- **Authentication**: Requires API Gateway headers

#### Public API (`make run-public`)
- **Purpose**: Public read-only access
- **Endpoints**: `/events/*`
- **Features**: Read published events only
- **Authentication**: Optional, read-only operations

### API Gateway Headers

Both services expect these headers from the API Gateway:
```
X-User-Id: user-uuid
X-User-Email: user@example.com  
X-User-Name: User Name
X-Brand-Id: brand-uuid           # Required for brand isolation
```

### Service Monitoring

Monitor services using Docker Compose:
```bash
# View service status
docker-compose ps

# View service logs
docker-compose logs -f event-console
docker-compose logs -f event-public

# Check resource usage
docker stats
```

## Development Commands

### Code Generation
```bash
# Generate gRPC code from proto files
make grpc
```

### Testing
```bash
# Run all tests
make test

# Run unit tests only (recommended)
make test-unit

# Run with coverage
make test-coverage

# Run integration tests (requires Docker)
make test-integration
```

### Code Quality
```bash
# Format and vet code
make gotool

# Individual commands
go fmt ./...
go vet ./...
```

## Troubleshooting

### Common Issues

#### 1. MongoDB Connection Failed
**Error**: `failed to connect to mongodb`
**Solution**: 
```bash
# Check MongoDB is running
brew services list | grep mongodb
# or
docker ps | grep mongo

# Test connection
mongosh --host localhost --port 27017
```

#### 2. Port Already in Use
**Error**: `bind: address already in use`
**Solution**:
```bash
# Find process using port 8081
lsof -i :8081
kill -9 <PID>

# Or change port in config
port: 8082
```

#### 3. gRPC Code Generation Failed
**Error**: `protoc: command not found`
**Solution**:
```bash
# Install Protocol Buffer compiler
brew install protobuf

# Verify installation
protoc --version
```

#### 4. Vulpes Dependency Issues
**Error**: `vulpes package not found`
**Solution**:
```bash
# Check go.mod replace directive
grep vulpes go.mod

# Re-download dependencies
go mod download
go mod tidy
```

#### 5. Docker Build Failed
**Error**: `failed to compute cache key`
**Solution**:
```bash
# Clean Docker cache
docker system prune -a

# Check Dockerfile paths
docker build --no-cache -t partivo_event:1.0 .
```

### Logging and Debugging

#### Enable Debug Logging
```yaml
# In config.yaml
log:
  level: "debug"
mode: "dev"
```

#### Check Application Logs
```bash
# Console output (development)
tail -f logs/app.log

# Docker logs
docker logs -f container_name
```

### Dependencies Check

#### Verify External Services
```bash
# Test Order Service connection
grpcurl -plaintext 192.168.1.134:8081 list

# Test MongoDB connection  
mongosh mongodb://localhost:27017/partivo_event
```

### Performance Monitoring

#### MongoDB Indexes
```bash
# Check if indexes are created
mongosh partivo_event --eval "db.events.getIndexes()"
mongosh partivo_event --eval "db.sessions.getIndexes()"
```

#### Memory and CPU Usage
```bash
# Monitor resource usage
docker stats partivo_event
top -p $(pgrep event-server)
```

---

## Notes

- `dao/mongodb/migration` can define collections and indexes, which will be created during initialization
- The service uses the Vulpes framework for shared utilities and middleware
- Brand isolation is implemented at the service layer - ensure all operations include brand filtering
- For production deployment, consider using a process manager like systemd or supervisord
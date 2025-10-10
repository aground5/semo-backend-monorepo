# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This project follows a **microservices architecture** using Go Workspaces for monorepo management. Each service is designed with clean/hexagonal architecture principles.

### Applications

- **geo**: Geolocation service (currently implemented)
- **api**: API gateway service (planned)
- **auth**: Authentication service (planned)
- **notification**: Notification service (planned)

Each service exposes both gRPC and HTTP APIs, with Protocol Buffers for service-to-service communication.

### Key Packages

**Shared Packages (`/pkg/`):**

- `config`: Common configuration handling using Viper
- `logger`: Logging implementations for Echo, GORM, gRPC, and Zap

**Service Structure Pattern:**

- `cmd/server`: Service entry point
- `internal/adapter`: Interface adapters (HTTP/gRPC handlers, repositories)
- `internal/domain`: Domain entities and repository interfaces
- `internal/infrastructure`: Infrastructure implementations
- `internal/usecase`: Business logic implementation
- `internal/config`: Service-specific configuration

## Common Commands

```bash
# Development
make setup          # Set up development environment
make run           # Run all services with docker-compose
make air-geo       # Run geo service with hot reload

# Building
make build         # Build all services
make docker-geo    # Build geo service Docker image

# Code Generation
make proto-gen     # Generate protobuf code
make mock          # Generate mocks using mockery

# Quality
make test          # Run all tests
make lint          # Run golangci-lint
make tidy          # Tidy all Go modules
```

## Folder Structure

```
/bin/              # Compiled binaries
/configs/          # Configuration files
  /dev/            # Development configs
  /example/        # Example config templates
/deployments/      # Deployment configurations
  /docker/         # Dockerfiles for each service
  /k8s/helm/       # Kubernetes Helm charts
/docs/             # Documentation
/pkg/              # Shared Go packages
/proto/            # Protocol Buffer definitions
/scripts/          # Utility scripts
/services/         # Service implementations
/tools/            # Go tools dependencies
```

## Technology Stack

- **Language**: Go 1.23.6
- **Web Framework**: Echo v4
- **RPC**: gRPC
- **Logging**: Zap (structured logging)
- **Configuration**: Viper
- **Database**: PostgreSQL 14
- **Container**: Docker with distroless images
- **Orchestration**: Kubernetes with Helm
- **Development**: Air (hot reload), Mockery (mocking)

## Key Development Patterns

1. **Clean Architecture**: Strict separation of concerns with layers
2. **Dependency Injection**: Interface-based with constructor injection
3. **Error Handling**: Structured error types with proper propagation
4. **Configuration**: Environment-based YAML configs
5. **Service Communication**: gRPC with Protocol Buffers
6. **API Design**: RESTful HTTP alongside gRPC endpoints

## Database

- **Type**: PostgreSQL 14
- **Pattern**: One database per service (e.g., geo_db, auth_db)
- **Initialization**: Use `scripts/create_multiple_dbs.sh` for setup
- **Configuration**: Database connections in service YAML configs

## Error Logging

- **Logger**: Zap with structured fields
- **Levels**: debug, info, warn, error, dpanic, panic, fatal
- **Format**: JSON (production) or console (development)
- **Features**:
  - Request/response middleware logging
  - Caller information in development
  - Context-aware logging

## Testing

- **Unit Tests**: Follow Go conventions with `_test.go` suffix
- **Mocking**: Mockery for generating mocks (see `.mockery.yaml`)
- **Running Tests**: `make test`
- **Example**: See `services/geo/internal/usecase/geo/examples_test.go`

## Deployment

### Local Development

```bash
make run           # Start all services with docker-compose
make air-geo       # Hot reload for geo service
```

### Production

- **Build**: Multi-stage Dockerfiles with distroless base images
- **Deploy**: Kubernetes using Helm charts
- **Charts**: Located in `/deployments/k8s/helm/`
- **Resources**: Deployment, Service, ConfigMap, Ingress, HPA

### Build Variables

Services are built with version information:

- Version (from git tag)
- Commit hash
- Build date

## Important Notes

1. **Service Independence**: Each service should be independently deployable
2. **Configuration**: Never commit secrets; use example configs as templates
3. **Logging**: Always use structured logging with appropriate levels
4. **Testing**: Write tests for new features following existing patterns
5. **Protocol Buffers**: Run `make proto-gen` after modifying `.proto` files
6. **Dependencies**: Run `make tidy` after adding new dependencies

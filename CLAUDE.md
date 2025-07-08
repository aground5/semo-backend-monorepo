# Project Structure Overview

This document describes the roles and purposes of each folder in the semo-backend-monorepo project.

## Root Directories

### `/bin/`

Contains compiled binary executables for the services. Currently contains:

- `geo` - The compiled geo service binary

### `/configs/`

Stores configuration files for different environments and services:

- `/dev/` - Development environment configurations
- `/example/` - Example configuration templates
- Contains YAML configs for: api, auth, notification, and geo services

### `/deployments/`

Infrastructure and deployment configurations:

- `/docker/` - Dockerfile definitions for containerizing each service (api, auth, geo, notification)
- `/k8s/helm/` - Kubernetes Helm charts for deploying services
  - Individual charts for: admin, api, auth, and worker deployments
  - Each chart includes standard Kubernetes resources (deployment, service, configmap, etc.)

### `/docs/`

Project documentation directory

### `/pkg/`

Shared Go packages used across multiple services:

- `/config/` - Common configuration handling
- `/logger/` - Logging implementations for different frameworks (Echo, GORM, gRPC, Zap)

### `/proto/`

Protocol Buffer definitions and generated Go code:

- `/api/v1/` - API service protobuf definitions
- `/auth/v1/` - Authentication service protobuf definitions
- `/geo/v1/` - Geolocation service protobuf definitions
- `/notification/v1/` - Notification service protobuf definitions

### `/scripts/`

Utility scripts for development and operations:

- `create_multiple_dbs.sh` - Database initialization script

### `/services/`

Main service implementations. Currently contains:

- `/geo/` - Geolocation service implementation

Note: Additional services (api, auth, notification) can be added following the same structure as the geo service.

### `/tools/`

Go tools dependencies management using Go modules. Contains tool imports for:

- Protocol buffer code generation
- gRPC code generation

## Configuration Files

- `docker-compose.yml` - Local development environment setup
- `go.work` & `go.work.sum` - Go workspace configuration for multi-module project
- `Makefile` - Build and development automation tasks

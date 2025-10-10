# Payment Service

This service handles payment processing for the SEMO backend.

## Structure

```
payment/
├── cmd/
│   └── server/
│       └── main.go          # Service entry point
├── internal/
│   ├── adapter/             # External interfaces implementation
│   │   ├── handler/         # Request handlers
│   │   │   ├── grpc/       # gRPC handlers
│   │   │   └── http/       # HTTP handlers
│   │   └── repository/      # Data persistence
│   ├── config/             # Configuration structures
│   ├── domain/             # Business logic
│   │   ├── entity/         # Domain models
│   │   └── repository/     # Repository interfaces
│   ├── infrastructure/     # Infrastructure setup
│   │   ├── grpc/          # gRPC server
│   │   └── http/          # HTTP server
│   └── usecase/           # Application business logic
├── go.mod
├── go.sum
└── README.md
```

## Running the Service

```bash
go run cmd/server/main.go
```

## Configuration

The service expects a configuration file at `./configs/payment_legacy.yaml` or the path specified in the `CONFIG_PATH` environment variable.  
You can embed environment variable placeholders (for example `${DATABASE_URL}`) inside the YAML; the loader expands them at runtime, which is useful when wiring Railway-provided credentials.  
For Postgres specifically, set `database.url` to your connection string (e.g. `${DATABASE_URL}`); if `url` is omitted, the legacy `host` / `port` / `user` fields are still supported for local development.

## API Endpoints

### HTTP
- `GET /health` - Health check endpoint
- `POST /api/v1/payments` - Create a new payment
- `GET /api/v1/payments/:id` - Get payment by ID
- `GET /api/v1/payments?user_id=xxx` - Get payments by user ID

### gRPC
The service also exposes gRPC endpoints for internal service communication.

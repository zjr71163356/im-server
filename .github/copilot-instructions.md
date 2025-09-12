# Copilot Instructions for im-server

## Overview

This project is a high-performance, distributed IM (Instant Messaging) server, designed for scalability, reliability, and extensibility. The architecture is modular, with clear service boundaries and a focus on real-world IM requirements (multi-device, offline, message queue, etc.).

## Architecture & Major Components

- **Gateway Layer**: Uses grpc-gateway for HTTP to gRPC conversion, providing REST API endpoints. See `gateway/grpc_gateway.go` and `cmd/gateway/main.go`.
- **Auth Service**: Handles user registration, login, and token verification. gRPC service with HTTP endpoints via grpc-gateway.
- **User Service**: Manages user search and profile operations. gRPC service with HTTP endpoints via grpc-gateway.
- **Connect Service**: Manages WebSocket/TCP long connections, user authentication, heartbeats, and connection binding. See `internal/connect/`.
- **Device Service**: Handles device management and connection status. Pure gRPC service without HTTP endpoints.
- **Message Queue**: Kafka/NATS/NSQ is used for decoupling message delivery and persistence (see `DESIGN.md`).
- **Storage**: MongoDB for message bodies, Redis for online state and offline messages, MySQL for user/device/friend/group data.
- **Push Service**: For offline push (APNs/FCM), see design notes in `DESIGN.md`.

## Data Flow Example

- Client connects via WebSocket → Gateway authenticates and binds user/device → Messages are routed via IM Core → Delivered to online users or queued for offline → Persisted in MongoDB/Redis/MySQL as appropriate.

## Developer Workflows

- **Build**: Standard Go build. Entrypoints: `cmd/auth/main.go`, `cmd/user/main.go`, `cmd/gateway/main.go`, etc.
- **Proto Generation**: Run `make proto` to generate Go code from all `.proto` files. Includes grpc-gateway HTTP handlers.
- **DB Migration**: Use `make migrate-up` and `make migrate-down` (see `Makefile`).
- **SQLC**: SQL queries in `db/queries/`, generate Go code with `make sqlc-generate` or `sqlc generate`.
- **Testing**: Use `scripts/test_api.sh` for comprehensive API testing of auth and user services.
- **grpc-gateway**: HTTP API server runs on port 8080, provides REST endpoints for auth and user services.

## Project Conventions & Patterns

- **gRPC**: Service boundaries are defined via Protobuf in `pkg/protocol/proto/`, with generated clients/servers in `pkg/protocol/pb/`.
- **Config**: Centralized in `pkg/config/config.go`, supports YAML/JSON, including per-service gRPC client config.
- **Connection Management**: Each user/device connection is tracked in memory (see `Conn`, `Session`), with device/user IDs as keys.
- **Message Handling**: All incoming messages are deserialized and dispatched via `Conn.HandleMessage`. Unauthorized commands are rejected early.
- **Error Logging**: Uses `log/slog` for structured logging throughout the codebase.
- **Extensibility**: New message types/commands are added via Protobuf and handled in the `switch` in `HandleMessage`.

## Integration Points

- **External Services**: Kafka/NATS (MQ), MongoDB, Redis, MySQL, APNs/FCM (push).
- **gRPC**: Internal service-to-service calls (see `DeviceIntService` and related clients).
- **WebSocket**: Client entrypoint, see `wsHandler` and `StartWSConn`.

## Key Files & Directories

- `cmd/auth/main.go`: Auth service entrypoint.
- `cmd/user/main.go`: User service entrypoint.
- `cmd/gateway/main.go`: Gateway service entrypoint (grpc-gateway).
- `gateway/grpc_gateway.go`: grpc-gateway server implementation.
- `internal/auth/`: Auth service logic and HTTP handlers.
- `internal/user/`: User service logic and handlers.
- `internal/connect/`: Connection/session/message handling.
- `pkg/protocol/proto/`: Protobuf definitions with grpc-gateway annotations.
- `pkg/protocol/pb/`: Generated Go code from proto.
- `db/queries/`: SQLC query files.
- `pkg/dao/`: SQLC-generated DB access code (models, queries, and interfaces).
- `scripts/`: Test scripts and utilities.
- `pkg/config/config.go`: Central config structs.
- `Makefile`: Build, proto, and migration commands.
- `DESIGN.md`: High-level architecture, rationale, and workflow notes.

## Database Schema

The database schema includes the following main tables:

- **user**: User accounts with authentication (username, email, phone_number, hashed_password, salt)
- **device**: User devices with connection info and online status
- **friend**: User friendship relationships
- **group**: Group chat entities
- **group_user**: Group membership relationships
- **message**: Message storage
- **user_message**: User-specific message indexing
- **seq**: Sequence numbers for message ordering

Key features:

- Multi-field unique constraints (username, email, phone_number)
- Password hashing with salt for security
- Device-based session management
- Group chat support
- Message sequencing for proper ordering

## Examples

- See `README.md` and `DESIGN.md` for architecture diagrams, workflow explanations, and development order.
- For message handling, see `Conn.HandleMessage` in `internal/connect/conn.go`.
- For gRPC client config, see `RPCClientConfig` in `pkg/config/config.go`.

---

If any section is unclear or missing, please provide feedback to improve these instructions.

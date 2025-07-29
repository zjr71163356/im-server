# Copilot Instructions for im-server

## Overview

This project is a high-performance, distributed IM (Instant Messaging) server, designed for scalability, reliability, and extensibility. The architecture is modular, with clear service boundaries and a focus on real-world IM requirements (multi-device, offline, message queue, etc.).

## Architecture & Major Components

- **Gateway Layer**: Manages WebSocket/TCP long connections, user authentication, heartbeats, and connection binding. See `internal/connect/` and `StartWSServer` in `cmd/main.go`.
- **IM Core Service**: Handles message routing, storage, and forwarding. Message processing logic is in `internal/connect/conn.go` (`Conn`, `HandleMessage`, `SignIn`, etc.).
- **Message Queue**: Kafka/NATS/NSQ is used for decoupling message delivery and persistence (see `DESIGN.md`).
- **User System**: Auth, session, and friend management. User/device info is in MySQL, with SQLC-generated code in `internal/db/`.
- **Storage**: MongoDB for message bodies, Redis for online state and offline messages, MySQL for user/device/friend data.
- **Push Service**: For offline push (APNs/FCM), see design notes in `DESIGN.md`.

## Data Flow Example

- Client connects via WebSocket → Gateway authenticates and binds user/device → Messages are routed via IM Core → Delivered to online users or queued for offline → Persisted in MongoDB/Redis/MySQL as appropriate.

## Developer Workflows

- **Build**: Standard Go build. Entrypoint: `cmd/main.go`.
- **Proto Generation**: Run `make proto` to generate Go code from all `.proto` files. Output is controlled by each proto's `option go_package`.
- **DB Migration**: Use `make migrate-up` and `make migrate-down` (see `Makefile`).
- **SQLC**: SQL queries in `db/queries/`, generate Go code with `sqlc generate`.
- **Testing**: (Add test conventions here if present.)

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

- `cmd/main.go`: Entrypoint, server startup.
- `internal/connect/`: Gateway logic, connection/session/message handling.
- `pkg/protocol/proto/`: Protobuf definitions.
- `pkg/protocol/pb/`: Generated Go code from proto.
- `db/queries/`: SQLC query files.
- `internal/db/`: SQLC-generated DB access code.
- `pkg/config/config.go`: Central config structs.
- `Makefile`: Build, proto, and migration commands.
- `DESIGN.md`: High-level architecture, rationale, and workflow notes.

## Examples

- See `README.md` and `DESIGN.md` for architecture diagrams, workflow explanations, and development order.
- For message handling, see `Conn.HandleMessage` in `internal/connect/conn.go`.
- For gRPC client config, see `RPCClientConfig` in `pkg/config/config.go`.

---

If any section is unclear or missing, please provide feedback to improve these instructions.

# Repository Guidelines
学习性质的项目，不需要太多细节
## language
使用中文
## logs
更新代码后更新/home/tyrfly1001/im-server/README.md中的### 待完善清单（落地优先）
## Project Structure & Module Organization
- `cmd/`: Service entrypoints (`auth`, `user`, `device`, `connect`, `gateway`, `message`).
- `internal/`: Business logic by domain (e.g., `connect/`, `device/`, `friend/`, `message/`).
- `pkg/`: Shared libs (config loading, `dao` via sqlc, Redis/Kafka clients, JWT, Mongo storage).
- `db/`: Migrations (`db/migrations/`) and SQL queries for sqlc (`db/queries/`).
- `frontend/`: Vue3 demo for login/WS validation.
- `docs/`: Notes, learning guides; `Makefile` aggregates common workflows.

## Build, Test, and Development Commands
- Dependencies: `docker compose up -d` (MySQL/Redis/Mongo/Kafka via `docker-compose.yml`).
- Migrations & codegen: `make migrate-up`, `make sqlc-generate`, `make proto` (protobufs), `make proto-openapi`.
- Build binaries: `make build-all` or per service (`make build-connect`, `make build-device`, etc.).
- Run locally: `make start` (deps + services), stop via `make stop`.
- Tests: `go test ./...`; for unit tests without Redis init use `IM_REDIS_SKIP_INIT=1 go test ./...`.
- Frontend: `cd frontend && pnpm install && pnpm dev`.

## Coding Style & Naming Conventions
- Go code must be `gofmt`-ed; prefer `context.Context` named `ctx` and short, explicit error messages.
- Package layout mirrors domains; keep shared helpers in `pkg/`, service-only logic in `internal/`.
- Proto/DB generated code lives under `pkg/`; do not edit generated files manually.
- Use lower_snake_case for SQL identifiers; Go variables/functions in mixedCaps; keep request/response structs near handlers.

## Testing Guidelines
- Framework: standard `go test`; DAO mocks via `make mockdb` (mockgen for `pkg/dao/querier.go`); Redis logic can use redismock.
- Add table-driven tests for handlers, DAO, and connect/message flows; validate error branches and idempotency keys.
- Naming: `<file>_test.go` with functions `TestXxx`; keep fixtures small and in-memory where possible.
- For integration, start deps (`docker compose up -d`) then exercise HTTP/WS paths; document steps in `docs/` if non-trivial.

## Commit & Pull Request Guidelines
- Commits: short imperative subject (e.g., `fix: handle offline cleanup`); group related changes; avoid committing generated artifacts alone.
- PRs: describe scope, risks, and how to validate; link issues if any; note commands/tests run. Frontend/UI changes should include brief before/after notes or screenshots.

## Security & Configuration Tips
- Secrets (DB DSN, JWT secret) come from `config.yaml`/env; never hardcode in code or commits.
- Redis/Kafka/MySQL ports are exposed via compose defaults; adjust via env if running locally.
- Log with care: avoid dumping tokens/passwords; prefer structured logging already used in services.

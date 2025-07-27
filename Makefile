migrate-up:
	migrate -database "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true" -path db/migrations up

migrate-down:
	migrate -database "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true" -path db/migrations down

.PHONY: migrate-up migrate-down
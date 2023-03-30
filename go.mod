module github.com/aaarkadev/collectalertagent

go 1.18

replace github.com/aaarkadev/collectalertagent/internal/handlers => ./internal/handlers

replace github.com/aaarkadev/collectalertagent/internal/repositories => ./internal/repositories

replace github.com/aaarkadev/collectalertagent/internal/storages => ./internal/storages

replace github.com/aaarkadev/collectalertagent/internal/types => ./internal/types

require (
	github.com/go-chi/chi/v5 v5.0.8
	github.com/jackc/pgx/v5 v5.3.1
	github.com/jmoiron/sqlx v1.3.5
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/text v0.7.0 // indirect
)

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
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shirou/gopsutil/v3 v3.23.3 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.7.0 // indirect
)

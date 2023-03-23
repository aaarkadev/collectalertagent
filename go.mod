module github.com/aaarkadev/collectalertagent

go 1.18

replace github.com/aaarkadev/collectalertagent/internal/handlers => ./internal/handlers

replace github.com/aaarkadev/collectalertagent/internal/repositories => ./internal/repositories

replace github.com/aaarkadev/collectalertagent/internal/storages => ./internal/storages

replace github.com/aaarkadev/collectalertagent/internal/types => ./internal/types

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-chi/chi/v5 v5.0.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

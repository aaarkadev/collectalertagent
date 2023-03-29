package main

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	/*
		errDb := db.QueryRow("select \"ID\",\"MType\" from metrics limit 1").Scan(&name, &typ)
		if errDb != nil {
			fmt.Println(errDb)
			panic("%")
		}
	*/
	config := configs.InitServerConfig()
	repo := storages.FileStorage{Config: config}
	repo.Init()
	defer func() {
		repo.Shutdown()
	}()

	serverData := servers.ServerHandlerData{}
	serverData.Repo = &repo
	serverData.Config = config
	if len(config.DSN) > 0 {
		//"postgres://dron:dron@localhost:5432/dron"
		conn, connErr := sql.Open("pgx", config.DSN)
		if connErr != nil {
			config.DSN = ""
		} else {
			serverData.DbConn = conn
			defer conn.Close()
		}
	}

	router := chi.NewRouter()
	router.Use(servers.GzipMiddleware)
	router.Use(servers.UnGzipMiddleware)
	//router.Use(middleware.DefaultCompress)
	//router.Use(middleware.Compress(6, "gzip"))

	router.Post("/update/{type}/{name}/{value}", servers.BindServerToHandler(&serverData, handlers.HandlerUpdateRaw))
	router.Post("/update/", servers.BindServerToHandler(&serverData, handlers.HandlerUpdateJSON))

	router.Get("/value/{type}/{name}", servers.BindServerToHandler(&serverData, handlers.HandlerFuncOneRaw))
	router.Post("/value/", servers.BindServerToHandler(&serverData, handlers.HandlerFuncOneJSON))
	router.Get("/", servers.BindServerToHandler(&serverData, handlers.HandlerFuncAll))
	router.Get("/ping", servers.BindServerToHandler(&serverData, handlers.HandlerPingDB))

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	defer mainCtxCancel()

	servers.StartServer(mainCtx, config, router)

	fmt.Println("SERVER END")

}

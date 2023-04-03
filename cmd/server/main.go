package main

import (
	"context"
	"fmt"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/go-chi/chi/v5"
)

func main() {

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	defer mainCtxCancel()

	config := configs.InitServerConfig()
	config.MainCtx = mainCtx

	var repo repositories.Repo
	repo = &storages.DBStorage{Config: config}
	// repo = repositories.Repo(&storages.DBStorage{Config: config})
	isInitSuccess := repo.Init()
	if !isInitSuccess {
		repo = &storages.FileStorage{Config: config}
		// repo = (*storages.FileStorage)(&storages.FileStorage{Config: config})
		isInitSuccess = repo.Init()
	}
	defer func() {
		repo.Shutdown()
	}()

	serverData := servers.ServerHandlerData{}
	serverData.Repo = repo
	serverData.Config = config

	router := chi.NewRouter()
	router.Use(servers.GzipMiddleware)
	router.Use(servers.UnGzipMiddleware)
	//router.Use(middleware.DefaultCompress)
	//router.Use(middleware.Compress(6, "gzip"))

	router.Post("/update/{type}/{name}/{value}", servers.BindServerToHandler(&serverData, handlers.HandlerUpdateRaw))
	router.Post("/update/", servers.BindServerToHandler(&serverData, handlers.HandlerUpdateJSON))
	router.Post("/updates/", servers.BindServerToHandler(&serverData, handlers.HandlerUpdatesJSON))

	router.Get("/value/{type}/{name}", servers.BindServerToHandler(&serverData, handlers.HandlerFuncOneRaw))
	router.Post("/value/", servers.BindServerToHandler(&serverData, handlers.HandlerFuncOneJSON))
	router.Get("/", servers.BindServerToHandler(&serverData, handlers.HandlerFuncAll))
	router.Get("/ping", servers.BindServerToHandler(&serverData, handlers.HandlerPingDB))

	servers.StartServer(mainCtx, config, router)

	fmt.Println("SERVER END")

}

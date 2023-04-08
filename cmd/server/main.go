package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

func main() {

	f, err := os.OpenFile("/home/dron/go/src/dron/collectalertagent/log.sever.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	defer mainCtxCancel()

	config := configs.InitServerConfig()
	config.MainCtx = mainCtx

	var repo repositories.Repo
	repo = &storages.DBStorage{Config: &config}
	isInitSuccess := repo.Init()
	if !isInitSuccess {
		log.Println(types.NewTimeError(fmt.Errorf("Init DB repo failed. falback to file.")))
		repo = &storages.FileStorage{Config: config}
		isInitSuccess = repo.Init()
		if !isInitSuccess {
			log.Println(types.NewTimeError(fmt.Errorf("Init File repo failed. falback to mem.")))
		}
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

	router.Post("/update/{type}/{name}/{value}", servers.BindServerToHandler(&serverData, handlers.HandlerUpdateRaw))
	router.Post("/update/", servers.BindServerToHandler(&serverData, handlers.HandlerUpdateJSON))
	router.Post("/updates/", servers.BindServerToHandler(&serverData, handlers.HandlerUpdatesJSON))

	router.Get("/value/{type}/{name}", servers.BindServerToHandler(&serverData, handlers.HandlerFuncOneRaw))
	router.Post("/value/", servers.BindServerToHandler(&serverData, handlers.HandlerFuncOneJSON))
	router.Get("/", servers.BindServerToHandler(&serverData, handlers.HandlerFuncAll))
	router.Get("/ping", servers.BindServerToHandler(&serverData, handlers.HandlerPingDB))

	servers.StartServer(mainCtx, config, router)

	log.Println(types.NewTimeError(fmt.Errorf("SERVER END")))
}

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

func main() {
	servers.SetupLog()
	log.Println(types.NewTimeError(fmt.Errorf("START")))

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	mainCtx = context.WithValue(mainCtx, "mainCtxCancel", mainCtxCancel)
	defer mainCtxCancel()

	config := configs.InitServerConfig()
	config.MainCtx = mainCtx

	repo, serverData := servers.Init(&config)
	defer func() {
		servers.StopServer(repo)
	}()

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

	log.Println(types.NewTimeError(fmt.Errorf("END")))
}

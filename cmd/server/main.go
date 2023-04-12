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
	mainCtx = context.WithValue(mainCtx, types.MainCtxCancelFunc, mainCtxCancel)
	defer mainCtxCancel()

	config := configs.InitServerConfig()

	repo, serverData := servers.Init(mainCtx, &config)
	defer func() {
		servers.StopServer(mainCtx, repo)
	}()

	router := chi.NewRouter()
	router.Use(servers.GzipMiddleware)
	router.Use(servers.UnGzipMiddleware)

	router.Post("/update/{type}/{name}/{value}", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerUpdateRaw))
	router.Post("/update/", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerUpdateJSON))
	router.Post("/updates/", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerUpdatesJSON))

	router.Get("/value/{type}/{name}", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerFuncOneRaw))
	router.Post("/value/", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerFuncOneJSON))
	router.Get("/", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerFuncAll))
	router.Get("/ping", servers.BindServerDataToHandler(mainCtx, &serverData, handlers.HandlerPingDB))

	servers.StartServer(mainCtx, config, router)

	log.Println(types.NewTimeError(fmt.Errorf("END")))
}

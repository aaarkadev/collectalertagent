package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

func initConfig() types.ServerConfig {

	config := types.ServerConfig{}

	defaultListenAddress := "127.0.0.1:8080"
	flag.StringVar(&config.ListenAddress, "a", defaultListenAddress, "address to listen on")

	defaultStoreInterval := 300 * time.Second
	flag.DurationVar(&config.StoreInterval, "i", defaultStoreInterval, "store interval")

	flag.BoolVar(&config.IsRestore, "r", false, "is restore DB")

	defaultStoreFile := "/tmp/devops-metrics-db.json"
	flag.StringVar(&config.StoreFileName, "f", defaultStoreFile, "store filepath")

	flag.Parse()

	envVal, envFound := os.LookupEnv("ADDRESS")
	if envFound {
		config.ListenAddress = envVal
	}
	envVal, envFound = os.LookupEnv("STORE_INTERVAL")
	if envFound {
		envDur, err := time.ParseDuration(envVal)
		if err == nil {
			config.StoreInterval = envDur
		}
	}
	envVal, envFound = os.LookupEnv("STORE_FILE")
	if envFound {
		config.StoreFileName = envVal
	}
	envVal, envFound = os.LookupEnv("RESTORE")
	if envFound {
		if envVal == "true" {
			config.IsRestore = true
		} else {
			config.IsRestore = false
		}
	}

	return config
}

func main() {

	config := initConfig()
	repo := storages.FileStorage{Config: config}
	repo.Init()
	defer func() {
		repo.Shutdown()
	}()

	serverData := servers.ServerHandlerData{}
	serverData.Repo = &repo

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

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	defer mainCtxCancel()

	servers.StartServer(mainCtx, config, router)

	fmt.Println("SERVER END")

}

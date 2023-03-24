package main

import (
	"log"
	"net/http"

	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

func initRepo(r repositories.Repo) {
	r.Init()
}

func initConfig() types.ServerConfig {

	config := types.ServerConfig{}

	config.ListenAddress = os.Getenv("ADDRESS")
	if len(config.ListenAddress) <= 0 {
		config.ListenAddress = "127.0.0.1:8080"
	}

	defaultStoreInterval := 300 * time.Second
	envVal, envErr := os.LookupEnv("STORE_INTERVAL")
	if !envErr {
		config.StoreInterval = defaultStoreInterval
	} else {
		envInt, err := strconv.Atoi(envVal)
		if err == nil {
			config.StoreInterval = time.Duration(envInt) * time.Second
		} else {
			config.StoreInterval = defaultStoreInterval
		}
	}

	defaultStoreFile := "/tmp/devops-metrics-db.json"
	envVal, envErr = os.LookupEnv("STORE_FILE")
	if !envErr {
		config.StoreFileName = defaultStoreFile
	} else {
		config.StoreFileName = envVal
	}

	envVal, envErr = os.LookupEnv("RESTORE")
	if !envErr {
		config.IsRestore = false
	} else {
		if envVal == "true" {
			config.IsRestore = true
		} else {
			config.IsRestore = false
		}
	}

	if len(config.StoreFileName) > 0 {
		fmode := os.O_RDWR | os.O_CREATE
		if !config.IsRestore {
			fmode = fmode | os.O_TRUNC
		}
		file, fileErr := os.OpenFile(config.StoreFileName, fmode, 0777)
		if fileErr == nil {
			config.StoreFile = file
		} else {
			config.StoreFileName = ""
		}
	}

	return config
}

func storeDBfunc(data string, config types.ServerConfig) {

	if len(config.StoreFileName) <= 0 {
		return
	}
	err := config.StoreFile.Truncate(0)
	if err != nil {
		return
	}
	_, err = config.StoreFile.Seek(0, 0)
	if err != nil {
		return
	}

	_, err = config.StoreFile.WriteString(data)
	if err != nil {
		return
	}
}

func storeDB(r repositories.Repo, config types.ServerConfig) {
	if config.StoreInterval == 0 {
		return
	}
	go func() {
		storeTicker := time.NewTicker(config.StoreInterval)
		defer storeTicker.Stop()
		for {
			select {
			case <-storeTicker.C:
				{
					txtM, err := json.Marshal(r.GetAll())
					if err == nil {
						storeDBfunc(string(txtM), config)
					}
				}
			}
		}
	}()
}

func loadDB(r repositories.Repo, config types.ServerConfig) {
	if !config.IsRestore {
		return
	}
	if len(config.StoreFileName) <= 0 {
		return
	}
	decoder := json.NewDecoder(config.StoreFile)

	oldMetrics := []types.Metrics{}

	if err := decoder.Decode(&oldMetrics); err != nil {
		return
	}

	for _, m := range oldMetrics {
		r.Set(m)
	}
}

func startServer(config types.ServerConfig, router http.Handler) *http.Server {
	server := &http.Server{Addr: config.ListenAddress, Handler: router}
	return server
}

func main() {

	r := storages.MemStorage{}
	initRepo(&r)
	config := initConfig()
	if len(config.StoreFileName) > 0 {
		defer config.StoreFile.Close()
	}
	loadDB(&r, config)
	storeDB(&r, config)
	router := chi.NewRouter()

	u := handlers.UpdateMetricsHandler{}
	u.Data = &r
	if config.StoreInterval == 0 {
		u.StoreFunc = func(data string) { storeDBfunc(data, config) }
	}

	router.Post("/update/{type}/{name}/{value}", u.HandlerRaw)
	router.Post("/update/", u.HandlerJson)

	g := handlers.GetMetricsHandler{}
	g.Data = &r
	router.Get("/value/{type}/{name}", g.HandlerFuncOneRaw)
	router.Post("/value/", g.HandlerFuncOneJson)
	router.Get("/", g.HandlerFuncAll)

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	defer mainCtxCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	server := startServer(config, router)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-sigChan

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(mainCtx, 10*time.Second)
	defer shutdownCtxCancel()
	defer func() {
		txtM, err := json.Marshal(r.GetAll())
		if err == nil {
			storeDBfunc(string(txtM), config)
		}
	}()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}

}

package main

import (
	"log"
	"net/http"

	"os"

	"github.com/aaarkadev/collectalertagent/internal/handlers"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/go-chi/chi/v5"
)

func initRepo(r repositories.Repo) {
	r.Init()
}

func main() {

	r := storages.MemStorage{}
	initRepo(&r)

	router := chi.NewRouter()

	u := handlers.UpdateMetricsHandler{}
	u.Data = &r

	router.Post("/update/{type}/{name}/{value}", u.HandlerRaw)
	router.Post("/update/", u.HandlerJson)

	g := handlers.GetMetricsHandler{}
	g.Data = &r
	router.Get("/value/{type}/{name}", g.HandlerFuncOneRaw)
	router.Post("/value/", g.HandlerFuncOneJson)
	router.Get("/", g.HandlerFuncAll)

	listenAddress := os.Getenv("ADDRESS")
	if len(listenAddress) <= 0 {
		listenAddress = "127.0.0.1:8080"
	}
	log.Fatal(http.ListenAndServe(listenAddress, router))
}

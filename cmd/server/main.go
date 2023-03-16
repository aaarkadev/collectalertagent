package main

import (
	//"fmt"
	"log"

	"net/http"

	"github.com/aaarkadev/collectalertagent/internal/handlers"
	. "github.com/aaarkadev/collectalertagent/internal/repositories"
	. "github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/go-chi/chi/v5"
)

func initRepo(r Repo) {
	r.Init()
}

func main() {

	r := MemStorage{}
	initRepo(&r)

	router := chi.NewRouter()

	u := handlers.UpdateMetricsHandler{}
	u.Data = &r
	router.Post("/update/{type}/{name}/{value}", u.HandlerFunc)

	g := handlers.GetMetricsHandler{}
	g.Data = &r
	router.Get("/value/{type}/{name}", g.HandlerFuncOne)
	router.Get("/", g.HandlerFuncAll)
	//log.Println("server start ")

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", router))
}

package main

import (
	//"fmt"
	"log"

	"net/http"

	"github.com/aaarkadev/collectalertagent/internal/handlers"
	. "github.com/aaarkadev/collectalertagent/internal/repositories"
	. "github.com/aaarkadev/collectalertagent/internal/storages"
)

func initRepo(r Repo) {
	r.Init()
	//fmt.Println("repo", r)
	/*
		x, _ := r.Get("BuckHashSys")
		fmt.Println("BuckHashSys", x)
		log.Print(".")*/
}

func main() {
	r := MemStorage{}
	initRepo(&r)

	mux := http.NewServeMux()
	h := handlers.UpdateMetricsHandlerStruct{RepoData: &r}
	mux.HandleFunc("/", h.Handler)

	//log.Println("server start http://127.0.0.1:8080")

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}

package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	. "github.com/aaarkadev/collectalertagent/internal/repositories"
	. "github.com/aaarkadev/collectalertagent/internal/types"
)

type UpdateMetricsHandlerStruct struct {
	RepoData Repo
}

func (hStruct UpdateMetricsHandlerStruct) Handler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	/*
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "wrong Content-Type!", http.StatusBadRequest)
			return
		}*/

	path := r.URL.EscapedPath()
	httpErr := http.StatusOK
	var pathParts []string
	if len(path) <= 0 || !utf8.ValidString(path) {
		httpErr = http.StatusNotFound
	} else {
		pathParts = strings.Split(strings.Trim(path, " /"), "/")
		if len(pathParts) != 4 || pathParts[0] != "update" {
			httpErr = http.StatusNotFound
		}
	}

	if httpErr == http.StatusOK && (pathParts[1] != "gauge" && pathParts[1] != "counter") {
		httpErr = http.StatusNotImplemented
	}

	intV := 0
	floatV := 0.0
	var parseErr error
	if httpErr == http.StatusOK && pathParts[1] == "counter" {
		if intV, parseErr = strconv.Atoi(pathParts[3]); parseErr != nil {
			httpErr = http.StatusBadRequest
		}
	}

	if httpErr == http.StatusOK && pathParts[1] == "gauge" {
		if floatV, parseErr = strconv.ParseFloat(pathParts[3], 64); parseErr != nil {
			httpErr = http.StatusBadRequest
		}
	}

	/*
		if !notFound {
			notFound = true
			allowedNames := []string{
				"Alloc",
				"BuckHashSys",
				"Frees",
				"GCCPUFraction",
				"GCSys",
				"HeapAlloc",
				"HeapIdle",
				"HeapInuse",
				"HeapObjects",
				"HeapReleased",
				"HeapSys",
				"LastGC",
				"LastGC",
				"MCacheInuse",
				"MCacheSys",
				"MSpanInuse",
				"MSpanSys",
				"Mallocs",
				"NextGC",
				"NumForcedGC",
				"NumGC",
				"OtherSys",
				"PauseTotalNs",
				"StackInuse",
				"StackSys",
				"Sys",
				"TotalAlloc",
				"PollCount",
				"RandomValue",
			}
			for _, v := range allowedNames {
				if v == pathParts[2] {
					httpErr = http.StatusNotFound
					break
				}
			}
		}
	*/

	if httpErr != http.StatusOK {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(httpErr)
		fmt.Fprintln(w, "Err")
		return
	}

	_, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if pathParts[1] == "gauge" {
		hStruct.RepoData.Set(Metric{Name: pathParts[2], Type: GaugeType, Source: OsSource, Val: float64(floatV)})
	} else {
		oldVal, oldValErr := hStruct.RepoData.Get(pathParts[2])
		if oldValErr != nil {
			hStruct.RepoData.Set(Metric{Name: pathParts[2], Type: CounterType, Source: IncrementSource, Val: int64(intV)})
		} else {
			hStruct.RepoData.Set(Metric{Name: pathParts[2], Type: CounterType, Source: IncrementSource, Val: (oldVal.Val.(int64) + int64(intV))})
		}
	}

	fmt.Println("RepoData", hStruct.RepoData)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

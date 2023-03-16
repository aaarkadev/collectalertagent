package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	. "github.com/aaarkadev/collectalertagent/internal/repositories"
	. "github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

type UpdateMetricsHandler struct {
	ServerHandlerData
}

func (hStruct UpdateMetricsHandler) HandlerFunc(w http.ResponseWriter, r *http.Request) {

	httpErr := http.StatusOK
	typeParam := chi.URLParam(r, "type")
	nameParam := chi.URLParam(r, "name")
	valueParam := chi.URLParam(r, "value")

	if typeParam != "gauge" && typeParam != "counter" {
		httpErr = http.StatusNotImplemented
	}

	intV := 0
	floatV := 0.0
	var parseErr error
	if httpErr == http.StatusOK && typeParam == "counter" {
		if intV, parseErr = strconv.Atoi(valueParam); parseErr != nil {
			httpErr = http.StatusBadRequest
		}
	}

	if httpErr == http.StatusOK && typeParam == "gauge" {
		if floatV, parseErr = strconv.ParseFloat(valueParam, 64); parseErr != nil {
			httpErr = http.StatusBadRequest
		}
	}

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

	repoData, ok := hStruct.Data.(Repo)
	if !ok {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if typeParam == "gauge" {
		repoData.Set(Metric{Name: nameParam, Type: GaugeType, Source: OsSource, Val: float64(floatV)})
	} else {
		oldVal, oldValErr := repoData.Get(nameParam)
		if oldValErr != nil {
			repoData.Set(Metric{Name: nameParam, Type: CounterType, Source: IncrementSource, Val: int64(intV)})
		} else {
			repoData.Set(Metric{Name: nameParam, Type: CounterType, Source: IncrementSource, Val: (oldVal.Val.(int64) + int64(intV))})
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

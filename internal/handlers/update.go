package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

type UpdateMetricsHandler struct {
	types.ServerHandlerData
}

func (hStruct UpdateMetricsHandler) HandlerJson(w http.ResponseWriter, r *http.Request) {

	bodyBytes, err := io.ReadAll(r.Body)
	bodyStr := strings.Trim(string(bodyBytes[:]), " /")
	if err != nil || len(bodyStr) <= 0 {
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}

	repoData, ok := hStruct.Data.(repositories.Repo)
	if !ok {
		http.Error(w, "handler data type assertion to Repo fail", http.StatusBadRequest)
		return
	}

	updateOneMetric := types.Metrics{}
	isUpdateOneMetric := false
	err = json.Unmarshal([]byte(bodyStr), &updateOneMetric)
	if err == nil {
		repoData.Set(updateOneMetric)
		isUpdateOneMetric = true
	}

	oldMetrics := repoData.GetAll()

	txtM := []byte{}
	if !isUpdateOneMetric {
		couterMetrics := map[string]int64{}
		for _, m := range oldMetrics {
			if types.DataType(m.MType) == types.CounterType {
				couterMetrics[m.ID] = *m.Delta
			}
		}
		err = json.Unmarshal([]byte(bodyStr), &oldMetrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for i, m := range oldMetrics {
			if types.DataType(m.MType) == types.CounterType {
				oldMetrics[i].Set(*m.Delta + couterMetrics[m.ID])
			}
			repoData.Set(oldMetrics[i])
		}
		txtM, err = json.Marshal(oldMetrics)
	} else {
		updateOneMetric, err = repoData.Get(updateOneMetric.ID)
		txtM, err = json.Marshal(updateOneMetric)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println("POST answer " + string(txtM))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(txtM))
}

func (hStruct UpdateMetricsHandler) HandlerRaw(w http.ResponseWriter, r *http.Request) {

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

	repoData, ok := hStruct.Data.(repositories.Repo)
	if !ok {
		http.Error(w, "handler data type assertion to Repo fail", http.StatusBadRequest)
		return
	}
	var newM *types.Metrics
	var newMerr error
	if typeParam == "gauge" {
		newM, newMerr = types.NewMetric(nameParam, types.DataType(typeParam), types.OsSource)
	} else {
		newM, newMerr = types.NewMetric(nameParam, types.DataType(typeParam), types.IncrementSource)
	}
	if newMerr != nil {
		panic("NewMetric error")
	}
	if typeParam == "gauge" {
		newM.Set(float64(floatV))
	} else {
		oldVal, oldValErr := repoData.Get(nameParam)
		if oldValErr != nil {
			oldVal = *newM
		}
		newM.Set((*oldVal.Delta + int64(intV)))
	}
	ok = repoData.Set(*newM)
	if !ok {
		panic("error repo set element")
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))

}

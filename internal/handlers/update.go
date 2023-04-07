package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

func HandlerUpdatesJSON(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {
	HandlerUpdateJSON(w, r, serverData)
}

func HandlerUpdateJSON(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

	bodyBytes, err := io.ReadAll(r.Body)
	bodyStr := strings.Trim(string(bodyBytes[:]), " /")

	if err != nil || len(bodyStr) <= 0 {
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}

	if serverData == nil || serverData.Repo == nil {
		http.Error(w, "Repo fail", http.StatusBadRequest)
		return
	}

	updateOneMetric := types.Metrics{}
	isUpdateOneMetric := false
	err = json.Unmarshal([]byte(bodyStr), &updateOneMetric)
	if err == nil {
		if !types.DataType(updateOneMetric.MType).IsValid() {
			http.Error(w, "DataType invalid", http.StatusBadRequest)
			return
		}

		if types.DataType(updateOneMetric.MType) == types.GaugeType && !updateOneMetric.IsValue() {
			http.Error(w, "empty value", http.StatusBadRequest)
			return
		}
		if types.DataType(updateOneMetric.MType) == types.CounterType && !updateOneMetric.IsDelta() {
			http.Error(w, "empty delta", http.StatusBadRequest)
			return
		}
		tmpHash := updateOneMetric
		tmpHash.GenHash(serverData.Config.HashKey)
		if len(updateOneMetric.Hash) > 0 && updateOneMetric.Hash != tmpHash.Hash {
			http.Error(w, "wrong hash", http.StatusBadRequest)
			return
		}
		serverData.Repo.Set(updateOneMetric)
		isUpdateOneMetric = true
	}

	var txtM []byte
	if !isUpdateOneMetric {
		newMetrics := []types.Metrics{}
		err = json.Unmarshal([]byte(bodyStr), &newMetrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, m := range newMetrics {
			tmpHash := m
			tmpHash.GenHash(serverData.Config.HashKey)
			if len(m.Hash) > 0 && m.Hash != tmpHash.Hash {
				fmt.Println("wrong hash")
				continue
			}
			serverData.Repo.Set(m)
		}
		hashedMetrics := []types.Metrics{}
		hashedMetrics = append(hashedMetrics, serverData.Repo.GetAll()...)
		for i, _ := range newMetrics {
			newMetrics[i].GenHash(serverData.Config.HashKey)
		}
		txtM, err = json.Marshal(newMetrics)
	} else {
		updateOneMetric, _ = serverData.Repo.Get(updateOneMetric.ID)
		updateOneMetric.GenHash(serverData.Config.HashKey)
		txtM, err = json.Marshal(updateOneMetric)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverData.Repo.FlushDB()

	w.Header().Set("Content-Type", "application/json")
	w.Write(txtM)
}

func HandlerUpdateRaw(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

	httpErr := http.StatusOK
	typeParam := chi.URLParam(r, "type")
	nameParam := chi.URLParam(r, "name")
	valueParam := chi.URLParam(r, "value")

	if types.DataType(typeParam) != types.GaugeType && types.DataType(typeParam) != types.CounterType {
		httpErr = http.StatusNotImplemented
	}

	intV := 0
	floatV := 0.0
	var parseErr error
	if httpErr == http.StatusOK && types.DataType(typeParam) == types.CounterType {
		if intV, parseErr = strconv.Atoi(valueParam); parseErr != nil {
			httpErr = http.StatusBadRequest
		}
	}

	if httpErr == http.StatusOK && types.DataType(typeParam) == types.GaugeType {
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

	if serverData == nil || serverData.Repo == nil {
		http.Error(w, "Repo fail", http.StatusBadRequest)
		return
	}

	var newM *types.Metrics
	var newMerr error
	if types.DataType(typeParam) == types.GaugeType {
		newM, newMerr = types.NewMetric(nameParam, types.DataType(typeParam), types.OsSource)
	} else {
		newM, newMerr = types.NewMetric(nameParam, types.DataType(typeParam), types.IncrementSource)
	}
	if newMerr != nil {
		panic("NewMetric error")
	}
	if types.DataType(typeParam) == types.GaugeType {
		newM.Set(float64(floatV))
	} else {
		newM.Set(int64(intV))
	}
	ok := serverData.Repo.Set(*newM)
	if !ok {
		panic("error repo set element")
	}

	serverData.Repo.FlushDB()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Ok"))

}

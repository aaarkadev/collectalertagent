package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
		errStr := "BadRequest. empty body"
		http.Error(w, errStr, http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): %v", errStr)))
		return
	}

	if serverData == nil || serverData.Repo == nil {
		repoErr := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): Repo fail"))
		http.Error(w, repoErr.Error(), http.StatusBadRequest)
		log.Fatalln(repoErr)
		return
	}

	updateOneMetric := types.Metrics{}
	isUpdateOneMetric := false
	err = json.Unmarshal([]byte(bodyStr), &updateOneMetric)
	if err == nil {
		if !types.DataType(updateOneMetric.MType).IsValid() {
			errStr := "DataType invalid"
			http.Error(w, errStr, http.StatusBadRequest)
			log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): %v", errStr)))
			return
		}

		if types.DataType(updateOneMetric.MType) == types.GaugeType && !updateOneMetric.IsValue() {
			errStr := "empty value"
			http.Error(w, errStr, http.StatusBadRequest)
			log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): %v", errStr)))
			return
		}
		if types.DataType(updateOneMetric.MType) == types.CounterType && !updateOneMetric.IsDelta() {
			errStr := "empty delta"
			http.Error(w, errStr, http.StatusBadRequest)
			log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): %v", errStr)))
			return
		}
		tmpHash := updateOneMetric
		tmpHash.GenHash(serverData.Config.HashKey)
		if len(updateOneMetric.Hash) > 0 && updateOneMetric.Hash != tmpHash.Hash {
			errStr := "wrong hash"
			http.Error(w, errStr, http.StatusBadRequest)
			log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): %v", errStr)))
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
			log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): fail: %w", err)))
			return
		}
		for _, m := range newMetrics {
			tmpHash := m
			tmpHash.GenHash(serverData.Config.HashKey)
			if len(m.Hash) > 0 && m.Hash != tmpHash.Hash {
				errStr := "wrong hash"
				log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): %v", errStr)))
				continue
			}
			err := serverData.Repo.Set(m)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): fail: %w", err)))
				return
			}
		}
		hashedMetrics := serverData.Repo.GetAll()
		for i, _ := range hashedMetrics {
			hashedMetrics[i].GenHash(serverData.Config.HashKey)
		}
		txtM, err = json.Marshal(hashedMetrics)
	} else {
		updateOneMetric, _ = serverData.Repo.Get(updateOneMetric.ID)
		updateOneMetric.GenHash(serverData.Config.HashKey)
		txtM, err = json.Marshal(updateOneMetric)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(): fail: %w", err)))
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
		errStr := "wrong type or err convert str to val"
		http.Error(w, errStr, httpErr)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerUpdateRaw(): fail: %v", errStr)))
		return
	}

	_, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	if serverData == nil || serverData.Repo == nil {
		repoErr := types.NewTimeError(fmt.Errorf("HandlerUpdateRaw(): Repo fail"))
		http.Error(w, repoErr.Error(), http.StatusBadRequest)
		log.Fatalln(repoErr)
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
		http.Error(w, newMerr.Error(), http.StatusBadRequest)
		log.Println(newMerr)
		return
	}
	if types.DataType(typeParam) == types.GaugeType {
		err = newM.Set(float64(floatV))
	} else {
		err = newM.Set(int64(intV))
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}
	err = serverData.Repo.Set(*newM)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	serverData.Repo.FlushDB()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Ok"))

}

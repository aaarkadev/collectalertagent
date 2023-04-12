package handlers

import (
	"context"
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

func HandlerUpdatesJSON(mainCtx context.Context, w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {
	_, err := getHandlerUpdateJSONResponse(mainCtx, w, r, serverData)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func HandlerUpdateJSON(mainCtx context.Context, w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {
	txtM, err := getHandlerUpdateJSONResponse(mainCtx, w, r, serverData)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(txtM))
}

func getHandlerUpdateJSONResponse(mainCtx context.Context, w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) (string, error) {

	bodyBytes, err := io.ReadAll(r.Body)
	bodyStr := strings.Trim(string(bodyBytes[:]), " /")

	if err != nil || len(bodyStr) <= 0 {
		e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(1): BadRequest. empty body"))
		http.Error(w, e.Error(), http.StatusBadRequest)
		log.Println(e)
		return "", e
	}

	if serverData == nil || serverData.Repo == nil {
		e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(2): Repo fail"))
		http.Error(w, e.Error(), http.StatusBadRequest)
		log.Fatalln(e)
		return "", e
	}

	updateOneMetric := types.Metrics{}
	isUpdateOneMetric := false
	err = json.Unmarshal([]byte(bodyStr), &updateOneMetric)
	if err == nil {
		if !types.DataType(updateOneMetric.MType).IsValid() {
			e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(3): DataType invalid"))
			http.Error(w, e.Error(), http.StatusBadRequest)
			log.Println(e)
			return "", e
		}

		if types.DataType(updateOneMetric.MType) == types.GaugeType && !updateOneMetric.IsValue() {
			e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(4): empty value"))
			http.Error(w, e.Error(), http.StatusBadRequest)
			log.Println(e)
			return "", e
		}
		if types.DataType(updateOneMetric.MType) == types.CounterType && !updateOneMetric.IsDelta() {
			e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(5): empty delta"))
			http.Error(w, e.Error(), http.StatusBadRequest)
			log.Println(e)
			return "", e
		}
		tmpHash := updateOneMetric
		tmpHash.GenHash(serverData.Config.HashKey)
		if len(updateOneMetric.Hash) > 0 && updateOneMetric.Hash != tmpHash.Hash {
			e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(6): wrong hash"))
			http.Error(w, e.Error(), http.StatusBadRequest)
			log.Println(e)
			return "", e
		}
		serverData.Repo.Set(updateOneMetric)
		isUpdateOneMetric = true
	}

	var txtM []byte
	if !isUpdateOneMetric {
		newMetrics := []types.Metrics{}
		err = json.Unmarshal([]byte(bodyStr), &newMetrics)
		if err != nil {
			e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(7): %w", err))
			http.Error(w, e.Error(), http.StatusBadRequest)
			log.Println(e)
			return "", e
		}

		for _, m := range newMetrics {

			tmpHash := m
			tmpHash.GenHash(serverData.Config.HashKey)

			if len(m.Hash) > 0 && m.Hash != tmpHash.Hash {
				e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(8): wrong hash %v", m.ID))
				log.Println(e)
				continue
			}
			err := serverData.Repo.Set(m)
			if err != nil {
				e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(9): %w", err))
				http.Error(w, e.Error(), http.StatusBadRequest)
				log.Println(e)
				return "", e
			}
		}
		hashedMetrics := serverData.Repo.GetAll()
		for i := range hashedMetrics {
			hashedMetrics[i].GenHash(serverData.Config.HashKey)
		}
		txtM, err = json.Marshal(hashedMetrics)
	} else {
		updateOneMetric, _ = serverData.Repo.Get(updateOneMetric.ID)
		updateOneMetric.GenHash(serverData.Config.HashKey)
		txtM, err = json.Marshal(updateOneMetric)
	}

	if err != nil {
		e := types.NewTimeError(fmt.Errorf("HandlerUpdateJSON(10): %w", err))
		http.Error(w, e.Error(), http.StatusBadRequest)
		log.Println(e)
		return "", e
	}

	serverData.Repo.FlushDB(mainCtx)

	return string(txtM), nil
}

func HandlerUpdateRaw(mainCtx context.Context, w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

	httpErr := http.StatusOK
	typeParam := chi.URLParam(r, "type")
	nameParam := chi.URLParam(r, "name")
	valueParam := chi.URLParam(r, "value")

	intV := 0
	floatV := 0.0
	var parseErr error
	if !types.DataType(typeParam).IsValid() {
		httpErr = http.StatusNotImplemented
	} else {
		if types.DataType(typeParam) == types.CounterType {
			if intV, parseErr = strconv.Atoi(valueParam); parseErr != nil {
				httpErr = http.StatusBadRequest
			}
		} else {
			if floatV, parseErr = strconv.ParseFloat(valueParam, 64); parseErr != nil {
				httpErr = http.StatusBadRequest
			}
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

	serverData.Repo.FlushDB(mainCtx)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Ok"))

}

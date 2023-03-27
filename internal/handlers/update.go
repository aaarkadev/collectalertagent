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

func HandlerUpdateJson(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

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
		serverData.Repo.Set(updateOneMetric)
		isUpdateOneMetric = true
	}

	txtM := []byte{}
	if !isUpdateOneMetric {
		newMetrics := []types.Metrics{}
		err = json.Unmarshal([]byte(bodyStr), &newMetrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, m := range newMetrics {
			serverData.Repo.Set(m)
		}
		newMetrics = serverData.Repo.GetAll()
		txtM, err = json.Marshal(newMetrics)
	} else {
		updateOneMetric, err = serverData.Repo.Get(updateOneMetric.ID)
		txtM, err = json.Marshal(updateOneMetric)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverData.Repo.FlushDB()

	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	w.Write([]byte(txtM))
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
	if httpErr == http.StatusOK && typeParam == "counter" {
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
	//w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))

}

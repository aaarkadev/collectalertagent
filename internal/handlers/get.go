package handlers

import (
	"fmt"
	"io"
	"net/http"

	"encoding/json"
	"strings"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

type GetMetricsHandler struct {
	types.ServerHandlerData
}

func (hStruct GetMetricsHandler) HandlerFuncAll(w http.ResponseWriter, r *http.Request) {
	body := `<!doctype html><html lang="ru">
			<body>
				<table width="50%%" border="1">
					%s
				</table>
			</body>
			</html>`
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

	metrics := repoData.GetAll()
	tableStr := []string{}
	for _, v := range metrics {
		tableStr = append(tableStr, "<tr><td>", v.ID, "</td><td>", v.Get(), "</td></tr>")
	}
	body = fmt.Sprintf(body, strings.Join(tableStr, "\r\n"))

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(body))
}

func (hStruct GetMetricsHandler) HandlerFuncOneJson(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "wrong Content-Type!", http.StatusBadRequest)
		return
	}

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

	metricVal := types.Metrics{}
	err = json.Unmarshal([]byte(bodyStr), &metricVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	oldMetrics := repoData.GetAll()
	isFound := false
	for _, m := range oldMetrics {
		if m.ID == metricVal.ID {
			metricVal = m
			isFound = true
			break
		}
	}
	if !isFound {
		http.Error(w, "Err", http.StatusNotFound)
		return
	}
	txtM, err := json.Marshal(metricVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(txtM))
}

func (hStruct GetMetricsHandler) HandlerFuncOneRaw(w http.ResponseWriter, r *http.Request) {

	httpErr := http.StatusOK
	typeParam := chi.URLParam(r, "type")
	nameParam := chi.URLParam(r, "name")

	if typeParam != "gauge" && typeParam != "counter" {
		httpErr = http.StatusNotImplemented
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

	oldVal, oldValErr := repoData.Get(nameParam)
	if oldValErr != nil {
		httpErr = http.StatusNotFound
		http.Error(w, "Err", httpErr)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(oldVal.Get()))
}

package handlers

import (
	"fmt"
	"io"
	"net/http"

	"encoding/json"
	"strings"

	"github.com/aaarkadev/collectalertagent/internal/servers"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
)

func HandlerFuncAll(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

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

	if serverData == nil || serverData.Repo == nil {
		http.Error(w, "Repo fail", http.StatusBadRequest)
		return
	}
	repoData := serverData.Repo

	metrics := repoData.GetAll()
	tableStr := []string{}
	for _, v := range metrics {
		tableStr = append(tableStr, "<tr><td>", v.ID, "</td><td>", v.Get(), "</td></tr>")
	}

	w.Header().Set("Content-Type", "text/html")

	io.WriteString(w, fmt.Sprintf(body, strings.Join(tableStr, "\r\n")))
}

func HandlerFuncOneJSON(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {
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

	if serverData == nil || serverData.Repo == nil {
		http.Error(w, "Repo fail", http.StatusBadRequest)
		return
	}
	repoData := serverData.Repo

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

	metricVal.GenHash(serverData.Config.HashKey)
	txtM, err := json.Marshal(metricVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(txtM)
}

func HandlerPingDB(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {
	if len(serverData.Config.DSN) < 1 {
		http.Error(w, "DSN empty or no connection to DB", http.StatusInternalServerError)
		return
	}
	pingErr := serverData.Repo.Ping()
	if pingErr != nil {
		http.Error(w, pingErr.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func HandlerFuncOneRaw(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

	httpErr := http.StatusOK
	typeParam := chi.URLParam(r, "type")
	nameParam := chi.URLParam(r, "name")

	if types.DataType(typeParam) != types.GaugeType && types.DataType(typeParam) != types.CounterType {
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

	if serverData == nil || serverData.Repo == nil {
		http.Error(w, "Repo fail", http.StatusBadRequest)
		return
	}
	repoData := serverData.Repo

	oldVal, oldValErr := repoData.Get(nameParam)
	if oldValErr != nil {
		httpErr = http.StatusNotFound
		http.Error(w, "Err", httpErr)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	w.Write([]byte(oldVal.Get()))
}

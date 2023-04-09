package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
		log.Println(err)
		return
	}

	if serverData == nil || serverData.Repo == nil {
		repoErr := types.NewTimeError(fmt.Errorf("HandlerFuncAll(): Repo fail"))
		http.Error(w, repoErr.Error(), http.StatusBadRequest)
		log.Fatalln(repoErr)
		return
	}
	repoData := serverData.Repo

	tableStr := []string{}
	for _, v := range repoData.GetAll() {
		tableStr = append(tableStr, "<tr><td>", v.ID, "</td><td>", v.Get(), "</td></tr>")
	}

	w.Header().Set("Content-Type", "text/html")

	io.WriteString(w, fmt.Sprintf(body, strings.Join(tableStr, "\r\n")))
}

func HandlerFuncOneJSON(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {

	if r.Header.Get("Content-Type") != "application/json" {
		errStr := "wrong Content-Type"
		http.Error(w, errStr, http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneJSON(): %v", errStr)))
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)

	bodyStr := strings.Trim(string(bodyBytes[:]), " /")
	if err != nil || len(bodyStr) <= 0 {
		errStr := "BadRequest. empty body"
		http.Error(w, errStr, http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneJSON(): %v", errStr)))
		return
	}

	if serverData == nil || serverData.Repo == nil {
		repoErr := types.NewTimeError(fmt.Errorf("HandlerFuncOneJSON(): Repo fail"))
		http.Error(w, repoErr.Error(), http.StatusBadRequest)
		log.Fatalln(repoErr)
		return
	}
	repoData := serverData.Repo

	metricVal := types.Metrics{}
	err = json.Unmarshal([]byte(bodyStr), &metricVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneJSON(): fail: %w", err)))
		return
	}
	metricVal, foundErr := repoData.Get(metricVal.ID)
	if foundErr != nil {
		http.Error(w, foundErr.Error(), http.StatusNotFound)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneJSON(): fail: %w", foundErr)))
		return
	}

	metricVal.GenHash(serverData.Config.HashKey)
	txtM, err := json.Marshal(metricVal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneJSON(): fail: %w", err)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(txtM)
}

func HandlerPingDB(w http.ResponseWriter, r *http.Request, serverData *servers.ServerHandlerData) {
	if serverData == nil || serverData.Repo == nil {
		repoErr := types.NewTimeError(fmt.Errorf("HandlerPingDB(): Repo fail"))
		http.Error(w, repoErr.Error(), http.StatusBadRequest)
		log.Fatalln(repoErr)
		return
	}
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

	if !types.DataType(typeParam).IsValid() {
		httpErr = http.StatusNotImplemented
	}
	if httpErr != http.StatusOK {
		errStr := "wrong type"
		http.Error(w, errStr, httpErr)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneRaw(): fail: %v", errStr)))
		return
	}

	_, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneRaw(): fail: %w", err)))
		return
	}

	if serverData == nil || serverData.Repo == nil {
		repoErr := types.NewTimeError(fmt.Errorf("HandlerFuncOneRaw(): Repo fail"))
		http.Error(w, repoErr.Error(), http.StatusBadRequest)
		log.Fatalln(repoErr)
		return
	}
	repoData := serverData.Repo

	oldVal, oldValErr := repoData.Get(nameParam)
	if oldValErr != nil {
		http.Error(w, oldValErr.Error(), http.StatusNotFound)
		log.Println(types.NewTimeError(fmt.Errorf("HandlerFuncOneRaw(): fail: %w", oldValErr)))
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	w.Write([]byte(oldVal.Get()))
}

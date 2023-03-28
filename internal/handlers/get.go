package handlers

import (
	"fmt"
	"io"
	"net/http"
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
		tableStr = append(tableStr, "<tr><td>", v.Name, "</td><td>", v.Get(), "</td></tr>")
	}
	body = fmt.Sprintf(body, strings.Join(tableStr, "\r\n"))

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(body))
}

func (hStruct GetMetricsHandler) HandlerFuncOne(w http.ResponseWriter, r *http.Request) {

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

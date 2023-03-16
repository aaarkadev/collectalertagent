package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context"

	"github.com/aaarkadev/collectalertagent/internal/handlers"
	. "github.com/aaarkadev/collectalertagent/internal/repositories"
	. "github.com/aaarkadev/collectalertagent/internal/storages"
	. "github.com/aaarkadev/collectalertagent/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestUserPostHandler(t *testing.T) {
	mem := MemStorage{}
	mem.Init()
	getHandler := handlers.UpdateMetricsHandler{}
	getHandler.Data = &mem
	t.Run("testUpdate", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/1234", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "counter")
		rctx.URLParams.Add("name", "PollCount")
		rctx.URLParams.Add("value", "1234")
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

		hFunc := http.HandlerFunc(getHandler.HandlerFunc)
		hFunc(w, request)

		result := w.Result()

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "text/plain", result.Header.Get("Content-Type"))

		bodyStr, err := ioutil.ReadAll(result.Body)
		require.NoError(t, err)
		err = result.Body.Close()
		require.NoError(t, err)

		assert.NotEmpty(t, string(bodyStr))
		metrica, metricaErr := mem.Get("PollCount")
		require.NoError(t, metricaErr)
		assert.Equal(t, "1234", metrica.Get())
	})
}

func TestUserGetHandler(t *testing.T) {
	mem := MemStorage{}
	mem.Init()
	getHandler := handlers.GetMetricsHandler{}
	getHandler.Data = &mem

	t.Run("testGetAll", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		hFunc := http.HandlerFunc(getHandler.HandlerFuncAll)
		hFunc(w, request)

		result := w.Result()

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "text/html", result.Header.Get("Content-Type"))

		bodyStr, err := ioutil.ReadAll(result.Body)
		require.NoError(t, err)
		err = result.Body.Close()
		require.NoError(t, err)

		assert.NotEmpty(t, string(bodyStr))
	})

	t.Run("testGetOne", func(t *testing.T) {
		mem.Set(Metric{Name: "PollCount", Type: CounterType, Source: IncrementSource, Val: int64(1234)})
		request := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
		w := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "counter")
		rctx.URLParams.Add("name", "PollCount")
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

		hFunc := http.HandlerFunc(getHandler.HandlerFuncOne)
		hFunc(w, request)

		result := w.Result()

		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "text/plain", result.Header.Get("Content-Type"))

		bodyStr, err := ioutil.ReadAll(result.Body)
		require.NoError(t, err)
		err = result.Body.Close()
		require.NoError(t, err)

		assert.NotEmpty(t, string(bodyStr))
		assert.Equal(t, "1234", strings.TrimSpace(string(bodyStr)))
	})

}

type MemStorageTestSuite struct {
	suite.Suite
	r Repo
}

func (suite *MemStorageTestSuite) SetupSuite() {
	m := MemStorage{}
	m.Init()
	suite.r = &m
}

func (suite *MemStorageTestSuite) TestSet() {

	ok := suite.r.Set(Metric{Name: "abc", Type: GaugeType, Source: OsSource, Val: float64(3.1416)})

	suite.Equal(true, ok)

	val, err := suite.r.Get("abc")
	require.NoError(suite.T(), err)
	suite.Equal(float64(3.1416), val.Val)
}

func TestMemStorageTestSuite(t *testing.T) {
	suite.Run(t, new(MemStorageTestSuite))
}

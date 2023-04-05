package agents

import (
	"bytes"
	"context"
	"encoding/json"

	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"

	"reflect"
	"runtime"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"golang.org/x/exp/constraints"
)

func numToFloat[T constraints.Integer | constraints.Float](a T) float64 {
	return float64(a)
}

func updateOne(m types.Metrics, statStructReflect reflect.Value) types.Metrics {
	switch m.Source {
	case types.IncrementSource:
		{
			m.Set((*m.Delta + int64(1)))
		}
	case types.RandSource:
		{
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			m.Set(r.Float64())
		}
	default:
		{
			structFieldVal := statStructReflect.FieldByName(m.ID)
			if structFieldVal.IsValid() {
				floatVal := float64(0.0)
				if structFieldVal.CanFloat() {
					floatVal = numToFloat(structFieldVal.Float())
				} else if structFieldVal.CanUint() {
					floatVal = numToFloat(structFieldVal.Uint())
				} else {
					floatVal = numToFloat(structFieldVal.Int())
				}
				m.Set(float64(floatVal))
			}
		}
	}

	return m
}

func UpdatelMetrics(rep repositories.Repo) bool {
	var osStats = runtime.MemStats{}
	runtime.ReadMemStats(&osStats)
	reflectVal := reflect.ValueOf(&osStats)
	if reflectVal.Kind() == reflect.Ptr {
		reflectVal = reflectVal.Elem()
	}
	if reflectVal.Kind() != reflect.Struct {
		return false
	}

	for _, mElem := range rep.GetAll() {
		mElem = updateOne(mElem, reflectVal)
		ok := rep.Set(mElem)
		if !ok {
			log.Fatal("error repo set element")
		}
	}
	return true
}

func InitAllMetrics(rep repositories.Repo) {
	rep.Init()
	var initVars = []struct {
		Name   string
		Type   types.DataType
		Source types.DataSource
	}{
		{Name: "Alloc", Type: types.GaugeType, Source: types.OsSource},
		{Name: "BuckHashSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Frees", Type: types.GaugeType, Source: types.OsSource},
		{Name: "GCCPUFraction", Type: types.GaugeType, Source: types.OsSource},
		{Name: "GCSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapAlloc", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapIdle", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapObjects", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapReleased", Type: types.GaugeType, Source: types.OsSource},
		{Name: "HeapSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "LastGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Lookups", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MCacheInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MCacheSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MSpanInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "MSpanSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Mallocs", Type: types.GaugeType, Source: types.OsSource},
		{Name: "NextGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "NumForcedGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "NumGC", Type: types.GaugeType, Source: types.OsSource},
		{Name: "OtherSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "PauseTotalNs", Type: types.GaugeType, Source: types.OsSource},
		{Name: "StackInuse", Type: types.GaugeType, Source: types.OsSource},
		{Name: "StackSys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "Sys", Type: types.GaugeType, Source: types.OsSource},
		{Name: "TotalAlloc", Type: types.GaugeType, Source: types.OsSource},
		{Name: "PollCount", Type: types.CounterType, Source: types.IncrementSource},
		{Name: "RandomValue", Type: types.GaugeType, Source: types.RandSource},
	}

	for _, v := range initVars {
		newM, err := types.NewMetric(v.Name, v.Type, v.Source)
		if err != nil {
			log.Fatal("NewMetric error")
		}
		ok := rep.Set(*newM)
		if !ok {
			log.Fatal("error repo set element")
		}
	}

}

func SendMetricsJSON(rep repositories.Repo, config configs.AgentConfig) {
	client := &http.Client{}
	client.Timeout = 10 * time.Second

	txtM, err := json.Marshal(rep.GetAll())
	if err != nil {
		panic(err)
	}
	url := fmt.Sprintf("http://%v/update/", config.SendAddress)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, rqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(txtM))
	if rqErr != nil {
		return
	}
	req.Header.Set("Content-Type", "Content-Type: application/json")

	response, doErr := client.Do(req)
	if doErr != nil {
		return
	}

	_, ioErr := io.Copy(io.Discard, response.Body)
	if ioErr != nil {
		return
	}
	response.Body.Close()
}

func sendMetricsRaw(rep repositories.Repo, config configs.AgentConfig) {
	client := &http.Client{}
	client.Timeout = 10 * time.Second

	for _, v := range rep.GetAll() {
		url := fmt.Sprintf("http://%v/update/%v/%v/%v", config.SendAddress, v.MType, v.ID, v.Get())

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		req, rqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		if rqErr != nil {
			continue
		}
		req.Header.Set("Content-Type", "Content-Type: text/plain")

		response, doErr := client.Do(req)
		if doErr != nil {
			continue
		}

		_, ioErr := io.Copy(io.Discard, response.Body)
		if ioErr != nil {
			continue
		}
		response.Body.Close()

	}
}

func StartAgent(rep repositories.Repo, config configs.AgentConfig) {
	pollTicker := time.NewTicker(config.PollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(config.ReportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			{
				go func() {
					UpdatelMetrics(rep)
				}()
			}
		case <-reportTicker.C:
			{
				go func() {
					SendMetricsJSON(rep, config)
				}()
			}
		}
	}
}

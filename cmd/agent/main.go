package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func getFloat(unk interface{}) (float64, error) {
	switch i := unk.(type) {
	case float64:
		return float64(i), nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int16:
		return float64(i), nil
	case int8:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint16:
		return float64(i), nil
	case uint8:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		f, err := strconv.ParseFloat(i, 64)
		if err != nil {
			return 0, err
		}
		return f, err
	default:
		return 0, fmt.Errorf("getFloat: unknown value is of incompatible type")
	}
}

func UpdateOne(m types.Metrics, statStructReflect reflect.Value) types.Metrics {
	switch m.Source {
	case types.IncrementSource:
		{
			m.Set((*m.Delta + int64(1)))
		}
	case types.RandSource:
		{
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			m.Set(float64(r.Float64()))
		}
	default:
		{
			structFieldVal := statStructReflect.FieldByName(m.ID)
			if structFieldVal.IsValid() {
				structFieldInterface := structFieldVal.Interface()
				v, err := getFloat(structFieldInterface)
				if err == nil {
					m.Set(float64(v))
				}
			}
		}
	}

	return m
}

func UpdatelMetrics(rep repositories.Repo) bool {
	var osStats = runtime.MemStats{}
	runtime.ReadMemStats(&osStats)
	rv := reflect.ValueOf(&osStats)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}

	for _, mElem := range rep.GetAll() {
		mElem = UpdateOne(mElem, rv)
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

var collectedMetric storages.MemStorage

func sendMetricsJson(rep repositories.Repo, config types.AgentConfig) {
	client := &http.Client{}
	client.Timeout = 10 * time.Second

	txtM, err := json.Marshal(rep.GetAll())

	if err != nil {
		panic(err)
	}
	url := fmt.Sprintf("http://%v/update/", config.SendAddress)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, rqErr := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(txtM))
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

func sendMetricsRaw(rep repositories.Repo, config types.AgentConfig) {
	client := &http.Client{}
	client.Timeout = 10 * time.Second

	for _, v := range rep.GetAll() {
		url := fmt.Sprintf("http://%v/update/%v/%v/%v", config.SendAddress, v.MType, v.ID, v.Get())

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		req, rqErr := http.NewRequestWithContext(ctx, "POST", url, nil)
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

func initConfig() types.AgentConfig {

	config := types.AgentConfig{}

	config.SendAddress = os.Getenv("ADDRESS")
	if len(config.SendAddress) <= 0 {
		config.SendAddress = "127.0.0.1:8080"
	}

	envVal, envErr := os.LookupEnv("REPORT_INTERVAL")
	if !envErr {
		config.ReportInterval = reportInterval
	} else {
		envInt, err := strconv.Atoi(envVal)
		if err == nil {
			config.ReportInterval = time.Duration(envInt) * time.Second
		} else {
			config.ReportInterval = reportInterval
		}
	}

	envVal, envErr = os.LookupEnv("POLL_INTERVAL")
	if !envErr {
		config.PollInterval = pollInterval
	} else {
		envInt, err := strconv.Atoi(envVal)
		if err == nil {
			config.PollInterval = time.Duration(envInt) * time.Second
		} else {
			config.PollInterval = pollInterval
		}
	}
	return config
}

func main() {
	collectedMetric = storages.MemStorage{}
	InitAllMetrics(&collectedMetric)
	config := initConfig()

	pollTicker := time.NewTicker(config.PollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(config.ReportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			{
				go func() {
					UpdatelMetrics(&collectedMetric)
				}()
			}
		case <-reportTicker.C:
			{
				go func() {
					sendMetricsJson(&collectedMetric, config)
				}()
			}
		}
	}

}

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"

	"time"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
	"golang.org/x/exp/constraints"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func numToFloat[T constraints.Integer | constraints.Float](a T) float64 {
	return float64(a)
}

func UpdateOne(m types.Metric, statStructReflect reflect.Value) types.Metric {
	switch m.Source {
	case types.IncrementSource:
		v, ok := m.Val.(int64)
		if ok {
			v = v + 1
			m.Val = int64(v)
		}
	case types.RandSource:
		{
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			m.Val = float64(r.Float64())
		}
	default:
		{
			structFieldVal := statStructReflect.FieldByName(m.Name)
			if structFieldVal.IsValid() {
				v := float64(0.0)
				if structFieldVal.CanFloat() {
					v = numToFloat(structFieldVal.Float())
				} else if structFieldVal.CanUint() {
					v = numToFloat(structFieldVal.Uint())
				} else {
					v = numToFloat(structFieldVal.Int())
				}
				//fieldType := structFieldVal.Type()
				//structFieldInterface := structFieldVal.Interface()
				m.Val = float64(v)

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
		{Name: "LastGC", Type: types.GaugeType, Source: types.OsSource},
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
		ok := rep.Set(types.Metric{Name: v.Name, Type: v.Type, Source: v.Source})
		if !ok {
			log.Fatal("error repo set element")
		}
	}
	//fmt.Println("init ", collectedMetric)
}

var collectedMetric storages.MemStorage

func sendMetrics(rep repositories.Repo) {
	client := &http.Client{}
	client.Timeout = 10 * time.Second

	for _, v := range rep.GetAll() {
		url := fmt.Sprintf("http://127.0.0.1:8080/update/%v/%v/%v", v.Type, v.Name, v.Val)
		//fmt.Println(url)
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
func main() {
	collectedMetric = storages.MemStorage{}
	InitAllMetrics(&collectedMetric)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(reportInterval)
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
					sendMetrics(&collectedMetric)
				}()
			}
		}
	}

}

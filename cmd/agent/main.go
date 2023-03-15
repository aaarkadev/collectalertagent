package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"

	. "github.com/aaarkadev/collectalertagent/internal/repositories"
	. "github.com/aaarkadev/collectalertagent/internal/storages"
	. "github.com/aaarkadev/collectalertagent/internal/types"
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

func UpdateOne(m Metric, statStructReflect reflect.Value) Metric {
	switch m.Source {
	case IncrementSource:
		v, ok := m.Val.(int64)
		if ok {
			v = v + 1
			m.Val = int64(v)
		}
	case RandSource:
		{
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			m.Val = float64(r.Float64())
		}
	default:
		{
			structFieldVal := statStructReflect.FieldByName(m.Name)
			if structFieldVal.IsValid() {
				structFieldInterface := structFieldVal.Interface()
				v, err := getFloat(structFieldInterface)
				if err == nil {
					m.Val = float64(v)
				}
			}
		}
	}

	return m
}

func UpdatelMetrics(rep Repo) bool {
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
			//fmt.Println("error upd [$v]", mElem.Name)
		}
	}
	return true
}

func InitAllMetrics(rep Repo) {
	rep.Init()
	var initVars = []struct {
		Name   string
		Type   DataType
		Source DataSource
	}{

		{Name: "Alloc", Type: GaugeType, Source: OsSource},
		{Name: "BuckHashSys", Type: GaugeType, Source: OsSource},
		{Name: "Frees", Type: GaugeType, Source: OsSource},
		{Name: "GCCPUFraction", Type: GaugeType, Source: OsSource},
		{Name: "GCSys", Type: GaugeType, Source: OsSource},
		{Name: "HeapAlloc", Type: GaugeType, Source: OsSource},
		{Name: "HeapIdle", Type: GaugeType, Source: OsSource},
		{Name: "HeapInuse", Type: GaugeType, Source: OsSource},
		{Name: "HeapObjects", Type: GaugeType, Source: OsSource},
		{Name: "HeapReleased", Type: GaugeType, Source: OsSource},
		{Name: "HeapSys", Type: GaugeType, Source: OsSource},
		{Name: "LastGC", Type: GaugeType, Source: OsSource},
		{Name: "LastGC", Type: GaugeType, Source: OsSource},
		{Name: "MCacheInuse", Type: GaugeType, Source: OsSource},
		{Name: "MCacheSys", Type: GaugeType, Source: OsSource},
		{Name: "MSpanInuse", Type: GaugeType, Source: OsSource},
		{Name: "MSpanSys", Type: GaugeType, Source: OsSource},
		{Name: "Mallocs", Type: GaugeType, Source: OsSource},
		{Name: "NextGC", Type: GaugeType, Source: OsSource},
		{Name: "NumForcedGC", Type: GaugeType, Source: OsSource},
		{Name: "NumGC", Type: GaugeType, Source: OsSource},
		{Name: "OtherSys", Type: GaugeType, Source: OsSource},
		{Name: "PauseTotalNs", Type: GaugeType, Source: OsSource},
		{Name: "StackInuse", Type: GaugeType, Source: OsSource},
		{Name: "StackSys", Type: GaugeType, Source: OsSource},
		{Name: "Sys", Type: GaugeType, Source: OsSource},
		{Name: "TotalAlloc", Type: GaugeType, Source: OsSource},
		{Name: "PollCount", Type: CounterType, Source: IncrementSource},
		{Name: "RandomValue", Type: GaugeType, Source: RandSource},
	}

	for _, v := range initVars {
		ok := rep.Set(Metric{Name: v.Name, Type: v.Type, Source: v.Source})
		if !ok {
			//fmt.Println("error init [$v]", v.Name)
		}
	}
	//fmt.Println("init ", collectedMetric)
}

var collectedMetric MemStorage

func sendMetrics(rep Repo) {
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
	collectedMetric = MemStorage{}
	InitAllMetrics(&collectedMetric)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case _ = <-pollTicker.C:
			{
				go func() {
					UpdatelMetrics(&collectedMetric)
				}()
			}
		case _ = <-reportTicker.C:
			{
				go func() {
					sendMetrics(&collectedMetric)
				}()
			}
		}
	}

}

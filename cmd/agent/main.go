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
)

type DataType string
type DataSource uint8

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

const (
	gaugeType   DataType = "gauge"
	counterType DataType = "counter"
)

func (s DataType) IsValid() bool {
	switch s {
	case gaugeType, counterType:
		return true
	default:
		return false
	}
}

func (s DataType) String() string {
	return string(s)
}

const (
	osSource DataSource = iota
	incrementSource
	randSource
)

func (s DataSource) IsValid() bool {
	switch s {
	case osSource, incrementSource, randSource:
		return true
	default:
		return false
	}
}

type Metric struct {
	Name   string
	Type   DataType
	Source DataSource
	Val    interface{}
}

func (m *Metric) Init() {
	switch m.Type {
	case counterType:
		m.Val = int64(0)
	default:
		m.Val = float64(0.0)
	}
}

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

func (m *Metric) Update(statStructReflect reflect.Value) {
	switch m.Source {
	case incrementSource:
		v, ok := m.Val.(int64)
		if ok {
			v = v + 1
			m.Val = int64(v)
		}
	case randSource:
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

}

func UpdatelMetrics() bool {
	var osStats = runtime.MemStats{}
	runtime.ReadMemStats(&osStats)
	rv := reflect.ValueOf(&osStats)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	metricsArray := GetMetrics()
	for i, _ := range *metricsArray {
		(*metricsArray)[i].Update(rv)
	}
	return true
}

func NewMetric(name string, typ DataType, source DataSource) (*Metric, error) {
	if !typ.IsValid() {
		return &Metric{}, fmt.Errorf("DataType[%v]: invalid", typ)
	}
	if !source.IsValid() {
		return &Metric{}, fmt.Errorf("DataSource[%v]: invalid", source)
	}

	m := &Metric{
		Name:   name,
		Type:   typ,
		Source: source,
		//Val:    val,
	}
	m.Init()
	//m.Update()
	return m, nil
}

func InitAllMetrics() {
	var initVars = []struct {
		Name   string
		Type   DataType
		Source DataSource
	}{

		{Name: "Alloc", Type: gaugeType, Source: osSource},
		{Name: "BuckHashSys", Type: gaugeType, Source: osSource},
		{Name: "Frees", Type: gaugeType, Source: osSource},
		{Name: "GCCPUFraction", Type: gaugeType, Source: osSource},
		{Name: "GCSys", Type: gaugeType, Source: osSource},
		{Name: "HeapAlloc", Type: gaugeType, Source: osSource},
		{Name: "HeapIdle", Type: gaugeType, Source: osSource},
		{Name: "HeapInuse", Type: gaugeType, Source: osSource},
		{Name: "HeapObjects", Type: gaugeType, Source: osSource},
		{Name: "HeapReleased", Type: gaugeType, Source: osSource},
		{Name: "HeapSys", Type: gaugeType, Source: osSource},
		{Name: "LastGC", Type: gaugeType, Source: osSource},
		{Name: "LastGC", Type: gaugeType, Source: osSource},
		{Name: "MCacheInuse", Type: gaugeType, Source: osSource},
		{Name: "MCacheSys", Type: gaugeType, Source: osSource},
		{Name: "MSpanInuse", Type: gaugeType, Source: osSource},
		{Name: "MSpanSys", Type: gaugeType, Source: osSource},
		{Name: "Mallocs", Type: gaugeType, Source: osSource},
		{Name: "NextGC", Type: gaugeType, Source: osSource},
		{Name: "NumForcedGC", Type: gaugeType, Source: osSource},
		{Name: "NumGC", Type: gaugeType, Source: osSource},
		{Name: "OtherSys", Type: gaugeType, Source: osSource},
		{Name: "PauseTotalNs", Type: gaugeType, Source: osSource},
		{Name: "StackInuse", Type: gaugeType, Source: osSource},
		{Name: "StackSys", Type: gaugeType, Source: osSource},
		{Name: "Sys", Type: gaugeType, Source: osSource},
		{Name: "TotalAlloc", Type: gaugeType, Source: osSource},
		{Name: "PollCount", Type: counterType, Source: incrementSource},
		{Name: "RandomValue", Type: gaugeType, Source: randSource},
	}
	metricsArray := GetMetrics()
	for _, v := range initVars {
		NewMetricElement, err := NewMetric(v.Name, v.Type, v.Source)
		if err != nil {
			fmt.Println(err)
		}
		*metricsArray = append(*metricsArray, *NewMetricElement)
	}
}

var collectedMetric = []Metric{}

func GetMetrics() *[]Metric {
	return &collectedMetric
}
func sendMetrics() {
	client := &http.Client{}
	client.Timeout = 10 * time.Second
	metricsArray := GetMetrics()
	for _, v := range *metricsArray {
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
	InitAllMetrics()
	//fmt.Println(*GetMetrics())

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()
	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case _ = <-pollTicker.C:
			{
				UpdatelMetrics()
			}
		case _ = <-reportTicker.C:
			{
				go func() {
					sendMetrics()
				}()
			}
		}
	}

}

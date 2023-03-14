package main

import (
	//"io"
	"fmt"
	"reflect"
	"runtime"
	"time"
	//"math/rand"
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
func (m *Metric) Update() {
	switch m.Source {
	case incrementSource:
		v, ok := m.Val.(int64)
		if ok {
			v = v + 1
			m.Val = v
		}
	case randSource:
		v, ok := m.Val.(float64)
		if ok {
			v = (1.0 + v) * 1.3
			m.Val = v
		}
	default:
		{
			//runtime.ReadMemStats(&osStats)
			rv := reflect.ValueOf(&osStats).Elem()
			osVal := rv.FieldByName(m.Name)
			m.Val = float64(osVal.Interface().(uint64))
		}
	}

}

func UpdatelMetrics() {
	runtime.ReadMemStats(&osStats)
	for i, _ := range collectedMetrics {
		collectedMetrics[i].Update()
	}
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

var collectedMetrics = []Metric{}
var osStats = runtime.MemStats{}

func InitAllMetrics() {
	var initVars = []struct {
		Name   string
		Type   DataType
		Source DataSource
	}{
		{Name: "HeapAlloc", Type: gaugeType, Source: osSource},
		{Name: "PollCount", Type: counterType, Source: incrementSource},
		{Name: "RandomValue", Type: gaugeType, Source: randSource},
	}

	for _, v := range initVars {
		NewMetricElement, err := NewMetric(v.Name, v.Type, v.Source)
		if err != nil {
			fmt.Println(err)
		}
		collectedMetrics = append(collectedMetrics, *NewMetricElement)
	}
}
func GetMetrics() []Metric {
	return collectedMetrics
}
func main() {
	InitAllMetrics()
	UpdatelMetrics()
	UpdatelMetrics()
	fmt.Println(GetMetrics())
}

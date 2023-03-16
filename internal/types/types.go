package types

import "fmt"

type DataType string
type DataSource uint8
type Metric struct {
	Name   string
	Type   DataType
	Source DataSource
	Val    interface{}
}

type ServerHandlerData struct {
	Data interface{}
}

const (
	GaugeType   DataType = "gauge"
	CounterType DataType = "counter"
)

const (
	OsSource DataSource = iota
	IncrementSource
	RandSource
)

func (s DataType) IsValid() bool {
	switch s {
	case GaugeType, CounterType:
		return true
	default:
		return false
	}
}

func (s DataType) String() string {
	return string(s)
}

func (s DataSource) IsValid() bool {
	switch s {
	case OsSource, IncrementSource, RandSource:
		return true
	default:
		return false
	}
}

func (m *Metric) Init() {
	switch m.Type {
	case CounterType:
		m.Val = int64(0)
	default:
		m.Val = float64(0.0)
	}
}

func (m *Metric) Get() string {
	s := ""
	switch m.Type {
	case CounterType:
		{

			v, ok := m.Val.(int64)
			if ok {
				s = fmt.Sprintf("%d", v)
			}
		}
	default:
		{
			v, ok := m.Val.(float64)
			if ok {
				s = fmt.Sprintf("%f", v)
			}
		}
	}

	return s
}

func (m *Metric) Set(val interface{}) bool {

	switch m.Type {
	case CounterType:
		{
			v, ok := val.(int64)
			if ok {
				m.Val = int64(v)
			} else {
				return false
			}
		}
	default:
		{
			v, ok := val.(float64)
			if ok {
				m.Val = float64(v)
			} else {
				return false
			}
		}
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
	return m, nil
}

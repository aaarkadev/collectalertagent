package types

import (
	"fmt"
	"time"
)

type DataType string
type DataSource uint8

type Metrics struct {
	ID     string     `json:"id"`
	MType  string     `json:"type"`
	Delta  *int64     `json:"delta,omitempty"`
	Value  *float64   `json:"value,omitempty"`
	Source DataSource `json:"-"`
}

type ServerConfig struct {
	ListenAddress string
	StoreInterval time.Duration
	StoreFileName string
	IsRestore     bool
}

type AgentConfig struct {
	SendAddress    string
	ReportInterval time.Duration
	PollInterval   time.Duration
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

func (m *Metrics) Init() {
	switch DataType(m.MType) {
	case CounterType:
		m.Delta = new(int64)
	default:
		m.Value = new(float64)
	}
}

func (m *Metrics) Get() string {
	s := ""
	switch DataType(m.MType) {
	case CounterType:
		{
			s = fmt.Sprintf("%d", *m.Delta)
		}
	default:
		{
			s = fmt.Sprintf("%.3f", *m.Value)
		}
	}

	return s
}

func (m *Metrics) SetMetric(newM Metrics) bool {
	if m.MType != newM.MType {
		return false
	}
	switch DataType(m.MType) {
	case CounterType:
		{
			*m.Delta += *newM.Delta
		}
	default:
		{
			m.Value = newM.Value
		}
	}
	return true
}

func (m *Metrics) Set(val interface{}) bool {

	switch DataType(m.MType) {
	case CounterType:
		{
			v, ok := val.(int64)
			if ok {
				*m.Delta = v
			} else {
				return false
			}
		}
	default:
		{
			v, ok := val.(float64)
			if ok {
				*m.Value = v
			} else {
				return false
			}
		}
	}
	return true
}

func NewMetric(name string, typ DataType, source DataSource) (*Metrics, error) {
	if !typ.IsValid() {
		return &Metrics{}, fmt.Errorf("DataType[%v]: invalid", typ)
	}
	if !source.IsValid() {
		return &Metrics{}, fmt.Errorf("DataSource[%v]: invalid", source)
	}

	m := &Metrics{
		ID:     name,
		MType:  string(typ),
		Source: source,
	}
	m.Init()
	return m, nil
}

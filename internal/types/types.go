package types

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"time"
	//"strings"
)

type DataType string
type DataSource uint8

type Metrics struct {
	ID     string     `json:"id" db:"ID"`
	MType  string     `json:"type" db:"MType"`
	Delta  *int64     `json:"delta,omitempty" db:"Delta,omitempty"`
	Value  *float64   `json:"value,omitempty" db:"Value,omitempty"`
	Hash   string     `json:"hash" db:"Hash"`
	Source DataSource `json:"-" db:"-"`
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

type CtxValues string

var MainCtxCancelFunc = CtxValues("mainCtxCancel")

type TimeError struct {
	Time time.Time
	Err  error
}

func (te *TimeError) Error() string {
	return fmt.Sprintf("%v %v", te.Time.Format(`2006/01/02 15:04:05`), te.Err)
}

func NewTimeError(err error) error {
	return &TimeError{
		Time: time.Now(),
		Err:  err,
	}
}

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

func (m *Metrics) GenHash(key []byte) {

	if len(key) < 1 {
		return
	}

	var strForHash string
	switch m.MType {
	case "counter":
		{
			strForHash = fmt.Sprintf("%s:%s:%d", m.ID, m.MType, m.GetDelta())
		}
	case "gauge":
		{
			strForHash = fmt.Sprintf("%s:%s:%f", m.ID, m.MType, m.GetValue())
		}
	}

	h := hmac.New(sha256.New, key)
	_, err := h.Write([]byte(strForHash))
	if err != nil {
		return
	}
	m.Hash = fmt.Sprintf("%x", h.Sum(nil))
}

func (m *Metrics) Get() string {
	s := ""
	switch DataType(m.MType) {
	case CounterType:
		{
			s = fmt.Sprintf("%d", m.GetDelta())
		}
	default:
		{
			s = fmt.Sprintf("%.3f", m.GetValue())
		}
	}

	return s
}

func (m *Metrics) GetDelta() int64 {
	if !m.IsDelta() {
		return int64(0)
	}
	return *m.Delta
}
func (m *Metrics) GetValue() float64 {
	if !m.IsValue() {
		return float64(0.0)
	}
	return *m.Value
}

func (m *Metrics) IsDelta() bool {
	return m.Delta != nil
}
func (m *Metrics) IsValue() bool {
	return m.Value != nil
}

func (m *Metrics) GetMetric() Metrics {
	newElem := Metrics{}
	newElem.ID = m.ID
	newElem.MType = m.MType
	if DataType(m.MType) == CounterType {
		delta := m.GetDelta()
		newElem.Delta = &delta
	} else {
		value := m.GetValue()
		newElem.Value = &value
	}

	newElem.Hash = m.Hash
	newElem.Source = m.Source
	return newElem
}

func (m *Metrics) SetMetric(newM Metrics) error {
	if m.MType != newM.MType {
		return NewTimeError(fmt.Errorf("Metric.SetMetric(): fail: type[%v]!=[%v]", m.MType, newM.MType))
	}
	if !newM.IsDelta() && !newM.IsValue() {
		return NewTimeError(fmt.Errorf("Metric.SetMetric(): fail: empty Delta and Value"))
	}
	switch DataType(m.MType) {
	case CounterType:
		{
			err := m.Set((m.GetDelta() + newM.GetDelta()))
			if err != nil {
				return NewTimeError(fmt.Errorf("Metric.SetMetric(): fail: %w", err))
			}
		}
	default:
		{
			err := m.Set(newM.GetValue())
			if err != nil {
				return NewTimeError(fmt.Errorf("Metric.SetMetric(): fail: %w", err))
			}
		}
	}
	//m.Hash = newM.Hash
	return nil
}

func (m *Metrics) Set(val interface{}) error {

	switch DataType(m.MType) {
	case CounterType:
		{
			v, ok := val.(int64)
			if ok {
				*m.Delta = v
			} else {
				return NewTimeError(fmt.Errorf("Metric.Set(%v): fail: type[%v]", val, m.MType))
			}
		}
	default:
		{
			v, ok := val.(float64)
			if ok {
				*m.Value = v
			} else {
				return NewTimeError(fmt.Errorf("Metric.Set(%v): fail: type[%v]", val, m.MType))
			}
		}
	}
	return nil
}

func NewMetric(name string, typ DataType, source DataSource) (*Metrics, error) {
	if !typ.IsValid() {
		return &Metrics{}, NewTimeError(fmt.Errorf("DataType[%v]: invalid", typ))

	}
	if !source.IsValid() {
		return &Metrics{}, NewTimeError(fmt.Errorf("DataSource[%v]: invalid", source))
	}

	m := &Metrics{
		ID:     name,
		MType:  string(typ),
		Source: source,
	}
	m.Init()
	return m, nil
}

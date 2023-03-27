package storages

import (
	"fmt"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type MemStorage struct {
	metrics []types.Metrics
}

var _ repositories.Repo = (*MemStorage)(nil)

func (m *MemStorage) Init() bool {
	m.metrics = make([]types.Metrics, 0, 0)
	return true
}

func (m *MemStorage) GetAll() []types.Metrics {
	return m.metrics
}

func (m *MemStorage) Get(k string) (types.Metrics, error) {
	for _, v := range m.metrics {
		if v.ID == k {
			return v, nil
		}
	}
	return types.Metrics{}, fmt.Errorf("k[%v]: not found in storage", k)
}

func (m *MemStorage) Set(mset types.Metrics) bool {
	if mset.Delta == nil && mset.Value == nil {
		return false
	}
	_, err := m.Get(mset.ID)

	if err != nil {
		newMetricElement, newErr := types.NewMetric(mset.ID, types.DataType(mset.MType), mset.Source)
		if newErr == nil {
			newMetricElement.SetMetric(mset)
			m.metrics = append(m.metrics, *newMetricElement)
		}
	} else {
		for i, v := range m.metrics {
			if v.ID == mset.ID {
				m.metrics[i].SetMetric(mset)
				break
			}
		}
	}

	return true
}

func (m *MemStorage) FlushDB() {
}

func (m *MemStorage) Shutdown() {

}

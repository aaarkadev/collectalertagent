package storages

import (
	"fmt"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type MemStorage struct {
	metrics []types.Metric
}

var _ repositories.Repo = (*MemStorage)(nil)

func (m *MemStorage) Init() bool {
	return true
}

func (m *MemStorage) GetAll() []types.Metric {
	return m.metrics
}

func (m *MemStorage) Get(k string) (types.Metric, error) {
	for _, v := range m.metrics {
		if v.Name == k {
			return v, nil
		}
	}
	return types.Metric{}, fmt.Errorf("k[%v]: not found in storage", k)
}

func (m *MemStorage) Set(mset types.Metric) bool {

	_, err := m.Get(mset.Name)

	if err != nil {
		newMetricElement, newErr := types.NewMetric(mset.Name, mset.Type, mset.Source)
		if newErr == nil {
			newMetricElement.Set(mset.Val)
			m.metrics = append(m.metrics, *newMetricElement)
		}
	} else {
		for i, v := range m.metrics {
			if v.Name == mset.Name {
				m.metrics[i].Set(mset.Val)
				break
			}
		}
	}

	return true
}

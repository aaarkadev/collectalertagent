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

func (repo *MemStorage) Init() bool {
	repo.metrics = make([]types.Metrics, 0)
	return true
}

func (repo *MemStorage) GetAll() []types.Metrics {
	return repo.metrics
}

func (repo *MemStorage) Get(k string) (types.Metrics, error) {
	for _, v := range repo.metrics {
		if v.ID == k {
			return v, nil
		}
	}
	return types.Metrics{}, fmt.Errorf("k[%v]: not found in storage", k)
}

func (repo *MemStorage) Set(mset types.Metrics) bool {
	if !mset.IsDelta() && !mset.IsValue() {
		return false
	}
	_, err := repo.Get(mset.ID)

	if err != nil {
		newMetricElement, newErr := types.NewMetric(mset.ID, types.DataType(mset.MType), mset.Source)
		if newErr == nil {
			newMetricElement.SetMetric(mset)
			repo.metrics = append(repo.metrics, *newMetricElement)
		}
	} else {
		for i, v := range repo.metrics {
			if v.ID == mset.ID {
				repo.metrics[i].SetMetric(mset)
				break
			}
		}
	}

	return true
}

func (repo *MemStorage) FlushDB() {
}

func (repo *MemStorage) Shutdown() {

}

func (repo *MemStorage) Ping() error {
	return nil
}

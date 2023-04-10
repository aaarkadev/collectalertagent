package storages

import (
	"fmt"
	"sync"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type MemStorage struct {
	metrics []types.Metrics
	mu      sync.RWMutex
}

var _ repositories.Repo = (*MemStorage)(nil)

func (repo *MemStorage) Init() bool {
	repo.metrics = make([]types.Metrics, 0)
	return true
}

func (repo *MemStorage) GetAll() []types.Metrics {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	copyValsMetrics := []types.Metrics{}
	for _, m := range repo.metrics {
		copyValsMetrics = append(copyValsMetrics, m.GetMetric())
	}
	return copyValsMetrics
}

func (repo *MemStorage) Get(k string) (types.Metrics, error) {
	allMetrics := repo.GetAll()
	for _, v := range allMetrics {
		if v.ID == k {
			return v, nil
		}
	}
	return types.Metrics{}, fmt.Errorf("k[%v]: not found in storage", k)
}

func (repo *MemStorage) Set(mset types.Metrics) error {

	_, err := repo.Get(mset.ID)

	allMetrics := repo.GetAll()

	if err != nil {
		newMetricElement, errNew := types.NewMetric(mset.ID, types.DataType(mset.MType), mset.Source)
		err = errNew
		if err == nil {
			err = newMetricElement.SetMetric(mset)
			if err == nil {
				allMetrics = append(allMetrics, *newMetricElement)
			}
		}
		if err != nil {
			err = types.NewTimeError(fmt.Errorf("MemStorage.Set(): fail: %w", err))
		}
	} else {
		for i, v := range allMetrics {
			if v.ID == mset.ID {
				err = allMetrics[i].SetMetric(mset)
				if err != nil {
					err = types.NewTimeError(fmt.Errorf("MemStorage.Set(): fail: %w", err))
				}
				break
			}
		}
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.metrics = allMetrics
	return err
}

func (repo *MemStorage) FlushDB() {
}

func (repo *MemStorage) Shutdown() {

}

func (repo *MemStorage) Ping() error {
	return nil
}

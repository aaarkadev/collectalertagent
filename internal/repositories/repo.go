package repositories

import (
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type Repo interface {
	Set(v types.Metric) bool
	Get(k string) (types.Metric, error)
	GetAll() []types.Metric
	Init() bool
}

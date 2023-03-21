package repositories

import (
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type Repo interface {
	Set(v types.Metrics) bool
	Get(k string) (types.Metrics, error)
	GetAll() []types.Metrics
	Init() bool
}

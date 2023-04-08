package repositories

import (
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type Repo interface {
	Set(v types.Metrics) error
	Get(k string) (types.Metrics, error)
	GetAll() []types.Metrics
	Init() bool
	Shutdown()
	FlushDB()
	Ping() error
}

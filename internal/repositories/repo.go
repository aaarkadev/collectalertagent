package repositories

import (
	"context"

	"github.com/aaarkadev/collectalertagent/internal/types"
)

type Repo interface {
	Set(v types.Metrics) error
	Get(k string) (types.Metrics, error)
	GetAll() []types.Metrics
	Init(context.Context) bool
	Shutdown(context.Context)
	FlushDB(context.Context)
	Ping(context.Context) error
}

package repositories

import (
	. "github.com/aaarkadev/collectalertagent/internal/types"
)

type Repo interface {
	Set(v Metric) bool
	Get(k string) (Metric, error)
	GetAll() []Metric
	Init() bool
}

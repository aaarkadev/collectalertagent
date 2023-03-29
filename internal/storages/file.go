package storages

import (
	"encoding/json"
	"os"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type FileStorage struct {
	mem       MemStorage
	Config    configs.ServerConfig
	StoreFile *os.File
}

var _ repositories.Repo = (*FileStorage)(nil)

func (repo *FileStorage) Init() bool {
	repo.mem = MemStorage{}
	repo.mem.Init()

	if len(repo.Config.StoreFileName) > 0 {
		fmode := os.O_RDWR | os.O_CREATE
		if !repo.Config.IsRestore {
			fmode |= os.O_TRUNC
		}
		file, fileErr := os.OpenFile(repo.Config.StoreFileName, fmode, 0777)
		if fileErr == nil {
			repo.StoreFile = file
		} else {
			repo.Config.StoreFileName = ""
		}
	}

	repo.loadDB()

	go func() {
		if repo.Config.StoreInterval == 0 {
			return
		}
		storeTicker := time.NewTicker(repo.Config.StoreInterval)
		defer storeTicker.Stop()
		for range storeTicker.C {
			repo.StoreDBfunc()
		}
	}()

	return true
}

func (repo *FileStorage) loadDB() {
	if !repo.Config.IsRestore {
		return
	}
	if len(repo.Config.StoreFileName) <= 0 {
		return
	}
	decoder := json.NewDecoder(repo.StoreFile)

	oldMetrics := []types.Metrics{}

	if err := decoder.Decode(&oldMetrics); err != nil {
		return
	}

	for _, m := range oldMetrics {
		repo.Set(m)
	}
}

func (repo *FileStorage) Shutdown() {
	repo.StoreDBfunc()
	if len(repo.Config.StoreFileName) > 0 {
		defer repo.StoreFile.Close()
	}
}

func (repo *FileStorage) GetAll() []types.Metrics {
	return repo.mem.metrics
}

func (repo *FileStorage) Get(k string) (types.Metrics, error) {
	return repo.mem.Get(k)
}

func (repo *FileStorage) Set(mset types.Metrics) bool {
	return repo.mem.Set(mset)
}

func (repo *FileStorage) StoreDBfunc() {
	if len(repo.Config.StoreFileName) <= 0 {
		return
	}
	err := repo.StoreFile.Truncate(0)
	if err != nil {
		return
	}
	_, err = repo.StoreFile.Seek(0, 0)
	if err != nil {
		return
	}

	storeTxt, err := json.Marshal(repo.GetAll())
	if err != nil {
		return
	}

	_, err = repo.StoreFile.WriteString(string(storeTxt[:]))
	if err != nil {
		return
	}

}

func (repo *FileStorage) FlushDB() {
	if repo.Config.StoreInterval == 0 {
		repo.StoreDBfunc()
		return
	}

}
package storages

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type FileStorage struct {
	mem       MemStorage
	Config    *configs.ServerConfig
	StoreFile *os.File
}

var _ repositories.Repo = (*FileStorage)(nil)

func (repo *FileStorage) Init(mainCtx context.Context) bool {
	repo.mem = MemStorage{}
	repo.mem.Init(mainCtx)

	if repo.Config == nil {
		log.Println(types.NewTimeError(fmt.Errorf("FileStorage.Init(): empty Config. falback to file")))
		return false
	}

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
			log.Println(types.NewTimeError(fmt.Errorf("FileStorage.Init(): open fail. falback to file. fail: %w", fileErr)))
			return false
		}
	}

	repo.loadDB(mainCtx)

	go func() {
		if repo.Config.StoreInterval == 0 {
			return
		}
		storeTicker := time.NewTicker(repo.Config.StoreInterval)
		defer storeTicker.Stop()
		for {
			select {
			case <-storeTicker.C:
				{
					repo.StoreDBfunc(mainCtx)
				}
			case <-mainCtx.Done():
				{
					runtime.Goexit()
					return
				}
			}
		}
	}()

	return true
}

func (repo *FileStorage) loadDB(mainCtx context.Context) {
	if !repo.Config.IsRestore {
		return
	}
	if len(repo.Config.StoreFileName) <= 0 {
		return
	}
	decoder := json.NewDecoder(repo.StoreFile)

	oldMetrics := []types.Metrics{}

	if err := decoder.Decode(&oldMetrics); err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("FileStorage.loadDB(): fail: %w", err)))
		return
	}

	for _, m := range oldMetrics {
		err := repo.Set(m)
		if err != nil {
			log.Fatalln(types.NewTimeError(fmt.Errorf("FileStorage.loadDB(): fail: %w", err)))
		}
	}
}

func (repo *FileStorage) Shutdown(mainCtx context.Context) {
	repo.StoreDBfunc(mainCtx)
	if len(repo.Config.StoreFileName) > 0 {
		defer repo.StoreFile.Close()
	}
}

func (repo *FileStorage) GetAll() []types.Metrics {
	return repo.mem.GetAll()
}

func (repo *FileStorage) Get(k string) (types.Metrics, error) {
	return repo.mem.Get(k)
}

func (repo *FileStorage) Set(mset types.Metrics) error {
	return repo.mem.Set(mset)
}

func (repo *FileStorage) StoreDBfunc(mainCtx context.Context) {
	if len(repo.Config.StoreFileName) <= 0 {
		return
	}
	err := repo.StoreFile.Truncate(0)
	if err != nil {
		return
	}
	_, err = repo.StoreFile.Seek(0, 0)
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("FileStorage.StoreDBfunc(): fail: %w", err)))
		return
	}

	storeTxt, err := json.Marshal(repo.GetAll())
	if err != nil {
		log.Fatalln(types.NewTimeError(fmt.Errorf("FileStorage.StoreDBfunc(): fail: %w", err)))
		return
	}

	_, err = repo.StoreFile.WriteString(string(storeTxt[:]))
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("FileStorage.StoreDBfunc(): fail: %w", err)))
		return
	}

}

func (repo *FileStorage) FlushDB(mainCtx context.Context) {
	if repo.Config.StoreInterval == 0 {
		repo.StoreDBfunc(mainCtx)
		return
	}

}

func (repo *FileStorage) Ping(mainCtx context.Context) error {
	return nil
}

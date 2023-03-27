package storages

import (
	"encoding/json"
	"os"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

type FileStorage struct {
	mem       MemStorage
	Config    types.ServerConfig
	StoreFile *os.File
}

var _ repositories.Repo = (*FileStorage)(nil)

func (m *FileStorage) Init() bool {
	m.mem = MemStorage{}
	m.mem.Init()

	if len(m.Config.StoreFileName) > 0 {
		fmode := os.O_RDWR | os.O_CREATE
		if !m.Config.IsRestore {
			fmode = fmode | os.O_TRUNC
		}
		file, fileErr := os.OpenFile(m.Config.StoreFileName, fmode, 0777)
		if fileErr == nil {
			m.StoreFile = file
		} else {
			m.Config.StoreFileName = ""
		}
	}

	m.loadDB()

	go func() {
		if m.Config.StoreInterval == 0 {
			return
		}
		storeTicker := time.NewTicker(m.Config.StoreInterval)
		defer storeTicker.Stop()
		for {
			select {
			case <-storeTicker.C:
				{
					m.StoreDBfunc()
				}
			}
		}
	}()

	return true
}

func (m *FileStorage) loadDB() {
	if !m.Config.IsRestore {
		return
	}
	if len(m.Config.StoreFileName) <= 0 {
		return
	}
	decoder := json.NewDecoder(m.StoreFile)

	oldMetrics := []types.Metrics{}

	if err := decoder.Decode(&oldMetrics); err != nil {
		return
	}

	for _, m := range oldMetrics {
		m.Set(m)
	}
}

func (m *FileStorage) Shutdown() {
	m.StoreDBfunc()
	if len(m.Config.StoreFileName) > 0 {
		defer m.StoreFile.Close()
	}
}

func (m *FileStorage) GetAll() []types.Metrics {
	return m.mem.metrics
}

func (m *FileStorage) Get(k string) (types.Metrics, error) {
	return m.mem.Get(k)
}

func (m *FileStorage) Set(mset types.Metrics) bool {
	return m.mem.Set(mset)
}

func (m *FileStorage) StoreDBfunc() {
	if len(m.Config.StoreFileName) <= 0 {
		return
	}
	err := m.StoreFile.Truncate(0)
	if err != nil {
		return
	}
	_, err = m.StoreFile.Seek(0, 0)
	if err != nil {
		return
	}

	storeTxt, err := json.Marshal(m.GetAll())
	if err != nil {
		return
	}

	_, err = m.StoreFile.WriteString(string(storeTxt[:]))
	if err != nil {
		return
	}

}

func (m *FileStorage) FlushDB() {
	if m.Config.StoreInterval == 0 {
		m.StoreDBfunc()
		return
	}

}

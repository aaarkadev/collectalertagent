package storages

import (
	"context"
	// "database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/repositories"
	"github.com/aaarkadev/collectalertagent/internal/types"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jmoiron/sqlx"
)

type DBStorage struct {
	mem    MemStorage
	Config configs.ServerConfig
	DbConn *sqlx.DB
}

var _ repositories.Repo = (*DBStorage)(nil)

const schemaSql = `DROP TABLE IF EXISTS "metrics";
CREATE TABLE  "metrics" (
    "ID"	varchar(255) NOT NULL,
    "MType" varchar(128) DEFAULT 'gauge' NOT NULL,
    "Delta" bigint,
    "Value" double precision,
    "Hash" varchar(128) DEFAULT '' NOT NULL,
    PRIMARY KEY ("ID")
);
CREATE INDEX "metrics_MType" ON  "metrics" USING btree ("MType");`

func (repo *DBStorage) Init() bool {
	repo.mem = MemStorage{}
	repo.mem.Init()

	if len(repo.Config.DSN) > 0 {
		//		"postgres://dron:dron@localhost:5432/dron"
		//conn, connErr := sql.Open("pgx", repo.Config.DSN)
		conn, connErr := sqlx.Open("pgx", repo.Config.DSN)
		if connErr != nil {
			repo.Config.DSN = ""
			return false
		} else {
			repo.DbConn = conn
		}

	}

	ctx, cancel := context.WithTimeout(repo.Config.MainCtx, configs.GlobalDefaultTimeout)
	defer cancel()
	if !repo.Config.IsRestore {
		_, err := repo.DbConn.ExecContext(ctx, schemaSql)
		if err != nil {
			repo.Config.DSN = ""
			return false
		}
	}

	repo.Ping()
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

func (repo *DBStorage) loadDB() {
	if !repo.Config.IsRestore {
		return
	}
	if len(repo.Config.DSN) <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(repo.Config.MainCtx, configs.GlobalDefaultTimeout)
	defer cancel()

	oldMetrics := []types.Metrics{}
	err := repo.DbConn.SelectContext(ctx, &oldMetrics, `SELECT * FROM "metrics"`)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, m := range oldMetrics {
		m.ID = strings.Trim(m.ID, " 	")
		m.MType = strings.Trim(m.MType, " 	")
		m.Hash = strings.Trim(m.Hash, " 	")
		repo.Set(m)
	}
}

func (repo *DBStorage) Shutdown() {
	repo.StoreDBfunc()
	if len(repo.Config.DSN) > 0 {
		defer repo.DbConn.Close()
	}
}

func (repo *DBStorage) GetAll() []types.Metrics {
	return repo.mem.metrics
}

func (repo *DBStorage) Get(k string) (types.Metrics, error) {
	return repo.mem.Get(k)
}

func (repo *DBStorage) Set(mset types.Metrics) bool {
	return repo.mem.Set(mset)
}

func (repo *DBStorage) StoreDBfunc() {
	if len(repo.Config.DSN) <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(repo.Config.MainCtx, configs.GlobalDefaultTimeout)
	defer cancel()

	var err error
	dbTx, err := repo.DbConn.BeginTxx(ctx, nil)
	if err != nil {
		fmt.Println("Err trans begin", err)
		return
	}

	_, err = dbTx.ExecContext(ctx, `TRUNCATE TABLE "metrics"`)
	if err != nil {
		fmt.Println("Err trunc", err)
		return
	}

	allMetrics := repo.GetAll()
	if len(allMetrics) > 0 {
		_, err = dbTx.NamedExecContext(ctx, `INSERT INTO "metrics" ("ID", "MType", "Delta", "Value", "Hash")
                                                    VALUES (:ID, :MType, :Delta, :Value, :Hash)`, allMetrics)
		if err != nil {
			fmt.Println("Err insert", err)
			return
		}
	}

	err = dbTx.Commit()
	if err != nil {
		fmt.Println("Err trans commit", err)
		return
	}
}

func (repo *DBStorage) FlushDB() {
	repo.StoreDBfunc()
}

func (repo *DBStorage) Ping() error {
	if len(repo.Config.DSN) <= 0 {
		return fmt.Errorf("DSN empty or no connection to DB")
	}
	return repo.DbConn.PingContext(repo.Config.MainCtx)
}

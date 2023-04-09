package storages

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	Config *configs.ServerConfig
	DBConn *sqlx.DB
}

var _ repositories.Repo = (*DBStorage)(nil)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS "metrics" (
    "ID"	varchar(255) NOT NULL,
    "MType" varchar(128) DEFAULT 'gauge' NOT NULL,
    "Delta" bigint,
    "Value" double precision,
    "Hash" varchar(128) DEFAULT '' NOT NULL,
    PRIMARY KEY ("ID")
);
CREATE INDEX IF NOT EXISTS "metrics_MType" ON  "metrics" USING btree ("MType");`

func (repo *DBStorage) Init() bool {
	repo.mem = MemStorage{}
	repo.mem.Init()

	if repo.Config == nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.Init(): empty Config. falback to file")))
		return false
	}
	if len(repo.Config.DSN) <= 0 {
		repo.Config.DSN = ""
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.Init(): empty Config.DSN falback to file")))
		return false
	}
	conn, connErr := sql.Open("pgx", repo.Config.DSN)
	if connErr != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.Init(): Cannot connect to DB. falback to file. fail: %w", connErr)))
		repo.Config.DSN = ""
		return false
	}
	connErr = conn.Ping()
	if connErr != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.Init(): Cannot ping DB. falback to file. fail: %w", connErr)))
		repo.Config.DSN = ""
		return false
	}
	repo.DBConn = sqlx.NewDb(conn, "pgx")

	ctx, cancel := context.WithTimeout(repo.Config.MainCtx, configs.GlobalDefaultTimeout)
	defer cancel()
	if !repo.Config.IsRestore {
		repo.DBConn.ExecContext(ctx, `DROP TABLE IF EXISTS "metrics";`)
	}
	_, err := repo.DBConn.ExecContext(ctx, schemaSQL)
	if err != nil {
		fmt.Println("Cannot load DB shema. falback to file. ", err)
		repo.Config.DSN = ""
		return false
	}
	_, err = repo.DBConn.ExecContext(ctx, `SELECT * FROM "metrics" LIMIT 1`)
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.Init(): Cannot find DB table. falback to file. fail: %w", err)))
		repo.Config.DSN = ""
		return false

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
	err := repo.DBConn.SelectContext(ctx, &oldMetrics, `SELECT * FROM "metrics"`)
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.loadDB(): empty table. fail: %w", err)))
		return
	}
	for _, m := range oldMetrics {
		m.ID = strings.Trim(m.ID, " 	")
		m.MType = strings.Trim(m.MType, " 	")
		m.Hash = strings.Trim(m.Hash, " 	")
		err := repo.Set(m)
		if err != nil {
			log.Fatalln(types.NewTimeError(fmt.Errorf("DBStorage.loadDB(): fail: %w", err)))
		}
	}
}

func (repo *DBStorage) Shutdown() {
	repo.StoreDBfunc()
	if len(repo.Config.DSN) > 0 {
		defer repo.DBConn.Close()
	}
}

func (repo *DBStorage) GetAll() []types.Metrics {
	return repo.mem.GetAll()
}

func (repo *DBStorage) Get(k string) (types.Metrics, error) {
	return repo.mem.Get(k)
}

func (repo *DBStorage) Set(mset types.Metrics) error {
	return repo.mem.Set(mset)
}

func (repo *DBStorage) StoreDBfunc() {
	if len(repo.Config.DSN) <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(repo.Config.MainCtx, configs.GlobalDefaultTimeout)
	defer cancel()

	var err error
	dbTx, err := repo.DBConn.BeginTxx(ctx, nil)
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.StoreDBfunc(): transaction begin fail: %w", err)))
		return
	}

	_, err = dbTx.ExecContext(ctx, `TRUNCATE TABLE "metrics"`)
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.StoreDBfunc(): truncate table fail: %w", err)))
		return
	}

	allMetrics := repo.GetAll()
	if len(allMetrics) > 0 {
		_, err = dbTx.NamedExecContext(ctx, `INSERT INTO "metrics" ("ID", "MType", "Delta", "Value", "Hash")
                                                    VALUES (:ID, :MType, :Delta, :Value, :Hash)`, allMetrics)
		if err != nil {
			log.Println(types.NewTimeError(fmt.Errorf("DBStorage.StoreDBfunc(): insert into table fail: %w", err)))
			return
		}
	}

	err = dbTx.Commit()
	if err != nil {
		log.Println(types.NewTimeError(fmt.Errorf("DBStorage.StoreDBfunc(): transaction commit fail: %w", err)))
		return
	}
}

func (repo *DBStorage) FlushDB() {
	repo.StoreDBfunc()
}

func (repo *DBStorage) Ping() error {
	if len(repo.Config.DSN) <= 0 {
		err := types.NewTimeError(fmt.Errorf("DBStorage.Ping(): DSN empty or no connection to DB"))
		log.Println(err)
		return err
	}
	return repo.DBConn.PingContext(repo.Config.MainCtx)
}

package configs

import (
	"context"
	"flag"
	"os"
	"time"
)

const GlobalDefaultTimeout = 15 * time.Second

type ServerConfig struct {
	ListenAddress string
	StoreInterval time.Duration
	StoreFileName string
	IsRestore     bool
	HashKey       []byte
	DSN           string
	MainCtx       context.Context
}

type AgentConfig struct {
	SendAddress    string
	ReportInterval time.Duration
	PollInterval   time.Duration
	HashKey        []byte
	DSN            string
}

func InitServerConfig() ServerConfig {

	config := ServerConfig{}

	defaultListenAddress := "127.0.0.1:8080"
	flag.StringVar(&config.ListenAddress, "a", defaultListenAddress, "address to listen on")

	defaultStoreInterval := 300 * time.Second
	flag.DurationVar(&config.StoreInterval, "i", defaultStoreInterval, "store interval")

	flag.BoolVar(&config.IsRestore, "r", false, "is restore DB")

	defaultStoreFile := "/tmp/devops-metrics-db.json"
	flag.StringVar(&config.StoreFileName, "f", defaultStoreFile, "store filepath")

	defaultHashKey := ""
	HashKeyStr := ""
	flag.StringVar(&HashKeyStr, "k", defaultHashKey, "hash key")

	defaultDSN := ""
	flag.StringVar(&config.DSN, "d", defaultDSN, "db DSN string")

	flag.Parse()

	config.HashKey = []byte(HashKeyStr)

	envVal, envFound := os.LookupEnv("ADDRESS")
	if envFound {
		config.ListenAddress = envVal
	}
	envVal, envFound = os.LookupEnv("STORE_INTERVAL")
	if envFound {
		envDur, err := time.ParseDuration(envVal)
		if err == nil {
			config.StoreInterval = envDur
		}
	}
	envVal, envFound = os.LookupEnv("STORE_FILE")
	if envFound {
		config.StoreFileName = envVal
	}
	envVal, envFound = os.LookupEnv("RESTORE")
	if envFound {
		if envVal == "true" {
			config.IsRestore = true
		} else {
			config.IsRestore = false
		}
	}
	envVal, envFound = os.LookupEnv("KEY")
	if envFound {
		config.HashKey = []byte(envVal)
	}

	envVal, envFound = os.LookupEnv("DATABASE_DSN")
	if envFound {
		config.DSN = envVal
	}

	return config
}

func InitAgentConfig() AgentConfig {

	config := AgentConfig{}

	defaultSendAddress := "127.0.0.1:8080"
	flag.StringVar(&config.SendAddress, "a", defaultSendAddress, "address to listen on")

	defaultReportInterval := 10 * time.Second
	flag.DurationVar(&config.ReportInterval, "r", defaultReportInterval, "report interval")

	defaultPollInterval := 2 * time.Second
	flag.DurationVar(&config.PollInterval, "p", defaultPollInterval, "poll interval")

	defaultHashKey := ""
	HashKeyStr := ""
	flag.StringVar(&HashKeyStr, "k", defaultHashKey, "hash key")

	defaultDSN := ""
	flag.StringVar(&config.DSN, "d", defaultDSN, "db DSN string")

	flag.Parse()

	config.HashKey = []byte(HashKeyStr)

	envVal, envFound := os.LookupEnv("ADDRESS")
	if envFound {
		config.SendAddress = envVal
	}

	envVal, envFound = os.LookupEnv("REPORT_INTERVAL")
	if envFound {
		envDur, err := time.ParseDuration(envVal)
		if err == nil {
			config.ReportInterval = envDur
		}
	}

	envVal, envFound = os.LookupEnv("POLL_INTERVAL")
	if envFound {
		envDur, err := time.ParseDuration(envVal)
		if err == nil {
			config.PollInterval = envDur
		}
	}

	envVal, envFound = os.LookupEnv("KEY")
	if envFound {
		config.HashKey = []byte(envVal)
	}

	envVal, envFound = os.LookupEnv("DATABASE_DSN")
	if envFound {
		config.DSN = envVal
	}

	return config
}

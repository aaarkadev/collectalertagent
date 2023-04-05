package main

import (
	"github.com/aaarkadev/collectalertagent/internal/agents"
	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/storages"
)

func main() {

	config := configs.InitAgentConfig()
	repo := storages.MemStorage{}
	agents.InitAllMetrics(&repo)

	agents.StartAgent(&repo, config)
}

package main

import (
	"fmt"
	"log"

	"github.com/aaarkadev/collectalertagent/internal/agents"
	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

func main() {

	agents.SetupLog()
	log.Println(types.NewTimeError(fmt.Errorf("START")))
	config := configs.InitAgentConfig()
	repo := storages.MemStorage{}
	agents.InitAllMetrics(&repo)

	defer func() {
		agents.StopAgent(&repo)
	}()

	agents.StartAgent(&repo, config)

	log.Println(types.NewTimeError(fmt.Errorf("END")))
}

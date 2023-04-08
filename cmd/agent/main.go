package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aaarkadev/collectalertagent/internal/agents"
	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/storages"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

func main() {
	f, err := os.OpenFile("/home/dron/go/src/dron/collectalertagent/log.agent.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	config := configs.InitAgentConfig()
	repo := storages.MemStorage{}
	agents.InitAllMetrics(&repo)

	agents.StartAgent(&repo, config)

	log.Println(types.NewTimeError(fmt.Errorf("AGENT END")))
}

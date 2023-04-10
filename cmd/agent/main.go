package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aaarkadev/collectalertagent/internal/agents"
	"github.com/aaarkadev/collectalertagent/internal/configs"
	"github.com/aaarkadev/collectalertagent/internal/types"
)

func main() {

	agents.SetupLog()
	log.Println(types.NewTimeError(fmt.Errorf("START")))

	mainCtx, mainCtxCancel := context.WithCancel(context.Background())
	mainCtx = context.WithValue(mainCtx, "mainCtxCancel", mainCtxCancel)
	defer mainCtxCancel()

	config := configs.InitAgentConfig()
	config.MainCtx = mainCtx
	repo := agents.Init(&config)

	defer func() {
		agents.StopAgent(repo)
	}()

	agents.StartAgent(repo, config)

	log.Println(types.NewTimeError(fmt.Errorf("END")))
}

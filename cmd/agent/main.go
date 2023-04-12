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
	mainCtx = context.WithValue(mainCtx, types.MainCtxCancelFunc, mainCtxCancel)
	defer mainCtxCancel()

	config := configs.InitAgentConfig()
	repo := agents.Init(mainCtx, &config)

	defer func() {
		agents.StopAgent(mainCtx, repo)
	}()

	agents.StartAgent(mainCtx, repo, config)

	log.Println(types.NewTimeError(fmt.Errorf("END")))
}

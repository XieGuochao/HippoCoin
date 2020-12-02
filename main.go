package main

import (
	"context"

	"github.com/withmandala/go-log"

	"os"

	. "github.com/XieGuochao/HippoCoin/host"
)

var version = "1.0"

var logger *log.Logger

func initLogger() {
	logger = log.New(os.Stdout)
	logger.WithDebug()
	logger.WithColor()
}

func main() {
	var (
		host   Host
		ctx    context.Context
		cancel context.CancelFunc

		config     HippoConfig
		configPath string
	)
	if len(os.Args) == 1 {
		configPath = "./host.yml"
	} else {
		configPath = os.Args[1]
	}

	config.Load(configPath)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	host = new(HippoHost)
	host.InitLogger(true)
	host.New(true, config.curve, true)

	host.InitLocals(ctx, Hash, config.miningFunction, 1,
		new(P2PClient), uint(config.BroadcastQueueLen), MiningCallbackBroadcastSave,
		BasicDifficulty, int64(config.MiningInterval), config.MiningCapacity,
		int64(config.MiningTTL), config.Protocol)
	host.InitNetwork(new(HippoBlock), config.MaxNeighbors, config.UpdateTimeBase, config.UpdateTimeRand,
		config.RegisterAddress, config.RegisterProtocol)
	host.Run()
}

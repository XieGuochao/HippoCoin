package main

import (
	"context"
	"crypto/elliptic"
	"os"
	"strconv"

	"github.com/withmandala/go-log"
)

var version = "1.0"

func initLogger() {
	logger = log.New(os.Stdout)
	logger.WithDebug()
	logger.WithColor()
}

func main() {
	var (
		host           Host
		ctx            context.Context
		cancel         context.CancelFunc
		miningInterval int64
		msi            int
		err            error
	)
	ms := os.Args[1]
	msi, err = strconv.Atoi(ms)
	if err != nil {
		msi = 30
	}
	miningInterval = int64(msi)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	host = new(HippoHost)
	host.InitLogger(true)
	host.New(true, elliptic.P224(), true)

	host.InitLocals(ctx, hash, new(singleMiningFunction), 1,
		new(P2PClient), 10, miningCallbackBroadcastSave,
		basicDifficulty, miningInterval, 5, 600, "tcp")
	host.InitNetwork(new(HippoBlock), 5, 4, 1,
		"localhost:9325", "tcp")
	host.Run()
}

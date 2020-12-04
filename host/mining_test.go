package host

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMiningOK(t *testing.T) {
	initTest(3)
	infoLogger.Debug("test mining ===============================================")
	balance := new(HippoBalance)
	// block
	var block Block
	block = new(HippoBlock)
	block.New([]byte{}, 233, testHashfunction, 0, balance, testCurve)

	// mining
	mining := new(HippoMining)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testMiningFunction.New(testHashfunction, 1)

	miningQueue := new(MiningQueue)
	// debugLogger.Debug("hash function:", &testHashfunction)
	miningQueue.New(ctx, func(has bool, block Block, storage Storage, bq BroadcastQueue) {
		infoLogger.Debug("has:", has)
		infoLogger.Debug("mine a block:", block)
		infoLogger.Debug("mine check:", block.CheckNonce(), block.Check())
		debugLogger.Debug("invoke stop")
		mining.Stop()
	}, testHashfunction, testMiningFunction)

	wg.Add(1)
	miningQueue.Run(&wg)

	transactionPool := new(HippoTransactionPool)
	transactionPool.New(balance, testBroadcastQueue)
	mining.New(miningQueue, transactionPool, StaticDifficulty,
		30, 100, 100, balance, testKeys[0])

	block = mining.Fetch(block)
	mining.Sign(block)
	mining.Mine(block)

	wg.Wait()
}

func TestMiningStop(t *testing.T) {
	initTest(3)
	infoLogger.Debug("test mining ===============================================")
	balance := new(HippoBalance)
	// block
	var block Block
	block = new(HippoBlock)
	block.New([]byte{}, 220, testHashfunction, 0, balance, testCurve)

	// mining
	mining := new(HippoMining)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testMiningFunction.New(testHashfunction, 1)

	miningQueue := new(MiningQueue)
	// debugLogger.Debug("hash function:", &testHashfunction)
	miningQueue.New(ctx, miningCallbackLog, testHashfunction, testMiningFunction)

	wg.Add(1)
	miningQueue.Run(&wg)

	transactionPool := new(HippoTransactionPool)
	transactionPool.New(balance, testBroadcastQueue)
	mining.New(miningQueue, transactionPool, StaticDifficulty,
		30, 100, 100, balance, testKeys[0])

	block = mining.Fetch(block)
	mining.Sign(block)
	mining.Mine(block)

	go func() {
		time.Sleep(time.Second * time.Duration(10))
		debugLogger.Debug("invoke stop")
		mining.Stop()
	}()

	wg.Wait()
}

func TestMiningMultipleOK(t *testing.T) {
	initTest(3)
	infoLogger.Debug("test mining ===============================================")
	balance := new(HippoBalance)
	// block
	var block Block
	block = new(HippoBlock)
	block.New([]byte{}, 233, testHashfunction, 0, balance, testCurve)

	// mining
	mining := new(HippoMining)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testMiningFunction = new(MultipleMiningFunction)
	testMiningFunction.New(testHashfunction, 4)

	miningQueue := new(MiningQueue)
	// debugLogger.Debug("hash function:", &testHashfunction)
	miningQueue.New(ctx, func(has bool, block Block, storage Storage, bq BroadcastQueue) {
		miningCallbackLog(has, block, storage, bq)
		mining.Stop()
	}, testHashfunction, testMiningFunction)

	wg.Add(1)
	miningQueue.Run(&wg)

	transactionPool := new(HippoTransactionPool)
	transactionPool.New(balance, testBroadcastQueue)
	mining.New(miningQueue, transactionPool, StaticDifficulty,
		30, 100, 100, balance, testKeys[0])

	block = mining.Fetch(block)
	mining.Sign(block)

	go func() {
		time.Sleep(time.Second * time.Duration(10))
		debugLogger.Debug("invoke stop")
		mining.Stop()
	}()

	mining.Mine(block)
	wg.Wait()
}

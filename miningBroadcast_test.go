package main

import (
	"testing"
)

func TestMiningBroadcast(t *testing.T) {
	initTest(1)
	logger.Info("test register ======================================")
	testWaitGroup.Add(1)
	initNetwork()
	testNetworkClient.SyncNeighbors()

	testBalance := new(HippoBalance)
	testBalance.New()

	var block Block
	block = new(HippoBlock)
	block.New([]byte{}, 232, testHashfunction, 0, testBalance, testCurve)

	// Storage & BQ
	testStorage = new(HippoStorage)
	testStorage.New()
	testBroadcastQueue = new(HippoBroadcastQueue)
	testBroadcastQueue.New(testContext, testProtocol, &testNetworkClient, &testP2PClientTemplate)
	testBroadcastQueue.Run()

	// mining
	testMining = new(HippoMining)
	testMiningFunction = new(multipleMiningFunction)
	testMiningFunction.New(testContext, testHashfunction, 4)

	testMiningQueue.New(testContext, miningCallbackBroadcastSave, testHashfunction, testMiningFunction)
	testMiningQueue.SetBroadcastQueue(testBroadcastQueue)
	testMiningQueue.SetStorage(testStorage)
	testMiningQueue.Run(&testWaitGroup)

	testTransactionPool = new(HippoTransactionPool)
	testTransactionPool.New(testBalance)
	testMining.New(&testMiningQueue, testTransactionPool, 10, 600, testBalance, testKeys[0])

	logger.Debug("going to fetch")

	block = testMining.Fetch(block)
	testMining.Sign(block)
	testMining.Mine(block)
	testWaitGroup.Wait()
}

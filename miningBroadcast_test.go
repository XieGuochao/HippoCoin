package main

import (
	"testing"
)

func TestMiningBroadcast(t *testing.T) {
	initTest(1)
	logger.Info("test register ======================================")
	testWaitGroup.Add(1)
	initPrenetwork()
	initNetwork()
	testNetworkClient.SyncNeighbors()

	var block Block
	block = new(HippoBlock)
	block.New([]byte{}, 230, testHashfunction, 0, testBalance, testCurve)

	logger.Debug("going to fetch")

	block = testMining.Fetch(block)
	testMining.Sign(block)
	testMining.Mine(block)
	testWaitGroup.Wait()
}

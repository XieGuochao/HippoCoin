package main

import (
	"testing"
	"time"
)

func TestMiningBroadcast(t *testing.T) {
	initTest(1)
	logger.Info("test register ======================================")
	initPrenetwork()
	initNetwork()
	initNetworkRun()
	testNetworkClient.SyncNeighbors()

	var block Block
	block = new(HippoBlock)
	block.New([]byte{}, 231, testHashfunction, 0, testBalance, testCurve)

	logger.Debug("going to fetch")

	go testMining.WatchSendNewBlock()
	block = testMining.Fetch(block)
	testMining.Sign(block)
	testMining.Mine(block)

	for {
		time.Sleep(time.Second * time.Duration(5))
		logger.Info("block hashes:", testStorage.AllHashesInLevel())
		logger.Info("balance:", testBalance.AllBalance())
	}
}

func TestListenBroadcast(t *testing.T) {
	initTest(1)
	logger.Info("test ping ======================================")

	initPrenetwork()
	initNetwork()
	testNetworkClient.SyncNeighbors()

	for {
		time.Sleep(time.Second * time.Duration(5))
		logger.Info("block hashes:", testStorage.AllHashesInLevel())
	}
}
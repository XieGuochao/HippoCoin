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

func TestQueryLevel(t *testing.T) {
	initTest(1)
	logger.Info("test query level ==============================")

	initPrenetwork()
	initNetwork()
	testNetworkClient.SyncNeighbors()

	for {
		var reply []string
		reply = make([]string, 10)
		time.Sleep(time.Second * time.Duration(5))
		neighbors := testNetworkClient.GetNeighbors()
		if len(neighbors) == 0 {
			continue
		}
		testNetworkClient.QueryLevel(neighbors[0], 0, 4, &reply)
		logger.Info("levels:", reply)
	}
}

func TestQueryGenisus(t *testing.T) {
	initTest(1)
	logger.Info("test query genisus ==============================")

	initPrenetwork()
	initNetwork()
	testNetworkClient.SyncNeighbors()

	for {
		var reply []string
		reply = make([]string, 10)

		var block Block
		time.Sleep(time.Second * time.Duration(5))
		neighbors := testNetworkClient.GetNeighbors()
		if len(neighbors) == 0 {
			continue
		}
		testNetworkClient.QueryLevel(neighbors[0], 0, 0, &reply)
		logger.Info("levels:", reply)
		if len(reply) > 0 {
			block = testNetworkClient.QueryByHash(neighbors[0], reply[0])
			logger.Info("block0:", block.Hash(), block.HashSignature())
		}
	}
}

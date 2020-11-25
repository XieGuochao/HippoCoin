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
	block.New([]byte{}, 232, testHashfunction, 0, testBalance, testCurve)

	logger.Debug("going to fetch")

	go testMining.WatchSendNewBlock()
	block = testMining.Fetch(block)
	testMining.Sign(block)
	testMining.Mine(block)

	for {
		time.Sleep(time.Second * time.Duration(5))
		logger.Info("block hashes: [", testStorage.MaxLevel(), "]", testStorage.AllHashesInLevel())
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

func TestQueryHashes(t *testing.T) {
	initTest(1)
	logger.Info("test query genisus ==============================")

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
		testNetworkClient.QueryLevel(neighbors[0], 0, 5, &reply)
		logger.Info("levels:", reply)
		newHashes := testStorage.FilterNewHashes(reply)
		newBlocks := testNetworkClient.QueryHashes(neighbors[0], newHashes)
		if newBlocks != nil {
			testStorage.AddBlocks(newBlocks)
			for _, b := range newBlocks {
				logger.Info("update new block:", b.Hash())
			}
		}
		logger.Info("block hashes: [", testStorage.MaxLevel(), "]", testStorage.AllHashesInLevel())
	}
}

func TestSyncBlocks(t *testing.T) {
	initTest(1)
	logger.Info("test sync blocks ==============================")

	initPrenetwork()
	initNetwork()
	testNetworkClient.SyncNeighbors()

	for {
		time.Sleep(time.Second * time.Duration(10))
		neighbors := testNetworkClient.GetNeighbors()
		if len(neighbors) == 0 {
			continue
		}

		testNetworkClient.SyncBlocks(neighbors[0], testStorage)
		logger.Info("block hashes: [", testStorage.MaxLevel(), "]", testStorage.AllHashesInLevel())
	}
}

func TestSyncAddressesN(t *testing.T) {
	initTest(1)
	logger.Info("test sync addresses n ==============================")

	initPrenetwork()
	initNetwork()
	testNetworkClient.SyncNeighbors()

	for {
		time.Sleep(time.Second * time.Duration(10))
		testNetworkClient.SyncAddressesN(3, testStorage)
		logger.Info("block hashes: [", testStorage.MaxLevel(), "]", testStorage.AllHashesInLevel())
	}
}

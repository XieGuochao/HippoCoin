package main

import (
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	initTest(0)
	logger.Info("test register ======================================")

	initNetwork()
	testNetworkClient.SyncNeighbors()

	go func() {
		time.Sleep(time.Second * time.Duration(30))
		testCancel()
	}()

	go func() {
		for {
			logger.Info("neighbors:", testNetworkClient.GetNeighbors())
			time.Sleep(time.Second)
		}
	}()

	for {
		select {
		case <-testContext.Done():
			logger.Info("test done.")
			break
		}
	}
}

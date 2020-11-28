package host

import (
	"testing"
	"time"
)

func TestPingSend(t *testing.T) {
	initTest(1)
	logger.Info("test ping ======================================")

	initPrenetwork()
	initNetwork()

	for {
		time.Sleep(time.Second * time.Duration(5))
		testNetworkClient.TryUpdateNeighbors()
		logger.Debug("neighbors:", testNetworkClient.GetNeighbors())

		addresses := testNetworkClient.GetNeighbors()
		for _, address := range addresses {
			t, ok := testNetworkClient.Ping(address)
			logger.Info("ping result:", address, t, ok)
		}
	}
}

func TestPingReceive(t *testing.T) {
	initTest(1)
	logger.Info("test ping ======================================")

	initPrenetwork()
	initNetwork()

	for {
		time.Sleep(time.Second * time.Duration(5))
		testNetworkClient.TryUpdateNeighbors()
		logger.Debug("neighbors:", testNetworkClient.GetNeighbors())
		logger.Info("running")
	}
}

func TestRegister(t *testing.T) {
	initTest(1)
	logger.Info("test register ======================================")

	initPrenetwork()
	initNetwork()
	initNetworkRun()
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
			return
		}
	}
}

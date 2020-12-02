package host

import (
	"testing"
	"time"
)

func TestPingSend(t *testing.T) {
	initTest(1)
	infoLogger.Debug("test ping ======================================")

	initPrenetwork()
	initNetwork()

	for {
		time.Sleep(time.Second * time.Duration(5))
		testNetworkClient.TryUpdateNeighbors()
		debugLogger.Debug("neighbors:", testNetworkClient.GetNeighbors())

		addresses := testNetworkClient.GetNeighbors()
		for _, address := range addresses {
			t, ok := testNetworkClient.Ping(address)
			infoLogger.Debug("ping result:", address, t, ok)
		}
	}
}

func TestPingReceive(t *testing.T) {
	initTest(1)
	infoLogger.Debug("test ping ======================================")

	initPrenetwork()
	initNetwork()

	for {
		time.Sleep(time.Second * time.Duration(5))
		testNetworkClient.TryUpdateNeighbors()
		debugLogger.Debug("neighbors:", testNetworkClient.GetNeighbors())
		infoLogger.Debug("running")
	}
}

func TestRegister(t *testing.T) {
	initTest(1)
	infoLogger.Debug("test register ======================================")

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
			infoLogger.Debug("neighbors:", testNetworkClient.GetNeighbors())
			time.Sleep(time.Second)
		}
	}()

	for {
		select {
		case <-testContext.Done():
			infoLogger.Debug("test done.")
			return
		}
	}
}

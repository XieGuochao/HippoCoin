package main

import (
	"context"
	"crypto/elliptic"

	registerlib "github.com/XieGuochao/HippoCoinRegister/lib"
)

var (
	testKeys           []Key
	testKeyNumber      int
	testCurve          elliptic.Curve
	testHashfunction   = hash
	testMiningFunction MiningFunction

	testContext context.Context
	testCancel  context.CancelFunc

	testRegister          Register
	testNetworkClient     NetworkClient
	testNetworkListener   NetworkListener
	testIP                string
	testPort              int
	testAddress           string
	testProtocol          string
	testP2PClientTemplate = P2PClient{}
	testP2PServer         P2PServiceInterface

	testRegisterAddress  string
	testRegisterProtocol string
)

func initKeys(number int) {
	testKeyNumber = number
	testKeys = make([]Key, testKeyNumber)
	testCurve = elliptic.P224()
	for i := range testKeys {
		testKeys[i].New(testCurve)
		testKeys[i].GenerateKey()
	}
}

func initTest(number int) {
	initLogger()
	logger.WithDebug()
	logger.WithColor()
	initKeys(number)
	testMiningFunction = new(singleMiningFunction)
}

func initNetwork() {
	testIP = registerlib.GetOutboundIP().String()
	testProtocol = "tcp"
	testRegisterAddress = "localhost:9325"
	testRegisterProtocol = "tcp"

	testContext, testCancel = context.WithCancel(context.Background())

	testNetworkListener = new(HippoNetworkListener)
	testNetworkListener.New(testContext, testIP, testProtocol)
	testNetworkListener.Listen()

	logger.Info("create listener")

	testAddress = testNetworkListener.NetworkAddress()
	logger.Info("listener address:", testAddress)

	testP2PServer = new(P2PServer)
	testP2PServer.new(testContext, testNetworkListener.Listener())
	testP2PServer.serve()

	testRegister = new(HippoRegister)
	testRegister.New(testContext, testRegisterAddress, testRegisterProtocol)

	logger.Info("create register")

	testNetworkClient = new(HippoNetworkClient)
	testNetworkClient.New(testContext, testAddress, testProtocol,
		10, testRegister, 5, 3, &testP2PClientTemplate)

	logger.Info("create network client")

}

package host

import (
	"context"
	"crypto/elliptic"
	"sync"
	"time"

	registerlib "github.com/XieGuochao/HippoCoinRegister/lib"
)

var (
	testKeys           []Key
	testKeyNumber      int
	testCurve          elliptic.Curve
	testHashfunction   = Hash
	testMiningFunction MiningFunction

	testContext context.Context
	testCancel  context.CancelFunc

	testRegister          Register
	testNetworkClient     NetworkClient
	testNetworkListener   NetworkListener
	testIP                string
	testPort              int
	testAddress           string
	testProtocol          = "tcp"
	testP2PClientTemplate = P2PClient{}
	testP2PServer         P2PServiceInterface

	testRegisterAddress  = "localhost:9325"
	testRegisterProtocol = "tcp"

	testWaitGroup sync.WaitGroup

	testBalance         Balance
	testMining          Mining
	testMiningQueue     MiningQueue
	testTransactionPool TransactionPool
	testStorage         Storage
	testBroadcastQueue  BroadcastQueue

	testBlockTemplate Block
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
	initLogger("")
	debugLogger.WithDebug()
	debugLogger.WithoutColor()
	initKeys(number)
	testMiningFunction = new(SingleMiningFunction)
	testBalance = new(HippoBalance)
	testBalance.New()
	testContext, testCancel = context.WithCancel(context.Background())
}

func initBalance() {
	testBalance = new(HippoBalance)
	testBalance.New()
}

func initStorage() {
	testStorage = new(HippoStorage)
	testStorage.New()
	testStorage.SetBalance(testBalance)
}

func initBroadcastQueue() {
	testBroadcastQueue = new(HippoBroadcastQueue)
	testBroadcastQueue.New(testContext, testProtocol, &testP2PClientTemplate, 5)
}

func initMining() {
	testMining = new(HippoMining)
	testMining.SetBroadcastQueue(testBroadcastQueue)
	testMining.SetStorage(testStorage)
	testMiningFunction = new(SingleMiningFunction)
	testMiningFunction.New(testHashfunction, 1)
}

func initMiningQueue() {
	testMiningQueue.New(testContext, MiningCallbackBroadcastSave, testHashfunction, testMiningFunction)
	testMiningQueue.SetBroadcastQueue(testBroadcastQueue)
	testMiningQueue.SetStorage(testStorage)
}

func initTransactionPool() {
	testTransactionPool = new(HippoTransactionPool)
	testTransactionPool.New(testBalance, testBroadcastQueue)
	testMining.New(&testMiningQueue, testTransactionPool, StaticDifficulty,
		30, 10, 600, testBalance, testKeys[0])
}

func initMinings() {
	initMining()
	initMiningQueue()
	initTransactionPool()
}

func initPrenetwork() {
	initBalance()
	initStorage()
	initBroadcastQueue()
	initMinings()
}

func initNetwork() {
	testIP = registerlib.GetOutboundIP().String()
	testBlockTemplate = new(HippoBlock)
	testBlockTemplate.New([]byte{}, 0, testHashfunction, 0, testBalance, testCurve)

	testContext, testCancel = context.WithCancel(context.Background())

	testNetworkListener = new(HippoNetworkListener)
	testNetworkListener.New(testContext, testIP, testProtocol)
	testNetworkListener.Listen()

	infoLogger.Debug("create listener")

	testAddress = testNetworkListener.NetworkAddress()
	infoLogger.Debug("listener address:", testAddress)

	testP2PServer = new(P2PServer)
	testP2PServer.new(testContext, testNetworkListener.Listener())
	testP2PServer.setBroadcastQueue(testBroadcastQueue)
	testP2PServer.setStorage(testStorage)
	testP2PServer.setBlockTemplate(testBlockTemplate)
	testP2PServer.serve()

	testRegister = new(HippoRegister)
	testRegister.New(testContext, testRegisterAddress, testRegisterProtocol)

	infoLogger.Debug("create register")

	testNetworkClient = new(HippoNetworkClient)
	testNetworkClient.New(testContext, testAddress, testProtocol,
		10, testRegister, 5, 2, &testP2PClientTemplate, testBlockTemplate)

	testBroadcastQueue.SetNetworkClient(testNetworkClient)
	infoLogger.Debug("create network client")

}

func initNetworkRun() {
	testBroadcastQueue.Run()
	testMiningQueue.Run(&testWaitGroup)
}

func watchStorageBalance(storage Storage, balance Balance,
	seconds int64) {
	for {
		time.Sleep(time.Second * time.Duration(seconds))
		infoLogger.Debug("block hashes: [", storage.MaxLevel(), "]",
			storage.AllHashesInLevel())
		infoLogger.Debug("balance:", balance.AllBalance())
	}
}

package main

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"sync"

	registerlib "github.com/XieGuochao/HippoCoinRegister/lib"
)

// Host ...
type Host interface {
	New(debug bool, curve elliptic.Curve, localMode bool)

	Run()
	InitLogger(debug bool)
	InitLocals(
		ctx context.Context,
		hashFunction HashFunction,
		miningFunction MiningFunction,
		miningThreads int,
		p2pClientTemplate P2PClientInterface,
		broadcastQueueLen uint,

		miningCallbackFunction MiningCallback,
		difficultyFunction DifficultyFunc,
		interval int64,

		miningCapacity int,
		miningTTL int64,
		protocol string,
	)
	InitNetwork(
		blockTemplate Block,
		maxNeighbors int,
		updateTimeBase int,
		updateTimeRand int,

		registerAddress string,
		registerProtocol string,
	)

	Close()
}

// HippoHost ...
type HippoHost struct {
	localMode bool

	// key and curve
	key   Key
	curve elliptic.Curve

	hashFunction   HashFunction
	miningFunction MiningFunction
	miningCallback MiningCallback

	ctx    context.Context
	cancel context.CancelFunc

	register        Register
	networkClient   NetworkClient
	networkListener NetworkListener

	IP                string
	port              int
	address           string
	protocol          string
	P2PClientTemplate P2PClientInterface
	P2PServer         P2PServiceInterface

	registerAddress  string
	registerProtocol string

	waitGroup sync.WaitGroup

	balance         Balance
	mining          Mining
	miningQueue     MiningQueue
	transactionPool TransactionPool
	storage         Storage
	broadcastQueue  BroadcastQueue
	blockTemplate   Block

	miningInterval int64
}

// InitKey ...
func (host *HippoHost) InitKey(curve elliptic.Curve) {
	host.curve = curve
	host.key.New(curve)
	host.key.GenerateKey()
	logger.Info("key:", host.key.ToAddress())
}

// InitLogger ...
func (host *HippoHost) InitLogger(debug bool) {
	initLogger()
	if debug {
		logger.WithDebug()
	} else {
		logger.WithoutDebug()
	}
	logger.WithColor()
}

// InitLocals ...
func (host *HippoHost) InitLocals(
	ctx context.Context,
	hashFunction HashFunction,
	miningFunction MiningFunction,
	miningThreads int,
	p2pClientTemplate P2PClientInterface,
	broadcastQueueLen uint,

	miningCallbackFunction MiningCallback,
	difficultyFunction DifficultyFunc,
	interval int64,

	miningCapacity int,
	miningTTL int64,
	protocol string,

) {
	host.ctx, host.cancel = context.WithCancel(ctx)
	host.hashFunction = hashFunction
	host.miningFunction = miningFunction
	host.miningFunction.New(host.hashFunction, miningThreads)
	host.miningCallback = miningCallbackFunction
	host.protocol = protocol
	host.miningInterval = interval

	host.balance = new(HippoBalance)
	host.balance.New()

	host.storage = new(HippoStorage)
	host.storage.New()
	host.storage.SetBalance(host.balance)

	host.P2PClientTemplate = p2pClientTemplate
	host.broadcastQueue = new(HippoBroadcastQueue)
	host.broadcastQueue.New(host.ctx, host.protocol,
		host.P2PClientTemplate, broadcastQueueLen)

	host.mining = new(HippoMining)
	host.mining.SetBroadcastQueue(host.broadcastQueue)
	host.mining.SetStorage(host.storage)

	host.miningQueue.New(host.ctx, host.miningCallback,
		host.hashFunction, host.miningFunction)
	host.miningQueue.SetBroadcastQueue(host.broadcastQueue)
	host.miningQueue.SetStorage(host.storage)

	host.transactionPool = new(HippoTransactionPool)
	host.transactionPool.New(host.balance)
	host.mining.New(&host.miningQueue, host.transactionPool,
		difficultyFunction, host.miningInterval, miningCapacity, miningTTL,
		host.balance, host.key)

}

// InitNetwork ...
func (host *HippoHost) InitNetwork(
	blockTemplate Block,
	maxNeighbors int,
	updateTimeBase int,
	updateTimeRand int,

	registerAddress string,
	registerProtocol string,
) {
	if host.localMode {
		host.IP = "localhost"
	} else {
		host.IP = registerlib.GetOutboundIP().String()
	}
	host.blockTemplate = blockTemplate
	host.blockTemplate.New([]byte{}, 0, host.hashFunction,
		0, host.balance, host.curve)

	host.networkListener = new(HippoNetworkListener)
	host.networkListener.New(host.ctx, host.IP, host.protocol)
	host.networkListener.Listen()

	logger.Info("listener: create")

	host.address = host.networkListener.NetworkAddress()
	logger.Info("listener:", host.address)

	host.P2PServer = new(P2PServer)
	host.P2PServer.new(host.ctx, host.networkListener.Listener())
	host.P2PServer.setBroadcastQueue(host.broadcastQueue)
	host.P2PServer.setStorage(host.storage)
	host.P2PServer.setBlockTemplate(host.blockTemplate)
	host.P2PServer.serve()

	host.registerAddress = registerAddress
	host.registerProtocol = registerProtocol
	host.register = new(HippoRegister)
	host.register.New(host.ctx, host.registerAddress, host.registerProtocol)

	logger.Info("register: create")

	host.networkClient = new(HippoNetworkClient)
	host.networkClient.New(host.ctx, host.address, host.protocol,
		maxNeighbors, host.register, updateTimeBase,
		updateTimeRand, host.P2PClientTemplate, host.blockTemplate)
	host.broadcastQueue.SetNetworkClient(host.networkClient)
	logger.Info("network client: created")
}

// Run ...
// Use `go host.Run()`
func (host *HippoHost) Run() {
	host.waitGroup.Add(1)
	host.broadcastQueue.Run()
	host.miningQueue.Run(&host.waitGroup)
	host.networkClient.SyncNeighbors()
	go host.mining.WatchSendNewBlock()

	host.networkClient.StartSyncBlocks(host.storage)

	genesisBlock := CreateGenesisBlock(host.hashFunction,
		host.curve, host.key)
	host.mining.Mine(&genesisBlock)

	go watchStorageBalance(host.storage, host.balance,
		20)
	logger.Info("host running")
	host.waitGroup.Wait()
}

// New ...
func (host *HippoHost) New(debug bool, curve elliptic.Curve,
	localMode bool) {
	host.InitLogger(debug)
	host.InitKey(curve)
	host.localMode = localMode
}

// Close ...
func (host *HippoHost) Close() {
	logger.Info("host: closed")
	host.cancel()
}

func hash(key []byte) []byte {
	bytes := sha256.Sum256(key)
	return bytes[:]
}

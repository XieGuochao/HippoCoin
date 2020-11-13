package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"log"
	"net"
	"sync"
	"time"

	"github.com/XieGuochao/HippoCoinRegister/lib"
)

// HashFunction ...
type HashFunction func([]byte) []byte

// ==============================

// MiningFunction ...
type MiningFunction func(ctx context.Context, close context.CancelFunc,
	PreviousHash []byte, numBytes uint, numThreads uint,
	hashFunction HashFunction) (nonce uint32, found bool)

// DifficultyFunction ...
type DifficultyFunction func(difficulty uint32) uint

// RewardFunction ...
type RewardFunction func(b *HippoBlock) uint32

// ===============================================

// P2PClients ...
// An string (address) to *P2PClient map
type P2PClients struct {
	m sync.Map
}

func (p *P2PClients) store(k string, v *P2PClient) {
	p.m.Store(k, v)
}

func (p *P2PClients) load(k string) (v *P2PClient, has bool) {
	value, has := p.m.Load(k)
	if has {
		v = value.(*P2PClient)
	}
	return
}

func (p *P2PClients) delete(k string) {
	p.m.Delete(k)
}

// Range ...
func (p *P2PClients) Range(f func(k string, v *P2PClient) bool) {
	p.m.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(*P2PClient))
	})
}

// func (m *P2PClients) store(v interface{}) {
// 	m.Store(v)
// }

// HippoCoinClient ...
// The client for HippoCoin
type HippoCoinClient struct {
	// Keys
	privateKey     *ecdsa.PrivateKey
	curve          elliptic.Curve
	privateKeyFile string

	// States
	balance            *Balance
	balanceShowContext context.Context
	balanceShowCancel  context.CancelFunc
	storage            *HippoStorage

	// Constants
	maxTransaction int
	hashFunction   HashFunction
	numberThreads  uint

	// Mining
	miningFunction MiningFunction
	// miningContext        context.Context
	// miningCancel         context.CancelFunc
	currentMining        *Mining
	difficultyToNumBytes DifficultyFunction
	MineDeadline         time.Duration
	rewardFunction       RewardFunction
	miningPool           *MiningPool
	mineMainContext      context.Context
	mineMainCancel       context.CancelFunc
	initDifficulty       uint32

	//Network
	registerClient *lib.Client
	// update neighbors by calling UpdateNeighbors and delete when we lose connection
	neighbors       *sync.Map
	maxNeighbors    int
	registerAddress string
	protocol        string
	listener        net.Listener
	outerIP         string
	port            int
	registerContext context.Context
	registerCancel  context.CancelFunc
	serverContext   context.Context
	serverCancel    context.CancelFunc
	p2pClients      *P2PClients

	// broadcast related
	broadcastChan     chan BroadcastStruct
	broadcastContext  context.Context
	broadcastCancel   context.CancelFunc
	maxBroadcastLevel uint
	maxQueryLevel     uint32

	// new block
	newBlock        chan *HippoBlock
	newBlockContext context.Context
	newBlockCancel  context.CancelFunc

	// sync
	syncQuery   chan QueryLevelStruct
	syncContext context.Context
	syncCancel  context.CancelFunc
}

// SHA256 ...
func SHA256(input []byte) []byte {
	s := crypto.SHA256.New()
	s.Write(input)
	hash := s.Sum(nil)
	return hash
}

// NewHippoCoinClient ...
func NewHippoCoinClient() *HippoCoinClient {
	return new(HippoCoinClient)
}

// Init ...
func (client *HippoCoinClient) Init(data map[string]interface{}) error {
	// Constant
	client.maxTransaction = 100
	client.hashFunction = SHA256

	nThreads, has := data["numberThreads"].(uint)
	if has {
		client.numberThreads = nThreads
	} else {
		client.numberThreads = 3
	}

	mineDeadline, has := data["mineDeadline"].(time.Duration)
	if has {
		client.MineDeadline = mineDeadline
	} else {
		client.MineDeadline = time.Minute * 10 // 10 minutes by default
	}

	maxNeighbor, has := data["maxNeighbor"].(int)
	if has {
		client.maxNeighbors = maxNeighbor
	} else {
		client.maxNeighbors = 10
	}

	client.neighbors = new(sync.Map)

	registerAddress, has := data["registerAddress"].(string)
	if has {
		client.registerAddress = registerAddress
	} else {
		client.registerAddress = "localhost:9325"
	}

	protocol, has := data["protocol"].(string)
	if has {
		client.protocol = protocol
	} else {
		client.protocol = "tcp"
	}

	client.maxBroadcastLevel = 15
	client.maxQueryLevel = 5

	miningFunction, has := data["miningFunction"].(MiningFunction)
	if has {
		client.miningFunction = miningFunction
	} else {
		client.miningFunction = SingleMine
	}

	rewardFunction, has := data["rewardFunction"].(RewardFunction)
	if has {
		client.rewardFunction = rewardFunction
	} else {
		client.rewardFunction = basicReward
	}

	initDifficulty, has := data["initDifficulty"].(uint32)
	if has {
		client.initDifficulty = initDifficulty
	} else {
		client.initDifficulty = 22
	}

	difficultyToNumBytes, has := data["difficultyToNumBytes"].(DifficultyFunction)
	if has {
		client.difficultyToNumBytes = difficultyToNumBytes
	} else {
		client.difficultyToNumBytes = DifficultyToNumBytes
	}

	client.storage = NewHippoStorage(client)

	maxPoolSize, has := data["maxPoolSize"].(int)
	if has {
		client.miningPool = NewMiningPool(maxPoolSize, client)
	} else {
		client.miningPool = NewMiningPool(50, client)
	}

	// Check if the public key and private key are parsed in data.
	// If not, generate a new key pair.
	client.curve = elliptic.P256()
	keyfile, has := data["privateKeyFile"].(string)
	if !has {
		client.privateKeyFile = "key-default.pem"
	} else {
		client.privateKeyFile = keyfile
	}

	if client.PrivateKey() == nil {
		if client.GenerateKeyPair() == nil {
			log.Fatalln("Error generating key pair.")
		}
	}

	client.balance = new(Balance)
	client.balance.init()
	client.balanceShowContext, client.balanceShowCancel = context.WithCancel(context.Background())

	client.mineMainContext, client.mineMainCancel = context.WithCancel(context.Background())
	log.Println("coin init done")

	return nil
}

// SyncMain ...
func (client *HippoCoinClient) SyncMain(wg *sync.WaitGroup) {
	client.syncContext, client.syncCancel = context.WithCancel(context.Background())
	client.syncQuery = make(chan QueryLevelStruct, 10)
	client.Sync()
	// Note: We may need to mine our own genesis.

	lastTime := time.Now()
	for {
		select {
		case <-client.syncContext.Done():
			log.Println("sync stop.")
			wg.Done()

			return
		default:
			if time.Since(lastTime) < time.Second*10 {
				time.Sleep((time.Second * 10).Truncate(time.Since(lastTime)))
			}
			lastTime = time.Now()
			client.Sync()
		}
	}
}

// Sync ...
// Sync by querying neighbors.
func (client *HippoCoinClient) Sync() {
	// First, get the current level.
	log.Println("sync start")
	level := client.storage.topLevel()

	for {
		var levelStruct = QueryLevelStruct{}
		if level <= 1 {
			levelStruct = QueryLevelStruct{
				Level0: 0,
				Level1: client.maxQueryLevel,
			}

		} else {
			levelStruct = QueryLevelStruct{
				Level0: uint32(level - 1),
				Level1: uint32(level-1) + client.maxQueryLevel,
			}
		}
		log.Println("sync level:", levelStruct)

		client.querySend(levelStruct)

		// Query from neighbors

		newLevel := client.storage.topLevel()
		log.Println("current top level:", newLevel)

		if newLevel == level {
			break
		}
		level = newLevel
	}
	log.Println("sync end")
}

// Run ...
func (client *HippoCoinClient) Run() error {
	var wg sync.WaitGroup
	wg.Add(5)
	client.StartNetwork()
	go client.Listen(&wg)
	go client.ListenBroadcast(&wg)
	go client.mineMain(&wg)
	go client.SyncMain(&wg)
	go client.ShowBalance(&wg)
	wg.Wait()
	log.Println("bye bye")
	return nil
}

// Stop ...
func (client *HippoCoinClient) Stop() {
	// client.currentMining.cancel()
	client.mineMainCancel()
	client.broadcastCancel()
	client.serverCancel()
	client.syncCancel()
	client.balanceShowCancel()
	log.Println("stopping")
}

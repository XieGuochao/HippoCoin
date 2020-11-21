package main

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"sync"
	"time"
)

// Host ...
type Host interface {
	New()
	Run()
	Close()
}

// HippoHost ...
type HippoHost struct {
	// mining
	transactionPool    TransactionPool
	mining             Mining
	miningQueue        *MiningQueue
	miningCallbackFunc miningCallback
	miningFunction     MiningFunction
	hashFunc           HashFunction

	// store
	balance Balance

	// key
	curve elliptic.Curve
	key   Key

	// context
	ctx    context.Context
	cancel context.CancelFunc
}

// New ...
func (host *HippoHost) New() {

	// const
	host.curve = elliptic.P256()
	host.key.New(elliptic.P224())
	host.key.GenerateKey()
	host.hashFunc = hash

	host.balance = new(HippoBalance)
	host.balance.New()

	host.ctx, host.cancel = context.WithCancel(context.Background())

	host.transactionPool = new(HippoTransactionPool)
	host.miningQueue = new(MiningQueue)
	host.miningQueue.New(host.ctx, host.miningCallbackFunc, host.hashFunc,
		host.miningFunction)

	host.mining = new(HippoMining)
	host.mining.New(host.miningQueue, host.transactionPool, 1000,
		int64(time.Hour.Seconds()), host.balance, host.key)
}

// Run ...
func (host *HippoHost) Run() {
	var wg sync.WaitGroup
	wg.Add(1)
	defer host.cancel()
	host.miningQueue.Run(&wg)
	wg.Wait()
	logger.Info("host has been closed.")
}

// Close ...
func (host *HippoHost) Close() {
	host.cancel()
}

func hash(key []byte) []byte {
	bytes := sha256.Sum256(key)
	return bytes[:]
}

package main

import (
	"context"
	"sync"
)

type miningCallback func(has bool, block Block, storage Storage, bq BroadcastQueue)

// MiningQueue ...
// Steps:
// 1. New(callback, hashFunction,miningFunc, threads)
// 2. Run(wg)
// 3. Add(block)
// 4. Cancel() or Close()
type MiningQueue struct {
	// const and variables
	threads int
	channel chan HippoBlock
	wg      *sync.WaitGroup

	// functions
	callback     miningCallback
	hashFunction HashFunction
	miningFunc   MiningFunction

	// context
	context       context.Context
	cancel        context.CancelFunc
	queueContext  context.Context
	queueCancel   context.CancelFunc
	parentContext context.Context

	storage        Storage
	broadcastQueue BroadcastQueue
}

// New ...
func (m *MiningQueue) New(parentContext context.Context, callback miningCallback,
	hashFunction HashFunction, miningFunc MiningFunction) {
	m.setCallback(callback)
	m.setMiningFunc(miningFunc)
	m.channel = make(chan HippoBlock, 30)
	m.hashFunction = hashFunction
	m.parentContext = parentContext
	m.context, m.cancel = context.WithCancel(parentContext)
}

// SetBroadcastQueue ...
func (m *MiningQueue) SetBroadcastQueue(bq BroadcastQueue) {
	m.broadcastQueue = bq
}

// SetStorage ...
func (m *MiningQueue) SetStorage(storage Storage) {
	m.storage = storage
}

// Run ...
func (m *MiningQueue) Run(wg *sync.WaitGroup) {
	m.wg = wg
	m.queueContext, m.queueCancel = context.WithCancel(m.parentContext)
	go m.main()
}

func (m *MiningQueue) main() {
	// var block HippoBlock
	defer logger.Debug("wg done")
	defer m.wg.Done()

	for {
		select {
		case block := <-m.channel:
			logger.Debug("new block to mining queue:", block.Hash())
			m.cancel()
			m.context, m.cancel = context.WithCancel(context.Background())

			// m.miningFunc.New(m.context, m.hashFunction, m.threads)
			result, newBlock := m.miningFunc.Solve(block)
			logger.Info("mining:", result)
			if result {
				logger.Info("mining result:", newBlock.Hash())
			}
			m.callback(result, &newBlock, m.storage, m.broadcastQueue)
		case <-m.queueContext.Done():
			logger.Debug("mining queue closed.")
			return
		}
	}
}

func (m *MiningQueue) setHashFunction(f HashFunction) {
	m.hashFunction = f
}

func (m *MiningQueue) setCallback(f miningCallback) {
	m.callback = f
}

func (m *MiningQueue) setMiningFunc(f MiningFunction) {
	m.miningFunc = f
}

// Add ...
func (m *MiningQueue) add(block HippoBlock) {
	m.channel <- block
	logger.Debug("add block to mining queue:", block.Hash())
}

// Cancel ...
func (m *MiningQueue) Cancel() {
	m.cancel()
}

// Close ...
func (m *MiningQueue) Close() {
	logger.Debug("mining queue Close()")
	m.queueCancel()
}

func miningCallbackLog(has bool, block Block, storage Storage, bq BroadcastQueue) {
	logger.Info("has:", has)
	if has {
		logger.Info("mine a block:", block)
		logger.Info("mine check:", block.CheckNonce(), block.Check())
	}
}

func miningCallbackBroadcastSave(has bool, block Block, storage Storage, bq BroadcastQueue) {
	if has {
		logger.Info("mine a block:", block, block.Check())
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			if storage != nil {
				storage.Add(block)
			} else {
				logger.Error("empty storage")
			}
			wg.Done()
		}()

		go func() {
			if bq != nil {
				broadcastBlock := BroadcastBlock{
					Block:     block,
					Level:     0,
					Addresses: make(map[string]bool),
				}
				bq.Add(broadcastBlock)
			} else {
				logger.Error("empty broadcast queue")
			}
			wg.Done()
		}()

		wg.Wait()
		logger.Infof("broadcast save block %s done.", block.Hash())
	}
}

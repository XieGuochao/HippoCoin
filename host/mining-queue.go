package host

import (
	"context"
	"sync"
)

// MiningCallback ...
type MiningCallback func(has bool, block Block, storage Storage, bq BroadcastQueue)

// MiningQueue ...
// Steps:
// 1. New(callback, hashFunction,miningFunc, threads)
// SetBroadcastQueue(bq)
// SetStorage(storage)
// SetTransactionPool(tp)
// 2. Run(wg)
// 3. Add(block)
// 4. Cancel() or Close()
type MiningQueue struct {
	// const and variables
	threads int
	channel chan HippoBlock
	wg      *sync.WaitGroup

	// functions
	callback     MiningCallback
	hashFunction HashFunction
	miningFunc   MiningFunction

	// context
	context       context.Context
	cancel        context.CancelFunc
	queueContext  context.Context
	queueCancel   context.CancelFunc
	parentContext context.Context

	storage         Storage
	broadcastQueue  BroadcastQueue
	transactionPool TransactionPool

	miningStatus chan bool
}

// New ...
func (m *MiningQueue) New(parentContext context.Context, callback MiningCallback,
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

// SetTransactionPool ...
func (m *MiningQueue) SetTransactionPool(transactionPool TransactionPool) {
	m.transactionPool = transactionPool
}

// Run ...
func (m *MiningQueue) Run(wg *sync.WaitGroup) {
	m.wg = wg
	m.queueContext, m.queueCancel = context.WithCancel(m.parentContext)
	go m.main()
}

func (m *MiningQueue) main() {
	// var block HippoBlock
	m.miningStatus = make(chan bool, 0)
	defer debugLogger.Debug("wg done")
	defer m.wg.Done()

	for {
		select {
		case block := <-m.channel:
			debugLogger.Debug("new block to mining queue:", block.Hash())
			m.cancel()
			m.context, m.cancel = context.WithCancel(m.queueContext)
			m.storage.SetMiningCancel(m.cancel)

			// m.miningFunc.New(m.context, m.hashFunction, m.threads)
			result, newBlock := m.miningFunc.Solve(m.context, block)
			infoLogger.Debug("mining:", result)
			if result {
				infoLogger.Info("mining success result:", newBlock.Hash())
			}
			m.callback(result, &newBlock, m.storage, m.broadcastQueue)
			infoLogger.Debug("mining continue to mine:")
			m.miningStatus <- true
			infoLogger.Debug("mining queue: trigger mining status")
		case <-m.queueContext.Done():
			debugLogger.Debug("mining queue closed.")
			return
		}
	}
}

// WaitMining ...
func (m *MiningQueue) WaitMining() { <-m.miningStatus }

func (m *MiningQueue) setHashFunction(f HashFunction) {
	m.hashFunction = f
}

func (m *MiningQueue) setCallback(f MiningCallback) {
	m.callback = f
}

func (m *MiningQueue) setMiningFunc(f MiningFunction) {
	m.miningFunc = f
}

// Add ...
func (m *MiningQueue) add(block HippoBlock) {
	m.channel <- block
	debugLogger.Debug("add block to mining queue:", block.Hash())
}

// Cancel ...
func (m *MiningQueue) Cancel() {
	m.cancel()
}

// Close ...
func (m *MiningQueue) Close() {
	debugLogger.Debug("mining queue Close()")
	m.queueCancel()
}

func miningCallbackLog(has bool, block Block, storage Storage, bq BroadcastQueue) {
	infoLogger.Debug("has:", has)
	if has {
		infoLogger.Debug("mine a block:", block)
		infoLogger.Debug("mine check:", block.CheckNonce(), block.Check())
	}
}

// MiningCallbackBroadcastSave ...
func MiningCallbackBroadcastSave(has bool, block Block, storage Storage, bq BroadcastQueue) {
	if has {
		infoLogger.Debug("mine a block:", block, block.Check())
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			if storage != nil {
				storage.Add(block)
			} else {
				infoLogger.Error("empty storage")
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
				infoLogger.Error("empty broadcast queue")
			}
			wg.Done()
		}()

		wg.Wait()
		infoLogger.Infof("broadcast save block %s done.", block.Hash())
	}
}

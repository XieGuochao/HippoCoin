package main

import (
	"sync"
	"time"
)

// Mining ...
// Steps:
// 1. Initialize miningQueue
// 2. Initialize transactionPool
// 3. Prepare balance
// 4. mining.New(q, tp, capacity, TTL, balance, key)
// mining.WatchSendNewBlock()
// 5. block = Fetch(block)
// 6. block = mining.Sign(block)
// 7. mining.Mine(block)
// 8. mining.Broadcast(block)
// 9. mining.Cancel()
// 10. mining.Stop()
type Mining interface {
	New(q *MiningQueue, tp TransactionPool, capacity int,
		TTL int64, balance Balance, key Key)
	SetStorage(storage Storage)
	SetBroadcastQueue(bq BroadcastQueue)
	Fetch(b Block) Block
	Sign(b Block)
	Mine(b Block)
	Broadcast(b Block)
	Stop()

	WatchSendNewBlock()
}

// HippoMining ...
// HippoMining is thread-safe.
type HippoMining struct {
	queue           *MiningQueue
	transactionPool TransactionPool
	storage         Storage
	broadcastQueue  BroadcastQueue

	minedHash     sync.Map
	blockCapacity int
	TTL           int64

	balance Balance
	// miner
	key Key
}

// New ...
func (m *HippoMining) New(q *MiningQueue, tp TransactionPool, capacity int,
	TTL int64, balance Balance, key Key) {
	m.queue = q
	m.blockCapacity = capacity
	m.transactionPool = tp
	m.TTL = TTL
	m.balance = balance
	m.key = key
}

// SetStorage ...
func (m *HippoMining) SetStorage(storage Storage) { m.storage = storage }

// SetBroadcastQueue ...
func (m *HippoMining) SetBroadcastQueue(bq BroadcastQueue) { m.broadcastQueue = bq }

// Fetch ...
// Fetch transactions into a block.
func (m *HippoMining) Fetch(b Block) Block {
	currentTime := time.Now().Unix()
	transactions := m.transactionPool.Fetch(m.blockCapacity, func(t Transaction) bool {
		if t.GetTimestamp()+m.TTL > currentTime {
			return false
		}
		if _, has := m.minedHash.Load(t.Hash()); has {
			return false
		}
		return true
	})
	b.SetTransactions(transactions)
	return b
}

// WatchSendNewBlock ...
// Watch and create new block to mine.
func (m *HippoMining) WatchSendNewBlock() {
	for {
		select {
		case <-m.queue.queueContext.Done():
			logger.Debug("mining: listen send new block stop.")
			return

		case <-m.queue.miningStatus:
			// logger.Info("channel length:", len(m.channel))
			logger.Info("mining queue chenged")

			if len(m.queue.channel) == 0 {
				if m.storage == nil {
					logger.Error("hippo mining: no storage.")
					continue
				}
				if m.transactionPool == nil {
					logger.Error("hippo mining: no transaction pool.")
					continue
				}

				var block Block
				block = new(HippoBlock)
				prevBlock := m.storage.GetTopBlock()
				block.New(prevBlock.HashBytes(), prevBlock.GetNumBytes(), prevBlock.GetHashFunction(),
					prevBlock.GetLevel()+1, prevBlock.GetBalance(), prevBlock.GetCurve())
				block = m.Fetch(block)

				block.Sign(m.key)
				logger.Info("block level:", block.GetLevel())
				m.Mine(block)
			}
			// default:
			// time.Sleep(time.Second)
		}

	}
}

// Sign ...
// Sign a block after fetching.
func (m *HippoMining) Sign(b Block) {
	b.Sign(m.key)
}

// Mine ...
func (m *HippoMining) Mine(b Block) {
	b0, ok := b.(*HippoBlock)
	if ok {
		m.queue.add(*b0)
	}
}

// Broadcast ...
func (m *HippoMining) Broadcast(b Block) {
	miningCallbackBroadcastSave(b != nil, b, m.storage, m.broadcastQueue)
}

// Cancel ...
func (m *HippoMining) Cancel() { m.queue.cancel() }

// Stop ...
func (m *HippoMining) Stop() {
	m.queue.Close()
	m.Cancel()
}

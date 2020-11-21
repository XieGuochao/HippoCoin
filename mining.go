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
// 5. block = Fetch(block)
// 6. block = mining.Sign(block)
// 7. mining.Mine(*block)
// 8. mining.Cancel()
// 9. mining.Stop()
type Mining interface {
	New(q *MiningQueue, tp TransactionPool, capacity int,
		TTL int64, balance Balance, key Key)
	Fetch(b *HippoBlock) *HippoBlock
	Sign(b *HippoBlock)
	MineBroadcast()
	Stop()
}

// HippoMining ...
// HippoMining is thread-safe.
type HippoMining struct {
	queue           *MiningQueue
	transactionPool TransactionPool

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

// Fetch ...
// Fetch transactions into a block.
func (m *HippoMining) Fetch(b *HippoBlock) *HippoBlock {
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
	b.Transactions = transactions
	return b
}

// Sign ...
// Sign a block after fetching.
func (m *HippoMining) Sign(b *HippoBlock) {
	b.Sign(m.key)
}

// Mine ...
func (m *HippoMining) Mine(b HippoBlock) {
	m.queue.add(b)
}

// MineBroadcast ...
func (m *HippoMining) MineBroadcast() {

}

// Cancel ...
func (m *HippoMining) Cancel() { m.queue.cancel() }

// Stop ...
func (m *HippoMining) Stop() {
	m.queue.Close()
	m.Cancel()
}

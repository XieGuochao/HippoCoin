package host

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
	New(q *MiningQueue, tp TransactionPool,
		difficultyFunction DifficultyFunc,
		miningInterval int64, capacity int,
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

	minedHash          sync.Map
	blockCapacity      int
	TTL                int64
	difficultyFunction DifficultyFunc
	miningInterval     int64

	balance Balance
	// miner
	key Key
}

// New ...
func (m *HippoMining) New(q *MiningQueue, tp TransactionPool,
	difficultyFunction DifficultyFunc,
	miningInterval int64, capacity int,
	TTL int64, balance Balance, key Key) {
	m.queue = q
	m.blockCapacity = capacity
	m.transactionPool = tp
	m.TTL = TTL
	m.balance = balance
	m.key = key
	m.difficultyFunction = difficultyFunction
	m.miningInterval = miningInterval
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
		// infoLogger.Warn(t.GetTimestamp(), m.TTL, currentTime)
		if t.GetTimestamp()+m.TTL < currentTime {
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
			debugLogger.Debug("mining: listen send new block stop.")
			return

		case <-m.queue.miningStatus:
			// infoLogger.Debug("channel length:", len(m.channel))
			infoLogger.Info("mining queue chenged")

			if len(m.queue.channel) == 0 {
				if m.storage == nil {
					infoLogger.Error("hippo mining: no storage.")
					continue
				}
				if m.transactionPool == nil {
					infoLogger.Error("hippo mining: no transaction pool.")
					continue
				}

				var block Block
				block = new(HippoBlock)
				var prevBlock Block
				for prevBlock = m.storage.GetTopBlock(); prevBlock == nil; prevBlock = m.storage.GetTopBlock() {
					infoLogger.Error("no top block")
					time.Sleep(time.Second * time.Duration(10))
				}
				newDifficulty := m.difficultyFunction(prevBlock, m.storage,
					m.miningInterval)
				block.New(prevBlock.HashBytes(), newDifficulty, prevBlock.GetHashFunction(),
					prevBlock.GetLevel()+1, prevBlock.GetBalance(), prevBlock.GetCurve())
				block = m.Fetch(block)

				block.Sign(m.key)
				debugLogger.Debug("block level:", block.GetLevel())
				m.Mine(block)
			}
		default:
			time.Sleep(time.Second)
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
	MiningCallbackBroadcastSave(b != nil, b, m.storage, m.broadcastQueue)
}

// Cancel ...
func (m *HippoMining) Cancel() { m.queue.cancel() }

// Stop ...
func (m *HippoMining) Stop() {
	m.queue.Close()
	m.Cancel()
}

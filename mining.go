package main

import (
	"container/heap"
	"context"
	"log"
	"sync"
	"time"
)

// Mining ...
type Mining struct {
	blockChan chan HippoBlock
	ctx       context.Context
	cancel    context.CancelFunc

	client *HippoCoinClient

	nonce uint32
	found bool
}

// NewMining ...
func NewMining(block HippoBlock, client *HippoCoinClient) *Mining {
	m := new(Mining)
	m.blockChan = make(chan HippoBlock, 10)
	m.client = client

	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
	return m
}

// Push ...
func (m *Mining) Push(block HippoBlock) {
	m.cancel()
	m.blockChan <- block
	log.Println("push block successfully")
}

// Start ...
func (m *Mining) Start() {
	log.Println("start mining:")
	// go func() {
	// 	for {
	// 		select {
	// 		case <-m.ctx.Done():
	// 			m.ctx, m.cancel = context.WithCancel(context.Background())
	// 		case block :<- m.blockChan:
	// 			nonce, found := m.block.Mine(block)
	// 			m.nonce, m.found = nonce, found

	// 		}
	// 	}

	// }()
}

// Cancel ...
func (m *Mining) Cancel() {
	log.Println("mining canceled:")
	m.cancel()
}

// =====================================

// MiningPool ...
// A collection of transactions.
type MiningPool struct {
	lock    sync.Mutex
	pq      TransactionPQ
	maxSize int
	client  *HippoCoinClient
}

// NewMiningPool ...
func NewMiningPool(maxSize int, client *HippoCoinClient) *MiningPool {
	pool := new(MiningPool)
	heap.Init(&pool.pq)
	pool.maxSize = maxSize
	pool.client = client
	return pool
}

// Add ...
func (pool *MiningPool) Add(t HippoTransaction) bool {
	if t.HashValue == [32]byte{} {
		t.UpdateHash(pool.client)
	}
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if pool.pq.In(t) {
		return false
	}

	if len(pool.pq) >= pool.maxSize {
		return false
	}

	pool.pq.Push(t)
	return true
}

// Delete ...
func (pool *MiningPool) Delete(t HippoTransaction) {
	if t.HashValue == [32]byte{} {
		t.UpdateHash(pool.client)
	}
	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.pq.Delete(t)
}

// DeleteBlock ...
func (pool *MiningPool) DeleteBlock(b *HippoBlock) {
	for _, t := range b.Transactions {
		pool.Delete(t)
	}
}

// Pop ...
func (pool *MiningPool) Pop() *HippoTransaction {
	value := heap.Pop(&pool.pq)
	if value == nil {
		return nil
	}
	return value.(*HippoTransaction)
}

// ======================================

// TransactionPQ ...
// A sync-safe priority queue for transactions.
type TransactionPQ []HippoTransaction

// Len ...
func (pq TransactionPQ) Len() int { return len(pq) }

// Less ...
func (pq TransactionPQ) Less(i, j int) bool { return pq[i].Fee > pq[j].Fee }

// Swap ...
func (pq TransactionPQ) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

// Push ...
func (pq *TransactionPQ) Push(x interface{}) {
	*pq = append(*pq, x.(HippoTransaction))
}

// Pop ...
func (pq *TransactionPQ) Pop() interface{} {
	n := len(*pq)
	x := (*pq)[n-1]
	*pq = (*pq)[0 : n-1]
	return x
}

// Delete ...
func (pq *TransactionPQ) Delete(x HippoTransaction) {
	for i, t := range *pq {
		if t.HashValue == x.HashValue {
			*pq = append((*pq)[:i], (*pq)[i+1:]...)
		}
	}
	heap.Init(pq)
}

// In ...
// Check if a transaction is in the pool.
func (pq *TransactionPQ) In(x HippoTransaction) bool {
	for _, b := range *pq {
		if b.HashValue == x.HashValue {
			return true
		}
	}
	return false
}

// =========================================

func (client *HippoCoinClient) mineMain(wg *sync.WaitGroup) {
	client.newBlockContext, client.newBlockCancel = context.WithCancel(context.Background())
	client.newBlock = make(chan *HippoBlock, 10)
	var newBlock *HippoBlock
	log.Println("start mining main.")

	// Start to mine our genesis
	go func() {
		block := NewHippoBlock([32]byte{}, client.initDifficulty, 0, client)
		client.newBlock <- block
	}()

	var currentHash, prevHash string
	for {
		select {
		case newBlock = <-client.newBlock:
			// We should handle the new block now
			currentHash = ByteToHexString(newBlock.Hash(client))
			prevHash = ByteToHexString(newBlock.PreviousHash[:])
			log.Println("new block for mining:", currentHash)
			client.newBlockCancel()
			client.newBlockContext, client.newBlockCancel = context.WithCancel(context.Background())
			log.Println("new block for mining:", currentHash)
			go mineBroadcast(client.newBlockContext, client.newBlockCancel,
				client, *newBlock, client.miningFunction)
		case <-client.mineMainContext.Done():
			log.Println("stop mining main.")
			wg.Done()
			return
		default:
			time.Sleep(time.Second)
			log.Printf("current mining: [%d] %s %s", client.storage.topLevel(), prevHash, currentHash)
		}
	}
}

func mineBroadcast(ctx context.Context, cancel context.CancelFunc,
	client *HippoCoinClient, block HippoBlock,
	miningFunction MiningFunction) {

	block.SignBlock(client)
	prevHash := block.GetPreviousHash(client)

	log.Println("start mine broadcast:", ByteToHexString(block.Hash(client)))

	nonce, found := miningFunction(ctx, cancel, prevHash,
		client.difficultyToNumBytes(block.Difficulty), client.numberThreads,
		client.hashFunction)
	if found {
		log.Println("successfully mine, now broadcast")
		block.Nonce = nonce

		client.storage.Add(&block)
		// broadcast
		broadcastStruct := BroadcastStruct{
			Data:  block,
			Level: 1,
			Type:  "block",
		}
		client.broadcastChan <- broadcastStruct
		log.Println("add to broadcast chan")
	} else {
		log.Println("fail to mine")
	}
}

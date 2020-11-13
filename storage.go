package main

import (
	"log"
	"sync"
)

// HippoStorageBlock ...
type HippoStorageBlock struct {
	block     *HippoBlock
	validated bool // Has found its previous all to root.
}

// HippoStorage ...
type HippoStorage struct {
	blockMap            *BlockMap
	blockLevel          *BlockLevel
	previousBlockMap    *PreviousBlockMap
	client              *HippoCoinClient
	currentLongestValid *HippoBlock
	addLock             sync.Mutex
}

func (st *HippoStorage) topLevel() int {
	if st.currentLongestValid == nil {
		return -1
	}
	return int(st.currentLongestValid.Level)
}

// NewHippoStorage ...
func NewHippoStorage(client *HippoCoinClient) (st *HippoStorage) {
	st = new(HippoStorage)
	st.blockMap = new(BlockMap)
	st.blockLevel = new(BlockLevel)
	st.previousBlockMap = new(PreviousBlockMap)
	st.client = client
	st.currentLongestValid = nil
	return
}

// Get ...
func (st *HippoStorage) Get(hash string) (block *HippoBlock, has bool, validated bool) {
	blockStorage, has := st.blockMap.load(hash)
	block, validated = blockStorage.block, blockStorage.validated
	return block, has, validated
}

// GetStorageBlock ...
func (st *HippoStorage) GetStorageBlock(hash string) (sb *HippoStorageBlock, has bool) {
	return st.blockMap.load(hash)
}

// GetInLevel ...
func (st *HippoStorage) GetInLevel(hash string, level uint32) (block *HippoBlock, has bool, validated bool) {
	blockLevel, has := st.blockLevel.load(level)
	if !has {
		return nil, false, false
	}
	sb, has := blockLevel.load(hash)
	if !has {
		return nil, false, false
	}
	block, validated = sb.block, sb.validated
	return
}

// In ...
// Check if the block is in storage using its level,
// if yes, also return the HippoStorage.
func (st *HippoStorage) In(b *HippoBlock) (*HippoStorageBlock, bool) {
	level := b.Level

	hash := ByteToHexString(b.Hash(st.client))
	log.Println("hash", hash)

	// block, has := st.blockMap.load(hash)
	// return block, has

	blockMap, has := st.blockLevel.load(level)
	log.Println("blockmap:", blockMap)
	if !has {
		return nil, has
	}

	block, has := blockMap.load(hash)
	log.Println(block, has)
	return block, has
}

// PreviousBlock ...
// Return the previous block, has, and validated.
func (st *HippoStorage) PreviousBlock(hash string) (previous *HippoBlock, has bool, validated bool) {
	previousHash, has := st.previousBlockMap.load(hash)
	if !has {
		return
	}
	previous, has, validated = st.Get(previousHash)
	return
}

// CheckValidated ...
// Check whether a block is validated by looking for its previous block only.
func (st *HippoStorage) CheckValidated(b *HippoBlock) bool {
	level := b.Level
	previousHash := ByteToHexString(b.PreviousHash[:])
	if level == 0 {
		return true
	}
	_, has, validated := st.GetInLevel(previousHash, uint32(level-1))
	log.Println("previous block:", previousHash, uint32(level-1))

	// If previous block not exists, return false
	if !has {
		log.Println("no previous block")
		return false
	}

	// If previous block is validated, return true
	if validated {
		return true
	}
	return false
}

// UpdateLongestValid ...
// Given a new block added, try to update the longest valid block.
func (st *HippoStorage) UpdateLongestValid(b *HippoStorageBlock) {
	log.Printf("update longest valid: [%d] %s", b.block.Level, ByteToHexString(b.block.Hash(st.client)))

	// First, if it is invalid, return
	if !b.validated {
		return
	}

	log.Println("update longest valid: pass validated")

	// Then, go further to its successor
	// There might be multiple,
	// Require DFS.
	level := b.block.Level
	blockLevel, has := st.blockLevel.load(level + 1)
	prevHash := b.block.Hash32(st.client)

	// No successor
	if !has {
		log.Println("update longest valid: no successor")
		if st.currentLongestValid == nil || level > st.currentLongestValid.Level {
			log.Println("update longest valid: current is longest")

			st.currentLongestValid = b.block

			// Switch block here! Stop mining and switch to this.
			st.SwitchCurrentBlock()
		}
		return
	}

	// DFS on successors
	blockLevel.Range(func(hash string, v *HippoStorageBlock) bool {
		if v.block.PreviousHash == prevHash {
			st.SetValidated(hash, true)
			st.UpdateLongestValid(v)
		}
		return true
	})
}

// SetValidated ...
// Set validated for a hash.
func (st *HippoStorage) SetValidated(hash string, value bool) {
	sb, has := st.GetStorageBlock(hash)
	if has {
		sb.validated = value
	}
}

// SwitchCurrentBlock ...
func (st *HippoStorage) SwitchCurrentBlock() {
	st.client.newBlockCancel()
	newBlock := st.client.GenerateNewBlock()
	st.client.newBlock <- &newBlock
}

// GenerateNewBlock ...
// Generate new block by
// 1. Pack transactions.
// 2. Sign the blbock.
func (client *HippoCoinClient) GenerateNewBlock() HippoBlock {
	log.Println("generate new block")

	var previousHash [32]byte
	copy(previousHash[:], client.storage.currentLongestValid.Hash(client))
	level := uint32(client.storage.topLevel() + 1)
	transactions := make([]*HippoTransaction, 0)

	// Add transactions !!!

	block := NewHippoBlock(previousHash, client.initDifficulty, level, client)
	for _, t := range transactions {
		block.AppendTransaction(t, client)
	}

	block.SignBlock(client)
	log.Println("generated a new block")
	return *block
}

// Add ...
// Add a block to storage,
// change the balance,
// and check if it has its previous or next block.
func (st *HippoStorage) Add(b *HippoBlock) (added bool, has bool) {
	st.addLock.Lock()
	sb := new(HippoStorageBlock)
	hash := ByteToHexString(b.Hash(st.client))

	// Check if exists
	_, existed := st.In(b)
	if existed {
		return true, true
	}

	// Validate the block
	if !b.Valid(st.client) {
		return false, false
	}

	// Change the balance
	go func(b *HippoBlock, balance *Balance, reward RewardFunction) {
		for _, t := range b.Transactions {
			t.UpdateBalance(b.MinerAddress, balance)
		}
		balance.increase(b.MinerAddress, int64(reward(b)))
	}(b, st.client.balance, st.client.rewardFunction)

	// Get validated value
	sb.validated = st.CheckValidated(b)
	sb.block = b

	// BlockMap
	st.blockMap.store(hash, sb)

	// BlockLevel
	blockMap, has := st.blockLevel.load(b.Level)
	if !has {
		blockMap = new(BlockMap)
		st.blockLevel.store(b.Level, blockMap)
	}
	blockMap.store(hash, sb)

	// PreviousHash
	st.previousBlockMap.store(ByteToHexString(b.PreviousHash[:]), hash)

	st.UpdateLongestValid(sb)

	log.Println("add block to storage:", hash)
	st.addLock.Unlock()
	return true, false
}

// HasLevel ...
// Check if a level has any block.
func (st *HippoStorage) HasLevel(level uint32) bool {
	if st.topLevel() < int(level) {
		return false
	}
	blockMap, has := st.blockLevel.load(level)
	if has {
		if !blockMap.empty() {
			return true
		}
	}
	return false
}

// HasGenesis ...
func (st *HippoStorage) HasGenesis() bool {
	return st.HasLevel(0)
}

// GetBlocks ...
// Get all validated blocks from level0 to level1.
func (st *HippoStorage) GetBlocks(level0, level1 uint32) []HippoBlock {
	blocks := make([]HippoBlock, 0)
	if level0 > level1 {
		return blocks
	}
	if level1 < level0+st.client.maxQueryLevel {
		level1 = level0 + st.client.maxQueryLevel
	}

	for l := level0; l <= level1; l++ {
		bm, has := st.blockLevel.load(l)
		if !has {
			break
		}
		validatedLevel := false
		bm.Range(func(hash string, v *HippoStorageBlock) bool {
			if v.validated {
				blocks = append(blocks, *(v.block))
				validatedLevel = true
			}
			return true
		})
		if !validatedLevel {
			break
		}
	}
	return blocks
}

// =======================================================

// BlockMap ...
// A string (current hash) to *HippoStorageBlock map
type BlockMap struct {
	m sync.Map
}

func (b *BlockMap) store(hash string, v *HippoStorageBlock) {
	b.m.Store(hash, v)
}

func (b *BlockMap) load(hash string) (v *HippoStorageBlock, has bool) {
	var value interface{}
	value, has = b.m.Load(hash)
	if has {
		v = value.(*HippoStorageBlock)
	}
	return
}

func (b *BlockMap) delete(hash string) {
	b.m.Delete(hash)
}

func (b *BlockMap) empty() bool {
	isEmpty := true
	b.m.Range(func(k, v interface{}) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}

// Range ...
func (b *BlockMap) Range(f func(hash string, v *HippoStorageBlock) bool) {
	b.m.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(*HippoStorageBlock))
	})
}

// PreviousBlockMap ...
// A hash to hash map
type PreviousBlockMap struct {
	m sync.Map
}

func (p *PreviousBlockMap) store(previous string, next string) {
	p.m.Store(previous, next)
}

func (p *PreviousBlockMap) load(previous string) (next string, has bool) {
	value, has := p.m.Load(previous)
	if has {
		next = value.(string)
	}
	return
}

func (p *PreviousBlockMap) delete(previous uint32) {
	p.m.Delete(previous)
}

// BlockLevel ...
// An int32 (Level) to *BlockMap map
type BlockLevel struct {
	m sync.Map
}

func (b *BlockLevel) store(level uint32, v *BlockMap) {
	b.m.Store(level, v)
}

func (b *BlockLevel) load(level uint32) (v *BlockMap, has bool) {
	value, has := b.m.Load(level)
	if has {
		v = value.(*BlockMap)
	}
	return
}

func (b *BlockLevel) delete(k uint32) {
	b.m.Delete(k)
}

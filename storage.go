package main

import (
	"context"
	"encoding/json"
	"sync"
)

// Storage ...
type Storage interface {
	New()
	Add(block Block) bool
	AddBlocks(blocks []Block)
	CheckVerified(hashkey string) bool
	UpdateVerified(hashkey string)
	Get(hashKey string) (Block, bool)
	GetOneFromLevel(levelNum int) Block
	UpdateChild(hashKey string)
	MaxLevel() int
	TryUpdateMaxLevel(level int) int
	GetTopBlock() Block
	GetBlocksLevel(level0, level1 int) []Block
	GetBlocksLevelHash(level0, level1 int) []string
	FilterNewHashes(hash []string) (result []string)

	GetLastInterval() int64

	Count() int
	AllHashes() []string
	AllHashesInLevel() map[int][]string

	SetMiningCancel(cancelFunc context.CancelFunc)
	CheckMiningCancel(level int) bool
	SetBalance(Balance)
}

// HippoStorage ...
type HippoStorage struct {
	// block
	blockLock sync.Mutex
	blocks    map[string]Block

	// Level
	levelLock sync.Mutex
	levels    map[int]map[Block]bool // It will not proceed to level N+1 if level N is empty.

	maxLevelLock sync.Mutex
	maxLevel     int

	// child
	child sync.Map // []string

	// verified
	verified sync.Map

	// mining
	miningCancel context.CancelFunc

	// balance
	balance Balance
}

// New ...
func (storage *HippoStorage) New() {
	storage.blocks = make(map[string]Block)
	storage.levels = make(map[int]map[Block]bool)
	storage.maxLevel = -1
}

// Locks ========================================

// LockBlock ...
func (storage *HippoStorage) LockBlock() {
	storage.blockLock.Lock()
}

// UnlockBlock ...
func (storage *HippoStorage) UnlockBlock() {
	storage.blockLock.Unlock()
}

// LockLevel ...
func (storage *HippoStorage) LockLevel() {
	storage.levelLock.Lock()
}

// UnlockLevel ...
func (storage *HippoStorage) UnlockLevel() {
	storage.levelLock.Unlock()
}

// ============================================

// SetBalance ...
func (storage *HippoStorage) SetBalance(balance Balance) { storage.balance = balance }

// Add ...
func (storage *HippoStorage) Add(block Block) bool {
	if !block.Check() {
		logger.Error("block check failed:", block.Hash())
		return false
	}

	logger.Info("storage add:", block.Hash())

	storage.LockBlock()
	h := block.Hash()

	parentHash := block.ParentHash()
	if _, has := storage.blocks[h]; has {
		// We have stored this block
		storage.UnlockBlock()
		return true
	}
	// A new block
	storage.blocks[h] = block
	storage.UnlockBlock()

	if storage.miningCancel != nil && storage.CheckMiningCancel(block.GetLevel()) {
		logger.Info("cancel mining and mine the new")
		storage.miningCancel()
		storage.miningCancel = nil
	}

	storage.LockLevel()
	l, has := storage.levels[block.GetLevel()]
	if !has {
		storage.levels[block.GetLevel()] = make(map[Block]bool)
		l = storage.levels[block.GetLevel()]
	}
	l[block] = true
	storage.UnlockLevel()

	// Update child information
	newChild := make([]string, 1)
	newChild[0] = h
	child, loaded := storage.child.LoadOrStore(parentHash, newChild)
	childSlice := child.([]string)
	if loaded {
		child = append(childSlice, h)
		storage.child.Store(parentHash, childSlice)
	}

	// Update child's verification
	if (block.GetLevel() == 0 && block.ParentHash() == "") || storage.CheckVerified(parentHash) {
		storage.UpdateVerified(h)
		storage.UpdateChild(h)
	}

	// Update balance from genesis
	balance := storage.balance
	if balance != nil {
		balance := storage.balance
		balance.Lock()
		balance.New()
		mainChain := storage.GetMainChain()
		logger.Debug("main chain:", mainChain)
		if mainChain != nil {
			for _, b := range mainChain {
				balanceChange := b.GetBalanceChange()
				for address, value := range balanceChange {
					balance.UpdateUnsafe(address, value)
				}
			}
		} else {
			logger.Error("storage: cannot update balance")
		}
		balance.Unlock()
	} else {
		logger.Error("storage: no balance")
	}
	logger.Debug("update balance end.")
	logger.Debug("balance:", storage.balance.AllBalance())

	return false
}

// AddBlocks ...
func (storage *HippoStorage) AddBlocks(blocks []Block) {
	for _, b := range blocks {
		storage.Add(b)
	}
}

// SetMiningCancel ...
func (storage *HippoStorage) SetMiningCancel(cancelFunc context.CancelFunc) {
	storage.miningCancel = cancelFunc
}

// CheckMiningCancel ...
func (storage *HippoStorage) CheckMiningCancel(level int) bool {
	currentTopLevel := storage.MaxLevel()
	return level > currentTopLevel
}

// CheckVerified ...
func (storage *HippoStorage) CheckVerified(hashkey string) bool {
	if _, has := storage.verified.Load(hashkey); has {
		return true
	}
	return false
}

// UpdateVerified ...
func (storage *HippoStorage) UpdateVerified(hashkey string) {
	storage.verified.Store(hashkey, true)
}

// Get ...
func (storage *HippoStorage) Get(hashKey string) (Block, bool) {
	storage.LockBlock()
	defer storage.UnlockBlock()
	if b, has := storage.blocks[hashKey]; has {
		return b, true
	}
	return nil, false
}

// GetOneFromLevel ...
func (storage *HippoStorage) GetOneFromLevel(levelNum int) Block {
	storage.LockLevel()
	defer storage.UnlockLevel()
	var (
		level map[Block]bool
		has   bool
	)
	if level, has = storage.levels[levelNum]; !has {
		return nil
	}
	for block := range level {
		return block
	}
	return nil
}

// UpdateChild ...
// Update the child verification by tracing to its child.
func (storage *HippoStorage) UpdateChild(hashKey string) {
	var (
		has bool
	)
	_, has = storage.verified.Load(hashKey)
	if !has {
		return
	}
	child, has := storage.child.Load(hashKey)
	if !has {
		block, _ := storage.Get(hashKey)
		storage.TryUpdateMaxLevel(block.GetLevel())
		return
	}
	childList := child.([]string)
	for _, childHash := range childList {
		storage.UpdateVerified(childHash)
		storage.UpdateChild(childHash)
	}

	return
}

// MaxLevel ...
func (storage *HippoStorage) MaxLevel() int {
	storage.maxLevelLock.Lock()
	defer storage.maxLevelLock.Unlock()
	return storage.maxLevel
}

// TryUpdateMaxLevel ...
func (storage *HippoStorage) TryUpdateMaxLevel(level int) int {
	storage.maxLevelLock.Lock()
	defer storage.maxLevelLock.Unlock()
	if level > storage.maxLevel {
		storage.maxLevel = level
	}
	return storage.maxLevel
}

// GetTopBlock ...
func (storage *HippoStorage) GetTopBlock() Block {
	maxLevel := storage.MaxLevel()
	if maxLevel == -1 {
		return nil
	}
	if block := storage.GetOneFromLevel(maxLevel); block != nil {
		return block
	}
	return nil
}

// GetBlocksLevel ...
func (storage *HippoStorage) GetBlocksLevel(level0, level1 int) (blocks []Block) {
	blocks = make([]Block, 0)
	if level1 < level0 || level0 > storage.MaxLevel() {
		return
	}
	storage.LockLevel()
	defer storage.UnlockLevel()
	for level := level0; level <= level1; level++ {
		levelBlocks, has := storage.levels[level]
		if !has {
			continue
		}
		for block := range levelBlocks {
			blocks = append(blocks, block)
		}
	}
	return
}

// GetBlocksLevelHash ...
func (storage *HippoStorage) GetBlocksLevelHash(level0, level1 int) (haashes []string) {
	haashes = make([]string, 0)
	if level1 < level0 || level0 > storage.MaxLevel() {
		return
	}
	storage.LockLevel()
	defer storage.UnlockLevel()
	for level := level0; level <= level1; level++ {
		levelBlocks, has := storage.levels[level]
		if !has {
			continue
		}
		for block := range levelBlocks {
			haashes = append(haashes, block.Hash())
		}
	}
	return
}

// GetMainChain ...
func (storage *HippoStorage) GetMainChain() []Block {
	var has bool
	block := storage.GetTopBlock()
	if block == nil {
		return make([]Block, 0)
	}
	level := block.GetLevel()
	blocks := make([]Block, level+1)

	for i := level; i >= 0; i-- {
		blocks[i] = block
		if i > 0 {
			block, has = storage.Get(block.ParentHash())
			if !has {
				logger.Error("storage: get main chain failed:", block.ParentHash())
				return nil
			}
		}
	}
	return blocks
}

// GetLastInterval ...
// Return -1 if not available.
func (storage *HippoStorage) GetLastInterval() int64 {
	topBlock := storage.GetTopBlock()
	if topBlock == nil {
		return -1
	}
	if topBlock.GetLevel() == 0 {
		return -1
	}
	prevBlock, has := storage.Get(topBlock.ParentHash())
	if !has {
		return -1
	}
	return topBlock.GetTimestamp() - prevBlock.GetTimestamp()
}

// FilterNewHashes ...
func (storage *HippoStorage) FilterNewHashes(hash []string) (result []string) {
	storage.LockBlock()
	defer storage.UnlockBlock()
	result = make([]string, 0)
	for _, h := range hash {
		if _, has := storage.blocks[h]; !has {
			result = append(result, h)
		}
	}
	return result
}

// Count ...
func (storage *HippoStorage) Count() int {
	storage.LockBlock()
	defer storage.UnlockBlock()
	return len(storage.blocks)
}

// AllHashes ...
func (storage *HippoStorage) AllHashes() []string {
	storage.LockBlock()
	defer storage.UnlockBlock()
	hashes := make([]string, len(storage.blocks))
	i := 0
	for h := range storage.blocks {
		hashes[i] = h
		i++
	}
	return hashes
}

// AllHashesInLevel ...
func (storage *HippoStorage) AllHashesInLevel() map[int][]string {
	storage.LockLevel()
	defer storage.UnlockLevel()
	newMap := make(map[int][]string)
	for k, v := range storage.levels {
		newMap[k] = make([]string, 0)
		for b := range v {
			newMap[k] = append(newMap[k], b.Hash())
		}
	}
	return newMap
}

// EncodeBlocks ...
func EncodeBlocks(blocks []Block) []byte {
	data, _ := json.Marshal(blocks)
	return data
}

// DecodeBlocks ...
func DecodeBlocks(bytes []byte) []Block {
	var blocks []Block
	json.Unmarshal(bytes, &blocks)
	return blocks
}

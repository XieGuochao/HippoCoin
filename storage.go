package main

import (
	"encoding/json"
	"sync"
)

// Storage ...
type Storage interface {
	New()
	Add(block Block) bool
	CheckVerified(hashkey string) bool
	UpdateVerified(hashkey string)
	Get(hashKey string) (Block, bool)
	GetOneFromLevel(levelNum int) Block
	UpdateChild(hashKey string)
	MaxLevel() int
	TryUpdateMaxLevel(level int) int
	GetTopBlock() Block
	GetBlocksLevel(level0, level1 int) []Block
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

// Add ...
func (storage *HippoStorage) Add(block Block) bool {
	if !block.Check() {
		return false
	}

	storage.LockBlock()
	h := block.Hash()

	parentHash := block.ParentHash()
	if _, has := storage.blocks[h]; has {
		storage.UnlockBlock()
		return true
	}
	storage.blocks[h] = block
	storage.UnlockBlock()

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

	return false
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

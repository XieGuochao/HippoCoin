package main

import (
	"context"
	"crypto/sha256"
	"math"
	"math/rand"
	"sync"
)

// MiningFunction ...
type MiningFunction interface {
	New(ctx context.Context, hashFunction HashFunction, threads int)
	Solve(block HippoBlock) (result bool, newBlock HippoBlock)
}

// single mining
type singleMiningFunction struct {
	ctx context.Context
	// block        HippoBlock
	hashFunction HashFunction
	callback     miningCallback
	seed         int64
}

func (m *singleMiningFunction) New(ctx context.Context,
	hashFunction HashFunction, threads int) {
	m.ctx, m.hashFunction = ctx, hashFunction
	logger.Debug("use single mining")
}

func (m *singleMiningFunction) SetSeed(seed int64) {
	m.seed = seed
}

func (m *singleMiningFunction) Solve(block HippoBlock) (result bool, newBlock HippoBlock) {
	found, nonce := mineBase(m.ctx, block.HashSignatureBytes(), block.NumBytes,
		m.hashFunction, m.seed, 0)
	if found {
		block.Nonce = nonce
		return true, block
	}
	return false, HippoBlock{}
}

// multiple mining
type multipleMiningFunction struct {
	ctx          context.Context
	hashFunction HashFunction
	threads      int
	seed         int64
}

func (m *multipleMiningFunction) New(ctx context.Context,
	hashFunction HashFunction, threads int) {
	m.ctx, m.hashFunction, m.threads = ctx, hashFunction, threads
	logger.Debug("use multiple mining:", threads)
}

func (m *multipleMiningFunction) SetThreads(threads int) {
	m.threads = threads
}

func (m *multipleMiningFunction) SetSeed(seed int64) {
	m.seed = seed
}

func (m *multipleMiningFunction) Solve(block HippoBlock) (result bool, newBlock HippoBlock) {
	wg := new(sync.WaitGroup)
	wg.Add(m.threads)
	var once sync.Once
	var totalNonce uint32

	miningContext, miningCancel := context.WithCancel(m.ctx)
	defer miningCancel()
	for i := 0; i < m.threads; i++ {
		go func(ctx context.Context, cancel context.CancelFunc, i int) {
			logger.Debug("start thread:", i)
			found, nonce := mineBase(ctx, block.HashSignatureBytes(), block.NumBytes,
				m.hashFunction, (m.seed+int64(i))%math.MaxInt64, i)
			if found {
				once.Do(func() {
					totalNonce = nonce
					result = true
					cancel()
				})
			}
			wg.Done()
		}(miningContext, miningCancel, i)
	}
	wg.Wait()
	logger.Info("multiple mining solved:", totalNonce, result)
	if result {
		block.Nonce = totalNonce
		return true, block
	}
	return false, HippoBlock{}
}

// HashFunction ...
type HashFunction func([]byte) []byte

func hashWithNonce(previousHash []byte, nonce uint32, hash HashFunction) []byte {
	full := append(previousHash, Uint32ToBytes(nonce)...)
	return hash(full)
}

// Check if the nonce hash satisfies the difficulty requirement.
func compareHashLen(hash []byte, numBytes uint) bool {
	return ByteToNumDigits(hash) < numBytes
}

func checkNonce(previousHash []byte, nonce uint32, numBytes uint, hash HashFunction) bool {
	sum := hashWithNonce(previousHash, nonce, hash)
	sb := sha256.Sum256([]byte(sum))
	sumBytes := sb[:]
	return compareHashLen(sumBytes, numBytes)
}

func mineBase(ctx context.Context, baseHash []byte, numBytes uint,
	hashFunction HashFunction, seed int64, threadID int) (found bool, nonce uint32) {
	logger.Debug("mineBase numBytes:", numBytes)
	logger.Debug("baseHash:", ByteToHexString(baseHash))
	rand.Seed(seed)
	found = false

	count := 0
	for {
		select {
		case <-ctx.Done():
			logger.Infof("[%d] mine finished", threadID)
			return
		default:
			for t := 0; t <= 1000; t++ {
				nonce = rand.Uint32()
				if checkNonce(baseHash, nonce, numBytes, hashFunction) {
					logger.Debugf("[%d] found: %d", threadID, nonce)
					return true, nonce
				}
			}
			count++
			if count%1000 == 0 {
				logger.Infof("[%d] current progress: %d * 10^6", threadID, count/1000)
			}
		}
	}
}

package host

import (
	"context"
	"crypto/sha256"
	"math"
	"math/rand"
	"sync"
)

// MiningFunction ...
type MiningFunction interface {
	New(hashFunction HashFunction, threads int)
	Solve(ctx context.Context, block HippoBlock) (result bool, newBlock HippoBlock)
}

// SingleMiningFunction ...
type SingleMiningFunction struct {
	// block        HippoBlock
	hashFunction HashFunction
	callback     MiningCallback
	seed         int64
}

func (m *SingleMiningFunction) New(hashFunction HashFunction, threads int) {
	m.hashFunction = hashFunction
	logger.Debug("use single mining")
}

func (m *SingleMiningFunction) SetSeed(seed int64) {
	m.seed = seed
}

func (m *SingleMiningFunction) Solve(ctx context.Context,
	block HippoBlock) (result bool, newBlock HippoBlock) {
	found, nonce := mineBase(ctx, block.HashSignatureBytes(), block.NumBytes,
		m.hashFunction, block.Level, m.seed, 0)
	if found {
		block.Nonce = nonce
		return true, block
	}
	return false, HippoBlock{}
}

// multiple mining
type MultipleMiningFunction struct {
	hashFunction HashFunction
	threads      int
	seed         int64
}

func (m *MultipleMiningFunction) New(hashFunction HashFunction, threads int) {
	m.hashFunction, m.threads = hashFunction, threads
	logger.Debug("use multiple mining:", threads)
}

func (m *MultipleMiningFunction) SetThreads(threads int) {
	m.threads = threads
}

func (m *MultipleMiningFunction) SetSeed(seed int64) {
	m.seed = seed
}

func (m *MultipleMiningFunction) Solve(ctx context.Context, block HippoBlock) (result bool, newBlock HippoBlock) {
	wg := new(sync.WaitGroup)
	wg.Add(m.threads)
	var once sync.Once
	var totalNonce uint32

	miningContext, miningCancel := context.WithCancel(ctx)
	defer miningCancel()
	for i := 0; i < m.threads; i++ {
		go func(ctx context.Context, cancel context.CancelFunc, i int) {
			logger.Debug("start thread:", i)
			found, nonce := mineBase(ctx, block.HashSignatureBytes(), block.NumBytes,
				m.hashFunction, block.Level, (m.seed+int64(i))%math.MaxInt64, i)
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
	return Hash(full)
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
	hashFunction HashFunction, level int, seed int64, threadID int) (found bool, nonce uint32) {
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
			if count%5000 == 0 {
				logger.Infof("[%d] current progress [%d %d]: %d * 10^6", threadID,
					numBytes, level, count/1000)
			}
		}
	}
}

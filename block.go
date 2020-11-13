package main

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

// HippoBlock ...
type HippoBlock struct {
	ID             uint64
	Level          uint32
	PreviousHash   [32]byte
	Timestamp      int64
	Difficulty     uint32
	Transactions   []HippoTransaction
	MinerAddress   string
	MinerPublicKey string
	MinerSignature string
	Nonce          uint32
}

// NewHippoBlock ...
// Initialize a HippoBlock based on the client.
func NewHippoBlock(PreviousHash [32]byte, difficulty uint32, level uint32, client *HippoCoinClient) *HippoBlock {
	b := new(HippoBlock)
	b.MinerAddress = client.Address()
	b.MinerPublicKey = publicKeyToString(&client.privateKey.PublicKey)

	b.PreviousHash = PreviousHash
	b.Difficulty = difficulty
	b.Level = level
	b.Timestamp = time.Now().Unix()
	return b
}

// NewHippoBlockFromJSONMap ...
func NewHippoBlockFromJSONMap(data interface{}) HippoBlock {
	block := HippoBlock{}
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Println("err in new block from json map:", err)
		return block
	}
	json.Unmarshal(bytes, &block)
	return block
}

// Encode ...
func (b *HippoBlock) Encode() []byte {
	data, _ := json.Marshal(*b)
	return data
}

// AppendTransaction ...
// Append a transaction into the current block.
func (b *HippoBlock) AppendTransaction(t *HippoTransaction, client *HippoCoinClient) error {
	if !t.Valid(client) {
		log.Println("Invalid block.")
		return errors.New("invalid block")
	}
	b.Transactions = append(b.Transactions)
	return nil
}

// String ...
func (b *HippoBlock) String(client *HippoCoinClient) string {
	result := strconv.FormatUint(b.ID, 10)
	result += strconv.FormatInt(int64(b.Level), 10)
	result += ByteToHexString(b.PreviousHash[:])
	result += strconv.FormatInt(b.Timestamp, 10)
	result += strconv.Itoa(int(b.Difficulty))
	for _, t := range b.Transactions {
		result += ByteToString(t.Hash(client))
	}
	result += b.MinerAddress + b.MinerPublicKey
	return result
}

// Byte ...
// Get the byte data of a block.
func (b *HippoBlock) Byte(client *HippoCoinClient) []byte {
	return []byte(b.String(client))
}

// HexString ...
// Get the hex string of a block.
func (b *HippoBlock) HexString(client *HippoCoinClient) string {
	return ByteToString(b.Byte(client))
}

// Hash ...
func (b *HippoBlock) Hash(client *HippoCoinClient) []byte {
	return client.hashFunction(b.Byte(client))
}

// Hash32 ...
func (b *HippoBlock) Hash32(client *HippoCoinClient) [32]byte {
	h := client.hashFunction(b.Byte(client))
	var h32 [32]byte
	copy(h32[:], h)
	return h32
}

// GetPreviousHash ...
// The previous hash after signing and before solving nonce
func (b *HippoBlock) GetPreviousHash(client *HippoCoinClient) []byte {
	bytes, _ := StringToByte(b.MinerSignature)
	return append(b.Hash(client), bytes...)
}

// Sign ...
// Sign the current block and return the signature.
func (b *HippoBlock) Sign(client *HippoCoinClient) string {
	sig, err := client.privateKey.Sign(cryptoRand.Reader, b.Hash(client), nil)
	if err != nil {
		log.Println("Sign error:", err)
		return ""
	}
	return ByteToString(sig)
}

// SignBlock ...
// Sign the current block.
func (b *HippoBlock) SignBlock(client *HippoCoinClient) error {
	sig := b.Sign(client)
	if sig == "" {
		return errors.New("fail to sign the block")
	}
	b.MinerSignature = sig
	return nil
}

// Mine ...
// Mine a block to satisfy the difficulty.
func (b *HippoBlock) Mine(client *HippoCoinClient) (nonce uint32, found bool) {
	// Mine with multiple go routines.
	log.Println("------ start mining ------")
	client.currentMining = NewMining(*b, client)
	// client.miningContext, client.miningCancel = context.WithTimeout(context.Background(), client.MineDeadline)
	nonce, found = client.miningFunction(client.currentMining.ctx, client.currentMining.cancel, b.GetPreviousHash(client), DifficultyToNumBytes(b.Difficulty), client.numberThreads, client.hashFunction)
	defer client.currentMining.cancel()

	if found {
		b.Nonce = nonce
		log.Printf("------ successfully mined %d ------\n", nonce)
	} else {
		log.Printf("------ failed to mine with difficulty %d ------\n", b.Difficulty)
	}
	return
}

// DifficultyToNumBytes ...
func DifficultyToNumBytes(difficulty uint32) uint {
	if difficulty >= 256 {
		return 1
	}
	return 256 - uint(difficulty)
}

// mineBase function
func mineBase(ctx context.Context, previousHash []byte, numBytes uint, hashFunction HashFunction, seed int64) (nonce uint32, found bool) {
	log.Println("numBytes:", numBytes)
	rand.Seed(seed)
	found = false
	var sum []byte
	t := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("finished")
			return
		default:
			{
				nonce = rand.Uint32()
				sum = NonceHash(previousHash, nonce, hashFunction)
				sumBytes := sha256.Sum256([]byte(sum))
				// digits := ByteToNumDigits(sumBytes[:])
				// log.Println("number of digits:", digits)
				if CheckNonce(sumBytes[:], numBytes) {
					log.Println("found:", nonce, ByteToNumDigits(sumBytes[:]))
					log.Printf("sum: %x\n", sum)
					log.Printf("sumBytes: %x\n", sumBytes)

					return nonce, true
				}
				t++
				if t%1000000 == 0 {
					log.Println("current progress:", t, ByteToNumDigits(sumBytes[:]))
				}
			}
		}

	}
}

// SingleMine ...
func SingleMine(ctx context.Context, close context.CancelFunc, previousHash []byte, numBytes uint, numThreads uint, hashFunction HashFunction) (nonce uint32, found bool) {
	return mineBase(ctx, previousHash, numBytes, hashFunction, time.Now().Unix())
}

// MultiMine ...
func MultiMine(ctx context.Context, close context.CancelFunc, previousHash []byte, numBytes uint, numThreads uint, hashFunction HashFunction) (nonce uint32, found bool) {
	found = false
	seed := time.Now().Unix()
	var wg sync.WaitGroup
	wg.Add(int(numThreads))
	var once sync.Once

	for i := 0; i < int(numThreads); i++ {
		go func(i int) {
			n, f := mineBase(ctx, previousHash, numBytes, hashFunction, (seed+int64(i))%math.MaxInt64)
			if f {
				once.Do(func() {
					nonce = n
					found = true
					close()
				})
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	log.Println("multi mine:", nonce, found)
	return
}

// UpdateNonce ...
func (b *HippoBlock) UpdateNonce(nonce uint32) {
	log.Println("update nonce:", nonce)
	b.Nonce = nonce
}

// NonceHash ...
// Calculate the hash with nonce.
func NonceHash(previousHash []byte, nonce uint32, hashFunction HashFunction) []byte {
	previousHash = append(previousHash, Uint32ToBytes(nonce)...)
	nonceHash := Uint32ToBytes(nonce)
	return hashFunction(append(previousHash, nonceHash...))
}

// CheckNonce ...
// Check if the nonce hash satisfies the difficulty requirement.
func CheckNonce(hash []byte, numBytes uint) bool {
	return ByteToNumDigits(hash) < numBytes
}

// Verify ...
// Verify checks only the signature ...
func (b *HippoBlock) Verify(client *HippoCoinClient) bool {
	// log.Println("B verify public key:", stringToPublicKey(b.MinerPublicKey, client.curve))
	sig, err := StringToByte(b.MinerSignature)
	if err != nil {
		log.Println("B verify sig error")
		return false
	}
	if !Verify(stringToPublicKey(b.MinerPublicKey, client.curve), sig, b.Hash(client), client) {
		log.Println("verify fail: invalid miner signature")
		return false
	}
	return true
}

// VerifyTransactions ...
// Verify all transactions
func (b *HippoBlock) VerifyTransactions(client *HippoCoinClient) bool {
	for _, t := range b.Transactions {
		if !t.Valid(client) {
			log.Println("invalid transaction:", t)
			return false
		}
	}
	return true
}

// Valid ...
// Valid checks everything including
// 1. Block Signature
// 2. Transactions
// 3. Nonce
func (b *HippoBlock) Valid(client *HippoCoinClient) bool {
	// Check block signature
	if !b.Verify(client) {
		log.Println("valid fail: invalid signature")
		return false
	}

	// Check transactions
	if !b.VerifyTransactions(client) {
		log.Println("valid fail: invalid transaction")
		return false
	}

	// Check nonce
	sum := NonceHash(b.GetPreviousHash(client), b.Nonce, client.hashFunction)
	log.Println("nonce:", b.Nonce)
	sumBytes := sha256.Sum256([]byte(sum))
	log.Printf("sum: %x\n", sum)
	log.Printf("sumBytes: %x\n", sumBytes)
	result := CheckNonce(sumBytes[:], DifficultyToNumBytes(b.Difficulty))
	log.Println("valid result:", result)
	return result
}

// Reward ...
// Return the reward of a block using a function.
func (b *HippoBlock) Reward(client *HippoCoinClient) uint32 {
	return client.rewardFunction(b)
}

// A reward function based on level.
func basicReward(b *HippoBlock) uint32 {
	e := 20 - int(b.Level/100)
	if e < 0 {
		e = 0
	}
	reward := uint32(1 << e)
	log.Println("reward:", reward, b.Level, e)
	return reward
}

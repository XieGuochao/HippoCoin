package host

import (
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"time"
)

// Block ...
// Steps:
// 1. New(previousHash, numBytes, hashFunction, Level, balance, curve)
// 2. SetTransactions(tr)
// 3. Sign(key)
// 4. SetNonce(nonce)
type Block interface {
	New(previousHash []byte, numBytes uint, hashFunction HashFunction,
		Level int, balance Balance, curve elliptic.Curve)
	Digest() string
	DigestSignature() string
	HashBytes() []byte
	Hash() string
	HashSignatureBytes() []byte
	HashSignature() string
	ParentHashBytes() []byte
	ParentHash() string
	SetTransactions(tr []Transaction)
	GetTransactions() (tr []Transaction)
	GetNumBytes() uint
	Sign(key Key)
	SetNonce(nonce uint32)
	SetBalance(b Balance)
	GetBalance() Balance
	SetHashFunction(hashFunction HashFunction)
	GetHashFunction() HashFunction
	SetCurve(curve elliptic.Curve)
	GetCurve() elliptic.Curve

	Signature() string
	CheckSignature() bool
	CheckTransactions() bool
	CheckNonce() bool
	Check() bool
	GetLevel() int
	GetBalanceChange() map[string]int64
	GetTimestamp() int64
	GetMiner() string
	GetNonce() uint32

	Encode() []byte

	CopyConstants(block Block)
	CloneConstants() Block
	CopyVariables(b Block)
}

// HippoBlock ...
type HippoBlock struct {
	transactions []Transaction
	PreviousHash []byte `json:"previousHash"`
	NumBytes     uint   `json:"numBytes"`
	Nonce        uint32 `json:"nonce"`
	hashFunction HashFunction
	Level        int   `json:"level"`
	Timestamp    int64 `json:"timestamp"`

	// miner
	MinerAddress   string `json:"minerAddress"`
	MinerSignature string `json:"minerSignature"`

	balance Balance

	curve elliptic.Curve
}

// New ...
func (b *HippoBlock) New(previousHash []byte, numBytes uint,
	hashFunction HashFunction, Level int, balance Balance, curve elliptic.Curve) {
	b.PreviousHash, b.NumBytes, b.hashFunction, b.Level, b.balance, b.curve =
		previousHash, numBytes, hashFunction, Level, balance, curve
	b.Timestamp = time.Now().Unix()
}

// Digest ...
func (b *HippoBlock) Digest() string {
	d := ""
	d += fmt.Sprintf("%d|%d|", b.Timestamp, b.Level)
	for _, t := range b.transactions {
		d += "|" + t.HashSignatures()
	}
	d += "|"
	return d
}

// DigestSignature ...
func (b *HippoBlock) DigestSignature() string { return b.Digest() + b.MinerSignature }

// HashBytes ...
func (b *HippoBlock) HashBytes() []byte {
	return b.hashFunction([]byte(b.Digest()))
}

// HashSignatureBytes ...
func (b *HippoBlock) HashSignatureBytes() []byte {
	return b.hashFunction([]byte(b.DigestSignature()))
}

// HashSignature ...
func (b *HippoBlock) HashSignature() string {
	return ByteToHexString(b.HashSignatureBytes())
}

// Hash ...
func (b *HippoBlock) Hash() string {
	return ByteToHexString(b.HashBytes())
}

// ParentHashBytes ...
func (b *HippoBlock) ParentHashBytes() []byte {
	return b.PreviousHash
}

// ParentHash ...
func (b *HippoBlock) ParentHash() string {
	return ByteToHexString(b.ParentHashBytes())
}

// SetTransactions ...
func (b *HippoBlock) SetTransactions(tr []Transaction) {
	b.transactions = tr
}

// GetTransactions ...
func (b *HippoBlock) GetTransactions() (tr []Transaction) {
	return b.transactions
}

// GetNumBytes ...
func (b *HippoBlock) GetNumBytes() uint { return b.NumBytes }

// SetNonce ...
func (b *HippoBlock) SetNonce(nonce uint32) {
	b.Nonce = nonce
}

// SetBalance ...
func (b *HippoBlock) SetBalance(balance Balance) { b.balance = balance }

// GetBalance ...
func (b *HippoBlock) GetBalance() Balance { return b.balance }

// GetHashFunction ...
func (b *HippoBlock) GetHashFunction() HashFunction { return b.hashFunction }

// SetHashFunction ...
func (b *HippoBlock) SetHashFunction(hashFunction HashFunction) { b.hashFunction = hashFunction }

// GetCurve ...
func (b *HippoBlock) GetCurve() elliptic.Curve { return b.curve }

// SetCurve ...
func (b *HippoBlock) SetCurve(curve elliptic.Curve) { b.curve = curve }

// GetNonce ...
func (b *HippoBlock) GetNonce() uint32 { return b.Nonce }

// CopyConstants ...
func (b *HippoBlock) CopyConstants(block Block) {
	b.SetCurve(block.GetCurve())
	b.SetBalance(block.GetBalance())
	b.SetHashFunction(block.GetHashFunction())
}

// CloneConstants ...
func (b *HippoBlock) CloneConstants() (block Block) {
	block = new(HippoBlock)
	block.CopyConstants(b)
	return block
}

// CopyVariables ...
func (b *HippoBlock) CopyVariables(newBlock Block) {
	b.transactions = newBlock.GetTransactions()
	b.Level = newBlock.GetLevel()
	b.MinerAddress = newBlock.GetMiner()
	b.MinerSignature = newBlock.Signature()
	b.Nonce = newBlock.GetNonce()
	b.NumBytes = newBlock.GetNumBytes()
	b.PreviousHash = newBlock.ParentHashBytes()
	b.Timestamp = newBlock.GetTimestamp()
}

// Sign ...
func (b *HippoBlock) Sign(key Key) {
	if sig, err := b.generateSignature(key); err == nil {
		b.MinerSignature = sig
		b.MinerAddress = key.ToAddress()
	}
}

// Signature ...
func (b *HippoBlock) Signature() string {
	return b.MinerSignature
}

func (b *HippoBlock) generateSignature(key Key) (string, error) {
	var (
		s   []byte
		err error
	)
	if s, err = key.Sign(b.HashBytes()); err != nil {
		infoLogger.Error("block generate sign failed:", err)
		return "", err
	}
	return ByteToString(s), nil
}

// CheckSignature ...
func (b *HippoBlock) CheckSignature() bool {
	var (
		result bool
		key    Key
	)
	key.LoadPublicKeyString(b.MinerAddress, b.curve)

	if result = key.CheckSignString(b.Hash(), b.Signature()); !result {
		infoLogger.Error("signature check failed:", b.Hash())
	}
	return result
}

// CheckTransactions ...
func (b *HippoBlock) CheckTransactions() bool {
	for _, t := range b.transactions {
		if !t.Check(b.balance) {
			infoLogger.Error("transaction check failed:", b.Hash())
			return false
		}
	}
	return true
}

// CheckNonce ...
func (b *HippoBlock) CheckNonce() bool {
	var (
		result bool
	)
	if result = checkNonce(b.HashSignatureBytes(), b.Nonce, b.NumBytes, b.hashFunction); !result {
		infoLogger.Error("nonce check failed:", b.Hash())
	}
	checkNonceShow(b.HashSignatureBytes(), b.Nonce, b.NumBytes, b.hashFunction)
	return result
}

// Check ...
func (b *HippoBlock) Check() bool {
	return b.CheckSignature() && b.CheckTransactions() && b.CheckNonce()
}

// GetLevel ...
func (b *HippoBlock) GetLevel() int { return b.Level }

// GetBalanceChange ...
func (b *HippoBlock) GetBalanceChange() map[string]int64 {
	balanceChange := make(map[string]int64)
	for _, tr := range b.transactions {
		for k, v := range tr.GetBalanceChange() {
			if k == "fee" {
				k = b.MinerAddress
			}
			if _, has := balanceChange[k]; !has {
				balanceChange[k] = v
			} else {
				balanceChange[k] += v
			}
		}
	}
	if _, has := balanceChange[b.MinerAddress]; !has {
		balanceChange[b.MinerAddress] = Reward(b)
	} else {
		balanceChange[b.MinerAddress] += Reward(b)
	}
	return balanceChange
}

// GetTimestamp ...
func (b *HippoBlock) GetTimestamp() int64 { return b.Timestamp }

// GetMiner ...
func (b *HippoBlock) GetMiner() string { return b.MinerAddress }

// =============================================================

// CreateGenesisBlock ...
func CreateGenesisBlock(hashFunction HashFunction,
	curve elliptic.Curve, key Key) HippoBlock {
	var block HippoBlock
	block.New([]byte{}, 235, hashFunction, 0, nil, curve)
	block.Sign(key)
	return block
}

// BlockEncoding ...
type BlockEncoding struct {
	Block        []byte
	Transactions [][]byte
}

// Encode ...
func (b *HippoBlock) Encode() []byte {
	var (
		blockBytes        []byte
		transactionsBytes [][]byte
		err               error
	)
	blockBytes, err = json.Marshal(*b)
	if err != nil {
		infoLogger.Error("encoding block:", err)
		return nil
	}
	transactionsBytes = make([][]byte, len(b.transactions))

	for i, tr := range b.transactions {
		transactionsBytes[i] = tr.Encode()
		if transactionsBytes[i] == nil {
			return nil
		}
	}

	be := BlockEncoding{
		Block:        blockBytes,
		Transactions: transactionsBytes,
	}
	blockBytes, _ = json.Marshal(be)

	debugLogger.Debug("block.Encode:", string(blockBytes))
	return blockBytes
}

// DecodeBlock ...
func DecodeBlock(bytes []byte, templateBlock Block) Block {
	var b Block

	b = templateBlock.CloneConstants()

	var be BlockEncoding
	err := json.Unmarshal(bytes, &be)
	debugLogger.Debug("decoding block 1:", string(bytes))
	if err != nil {
		infoLogger.Error("decode block error:", err)
		return nil
	}

	debugLogger.Debug("decoding block 2:", string(be.Block))
	err = json.Unmarshal(be.Block, b)
	if err != nil {
		infoLogger.Error("decode block error:", err)
		return nil
	}
	b.CopyConstants(templateBlock)

	// Set transactions
	trs := make([]Transaction, len(be.Transactions))
	for i, transactionBytes := range be.Transactions {
		trs[i] = DecodeTransaction(transactionBytes, b.GetHashFunction(),
			b.GetCurve())
		if trs[i] == nil {
			return nil
		}
	}
	b.SetTransactions(trs)

	debugLogger.Debug("decode a block:", b)
	// return b
	return b
}

// DifficultyFunc ...
type DifficultyFunc func(block Block, storage Storage,
	baseInterval int64) uint

// BasicDifficulty ...
func BasicDifficulty(block Block, storage Storage,
	baseInterval int64) uint {
	lastInterval := storage.GetLastInterval()
	debugLogger.Debug("difficulty: lastinterval", lastInterval)
	if lastInterval == -1 {
		return block.GetNumBytes()
	}
	if lastInterval > baseInterval/2 && block.GetNumBytes() < 255 {
		return block.GetNumBytes() + 1
	}
	if lastInterval < baseInterval*2 && block.GetNumBytes() > 0 {
		return block.GetNumBytes() - 1
	}
	return block.GetNumBytes()
}

// StaticDifficulty ...
func StaticDifficulty(block Block, storage Storage,
	baseInterval int64) uint {
	return block.GetNumBytes()
}

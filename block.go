package main

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
	CopyConstants(b Block)
	Signature() string
	CheckSignature() bool
	CheckTransactions() bool
	CheckNonce() bool
	Check() bool
	GetLevel() int
	GetBalanceChange() map[string]int64

	Encode() []byte
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
		d += "|" + t.Hash()
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

// CopyConstants ...
func (b *HippoBlock) CopyConstants(block Block) {
	b.curve = block.GetCurve()
	b.balance = block.GetBalance()
	b.hashFunction = block.GetHashFunction()
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
		logger.Error("block generate sign failed:", err)
		return "", err
	}
	return ByteToString(s), nil
}

// CheckSignature ...
func (b *HippoBlock) CheckSignature() bool {
	var key Key
	key.LoadPublicKeyString(b.MinerAddress, b.curve)
	return key.CheckSignString(b.Hash(), b.Signature())
}

// CheckTransactions ...
func (b *HippoBlock) CheckTransactions() bool {
	for _, t := range b.transactions {
		if !t.Check(b.balance) {
			return false
		}
	}
	return true
}

// CheckNonce ...
func (b *HippoBlock) CheckNonce() bool {
	return checkNonce(b.HashSignatureBytes(), b.Nonce, b.NumBytes, b.hashFunction)
}

// Check ...
func (b *HippoBlock) Check() bool {
	return b.CheckSignature() && b.CheckNonce() && b.CheckTransactions()
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

// CreateGenesisBlock ...
func CreateGenesisBlock(numBytes uint, hashFunction HashFunction,
	curve elliptic.Curve, key Key) HippoBlock {
	var block HippoBlock
	block.New([]byte{}, numBytes, hashFunction, 0, nil, curve)
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
		logger.Error("encoding block:", err)
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
	return blockBytes
}

// DecodeBlock ...
func DecodeBlock(bytes []byte, tempalteBlock Block) Block {
	var b Block
	b = new(HippoBlock)

	var be BlockEncoding
	err := json.Unmarshal(bytes, &be)
	if err != nil {
		logger.Error("decode block error:", err)
		return nil
	}

	err = json.Unmarshal(be.Block, b)
	if err != nil {
		logger.Error("decode block error:", err)
		return nil
	}

	b.CopyConstants(tempalteBlock)

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

	// return b
	return b
}

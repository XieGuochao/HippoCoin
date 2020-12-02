package host

import (
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"time"
)

// Transaction ...
// Steps to use a transaction:
// 1. New(hashFunction, curve)
// 2. SetSender(senderAddresses, senderAmounts)
// 3. SetReceiver(receiverAddresses, receiverAmounts)
// 4. UpdateFee()
// 5. Sign() for all senders.
type Transaction interface {
	New(hashFunction HashFunction, curve elliptic.Curve)
	SetSender(senderAddresses []string,
		senderAmounts []uint64) bool
	SetReceiver(receiverAddresses []string,
		receiverAmounts []uint64) bool
	SenderSum() uint64
	ReceiverSum() uint64
	GetBalanceChange() map[string]int64
	UpdateFee() bool
	CheckFee() bool
	CheckBalance(balance Balance) bool
	Sign(key Key) bool
	SetSignature(address string, signature string) bool
	CheckSignatures() bool
	Check(balance Balance) bool
	Digest() string
	HashBytes() []byte
	Hash() string

	GetTimestamp() int64
	GetFee() uint64

	Encode() []byte
}

// HippoTransaction ...
type HippoTransaction struct {
	SenderAddresses   []string `json:"senderAddresses"`
	SenderAmounts     []uint64 `json:"senderAmounts"`
	ReceiverAddresses []string `json:"receiverAddresses"`
	ReceiverAmounts   []uint64 `json:"receiverAmounts"`
	Fee               uint64   `json:"fee"`
	Timestamp         int64    `json:"timestamp"`
	hashFunction      HashFunction

	SenderSignatures []string `json:"senderSignatures"`
	curve            elliptic.Curve
}

// New ...
func (t *HippoTransaction) New(hashFunction HashFunction, curve elliptic.Curve) {
	t.Timestamp = time.Now().Unix()
	t.hashFunction = hashFunction
	t.curve = curve
}

// SetSender ...
func (t *HippoTransaction) SetSender(senderAddresses []string,
	senderAmounts []uint64) bool {
	if len(senderAddresses) != len(senderAmounts) || len(senderAddresses) <= 0 {
		return false
	}
	t.SenderAddresses, t.SenderAmounts = senderAddresses,
		senderAmounts
	t.SenderSignatures = make([]string, len(senderAddresses))
	return true
}

// SetReceiver ...
func (t *HippoTransaction) SetReceiver(receiverAddresses []string,
	receiverAmounts []uint64) bool {
	if len(receiverAddresses) != len(receiverAmounts) {
		return false
	}
	t.ReceiverAddresses, t.ReceiverAmounts = receiverAddresses, receiverAmounts
	return true
}

// SenderSum ...
func (t *HippoTransaction) SenderSum() uint64 {
	senderSum := uint64(0)
	for _, a := range t.SenderAmounts {
		senderSum += a
	}
	return senderSum
}

// ReceiverSum ...
func (t *HippoTransaction) ReceiverSum() uint64 {
	receiverSum := uint64(0)
	for _, a := range t.ReceiverAmounts {
		receiverSum += a
	}
	return receiverSum
}

// UpdateFee ...
// Return senderSum >= receiverSum
func (t *HippoTransaction) UpdateFee() bool {
	senderSum, receiverSum := t.SenderSum(), t.ReceiverSum()
	if senderSum < receiverSum {
		return false
	}
	t.Fee = senderSum - receiverSum
	return true
}

// CheckFee ...
func (t *HippoTransaction) CheckFee() bool {
	t.UpdateFee()
	return t.SenderSum() == t.ReceiverSum()+t.Fee
}

// CheckBalance ...
// Check with the address balance.
func (t *HippoTransaction) CheckBalance(balance Balance) bool {
	if balance == nil {
		infoLogger.Error("no balance for transaction check")
		return false
	}
	balance.Lock()
	defer balance.Unlock()
	safe := true
	for i, address := range t.SenderAddresses {
		debugLogger.Debug(address, balance.GetUnsafe(address))
		if balance.GetUnsafe(address) < t.SenderAmounts[i] {
			safe = false
			break
		}
	}
	return safe
}

func (t *HippoTransaction) findAddress(address string) int {
	pos := -1
	for i, a := range t.SenderAddresses {
		if a == address {
			pos = i
			break
		}
	}
	return pos
}

// Sign ...
func (t *HippoTransaction) Sign(key Key) bool {
	var (
		err       error
		signature string
	)

	// Check address
	address := key.ToAddress()
	pos := t.findAddress(address)
	if pos == -1 {
		infoLogger.Debug("Cannot sign the transaction:", t.Hash())
		return false
	}
	if signature, err = key.SignString(t.HashBytes()); err == nil {
		t.SenderSignatures[pos] = signature
		return true
	}
	infoLogger.Error("sign transaction error:", err)
	return false
}

// SetSignature ...
func (t *HippoTransaction) SetSignature(address string, signature string) bool {
	var key Key
	pos := t.findAddress(address)
	if pos == -1 {
		debugLogger.Debug("set signature failed: no such address")
		return false
	}

	key.LoadAddress(address, t.curve)
	if !key.CheckSignString(t.Hash(), signature) {
		debugLogger.Debug("set signature failed:", address)
		return false
	}
	t.SenderSignatures[pos] = signature
	return true
}

// CheckSignatures ...
func (t *HippoTransaction) CheckSignatures() bool {
	var key Key
	h := t.Hash()
	for pos, address := range t.SenderAddresses {
		key.LoadAddress(address, t.curve)
		if !key.CheckSignString(h, t.SenderSignatures[pos]) {
			debugLogger.Debug("check signatures failed:", pos, address)
			return false
		}
	}
	return true
}

// Check ...
// Check fee + check balance + check signatures
func (t *HippoTransaction) Check(balance Balance) bool {
	return t.CheckFee() && t.CheckBalance(balance) && t.CheckSignatures()
}

// Digest ...
func (t *HippoTransaction) Digest() string {
	result := ""
	for i := range t.SenderAddresses {
		result += "|" + t.SenderAddresses[i]
		result += fmt.Sprintf("|%d", t.SenderAmounts[i])
	}
	result += "|"
	for i := range t.ReceiverAddresses {
		result += "|" + t.ReceiverAddresses[i]
		result += fmt.Sprintf("|%d", t.ReceiverAmounts[i])
	}
	result += fmt.Sprintf("||%d", t.Timestamp)
	return result
}

// HashBytes ...
func (t *HippoTransaction) HashBytes() []byte {
	return t.hashFunction([]byte(t.Digest()))
}

// Hash ...
func (t *HippoTransaction) Hash() string {
	return ByteToHexString(t.HashBytes())
}

// GetTimestamp ...
func (t *HippoTransaction) GetTimestamp() int64 { return t.Timestamp }

// GetFee ...
func (t *HippoTransaction) GetFee() uint64 { return t.Fee }

// GetBalanceChange ...
func (t *HippoTransaction) GetBalanceChange() map[string]int64 {
	t.UpdateFee()
	result := make(map[string]int64)
	for i := range t.SenderAddresses {
		if v, has := result[t.SenderAddresses[i]]; !has {
			result[t.SenderAddresses[i]] = -int64(v)
		} else {
			result[t.SenderAddresses[i]] -= v
		}
	}
	for i := range t.ReceiverAddresses {
		if v, has := result[t.ReceiverAddresses[i]]; !has {
			result[t.ReceiverAddresses[i]] = int64(v)
		} else {
			result[t.ReceiverAddresses[i]] += v
		}
	}
	result["fee"] = int64(t.Fee)
	return result
}

// Encode ...
func (t *HippoTransaction) Encode() []byte {
	bytes, err := json.Marshal(*t)
	if err != nil {
		infoLogger.Error("encode transaction:", err)
		return nil
	}
	return bytes
}

// DecodeTransaction ...
func DecodeTransaction(bytes []byte, hash HashFunction, curve elliptic.Curve) Transaction {
	tr := new(HippoTransaction)
	err := json.Unmarshal(bytes, tr)
	if err != nil {
		infoLogger.Error("decode transaction error:", err)
		return nil
	}
	tr.hashFunction = hash
	tr.curve = curve
	return tr
}

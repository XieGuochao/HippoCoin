package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
)

// HippoTransaction ...
type HippoTransaction struct {
	FromAddress   []string
	FromAmount    []uint64
	FromPublicKey []string
	FromSignature []string
	ToAddress     []string
	ToAmount      []uint64
	Fee           uint64
	Timestamp     int64
	HashValue     [32]byte
}

// NewHippoTransaction ...
func NewHippoTransaction() *HippoTransaction {
	h := new(HippoTransaction)
	h.Timestamp = time.Now().Unix()
	return h
}

// AppendFrom ...
func (t *HippoTransaction) AppendFrom(address string, amount uint64, publicKey string) {
	t.FromAddress = append(t.FromAddress, address)
	t.FromAmount = append(t.FromAmount, amount)
	t.FromPublicKey = append(t.FromPublicKey, publicKey)
}

// AppendTo ...
func (t *HippoTransaction) AppendTo(address string, amount uint64) {
	t.ToAddress = append(t.ToAddress, address)
	t.ToAmount = append(t.ToAmount, amount)
}

// CalculateFee ...
func (t *HippoTransaction) CalculateFee() error {
	fromSum := uint64(0)
	toSum := uint64(0)
	for _, s := range t.FromAmount {
		fromSum += s
	}
	for _, s := range t.ToAmount {
		toSum += s
	}
	if toSum > fromSum {
		return errors.New("invalid Fee")
	}
	t.Fee = fromSum - toSum
	return nil
}

// SignAll ...
func (t *HippoTransaction) SignAll(privateKeys []*ecdsa.PrivateKey, client *HippoCoinClient) error {
	if len(privateKeys) != len(t.FromAddress) {
		log.Println("Not sufficient privateKeys.")
		return errors.New("not sufficient privateKeys")
	}
	t.FromSignature = make([]string, len(privateKeys))
	for i, privateKey := range privateKeys {
		t.FromSignature[i] = t.Sign(privateKey, client)
	}
	t.UpdateHash(client)
	return nil
}

func checkSliceUnique(slice []string) bool {
	m := make(map[string]bool)
	for _, s := range slice {
		if _, has := m[s]; has {
			return false
		}
		m[s] = true
	}
	return true
}

// Valid ...
// Check the whole transaction and all its from and to information
func (t *HippoTransaction) Valid(client *HippoCoinClient) bool {
	b := client.balance
	curve := client.curve

	// Check the length
	if len(t.FromAddress) != len(t.FromAmount) || len(t.FromPublicKey) != len(t.FromSignature) || len(t.FromAddress) != len(t.FromPublicKey) || len(t.ToAddress) != len(t.ToAmount) {
		log.Println("Transaction length does not match.")
		return false
	}

	if !checkSliceUnique(t.FromAddress) || !checkSliceUnique(t.ToAddress) {
		log.Println("Transaction duplicated addresses.")
		return false
	}

	// Check the sum should be equal
	fromSum := uint64(0)
	toSum := t.Fee
	for _, a := range t.FromAmount {
		fromSum += a
	}
	for _, a := range t.ToAmount {
		toSum += a
	}
	if fromSum != toSum {
		log.Println("Transaction is not balanced.")
		return false
	}

	// Check balance, address, and signature !!!
	for i, address := range t.FromAddress {
		publicKey := stringToPublicKey(t.FromPublicKey[i], curve)
		if currentBalance, has := b.load(address); !has || currentBalance < t.FromAmount[i] {
			log.Println("balance check fail:", currentBalance, t.FromAmount[i])
			return false
		}

		if !ValidatePublicKeyAddress(publicKey, address) {
			log.Println("invalid public key address", publicKey, address)
			return false
		}

		// Check signature
		bytes, err := StringToByte(t.FromSignature[i])
		if err != nil {
			log.Println("encode signature error")
			return false
		}
		if !t.Verify(publicKey, bytes, client) {
			log.Println("signature fail", publicKey)
			return false
		}
	}

	return true
}

// String ...
// Marshall the transaction into a string.
func (t *HippoTransaction) String() (result string) {
	result += strings.Join(t.FromAddress, " ")
	result += strings.Join(t.FromPublicKey, " ")
	for _, a := range t.FromAmount {
		result += strconv.FormatUint(a, 10)
	}
	result += strings.Join(t.ToAddress, " ")
	for _, a := range t.ToAmount {
		result += strconv.FormatUint(a, 10)
	}
	result += strconv.FormatUint(t.Fee, 10)
	result += strconv.FormatInt(t.Timestamp, 10)
	return
}

// Byte ...
// Marshall the transaction into a byte.
func (t *HippoTransaction) Byte() []byte {
	return []byte(t.String())
}

// HexString ...
// Marshall the transaction into a hex string.
func (t *HippoTransaction) HexString() string {
	return ByteToString(t.Byte())
}

// Hash ...
// Get the hash of a transaction.
func (t *HippoTransaction) Hash(client *HippoCoinClient) []byte {
	return client.hashFunction(t.Byte())
}

// Sign ...
// Sign the current transaction using a public key.
func (t *HippoTransaction) Sign(privateKey *ecdsa.PrivateKey, client *HippoCoinClient) string {
	t.UpdateHash(client)
	hash := t.HashValue[:]

	sig, err := privateKey.Sign(rand.Reader, hash, nil)
	if err != nil {
		log.Println("Sign error:", err)
		return ""
	}
	return ByteToString(sig)
}

// Verify ...
// Verify the current transaction, the current publicKey, and the current signature.
func (t *HippoTransaction) Verify(publicKey *ecdsa.PublicKey, sig []byte, client *HippoCoinClient) bool {
	log.Println("T verify public key:", publicKeyToString(publicKey))
	t.UpdateHash(client)
	return Verify(publicKey, sig, t.HashValue[:], client)
}

// UpdateBalance ...
// Update the balance for the current block.
func (t *HippoTransaction) UpdateBalance(miner string, balance *Balance) {
	for i, a := range t.FromAddress {
		balance.increase(a, -int64(t.FromAmount[i]))
	}
	for i, a := range t.ToAddress {
		balance.increase(a, int64(t.ToAmount[i]))
	}
	balance.increase(miner, int64(t.Fee))
}

// UpdateHash ...
// Update the hash value of a transaction.
func (t *HippoTransaction) UpdateHash(client *HippoCoinClient) {
	copy(t.HashValue[:], t.Hash(client))
}

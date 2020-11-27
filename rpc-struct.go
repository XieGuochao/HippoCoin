package main

import "encoding/json"

// NetworkSendInterface ...
type NetworkSendInterface interface {
	Encode() []byte
	SetAddresses(map[string]bool)
	SetLevel(uint)
}

// NetworkReceiveInterface ...
type NetworkReceiveInterface interface {
	Decode(*interface{})
	GetAddresses() map[string]bool
	GetLevel() uint
}

// ==============================================================

// QueryLevelStruct ...
type QueryLevelStruct struct {
	Level0, Level1 int
}

// Encode ...
func (q QueryLevelStruct) Encode() []byte {
	bytes, err := json.Marshal(q)
	if err != nil {
		return make([]byte, 0)
	}
	return bytes
}

// SetAddresses ...
func (q *QueryLevelStruct) SetAddresses(_ map[string]bool) {}

// SetLevel ...
func (q *QueryLevelStruct) SetLevel(_ uint) {}

// ==============================================================

// QueryResponse ...
type QueryResponse struct {
	Data []byte
}

// Decode ...
func (r QueryResponse) Decode(data *[]HippoBlock) {
	json.Unmarshal(r.Data, data)
}

// GetAddresses ...
func (r QueryResponse) GetAddresses() map[string]bool { return nil }

// GetLevel ...
func (r QueryResponse) GetLevel() uint { return 0 }

// ==============================================================

// BroadcastBlock ...
type BroadcastBlock struct {
	Block        Block
	Transactions []Transaction
	Level        uint
	Addresses    map[string]bool
}

// Encode ...
func (b *BroadcastBlock) Encode() []byte {
	var (
		bytes []byte
		err   error
	)
	b.Transactions = b.Block.GetTransactions()
	b.Block.SetTransactions([]Transaction{})
	bytes, err = json.Marshal(b)
	logger.Debug("encode:", err)
	return bytes
}

// SetAddresses ...
func (b *BroadcastBlock) SetAddresses(a map[string]bool) { b.Addresses = a }

// SetLevel ...
func (b *BroadcastBlock) SetLevel(l uint) { b.Level = l }

// ===================================================

// ReceiveBlock ...
type ReceiveBlock struct {
	Data      []byte
	block     *HippoBlock
	addresses map[string]bool
	level     uint
}

// Decode ...
func (r *ReceiveBlock) Decode(b *BroadcastBlock) {
	// logger.Debug("decode:", string(r.Data))
	var (
		dataMap map[string]interface{}
	)
	if r.block == nil {
		r.block = new(HippoBlock)
		err := json.Unmarshal(r.Data, &dataMap)
		if err != nil {
			return
		}
		bytes, err := json.Marshal(dataMap["Block"])

		err = json.Unmarshal(bytes, r.block)

		var trs []HippoTransaction
		bytes, err = json.Marshal(dataMap["Transactions"])

		json.Unmarshal(bytes, &trs)
		r.block.transactions = make([]Transaction, len(trs))
		for i, tr := range trs {
			r.block.transactions[i] = &tr
		}

		bytes, err = json.Marshal(dataMap["Addresses"])
		json.Unmarshal(bytes, &r.addresses)

		bytes, err = json.Marshal(dataMap["Level"])
		json.Unmarshal(bytes, &r.level)
	}
	// logger.Debug("decode:", r.block)
	b.Block = r.block
	b.Transactions = r.block.transactions
	b.Addresses = r.addresses
	b.Level = r.level
}

// GetAddresses ...
func (r *ReceiveBlock) GetAddresses() map[string]bool {
	return r.addresses
}

// GetLevel ...
func (r *ReceiveBlock) GetLevel() uint {
	return r.level
}

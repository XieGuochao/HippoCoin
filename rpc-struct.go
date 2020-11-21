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
	Block     Block
	Level     uint
	Addresses map[string]bool
}

// Encode ...
func (b *BroadcastBlock) Encode() []byte {
	var bytes []byte
	bytes, _ = json.Marshal(*b)
	return bytes
}

// SetAddresses ...
func (b *BroadcastBlock) SetAddresses(a map[string]bool) { b.Addresses = a }

// SetLevel ...
func (b *BroadcastBlock) SetLevel(l uint) { b.Level = l }

// ===================================================

// ReceiveBlock ...
type ReceiveBlock struct {
	Data           []byte
	broadcastBlock *BroadcastBlock
}

// Decode ...
func (r *ReceiveBlock) Decode(b *BroadcastBlock) {
	if r.broadcastBlock == nil {
		json.Unmarshal(r.Data, r.broadcastBlock)
	}
	b = r.broadcastBlock
}

// GetAddresses ...
func (r *ReceiveBlock) GetAddresses() map[string]bool {
	if r.broadcastBlock == nil {
		json.Unmarshal(r.Data, r.broadcastBlock)
	}
	return r.broadcastBlock.Addresses
}

// GetLevel ...
func (r *ReceiveBlock) GetLevel() uint {
	if r.broadcastBlock == nil {
		json.Unmarshal(r.Data, r.broadcastBlock)
	}
	return r.broadcastBlock.Level
}

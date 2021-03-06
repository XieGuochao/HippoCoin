package host

import (
	"sync"
)

// Balance ...
type Balance interface {
	New()
	Lock()
	Unlock()
	Store(address string, value uint64)
	StoreUnsafe(address string, value uint64)
	Get(address string) uint64
	GetUnsafe(address string) uint64
	AllBalance() map[string]uint64
	Update(address string, change int64) (uint64, bool)
	UpdateUnsafe(address string, change int64) (uint64, bool)
}

// HippoBalance ...
type HippoBalance struct {
	lock    sync.Mutex
	balance map[string]uint64
}

// New ...
func (b *HippoBalance) New() {
	b.balance = make(map[string]uint64)
}

// Lock ...
func (b *HippoBalance) Lock() {
	b.lock.Lock()
}

// Unlock ...
func (b *HippoBalance) Unlock() {
	b.lock.Unlock()
}

// Store ...
func (b *HippoBalance) Store(address string, value uint64) {
	b.Lock()
	defer b.Unlock()
	b.StoreUnsafe(address, value)
}

// StoreUnsafe ...
func (b *HippoBalance) StoreUnsafe(address string, value uint64) {
	b.balance[address] = value
}

// Get ...
func (b *HippoBalance) Get(address string) uint64 {
	b.Lock()
	defer b.Unlock()
	return b.GetUnsafe(address)
}

// GetUnsafe ...
func (b *HippoBalance) GetUnsafe(address string) uint64 {
	var (
		value uint64
		has   bool
	)
	if value, has = b.balance[address]; !has {
		value = 0
		b.balance[address] = 0
	}
	return value
}

// AllBalance ...
func (b *HippoBalance) AllBalance() map[string]uint64 {
	b.Lock()
	defer b.Unlock()
	balanceMap := make(map[string]uint64)
	for key, value := range b.balance {
		balanceMap[key] = value
	}
	return balanceMap
}

// Update ...
func (b *HippoBalance) Update(address string, change int64) (uint64, bool) {
	b.Lock()
	defer b.Unlock()
	return b.UpdateUnsafe(address, change)
}

// UpdateUnsafe ...
func (b *HippoBalance) UpdateUnsafe(address string, change int64) (uint64, bool) {
	var (
		value uint64
		has   bool
	)
	if value, has = b.balance[address]; !has {
		value = 0
		b.balance[address] = 0
	}
	if int64(value)+change > 0 {
		value = uint64(int64(value) + change)
		b.balance[address] = value
		return value, true
	}
	return value, false
}

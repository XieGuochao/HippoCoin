package main

import (
	// "log"
	"os"
	"sync"
	"time"

	"github.com/withmandala/go-log"
)

var logger *log.Logger

// ==============================

// Balance ...
// type Balance map[string]uint64
type Balance struct {
	m          sync.Map
	updateLock sync.Mutex
}

func (b *Balance) init() {
	logger = log.New(os.Stderr)
	logger.WithColor()
}

func (b *Balance) load(address string) (uint64, bool) {
	v, has := b.m.Load(address)
	if !has {
		return 0, false
	}
	return v.(uint64), true
}

func (b *Balance) store(address string, value uint64) {
	b.updateLock.Lock()
	defer b.updateLock.Unlock()
	b.m.Store(address, value)
}

func (b *Balance) increase(address string, amount int64) bool {
	b.updateLock.Lock()
	defer b.updateLock.Unlock()
	v, has := b.load(address)
	if !has {
		if amount >= 0 {
			b.m.Store(address, uint64(amount))
			return true
		}
		return false
	}
	if int64(v)+amount >= 0 {
		b.m.Store(address, uint64(int64(v)+amount))
		return true
	}
	return false
}

// Range ...
func (b *Balance) Range(f func(k string, v uint64) bool) {
	b.m.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(uint64))
	})
}

// ToMap ...
func (b *Balance) ToMap() (m map[string]uint64) {
	m = make(map[string]uint64)
	b.Range(func(k string, v uint64) bool {
		m[k] = v
		return true
	})
	return
}

// ShowBalance ...
func (client *HippoCoinClient) ShowBalance(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-client.balanceShowContext.Done():
			logger.Warn("stop show balance")
			return
		default:
			logger.Info("balance:", client.balance.ToMap())
			time.Sleep(time.Duration(10) * time.Second)
		}
	}
}

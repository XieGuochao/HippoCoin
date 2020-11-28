package host

import (
	"container/heap"
	"sync"
)

// TransactionPool ...
// Steps:
// 1. New(balance)
// 2. Push(t)
// 3. result := Fetch(n, checkFunc)
type TransactionPool interface {
	New(balance Balance)
	Lock()
	Unlock()
	Push(t HippoTransaction) bool
	Pop() Transaction
	Len() int
	Fetch(n int, checkFunc transactionPoolCheck) (result []Transaction)
}

// HippoTransactionPool ...
type HippoTransactionPool struct {
	lock sync.Mutex
	heap transactionHeap
	hash map[string]bool

	balance Balance
}

// New ...
func (tp *HippoTransactionPool) New(balance Balance) {
	heap.Init(&tp.heap)
	tp.balance = balance
}

// Lock ...
func (tp *HippoTransactionPool) Lock() {
	tp.lock.Lock()
}

// Unlock ...
func (tp *HippoTransactionPool) Unlock() {
	tp.lock.Unlock()
}

// Push ...
// Should pass the check first.
// 1. Add to the transaction heap.
// 2. Add to the hash map.
func (tp *HippoTransactionPool) Push(t HippoTransaction) bool {
	if !t.Check(tp.balance) {
		return false
	}
	tp.Lock()
	defer tp.Unlock()
	hash := t.Hash()
	if _, has := tp.hash[hash]; !has {
		tp.heap.Push(t)
		tp.hash[t.Hash()] = true
	}
	return true
}

// Pop ...
// 1. Remove from the transaction heap
// 2. Remove from the hash map
func (tp *HippoTransactionPool) Pop() Transaction {
	tp.Lock()
	defer tp.Unlock()
	if len(tp.heap) == 0 {
		return nil
	}
	t := tp.heap.Pop().(Transaction)
	delete(tp.hash, t.Hash())
	return t
}

// Len ...
func (tp *HippoTransactionPool) Len() int {
	return tp.heap.Len()
}

type transactionPoolCheck func(t Transaction) bool

// Fetch ...
// Fetch a number of transactions.
func (tp *HippoTransactionPool) Fetch(n int, checkFunc transactionPoolCheck) (result []Transaction) {
	tp.Lock()
	defer tp.Unlock()
	count := 0
	result = make([]Transaction, n)
	for count < n && tp.Len() > 0 {
		b := tp.Pop()
		if checkFunc(b) {
			result[count] = b
			count++
		}
	}
	result = result[:count]
	return
}

// =================================
type transactionHeap []Transaction

func (th transactionHeap) Len() int           { return len(th) }
func (th transactionHeap) Less(i, j int) bool { return th[i].GetFee() > th[i].GetFee() }
func (th transactionHeap) Swap(i, j int)      { th[i], th[j] = th[j], th[i] }
func (th *transactionHeap) Push(x interface{}) {
	item := x.(Transaction)
	*th = append(*th, item)
}

func (th *transactionHeap) Pop() interface{} {
	old := *th
	n := len(*th)
	item := old[n-1]
	*th = old[:n-1]
	return item
}

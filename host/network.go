package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	registerlib "github.com/XieGuochao/HippoCoinRegister/lib"
)

// Include network listener and network client.

// NetworkListener ...
// Steps:
// 1. New(ctx, ip, protocol)
// 2. Listen()   It will generate port.
// 3. Stop()
// 4. SetIP(ip) SetPort(port) NetworkAddress()
type NetworkListener interface {
	New(ctx context.Context, ip, protocol string)
	Listen()
	Listener() net.Listener
	SetIP(ip string)
	SetPort(port int)
	NetworkAddress() string

	Stop()
}

// HippoNetworkListener ...
type HippoNetworkListener struct {
	ip, protocol string
	port         int
	ctx          context.Context
	cancel       context.CancelFunc
	listener     net.Listener
	p2pServer    P2PServiceInterface
}

// New ...
func (l *HippoNetworkListener) New(ctx context.Context, ip, protocol string) {
	l.ip, l.protocol = ip, protocol
	l.ctx, l.cancel = context.WithCancel(ctx)
}

// SetIP ...
func (l *HippoNetworkListener) SetIP(ip string) {
	l.ip = ip
}

// SetPort ...
func (l *HippoNetworkListener) SetPort(port int) {
	l.port = port
}

// NetworkAddress ...
func (l *HippoNetworkListener) NetworkAddress() string {
	return fmt.Sprintf("%s:%d", l.ip, l.port)
}

// Stop ...
func (l *HippoNetworkListener) Stop() {
	l.cancel()
}

// Listen ...
func (l *HippoNetworkListener) Listen() {
	var err error
	l.listener, err = net.Listen(l.protocol, ":0")
	if err != nil {
		infoLogger.Fatal(err, l.protocol)
	}
	l.port = l.listener.Addr().(*net.TCPAddr).Port
	infoLogger.Debug("create register listener:", l.NetworkAddress())
}

// Listener ...
func (l *HippoNetworkListener) Listener() net.Listener {
	return l.listener
}

// NetworkClient ...
// Steps:
// 1. New(ctx, address, protocol, maxNeighbors, register, updateTimeBase, updateTimeRand, p2pClient)
// p2pClient is only a template.
// 1.(1) SetMaxPing(int64)
// 2. SyncNeighbors()
// 3. StopSyncNeighbors()
// 4. CountNeighbors()  UpdateNeighbors()  Ping(address)
// 5. StartSyncBlocks(storage)
// 6. StopSyncBlocks()
type NetworkClient interface {
	New(ctx context.Context, address string, protocol string, maxNeighbors int,
		register Register, updateTimeBase, updateTimeRand int, p2pClient P2PClientInterface,
		templateBlock Block)
	SetMaxPing(int64)
	CountNeighbors() int
	UpdateNeighbors()
	GetNeighbors() []string
	SyncNeighbors()
	StopSyncNeighbors()
	GetAddress() string

	TryUpdateNeighbors()
	Ping(address string) (int64, bool)
	BroadcastBlock(address string, broadcastBlock BroadcastBlock, reply *string) error
	QueryLevel(address string, level0, level1 int, reply *[]string) error
	QueryByHash(address string, hashValue string) Block
	QueryHashes(address string, hashes []string) (blocks []Block)
	SyncBlocks(address string, storage Storage)
	SyncAddressesN(n int, storage Storage)

	SetSyncBlockCount(count int)
	StartSyncBlocks(storage Storage)
	StopSyncBlocks()

	SetSyncPeriod(int64)
}

// HippoNetworkClient ...
type HippoNetworkClient struct {
	ctx                 context.Context
	address, protocol   string
	neighbors           sync.Map
	maxNeighbors        int
	register            Register
	syncNeighborsCtx    context.Context
	syncNeighborsCancel context.CancelFunc
	syncBlockCtx        context.Context
	syncBlockCancel     context.CancelFunc
	syncBlockCount      int
	updateTimeBase      int
	updateTimeRand      int
	p2pClient           P2PClientInterface
	maxPing             int64

	syncBlockPeriod int64

	networkPool NetworkPool

	templateBlock Block
}

// New ...
func (c *HippoNetworkClient) New(ctx context.Context, address string, protocol string,
	maxNeighbors int, register Register, updateTimeBase, updateTimeRand int,
	p2pClient P2PClientInterface, templateBlock Block) {
	c.ctx = ctx
	c.syncNeighborsCtx, c.syncNeighborsCancel = context.WithCancel(ctx)
	c.address, c.protocol = address, protocol
	c.register = register
	c.maxNeighbors = maxNeighbors
	c.updateTimeBase, c.updateTimeRand = updateTimeBase, updateTimeRand
	c.p2pClient = p2pClient
	c.maxPing = 1e4 // 10 seconds
	c.syncBlockCount = 5

	c.syncBlockPeriod = 5

	c.templateBlock = templateBlock

	c.networkPool.New(c.ctx, c.p2pClient, protocol, templateBlock)
}

// SetMaxPing ...
func (c *HippoNetworkClient) SetMaxPing(t int64) { c.maxPing = t }

// CountNeighbors ...
func (c *HippoNetworkClient) CountNeighbors() (count int) {
	c.neighbors.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// UpdateNeighbors ...
func (c *HippoNetworkClient) UpdateNeighbors() {
	var reply []byte
	registerClient := c.register.Client()
	err := registerClient.AddressesRefresh(registerlib.RefreshStruct{
		Number:  c.maxNeighbors,
		Address: c.address,
	}, &reply)
	if err != nil {
		infoLogger.Error("update neighbor error:", err)
		return
	}

	var neighbors []string
	json.Unmarshal(reply, &neighbors)
	infoLogger.Debug("update neighbor:", neighbors)
	for _, n := range neighbors {
		c.Ping(n)
	}
}

// TryUpdateNeighbors ...
// Count the number of neighbors and update.
func (c *HippoNetworkClient) TryUpdateNeighbors() {
	if c.CountNeighbors() >= c.maxNeighbors {
		return
	}
	c.UpdateNeighbors()
}

// Ping ...
// Ping and update neighbors.
func (c *HippoNetworkClient) Ping(address string) (t int64, ok bool) {
	var p2pClient P2PClientInterface
	var err error
	debugLogger.Debug("ping", address)

	ctx, cancel := context.WithTimeout(c.ctx, time.Millisecond*time.Duration(c.maxPing))
	done := make(chan error, 1)

	defer cancel()
	t0 := time.Now()
	t = c.maxPing + 1
	ok = false

	go func(done chan error) {
		p2pClient = c.networkPool.Get(address)
		if p2pClient != nil {
			var reply string
			err = p2pClient.Ping("", &reply)
			if err == nil {
				t = time.Since(t0).Milliseconds()
				ok = true
				done <- nil
				return
			}
		}
		infoLogger.Error(err)
		c.networkPool.Update(address)
		done <- nil
		return
	}(done)

	select {
	case <-done:
		debugLogger.Debug("ping finished.")
	case <-ctx.Done():
		debugLogger.Debug("ping timeout")
	}

	// debugLogger.Debug("ping done", address)

	if ok {
		c.neighbors.Store(address, t)
	} else {
		c.neighbors.Delete(address)
	}
	return t, true
}

// BroadcastBlock ...
func (c *HippoNetworkClient) BroadcastBlock(address string, block BroadcastBlock,
	reply *string) error {
	var p2pClient P2PClientInterface
	var err error
	var ok bool
	debugLogger.Debug("netowrk client: broadcast block", address)

	ctx, cancel := context.WithTimeout(c.ctx, time.Millisecond*time.Duration(c.maxPing))
	done := make(chan error, 1)

	defer cancel()
	ok = false

	go func(done chan error) {
		p2pClient = c.networkPool.Get(address)
		if p2pClient != nil {
			err = p2pClient.BroadcastBlock(&block, reply)
			if err == nil {
				ok = true
				done <- nil
				return
			}
		}
		infoLogger.Error(err)
		done <- nil
		return
	}(done)

	select {
	case <-done:
		debugLogger.Debug("netowrk client: broadcast block finished.")
		if ok {
			return nil
		}
		return err
	case <-ctx.Done():
		debugLogger.Debug("netowrk client: broadcast block timeout")
		return errors.New("netowrk client: broadcast block timeout")
	}
}

// QueryLevel ...
func (c *HippoNetworkClient) QueryLevel(address string, level0,
	level1 int, reply *[]string) error {
	var p2pClient P2PClientInterface
	var err error
	var ok bool
	debugLogger.Debug("netowrk client: query level", address, level0, level1)

	ctx, cancel := context.WithTimeout(c.ctx, time.Millisecond*time.Duration(c.maxPing))
	done := make(chan error, 1)

	defer cancel()
	ok = false

	go func(done chan error) {
		p2pClient = c.networkPool.Get(address)
		if p2pClient != nil {
			err = p2pClient.QueryLevel(level0, level1, reply)
			if err == nil {
				ok = true
				done <- nil
				return
			} else {
				c.networkPool.Update(address)
			}
		}
		infoLogger.Error(err)
		done <- nil
		return
	}(done)

	select {
	case <-done:
		debugLogger.Debug("netowrk client: query level finished.")
		if ok {
			return nil
		}
		return err
	case <-ctx.Done():
		debugLogger.Debug("netowrk client: query level timeout")
		return errors.New("netowrk client: query level timeout")
	}
}

// QueryByHash ...
func (c *HippoNetworkClient) QueryByHash(address string, hashValue string) (block Block) {
	var p2pClient P2PClientInterface
	var err error
	var ok bool
	debugLogger.Debug("netowrk client: query by hash", address, hashValue)

	ctx, cancel := context.WithTimeout(c.ctx, time.Millisecond*time.Duration(c.maxPing))
	done := make(chan error, 1)

	defer cancel()
	ok = false

	go func(done chan error) {
		p2pClient = c.networkPool.Get(address)
		if p2pClient != nil {
			block = p2pClient.QueryByHash(hashValue)
			if block != nil {
				ok = true
				done <- nil
				return
			}
		}
		infoLogger.Error(err)
		done <- nil
		return
	}(done)

	select {
	case <-done:
		debugLogger.Debug("netowrk client: query by hash finished.")
		if ok {
			return block
		}
		return nil
	case <-ctx.Done():
		debugLogger.Debug("netowrk client: query by hash timeout")
		return nil
	}
}

// QueryHashes ...
func (c *HippoNetworkClient) QueryHashes(address string, hashes []string) (blocks []Block) {
	var p2pClient P2PClientInterface
	var err error
	var ok bool
	debugLogger.Debug("netowrk client: query hashes", address, hashes)

	ctx, cancel := context.WithTimeout(c.ctx, time.Millisecond*time.Duration(c.maxPing*5))
	done := make(chan error, 1)

	defer cancel()
	ok = false

	go func(done chan error) {
		p2pClient = c.networkPool.Get(address)
		if p2pClient != nil {
			blocks = p2pClient.QueryHashes(hashes)
			if blocks != nil {
				ok = true
				done <- nil
				return
			}
		}
		infoLogger.Error(err)
		done <- nil
		return
	}(done)

	select {
	case <-done:
		debugLogger.Debug("netowrk client: query hashes finished.")
		if ok {
			return blocks
		}
		return nil
	case <-ctx.Done():
		debugLogger.Debug("netowrk client: query hashes timeout")
		return nil
	}
}

// SyncBlocks ...
// Sync blocks from the first level up to the latest one.
func (c *HippoNetworkClient) SyncBlocks(address string, storage Storage) {
	level0, level1 := 0, 4
	var hashes []string
	var err error
	var newBlocks []Block
	for {
		infoLogger.Infof("sync blocks %s %d-%d", address, level0, level1)
		err = c.QueryLevel(address, level0, level1, &hashes)
		if err != nil {
			infoLogger.Error("sync block error:", err)
			return
		}
		hashes = storage.FilterNewHashes(hashes)
		if len(hashes) == 0 {
			infoLogger.Infof("syncBlocks %s done", address)
			break
		}
		newBlocks = c.QueryHashes(address, hashes)
		if len(newBlocks) == 0 {
			infoLogger.Infof("syncBlocks %s done", address)
			break
		}
		storage.AddBlocks(newBlocks)
		level0, level1 = level1+1, level1+5
	}
}

// SyncAddressesN ...
func (c *HippoNetworkClient) SyncAddressesN(n int, storage Storage) {
	addresses := c.GetNeighbors()
	if n > len(addresses) {
		n = len(addresses)
	}
	addresses = addresses[:n]
	for _, address := range addresses {
		go c.SyncBlocks(address, storage)
	}
}

// SetSyncBlockCount ...
func (c *HippoNetworkClient) SetSyncBlockCount(count int) { c.syncBlockCount = count }

// StartSyncBlocks ...
func (c *HippoNetworkClient) StartSyncBlocks(storage Storage) {
	c.syncBlockCtx, c.syncBlockCancel = context.WithCancel(c.ctx)

	go func() {
		for {
			select {
			case <-c.syncBlockCtx.Done():
				infoLogger.Debug("stop sync blocks")
				return
			default:
				go c.SyncAddressesN(c.syncBlockCount, storage)
				time.Sleep(time.Second * time.Duration(c.syncBlockPeriod))
			}
		}
	}()
}

// StopSyncBlocks ...
func (c *HippoNetworkClient) StopSyncBlocks() { c.syncBlockCancel() }

// NeighborPing ...
type NeighborPing struct {
	Address string
	Ping    int64
}

// EvictNeighbors ...
// Evict neighbors based on the ping time.
func (c *HippoNetworkClient) EvictNeighbors() {
	neighbors := make([]NeighborPing, 0)
	c.neighbors.Range(func(k, v interface{}) bool {
		neighbors = append(neighbors, NeighborPing{
			Address: k.(string),
			Ping:    v.(int64),
		})
		return true
	})
	debugLogger.Debug(neighbors)
	if len(neighbors) < c.maxNeighbors {
		return
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Ping < neighbors[j].Ping
	})

	// Delete those slow neighbors
	for _, n := range neighbors[c.maxNeighbors:] {
		c.neighbors.Delete(n.Address)
	}
}

// GetNeighbors ...
func (c *HippoNetworkClient) GetNeighbors() (neighbors []string) {
	c.neighbors.Range(func(key, value interface{}) bool {
		neighbors = append(neighbors, key.(string))
		return true
	})
	return neighbors
}

// SyncNeighbors ...
// Run SyncNeighbors in background.
func (c *HippoNetworkClient) SyncNeighbors() {
	go func() {
		for {
			select {
			case <-c.syncNeighborsCtx.Done():
				infoLogger.Debug("stop sync neighbors")
				return
			default:
				c.TryUpdateNeighbors()
				debugLogger.Debug("neighbors:", c.GetNeighbors())

				c.EvictNeighbors()
				debugLogger.Debug("neighbors after eviction:", c.GetNeighbors())

				seconds := c.updateTimeBase + rand.Intn(c.updateTimeRand)
				debugLogger.Debug("sync neighbors stop:", seconds)
				time.Sleep(time.Second * time.Duration(seconds))
			}
		}
	}()
	infoLogger.Debug("start sync neighbors")
}

// StopSyncNeighbors ...
func (c *HippoNetworkClient) StopSyncNeighbors() {
	c.syncNeighborsCancel()
}

// GetAddress ...
func (c *HippoNetworkClient) GetAddress() string { return c.address }

// SetSyncPeriod ...
func (c *HippoNetworkClient) SetSyncPeriod(p int64) { c.syncBlockPeriod = p }

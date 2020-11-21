package main

import (
	"context"
	"encoding/json"
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
		logger.Fatal(err)
	}
	l.port = l.listener.Addr().(*net.TCPAddr).Port
	logger.Info("create register listener:", l.NetworkAddress())
}

// Listener ...
func (l *HippoNetworkListener) Listener() net.Listener {
	return l.listener
}

// NetworkClient ...
// Steps:
// 1. New(ctx, address, protocol, maxNeighbors, register, updateTimeBase, updateTimeRand, p2pClient)
// p2pClient is only a template.
// 2. SyncNeighbors()
// 3. StopSyncNeighbors()
// 4. CountNeighbors()  UpdateNeighbors()  Ping(address)
type NetworkClient interface {
	New(ctx context.Context, address string, protocol string, maxNeighbors int,
		register Register, updateTimeBase, updateTimeRand int, p2pClient P2PClientInterface)
	CountNeighbors() int
	UpdateNeighbors()
	GetNeighbors() []string
	SyncNeighbors()
	StopSyncNeighbors()
	Ping(address string) (int64, bool)
}

// HippoNetworkClient ...
type HippoNetworkClient struct {
	ctx               context.Context
	address, protocol string
	neighbors         sync.Map
	maxNeighbors      int
	register          Register
	syncCtx           context.Context
	syncCancel        context.CancelFunc
	updateTimeBase    int
	updateTimeRand    int
	p2pClient         P2PClientInterface
}

// New ...
func (c *HippoNetworkClient) New(ctx context.Context, address string, protocol string,
	maxNeighbors int, register Register, updateTimeBase, updateTimeRand int, p2pClient P2PClientInterface) {
	c.ctx = ctx
	c.syncCtx, c.syncCancel = context.WithCancel(ctx)
	c.address, c.protocol = address, protocol
	c.register = register
	c.maxNeighbors = maxNeighbors
	c.updateTimeBase, c.updateTimeRand = updateTimeBase, updateTimeRand
	c.p2pClient = p2pClient
}

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
		logger.Error("update neighbor error:", err)
		return
	}

	var neighbors []string
	json.Unmarshal(reply, &neighbors)
	logger.Info("update neighbor:", neighbors)
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
func (c *HippoNetworkClient) Ping(address string) (int64, bool) {
	p2pClient := c.p2pClient.Empty()
	logger.Debug("ping", address)
	err := p2pClient.New(c.ctx, c.protocol, address)
	if err != nil {
		logger.Error(err)
		c.neighbors.Delete(address)
		return 0, false
	}
	defer p2pClient.Close()
	t0 := time.Now()

	var reply string
	err = p2pClient.Ping("", &reply)
	if err != nil {
		logger.Error(err)
		c.neighbors.Delete(address)
		return 0, false
	}
	t := time.Since(t0).Nanoseconds()
	c.neighbors.Store(address, t)
	return t, true
}

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
			case <-c.syncCtx.Done():
				logger.Info("stop sync neighbors")
				return
			default:
				c.TryUpdateNeighbors()
				c.EvictNeighbors()
				seconds := c.updateTimeBase + rand.Intn(c.updateTimeRand)
				time.Sleep(time.Second * time.Duration(seconds))
			}
		}
	}()
	logger.Info("start sync neighbors")
}

// StopSyncNeighbors ...
func (c *HippoNetworkClient) StopSyncNeighbors() {
	c.syncCancel()
}

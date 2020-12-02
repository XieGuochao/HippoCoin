package host

import (
	"context"
)

// BroadcastQueue ...
// Steps:
// 1. New(ctx, protocol, p2pClient)
// 2. (init network client)
// 3. SetNetworkClient(networkClient)
// 4. Run()
// 5. Add(block)
// 6. Stop()
type BroadcastQueue interface {
	New(ctx context.Context, protocol string, p2pClient P2PClientInterface,
		maxBroadcastLevel uint)
	SetNetworkClient(networkClient NetworkClient)
	Add(b BroadcastBlock)
	Run()
	Stop()
	// BroadcastBlockSend(block BroadcastBlock)
}

// HippoBroadcastQueue ...
type HippoBroadcastQueue struct {
	ctx           context.Context
	cancel        context.CancelFunc
	channel       chan BroadcastBlock
	networkClient NetworkClient
	protocol      string
	p2pClient     P2PClientInterface

	maxBroadcastLevel uint
}

// New ...
func (bq *HippoBroadcastQueue) New(ctx context.Context, protocol string,
	p2pClient P2PClientInterface, maxBroadcastLevel uint) {
	bq.ctx, bq.cancel = context.WithCancel(ctx)
	bq.channel = make(chan BroadcastBlock, 10)
	bq.protocol = protocol
	bq.p2pClient = p2pClient
	bq.maxBroadcastLevel = maxBroadcastLevel
}

// SetNetworkClient ...
func (bq *HippoBroadcastQueue) SetNetworkClient(networkClient NetworkClient) {
	bq.networkClient = networkClient
}

// Add ...
func (bq *HippoBroadcastQueue) Add(b BroadcastBlock) {
	if b.Level < bq.maxBroadcastLevel {
		bq.channel <- b
		debugLogger.Debug("broadcast queue add:", b.Block.Hash())
	}
}

// Run ...
func (bq *HippoBroadcastQueue) Run() {
	go func() {
		for {
			select {
			case <-bq.ctx.Done():
				infoLogger.Debug("broadcast queue closed.")
				return
			case block := <-bq.channel:
				debugLogger.Debug("broadcast queue receive broadcast block")

				bq.broadcastBlockSend(block)
			}
		}
	}()
}

// Stop ...
func (bq *HippoBroadcastQueue) Stop() {
	bq.cancel()
}

func (bq *HippoBroadcastQueue) broadcastBlockSend(block BroadcastBlock) {
	debugLogger.Debug("receive broadcast block")
	addresses := bq.networkClient.GetNeighbors()
	debugLogger.Debug("neighbors all:", addresses)

	addressesToSend := make(map[string]bool)
	for _, address := range addresses {
		addressesToSend[address] = true
	}
	for address := range block.Addresses {
		delete(addressesToSend, address)
	}
	delete(addressesToSend, bq.networkClient.GetAddress())

	for address := range addressesToSend {
		block.Addresses[address] = true
	}
	debugLogger.Debug("neighbors to send:", addressesToSend)
	for address := range addressesToSend {
		debugLogger.Debug("send broadcast block to", address)
		var reply string
		if bq.networkClient == nil {
			infoLogger.Error("broadcast-queue: no netowrk client")
			break
		}

		bq.networkClient.BroadcastBlock(address, block, &reply)
	}
	debugLogger.Debug("broadcast send done.")
}

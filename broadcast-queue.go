package main

import (
	"context"
)

// BroadcastQueue ...
type BroadcastQueue interface {
	New(ctx context.Context, protocol string,
		networkClient *NetworkClient, p2pClient P2PClientInterface)
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
	networkClient *NetworkClient
	protocol      string
	p2pClient     P2PClientInterface
}

// New ...
func (bq *HippoBroadcastQueue) New(ctx context.Context, protocol string,
	networkClient *NetworkClient, p2pClient P2PClientInterface) {
	bq.ctx, bq.cancel = context.WithCancel(ctx)
	bq.channel = make(chan BroadcastBlock, 10)
	bq.protocol = protocol
	bq.networkClient = networkClient
	bq.p2pClient = p2pClient
}

// Add ...
func (bq *HippoBroadcastQueue) Add(b BroadcastBlock) {
	bq.channel <- b
	logger.Debug("broadcast queue add:", b.Block.Hash())
}

// Run ...
func (bq *HippoBroadcastQueue) Run() {
	go func() {
		for {
			select {
			case <-bq.ctx.Done():
				logger.Info("broadcast queue closed.")
				return
			case block := <-bq.channel:
				logger.Debug("receive broadcast block")

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
	logger.Debug("receive broadcast block")
	addresses := (*bq.networkClient).GetNeighbors()
	for _, address := range addresses {
		block.Addresses[address] = true
	}
	for _, address := range addresses {
		var reply string
		var client P2PClientInterface
		client = bq.p2pClient.Empty()
		if client.New(bq.ctx, bq.protocol, address) == nil {
			client.BroadcastBlock(&block, &reply)
			client.Close()
		}
	}
	logger.Debug("broadcast send done.")
}

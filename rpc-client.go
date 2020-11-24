package main

import (
	"context"
	"net/rpc"
)

// P2PServiceName ...
const P2PServiceName = "github.com/XieGuochao/HippoCoin"

// ==============================================================

// P2PClientInterface ...
// A P2PClient should be able to send P2P queries without much work.
// One connection should handle multiple queries before close.
// P2P queries include:
// - Ping
// - Broadcast
// - QueryLevel
type P2PClientInterface interface {
	Empty() P2PClientInterface
	New(ctx context.Context, protocol string, address string) error
	Copy() P2PClientInterface
	Close()
	Ping(request string, reply *string) error
	BroadcastBlock(data NetworkSendInterface, reply *string) error
	QueryLevel(level0, level1 int, reply *[]Block) error
}

// P2PClient ...
type P2PClient struct {
	c         *rpc.Client
	ctx       context.Context
	cancel    context.CancelFunc
	parentCtx context.Context
	protocol  string
	address   string
}

// Empty ...
func (c *P2PClient) Empty() P2PClientInterface {
	return new(P2PClient)
}

// New ...
func (c *P2PClient) New(ctx context.Context, protocol string, address string) (err error) {
	c.protocol, c.address = protocol, address
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.parentCtx = ctx
	c.c, err = rpc.DialHTTP(protocol, address)
	if err != nil {
		logger.Error(err, protocol, address)
	}
	return
}

// Copy ...
func (c *P2PClient) Copy() P2PClientInterface {
	newClient := new(P2PClient)
	if newClient.New(c.parentCtx, c.protocol, c.address) != nil {
		return nil
	}
	return newClient
}

// Close ...
func (c *P2PClient) Close() {
	defer logger.Debug("ping close.")

	c.cancel()
	if c.c != nil {
		c.c.Close()
	}
}

// Ping ...
func (c *P2PClient) Ping(request string, reply *string) error {
	return c.c.Call(P2PServiceName+".Ping", request, reply)
}

// QueryLevel ...
func (c *P2PClient) QueryLevel(level0, level1 int, reply *[]Block) error {
	var bytes []byte
	err := c.c.Call(P2PServiceName+".QueryLevel",
		QueryLevelStruct{
			Level0: level0,
			Level1: level1,
		}, bytes)
	if err != nil {
		return err
	}
	*reply = DecodeBlocks(bytes)
	return nil
}

// // BroadcastData ...
// func (c *P2PClient) BroadcastData(data NetworkSendInterface, reply *string) error {
// 	return c.c.Call(P2PServiceName+".BroadcastData", data.Encode(), reply)
// }

// BroadcastBlock ...
func (c *P2PClient) BroadcastBlock(data NetworkSendInterface, reply *string) error {
	logger.Debug("broadcastBlock to send", data)
	return c.c.Call(P2PServiceName+".BroadcastBlock", data.Encode(), reply)
}

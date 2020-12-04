package host

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
// - QueryByHash: require SetTemplateBlock(block)
// - QueryHashes: require SetTemplateBlock(block)
type P2PClientInterface interface {
	Empty() P2PClientInterface
	New(ctx context.Context, protocol string, address string) error
	Copy() P2PClientInterface
	Close()

	SetTemplateBlock(b Block)

	Ping(request string, reply *string) error
	BroadcastBlock(data NetworkSendInterface, reply *string) error
	BroadcastTransaction(data NetworkSendInterface, reply *string) error
	QueryLevel(level0, level1 int, reply *[]string) error
	QueryByHash(hashValue string) (block Block)
	QueryHashes(hashes []string) (block []Block)
}

// P2PClient ...
type P2PClient struct {
	c             *rpc.Client
	ctx           context.Context
	cancel        context.CancelFunc
	parentCtx     context.Context
	protocol      string
	address       string
	templateBlock Block
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
		infoLogger.Error(err, protocol, address)
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
	defer debugLogger.Debug("ping close.")

	c.cancel()
	if c.c != nil {
		c.c.Close()
	}
}

// SetTemplateBlock ...
func (c *P2PClient) SetTemplateBlock(block Block) { c.templateBlock = block }

// Ping ...
func (c *P2PClient) Ping(request string, reply *string) error {
	return c.c.Call(P2PServiceName+".Ping", request, reply)
}

// QueryLevel ...
func (c *P2PClient) QueryLevel(level0, level1 int, reply *[]string) error {
	err := c.c.Call(P2PServiceName+".QueryLevel",
		QueryLevelStruct{
			Level0: level0,
			Level1: level1,
		}, reply)
	if err != nil {
		return err
	}
	return nil
}

// QueryByHash ...
func (c *P2PClient) QueryByHash(hashValue string) (block Block) {
	var reply []byte
	err := c.c.Call(P2PServiceName+".QueryByHash",
		hashValue, &reply)
	if err != nil {
		infoLogger.Error("query by hash: cannot decode block:", err)
		return nil
	}

	if c.templateBlock == nil {
		infoLogger.Error("query by hash: no template block")
		return nil
	}
	block = DecodeBlock(reply, c.templateBlock)
	if block == nil {
		infoLogger.Error("query by hash: cannot decode block")
	}
	return
}

// QueryHashes ...
func (c *P2PClient) QueryHashes(hashes []string) (block []Block) {
	block = make([]Block, 0)
	for _, h := range hashes {
		if b := c.QueryByHash(h); b != nil {
			block = append(block, b)
		}
	}
	return block
}

// // BroadcastData ...
// func (c *P2PClient) BroadcastData(data NetworkSendInterface, reply *string) error {
// 	return c.c.Call(P2PServiceName+".BroadcastData", data.Encode(), reply)
// }

// BroadcastBlock ...
func (c *P2PClient) BroadcastBlock(data NetworkSendInterface, reply *string) error {
	// debugLogger.Debug("broadcastBlock to send", data)
	return c.c.Call(P2PServiceName+".BroadcastBlock", data.Encode(), reply)
}

// BroadcastTransaction ...
func (c *P2PClient) BroadcastTransaction(data NetworkSendInterface, reply *string) error {
	// debugLogger.Debug("broadcastBlock to send", data)
	return c.c.Call(P2PServiceName+".BroadcastTransaction", data.Encode(), reply)
}

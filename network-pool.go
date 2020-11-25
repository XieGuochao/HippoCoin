package main

import (
	"context"
	"sync"
)

// NetworkPool ...
type NetworkPool struct {
	data           sync.Map
	clientTemplate P2PClientInterface
	parentCtx      context.Context
	protocol       string
	tempalteBlock  Block
}

// New ...
func (n *NetworkPool) New(ctx context.Context, clientTemplate P2PClientInterface,
	protocol string, templateBlock Block) {
	n.clientTemplate = clientTemplate.Empty()
	n.parentCtx = ctx
	n.protocol = protocol
	n.tempalteBlock = templateBlock
}

// Get ...
func (n *NetworkPool) Get(address string) P2PClientInterface {
	var clientInterface interface{}
	var client P2PClientInterface
	var has bool
	var err error
	clientInterface, has = n.data.Load(address)
	if !has {
		client = n.clientTemplate.Empty()
		err = client.New(n.parentCtx, n.protocol, address)
		client.SetTemplateBlock(n.tempalteBlock)
		if err == nil {
			n.data.Store(address, client)
		} else {
			client = nil
		}
	} else {
		client = clientInterface.(P2PClientInterface)
	}
	return client
}

// Update ...
func (n *NetworkPool) Update(address string) P2PClientInterface {
	var clientInterface interface{}
	var client P2PClientInterface
	var has bool
	var err error
	clientInterface, has = n.data.Load(address)
	if has {
		client = clientInterface.(P2PClientInterface)
		client.Close()
	}

	client = n.clientTemplate.Empty()
	err = client.New(n.parentCtx, n.protocol, address)
	client.SetTemplateBlock(n.tempalteBlock)

	if err == nil {
		n.data.Store(address, client)
	} else {
		client = nil
	}

	return client
}

// Reset ...
func (n *NetworkPool) Reset() {
	n.data.Range(func(k, v interface{}) bool {
		client := v.(P2PClientInterface)
		client.Close()
		return true
	})
	n.data = sync.Map{}
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/rpc"
	"sync"
	"time"
)

const (
	// HippoP2PServiceName ...
	HippoP2PServiceName = "github.com/XieGuochao/HippoCoin"
)

// HippoP2PServiceInterface ...
type HippoP2PServiceInterface = interface {
	Ping(request string, reply *string) error
}

// InitP2PServer ...
func (client *HippoCoinClient) InitP2PServer() {
	log.Println("init p2p sersver")
	service := new(ServiceStruct)
	service.client = client
	RegisterP2PService(service)
	client.CreateListener()
	client.registerContext, client.registerCancel = context.WithCancel(context.Background())
	client.serverContext, client.serverCancel = context.WithCancel(context.Background())
	go func() {
		for {
			conn, err := client.listener.Accept()

			if err != nil {
				log.Fatal("p2p server accept error:", err)
			}

			go rpc.ServeConn(conn)
		}
	}()
}

// Listen ...
func (client *HippoCoinClient) Listen(wg *sync.WaitGroup) {
	defer client.listener.Close()
	defer wg.Done()

	for {
		select {
		case <-client.serverContext.Done():
			log.Println("stop p2p server listen")
			break

		default:
			time.Sleep(time.Second)
		}
	}
}

// RegisterP2PService ...
func RegisterP2PService(svc HippoP2PServiceInterface) error {
	return rpc.RegisterName(HippoP2PServiceName, svc)
}

// ServiceStruct ...
type ServiceStruct struct {
	client *HippoCoinClient
}

// Ping ...
func (s *ServiceStruct) Ping(request string, reply *string) error {
	log.Println("ping:", request)
	*reply = request + "!"
	return nil
}

// ===================================

// BroadcastStruct ...
type BroadcastStruct struct {
	Addresses map[string]bool
	Data      interface{}
	Type      string
	Level     uint
}

// BroadcastStructByte ...
type BroadcastStructByte []byte

// Encode ...
func (b *BroadcastStruct) Encode() ([]byte, error) {
	return json.Marshal(*b)
}

// Decode ...
func (bytes *BroadcastStructByte) Decode() (b *BroadcastStruct) {
	b = new(BroadcastStruct)
	json.Unmarshal(*bytes, b)
	return
}

// Broadcast ...
func (s *ServiceStruct) Broadcast(bytes BroadcastStructByte, reply *bool) error {
	bcdata := bytes.Decode()
	c := s.client

	// First: try to decode to a block
	if bcdata.Type == "block" {
		block := NewHippoBlockFromJSONMap(bcdata.Data)
		// Add to local storage
		// Check if it is a block, and add it.
		log.Println("receive broadcast block:", ByteToHexString(block.Hash(s.client)))
		newBlock := new(HippoBlock)
		*newBlock = block
		if newBlock.Valid(s.client) {
			s.client.storage.Add(newBlock)
			s.client.miningPool.DeleteBlock(newBlock)
		}
	} else {
		log.Println("receive broadcast:", bcdata)
	}

	go func() {

	}()

	bcdata.Level++
	if bcdata.Level < c.maxBroadcastLevel {
		c.Broadcast(*bcdata)
	}
	*reply = true
	return nil
}

// =============================

// QueryLevelStruct ...
type QueryLevelStruct struct {
	Level0, Level1 uint32
}

// QueryResponse ...
type QueryResponse []HippoBlock

// // QueryResponseByte ...
// type QueryResponseByte []byte

// QueryLevel ...
func (s *ServiceStruct) QueryLevel(q QueryLevelStruct, reply *QueryResponse) error {
	*reply = s.client.storage.GetBlocks(q.Level0, q.Level1)
	return nil
}

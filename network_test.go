package main

import (
	"crypto/ecdsa"
	"log"
	"testing"
	"time"
)

func TestNetwork(t *testing.T) {
	var result bool

	initClients(1)
	client := &clients[0]
	client.CreateListener()
	client.InitRegisterClient()
	client.Register()

	log.Println("sleep for 5 seconds")
	time.Sleep(time.Second * 5)
	log.Println("try update neighbors")

	result = client.TryUpdateNeighbors()
	log.Println("update neighbors:", result)

	log.Println("neighbors:", client.neighbors)
}

// You should start the register server first.
// Then start several tests sequentially.
func TestNeighbors(t *testing.T) {
	var result bool

	initClients(1)
	client := &clients[0]
	client.CreateListener()
	client.InitRegisterClient()
	client.Register()

	for i := 0; i < 10; i++ {
		log.Println("sleep for 5 seconds")
		time.Sleep(time.Second * 5)
		log.Println("try update neighbors")

		result = client.TryUpdateNeighbors()
		log.Println("update neighbors:", result)
		log.Println("neighbors:", client.neighbors)
	}
}

func TestP2PBroadcastString(t *testing.T) {
	// var result bool

	initClients(1)
	client := &clients[0]
	client.StartNetwork()

	data := "hello world from " + client.NetworkAddress()
	go func() {
		time.Sleep(10 * time.Second)
		client.Broadcast(BroadcastStruct{
			Data: data,
		})
		time.Sleep(10 * time.Second)
		client.Broadcast(BroadcastStruct{
			Data: "2:" + data,
		})
	}()
	client.Run()
}

func TestP2PSendBlock(t *testing.T) {
	// var result bool

	var (
		check  bool
		t1, t2 int64
	)

	initClients(2)
	client := &clients[0]
	client.StartNetwork()

	tr := NewHippoTransaction()

	client.balance.store(client.Address(), 1000)

	publicKey := publicKeyToString(client.PublicKey())
	tr.AppendFrom(client.Address(), 100, publicKey)

	client2 := clients[1]
	tr.AppendTo(client2.Address(), 90)

	tr.CalculateFee()

	privateKeys := make([]*ecdsa.PrivateKey, 1)
	privateKeys[0] = client.privateKey
	tr.SignAll(privateKeys, client)

	log.Println("sign:", *tr)
	log.Println("validate:", tr.Valid(client))

	block := NewHippoBlock([32]byte{}, 23, 0, client)
	block.AppendTransaction(tr, client)

	log.Println("block append:", *block)
	assertT(block.SignBlock(client) == nil, t)
	log.Println("block sign:", *block)
	log.Println("block verify:", block.Verify(client))

	log.Println("block valid should false", block.Valid(client))
	assertT(!block.Valid(client), t)

	client.miningFunction = SingleMine

	log.Println("====== start mining single ======")
	t1 = time.Now().Unix()
	block.Mine(client)
	t2 = time.Now().Unix()
	log.Printf("====== end mining ======\n\n")

	log.Println("mine single thread:", t2-t1, "seconds")
	log.Println("mined:", block.Nonce)

	check = block.Valid(client)
	log.Println("block valid should true:", check)
	assertT(check, t)

	data := BroadcastStruct{
		Data:  *block,
		Level: 0,
	}

	go func() {
		time.Sleep(10 * time.Second)
		client.Broadcast(data)
	}()

	client.Run()
}

func TestP2PRun(t *testing.T) {
	initClients(1)
	client := &clients[0]
	client.StartNetwork()
	client.Run()
}

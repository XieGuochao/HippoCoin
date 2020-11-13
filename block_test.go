package main

import (
	"crypto/ecdsa"
	"log"
	"testing"
	"time"
)

func TestBlock(t *testing.T) {
	tr := NewHippoTransaction()

	initClients(2)
	client := clients[0]
	client.balance.store(client.Address(), 1000)

	// log.Println("privateKey:", *client.privateKey)
	publicKey := publicKeyToString(client.PublicKey())
	// log.Println("publicKey:", publicKey)
	tr.AppendFrom(client.Address(), 100, publicKey)
	// log.Println("append from:", *tr)

	client2 := clients[1]
	tr.AppendTo(client2.Address(), 90)
	// log.Println("after append to", *tr)

	tr.CalculateFee()
	// log.Println("after calculate fee", *tr)

	privateKeys := make([]*ecdsa.PrivateKey, 1)
	privateKeys[0] = client.privateKey
	tr.SignAll(privateKeys, &client)

	log.Println("sign:", *tr)
	log.Println("validate:", tr.Valid(&client))

	block := NewHippoBlock([32]byte{}, 10, 0, &client)
	block.AppendTransaction(tr, &client)

	log.Println("block append:", *block)
	assertT(block.SignBlock(&client) == nil, t)
	log.Println("block sign:", *block)
	log.Println("block verify:", block.Verify(&client))

	log.Println("block valid should false", block.Valid(&client))
	assertT(!block.Valid(&client), t)
}

func TestMineSingleBlock(t *testing.T) {
	var (
		check  bool
		t1, t2 int64
	)

	tr := NewHippoTransaction()

	initClients(2)
	client := clients[0]
	client.balance.store(client.Address(), 1000)

	publicKey := publicKeyToString(client.PublicKey())
	tr.AppendFrom(client.Address(), 100, publicKey)

	client2 := clients[1]
	tr.AppendTo(client2.Address(), 90)

	tr.CalculateFee()

	privateKeys := make([]*ecdsa.PrivateKey, 1)
	privateKeys[0] = client.privateKey
	tr.SignAll(privateKeys, &client)

	log.Println("sign:", *tr)
	log.Println("validate:", tr.Valid(&client))

	block := NewHippoBlock([32]byte{}, 23, 0, &client)
	block.AppendTransaction(tr, &client)

	log.Println("block append:", *block)
	assertT(block.SignBlock(&client) == nil, t)
	log.Println("block sign:", *block)
	log.Println("block verify:", block.Verify(&client))

	log.Println("block valid should false", block.Valid(&client))
	assertT(!block.Valid(&client), t)

	client.miningFunction = SingleMine

	log.Println("====== start mining single ======")
	t1 = time.Now().Unix()
	block.Mine(&client)
	t2 = time.Now().Unix()
	log.Printf("====== end mining ======\n\n")

	log.Println("mine single thread:", t2-t1, "seconds")
	log.Println("mined:", block.Nonce)

	check = block.Valid(&client)
	log.Println("block valid should true:", check)
	assertT(check, t)

	log.Println("reset nonce")
	block.Nonce = 0

	client.miningFunction = MultiMine

	log.Println("====== start mining multiple ======")
	t1 = time.Now().Unix()
	block.Mine(&client)
	t2 = time.Now().Unix()
	log.Printf("====== end mining ======\n\n")

	log.Println("mine single thread:", t2-t1, "seconds")
	log.Println("mined:", block.Nonce)

	check = block.Valid(&client)
	log.Println("block valid should true:", check)
	assertT(check, t)
}

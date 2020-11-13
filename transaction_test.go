package main

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"testing"
)

var (
	clients    []HippoCoinClient
	numThreads uint
)

func initClients(n uint) {
	clients = make([]HippoCoinClient, n)
	for i := uint(0); i < n; i++ {
		clients[i].Init(map[string]interface{}{
			"numberThreads":  1,
			"privateKeyFile": fmt.Sprintf("key-%d.pem", i),
		})
	}
	numThreads = n
	log.Println("Init clients done.")
}

func TestTransaction(t *testing.T) {
	tr := NewHippoTransaction()

	initClients(2)
	client := clients[0]
	client.balance.store(client.Address(), 1000)

	log.Println("privateKey:", *client.privateKey)
	publicKey := publicKeyToString(client.PublicKey())
	log.Println("publicKey:", publicKey)
	tr.AppendFrom(client.Address(), 100, publicKey)
	log.Println("append from:", *tr)

	client2 := clients[1]
	tr.AppendTo(client2.Address(), 90)
	log.Println("after append to", *tr)

	tr.CalculateFee()
	log.Println("after calculate fee", *tr)

	privateKeys := make([]*ecdsa.PrivateKey, 1)
	privateKeys[0] = client.privateKey
	tr.SignAll(privateKeys, &client)

	log.Println("sign:", *tr)
	validResult := tr.Valid(&client)
	log.Println("validate:", validResult)

}

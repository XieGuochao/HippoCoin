package main

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
)

func TestCoin(t *testing.T) {
	params := make(map[string]interface{})
	randomNumber := rand.Uint32()
	params["privateKeyFile"] = fmt.Sprintf("%d-key.pem", randomNumber)
	coin := HippoCoinClient{}
	coin.Init(params)

	originalKey := coin.privateKey
	loadedKey := coin.LoadPrivateKey()
	log.Println(originalKey)
	log.Println(loadedKey)
	log.Println("finish testing")
}

func TestCoinStart(t *testing.T) {
	params := make(map[string]interface{})
	randomNumber := rand.Uint32()
	params["privateKeyFile"] = fmt.Sprintf("%d-key.pem", randomNumber)
	coin := HippoCoinClient{}
	coin.Init(params)

	coin.Run()
	log.Println("finish testing")
}

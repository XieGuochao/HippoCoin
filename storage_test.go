package main

import (
	"log"
	"testing"
)

var (
	client *HippoCoinClient
	st     *HippoStorage
)

func TestStorageInit(t *testing.T) {
	client = NewHippoCoinClient()
	client.Init(nil)
	st = NewHippoStorage(client)
	assertT(st.HasGenesis() == false, t)
}

func TestStorageBlock(t *testing.T) {
	client = NewHippoCoinClient()
	client.Init(nil)
	st = NewHippoStorage(client)
	assertT(st.HasGenesis() == false, t)

	b := NewHippoBlock([32]byte{}, 10, 0, client)
	b.SignBlock(client)
	b.Mine(client)
	added, has := st.Add(b)
	assertT(added, t)
	assertT(!has, t)

	log.Println("pass add storage test.")
	log.Println("now check the storage metrics")

	log.Println("topLevel:", st.topLevel())
	b2, has := st.In(b)
	log.Println("find block:", b2)
	assertT(has, t)
	assertT(st.HasGenesis(), t)
	assertT(st.HasLevel(0), t)
	assertT(!st.HasLevel(1), t)
}

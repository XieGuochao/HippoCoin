package main

import (
	"context"
	"log"
	"net/rpc"
	"strings"
	"sync"
)

// P2PClient ...
type P2PClient struct {
	c *rpc.Client
	h *HippoCoinClient
}

// InitP2PClient ...
// This will init the p2pClients.
func (h *HippoCoinClient) InitP2PClient() {
	h.p2pClients = new(P2PClients)
	h.broadcastChan = make(chan BroadcastStruct, 20)
	h.broadcastContext, h.broadcastCancel = context.WithCancel(context.Background())
	log.Println("p2p client set up done.")
}

// P2PConnect ...
func (h *HippoCoinClient) P2PConnect(address string) (client *P2PClient, err error) {
	client, err = CreateP2PClient(h.protocol, address)
	if err != nil {
		log.Println("cannot create p2p connect to", address)
		return nil, err
	}
	h.p2pClients.store(address, client)
	return client, nil
}

// P2PClose ...
func (h *HippoCoinClient) P2PClose(address string) {
	client, has := h.p2pClients.load(address)
	if has {
		client.Close()
		h.p2pClients.delete(address)
	}
}

// ======================================================

// ListenBroadcast ...
func (h *HippoCoinClient) ListenBroadcast(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-h.broadcastContext.Done():
			log.Println("listen broadcast done.")
			return
		case bcdata := <-h.broadcastChan:
			log.Println("listen broadcast:", bcdata.Level)
			go h.broadcastSend(bcdata)
		}
	}
}

// Broadcast ...
func (h *HippoCoinClient) Broadcast(bcdata BroadcastStruct) {
	h.broadcastChan <- bcdata
}

// broadcastSend ...
func (h *HippoCoinClient) broadcastSend(bcdata BroadcastStruct) {
	if bcdata.Level >= h.maxBroadcastLevel {
		return
	}
	addresses := bcdata.Addresses
	myAddress := h.NetworkAddress()
	newAddresses := make(map[string]bool) // store all previous addresses and my neighhbors
	newAddresses[myAddress] = true
	newAddressesList := make([]string, 0)

	h.neighbors.Range(func(k, v interface{}) bool {
		ad := k.(string)
		_, has := addresses[ad]
		newAddresses[ad] = true

		if !has {
			newAddressesList = append(newAddressesList, ad)
		}
		return true
	})

	bcdata.Addresses = newAddresses
	log.Printf("going to send to addresses\n%s\n", strings.Join(newAddressesList, "\n"))

	var wg sync.WaitGroup
	wg.Add(len(newAddresses))
	for _, ad := range newAddressesList {
		if ad == myAddress {
			continue
		}
		go func(ad string, wg *sync.WaitGroup) {
			client, err := CreateP2PClient(h.protocol, ad)
			if err != nil {
				log.Println("broadcast error to address", ad)
				return
			}
			var reply bool
			err = client.Broadcast(bcdata, &reply)
			if err == nil && reply {
				log.Println("broadcast success to", ad)
			} else {
				log.Println("broadcast error", ad, err, reply)
			}
			defer wg.Done()
		}(ad, &wg)
	}
	wg.Wait()
	log.Println("broadcast send done.")
}

// =========================================

// ListenQueryLevel ...
func (h *HippoCoinClient) ListenQueryLevel(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-h.syncContext.Done():
			log.Println("listen sync done.")
			return
		case query := <-h.syncQuery:
			log.Println("listen sync:", query)
			go h.query(query)
		}
	}
}

func (h *HippoCoinClient) query(q QueryLevelStruct) {
	h.syncQuery <- q
}

// querySend ...
func (h *HippoCoinClient) querySend(q QueryLevelStruct) {
	newAddressesList := make([]string, 0)

	h.neighbors.Range(func(k, v interface{}) bool {
		ad := k.(string)
		newAddressesList = append(newAddressesList, ad)
		return true
	})
	var wg sync.WaitGroup
	wg.Add(len(newAddressesList))
	for _, address := range newAddressesList {
		go func(ad string, wg *sync.WaitGroup) {
			defer wg.Done()
			client, err := CreateP2PClient(h.protocol, ad)
			if err != nil {
				log.Println("query level error to address", ad)
				return
			}
			var reply QueryResponse
			err = client.QueryLevel(q.Level0, q.Level1, &reply)
			if err == nil {
				log.Println("query level success to", ad)
			} else {
				log.Println("query level error", ad, err, reply)
			}

			// Add to storage
			for _, b := range reply {
				h.storage.Add(&b)
			}
		}(address, &wg)
	}
	wg.Wait()
	log.Println("query level send done.")
}

// ======================================================

// CreateP2PClient ...
// This should be run after HippoCoinClient has init P2P server
// and has its address.
func CreateP2PClient(protocol, regAddress string) (client *P2PClient, err error) {
	client = new(P2PClient)
	client.c, err = rpc.Dial(protocol, regAddress)
	if err != nil {
		log.Println("create p2p client error:", err)
		return nil, err
	}
	return client, err
}

// Ping ...
func (c *P2PClient) Ping(request string, reply *string) error {
	return c.c.Call(HippoP2PServiceName+".Ping", request, reply)
}

// Close ...
func (c *P2PClient) Close() {
	c.c.Close()
}

// =========================================================

// Broadcast ...
func (c *P2PClient) Broadcast(bcdata BroadcastStruct, reply *bool) error {
	encode, err := bcdata.Encode()
	if err != nil {
		return err
	}
	return c.c.Call(HippoP2PServiceName+".Broadcast", encode, reply)
}

// QueryLevel ...
func (c *P2PClient) QueryLevel(level0, level1 uint32, reply *QueryResponse) error {
	return c.c.Call(HippoP2PServiceName+".QueryLevel",
		QueryLevelStruct{
			Level0: level0,
			Level1: level1,
		}, reply)
}

package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"time"

	registerlib "github.com/XieGuochao/HippoCoinRegister/lib"
)

// InitRegisterClient ...
func (client *HippoCoinClient) InitRegisterClient() (err error) {
	log.Println("init register client:", client.registerAddress)
	client.registerClient, err = registerlib.CreateClient(client.protocol, client.registerAddress)
	if err != nil {
		log.Fatalln("init register client error:", err)
	}
	var response string
	res := client.registerClient.Ping("", &response)
	assert(res == nil)
	log.Println("init register client done")
	return
}

// CreateListener ...
func (client *HippoCoinClient) CreateListener() {
	log.Println("create listener")

	listener, err := net.Listen(client.protocol, ":0")
	if err != nil {
		log.Fatalln(err)
	}
	client.listener = listener

	outerIP := registerlib.GetOutboundIP().String()
	client.outerIP = outerIP
	client.port = listener.Addr().(*net.TCPAddr).Port
	log.Println("create listener done:", client.outerIP, client.port)
}

// NetworkAddress ...
func (client *HippoCoinClient) NetworkAddress() string {
	return client.outerIP + ":" + strconv.Itoa(client.port)
}

// Register ...
func (client *HippoCoinClient) Register() error {
	c := client.registerClient
	var reply string
	err := c.Register(client.NetworkAddress(), &reply)
	log.Println("register:", err, reply)
	return err
}

// CountNeighbors ...
func (client *HippoCoinClient) CountNeighbors() (count int) {
	client.neighbors.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// UpdateNeighbors ...
func (client *HippoCoinClient) UpdateNeighbors() error {
	c := client.registerClient
	var reply []byte
	err := c.AddressesRefresh(registerlib.RefreshStruct{
		Number:  client.maxNeighbors,
		Address: client.NetworkAddress(),
	}, &reply)

	if err != nil {
		log.Println("update neighbor error:", err)
		return err
	}

	var neighbors []string
	json.Unmarshal(reply, &neighbors)
	log.Println("update neighbor get:", neighbors)

	for _, n := range neighbors {
		client.Ping(n)
	}
	return err
}

// ping
func ping(address string, c *HippoCoinClient) (int64, bool) {
	t := time.Now()
	// do ping here
	return time.Since(t).Nanoseconds(), true
}

// Ping ...
// Ping the address and store the result to neighbors.
func (client *HippoCoinClient) Ping(address string) {
	// c := client.registerClient
	t, ok := ping(address, client)
	log.Printf("ping %s : %s %d", address, strconv.FormatBool(ok), t)
	if ok {
		client.neighbors.Store(address, t)
	} else {
		client.neighbors.Delete(address)
	}
}

// TryUpdateNeighbors ...
// Count the number of neighbors and update the neighbor list.
// Return whether neighbors are updated or not.
func (client *HippoCoinClient) TryUpdateNeighbors() bool {
	count := client.CountNeighbors()
	if count >= client.maxNeighbors {
		return false
	}
	err := client.UpdateNeighbors()
	if err != nil {
		log.Println("try update neighbors error:", err)
		return false
	}
	return true
}

// NeighborPing ...
type NeighborPing struct {
	Address string
	Ping    int64
}

// EvictNeighbors ...
// Evict neighbors based on the ping time.
func (client *HippoCoinClient) EvictNeighbors() {
	neighbors := make([]NeighborPing, 0)
	client.neighbors.Range(func(k, v interface{}) bool {
		neighbors = append(neighbors, NeighborPing{
			Address: k.(string),
			Ping:    v.(int64),
		})
		return true
	})
	if len(neighbors) < client.maxNeighbors {
		return
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Ping < neighbors[j].Ping
	})

	// Delete those slow neighbors
	for _, n := range neighbors[client.maxNeighbors:] {
		client.neighbors.Delete(n.Address)
	}
}

// StartSyncRegister ...
func (client *HippoCoinClient) StartSyncRegister() {
	client.InitRegisterClient()
	client.Register()

	for {
		select {
		case <-client.registerContext.Done():
			log.Println("Stop sync with register")
			return
		default:
			client.TryUpdateNeighbors()
			client.EvictNeighbors()
			seconds := 5 + rand.Intn(4)
			time.Sleep(time.Second * time.Duration(seconds))
		}
	}
}

// StartNetwork ...
// Start listener
// Start sync with register
func (client *HippoCoinClient) StartNetwork() {
	client.InitP2PServer()
	client.InitP2PClient()
	go client.StartSyncRegister()
}

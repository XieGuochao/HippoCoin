package main

import (
	"context"
	"net"
	"net/http"
	"net/rpc"
)

// P2PServiceInterface ...
// Steps:
// 1. new(ctx)
// 2. setStorage(storage)
// 3. setBroadcastQueue(broadcastQueue)
// 4. serve()
type P2PServiceInterface interface {
	new(context.Context, net.Listener)
	setStorage(Storage)
	setBroadcastQueue(BroadcastQueue)
	Ping(request string, reply *string) error
	BroadcastBlock(sendBlockByte []byte, reply *string) error
	QueryLevel(q QueryLevelStruct, reply *[]byte) error
	serve()
}

// RegisterP2PService ...
func RegisterP2PService(svc P2PServiceInterface) error {
	return rpc.RegisterName(P2PServiceName, svc)
}

// P2PServer ...
type P2PServer struct {
	storage        Storage
	broadcastQueue BroadcastQueue
	ctx            context.Context
	cancel         context.CancelFunc
	listener       net.Listener
}

// new ...
func (s *P2PServer) new(parentContext context.Context, listener net.Listener) {
	s.ctx, s.cancel = context.WithCancel(parentContext)
	s.listener = listener
	logger.Info("register p2p server error:", RegisterP2PService(s))
	rpc.HandleHTTP()
	go http.Serve(listener, nil)
}

// serve ...
func (s *P2PServer) serve() {
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				logger.Info("p2p server close.")
				return
			default:
				conn, err := s.listener.Accept()
				if err != nil {
					logger.Error("p2p server accept error:", err)
				} else {
					go rpc.ServeConn(conn)
				}
			}
		}
	}()
}

// setStorage ...
func (s *P2PServer) setStorage(storage Storage) { s.storage = storage }

// setBroadcastQueue ...
func (s *P2PServer) setBroadcastQueue(broadcastQueue BroadcastQueue) {
	s.broadcastQueue = broadcastQueue
}

// Ping ...
func (s *P2PServer) Ping(request string, reply *string) error {
	*reply = request
	return nil
}

// // BroadcastData ...
// func (s *P2PServer) BroadcastData(sendByte []byte, reply *string) error {
// 	return nil
// }

// BroadcastBlock ...
func (s *P2PServer) BroadcastBlock(sendBlockByte []byte,
	reply *string) error {
	var receiveBlock ReceiveBlock
	receiveBlock.Data = sendBlockByte
	var (
		broadcastBlock BroadcastBlock
		block          Block
	)
	logger.Debug("receive bytes:", sendBlockByte)
	receiveBlock.Decode(&broadcastBlock)
	logger.Debug("receive block:", broadcastBlock)
	block = broadcastBlock.Block

	// Check block
	if !block.Check() {
		*reply = "check fail"
		return nil
	}

	// If check block ok, add to storage
	if s.storage != nil {
		s.storage.Add(block)
	} else {
		logger.Debug("no storage in rpc server")
	}

	// and broadcast.
	if s.broadcastQueue != nil {
		s.broadcastQueue.Add(broadcastBlock)
	} else {
		logger.Debug("no broadcast queue in rpc server")
	}

	return nil
}

// QueryLevel ...
func (s *P2PServer) QueryLevel(q QueryLevelStruct, reply *[]byte) error {
	var blocks []Block
	if s.storage != nil {
		blocks = s.storage.GetBlocksLevel(q.Level0, q.Level1)
	}
	*reply = EncodeBlocks(blocks)
	return nil
}

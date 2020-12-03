package host

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
// 4. setBlockTemplate(block) : curve, hashFunction, balance
// 5. serve()
type P2PServiceInterface interface {
	new(context.Context, net.Listener)
	setStorage(Storage)
	setBroadcastQueue(BroadcastQueue)
	setBlockTemplate(block Block)
	Ping(request string, reply *string) error
	BroadcastBlock(sendBlockByte []byte, reply *string) error
	QueryLevel(q QueryLevelStruct, reply *[]string) error
	QueryByHash(h string, blockBytes *[]byte) error
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

	blockTemplate Block
}

// new ...
func (s *P2PServer) new(parentContext context.Context, listener net.Listener) {
	s.ctx, s.cancel = context.WithCancel(parentContext)
	s.listener = listener
	infoLogger.Debug("register p2p server error:", RegisterP2PService(s))
	rpc.HandleHTTP()
	infoLogger.Debug("start serving HTTP")
	go http.Serve(listener, nil)
}

func (s *P2PServer) setBlockTemplate(block Block) { s.blockTemplate = block }

// serve ...
func (s *P2PServer) serve() {
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				infoLogger.Debug("p2p server close.")
				return
			default:
				conn, err := s.listener.Accept()
				infoLogger.Debug("p2p server receive conn:", err)
				if err != nil {
					infoLogger.Error("p2p server accept error:", err)
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
	debugLogger.Debug("receive ping:", request)
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
	broadcastBlock.Block = s.blockTemplate
	// debugLogger.Debug("receive bytes:", sendBlockByte)
	receiveBlock.Decode(&broadcastBlock)
	// debugLogger.Debug("receive block:", broadcastBlock)
	block = broadcastBlock.Block

	block.CopyConstants(s.blockTemplate)

	// Check block
	if !block.Check() {
		*reply = "check fail"
		return nil
	}

	// If check block ok, add to storage
	if s.storage != nil {
		s.storage.Add(block)
	} else {
		debugLogger.Debug("no storage in rpc server")
	}

	// and broadcast.
	if s.broadcastQueue != nil {
		broadcastBlock.Level++
		s.broadcastQueue.Add(broadcastBlock)
	} else {
		debugLogger.Debug("no broadcast queue in rpc server")
	}

	return nil
}

// QueryLevel ...
func (s *P2PServer) QueryLevel(q QueryLevelStruct, reply *[]string) error {
	var hashes []string
	if s.storage != nil {
		hashes = s.storage.GetBlocksLevelHash(q.Level0, q.Level1)
	}
	*reply = hashes
	return nil
}

// QueryByHash ...
func (s *P2PServer) QueryByHash(h string, blockBytes *[]byte) error {
	if s.storage != nil {
		block, has := s.storage.Get(h)
		if has {
			*blockBytes = block.Encode()
		}
	}
	return nil
}

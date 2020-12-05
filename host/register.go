package host

import (
	"context"

	registerlib "github.com/XieGuochao/HippoCoinRegister/lib"
)

// Register ...
// Steps:
// 0. Get the ip and port: NetworkListener
// 1. New(ctx, address)
// 2. Register()
// 3. Stop()
// 4. Client()
type Register interface {
	New(ctx context.Context, address, protocol string)
	// Register() error
	Client() *registerlib.Client
	Stop()
}

// HippoRegister ...
type HippoRegister struct {
	ctx      context.Context
	cancel   context.CancelFunc
	address  string
	protocol string
	client   *registerlib.Client
}

// New ...
func (r *HippoRegister) New(ctx context.Context, address, protocol string) {
	r.address, r.protocol = address, protocol
	r.ctx, r.cancel = context.WithCancel(ctx)

	// var response string
	// err = r.client.Ping("", &response)
	// if err != nil {
	// 	infoLogger.Fatal("new register client error:", err)
	// }
	// debugLogger.Debug("register ping success")

	// p2p service register

	infoLogger.Info("new register client done.")
}

// Register ...
func (r *HippoRegister) Register() error {
	var (
		reply string
		err   error
	)
	r.client, err = registerlib.CreateClient(r.protocol, r.address)
	if err != nil {
		infoLogger.Fatal("new register client error:", err)
	}
	debugLogger.Debug("create register client success", r.client)

	err = r.client.Register(r.address, &reply)
	debugLogger.Debug("register:", reply)
	return err
}

// Client ...
func (r *HippoRegister) Client() *registerlib.Client {
	var err error
	r.client, err = registerlib.CreateClient(r.protocol, r.address)
	if err != nil {
		infoLogger.Fatal("new register client error:", err)
		return nil
	}
	debugLogger.Debug("create register client success", r.client)

	return r.client
}

// Stop ...
func (r *HippoRegister) Stop() {
	r.cancel()
	r.client.Close()
}

// Refresh ...
func (r *HippoRegister) Refresh() {
	var err error
	r.client.Close()
	r.client, err = registerlib.CreateClient(r.protocol, r.address)
	if err != nil {
		infoLogger.Fatal("new register client error:", err)
	}
	debugLogger.Debug("create register client success", r.client)

	infoLogger.Info("refresh register client done.")
}

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/withmandala/go-log"

	"os"

	. "github.com/XieGuochao/HippoCoin/host"
	"github.com/XieGuochao/HippoCoin/ui"
)

var version = "1.0"

var (
	debugLogger *log.Logger
	debugFile   *os.File
	infoLogger  *log.Logger

	u ui.UI
)

func initLogger(debugPath string) {
	var err error

	if debugPath == "" {
		debugLogger = log.New(os.Stdout)
	} else {
		debugFile, err = os.Create(debugPath)
		if err != nil {
			_ = fmt.Errorf("error: %s", err)
			return
		}
		debugLogger = log.New(debugFile)
	}

	debugLogger.WithDebug()
	debugLogger.WithoutColor()

	infoLogger = log.New(os.Stdout)
	infoLogger.WithoutDebug()
	infoLogger.WithColor()
}

func main() {
	fmt.Printf("Hippo Coin v%s\n", version)
	fmt.Println("By Guochao Xie")
	fmt.Println(`                                                     
                         @@@.  .@@                    
                        @@ .@@@@@@@                   
              @@@@@@@@@@@@        @@.                 
           @@@@               @@    @@@@@@@@.@@@      
         @@@.                 .                @@.    
        @@                                      @@    
       @@                                      .@@    
      @@@                   @                  .@@    
    ..@@@.                   @                  @@    
        @..                   @@               @@     
        .@ ...              . @@@@@ .  .   . @@@      
         .@@. .    @@@       .@@   @@@@@@@@@@         
           @@@@@@@@. .@@@@@@@@@                       
         ........................                
`)
	time.Sleep(time.Second)
	var (
		host   Host
		ctx    context.Context
		cancel context.CancelFunc

		config     HippoConfig
		configPath string
	)
	if len(os.Args) == 1 {
		configPath = "./host.yml"
	} else {
		configPath = os.Args[1]
	}

	config.Load(configPath)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	host = new(HippoHost)

	t := time.Now().Format("2006-01-02-15-04-05")

	var (
		debugPath = config.DebugFileTemplate
		infoPath  = config.InfoFileTemplate
	)
	if debugPath != "" && debugPath != "STDOUT" {
		debugPath = fmt.Sprintf(debugPath, t)
	}
	if infoPath != "" && infoPath != "STDOUT" {
		infoPath = fmt.Sprintf(infoPath, t)
	}

	host.New(true, debugPath, infoPath, config.curve, config.LocalMode)
	host.InitLogger(true)
	debugLogger, infoLogger = host.GetLoggers()
	infoLogger.Info("localmode:", config.LocalMode)

	fmt.Println("output to debug file:", t+"-debug.out")

	runtime.GOMAXPROCS(config.MiningThreads + 1)
	fmt.Println("set max procs:", config.MiningThreads+2)
	host.InitLocals(ctx, Hash, config.miningFunction, config.MiningThreads,
		new(P2PClient), uint(config.BroadcastQueueLen), MiningCallbackBroadcastSave,
		BasicDifficulty, int64(config.MiningInterval), config.MiningCapacity,
		int64(config.MiningTTL), config.Protocol)
	host.InitNetwork(new(HippoBlock), new(HippoTransaction), config.MaxNeighbors, config.UpdateTimeBase, config.UpdateTimeRand,
		config.RegisterAddress, config.RegisterProtocol, config.ListenerPort)

	u.New(debugLogger, infoLogger, host)
	u.Main(config.UIPort)

	host.Run()
}

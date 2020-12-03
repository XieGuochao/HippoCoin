package ui

import (
	"fmt"

	"github.com/withmandala/go-log"

	"github.com/XieGuochao/HippoCoin/host"
	"github.com/gin-gonic/gin"
)

// UI ...
type UI struct {
	h           host.Host
	r           *gin.Engine
	debugLogger *log.Logger
	infoLogger  *log.Logger
}

// UIBlock ...
type UIBlock struct {
	Hash         string
	ParentHash   string
	Transactions []UITransaction
	Level        int
	Miner        string
}

// UITransaction ...
type UITransaction struct {
	SenderAddreesses  []string
	SenderAmounts     []uint64
	ReceiverAddresses []string
	ReceiverAmounts   []uint64
	Fee               uint64
	TimeStamp         int64
}

// New ...
func (u *UI) New(debugLogger, infoLogger *log.Logger, h host.Host) {
	u.debugLogger, u.infoLogger = debugLogger, infoLogger
	fmt.Println("ui new:", u.debugLogger != nil, u.infoLogger != nil)
	u.h = h
	u.r = gin.Default()
	u.r.LoadHTMLGlob("./templates/*")

	u.r.Use(func(c *gin.Context) {
		if u.h != nil {
			c.Set("public-key", u.h.PublicKey())
			c.Set("address", u.h.Address())
			c.Set("private-key", u.h.PrivateKey())
			c.Set("balance", u.h.GetBalance())
		} else {
			c.Set("public-key", "no public key")
			c.Set("address", "no address")
			c.Set("private-key", "no private key")
			c.Set("balance", map[string]uint64{})
		}
	})

	u.r.GET("/myaccount", func(c *gin.Context) {
		var balance = make(map[string]uint64)
		balanceInterface, has := c.Get("balance")
		if has {
			balance = balanceInterface.(map[string]uint64)
		}

		myBalance, has := balance[c.GetString("public-key")]
		if !has {
			myBalance = 0
		}
		c.HTML(200, "myaccount.html", gin.H{
			"address":    c.GetString("address"),
			"privateKey": c.GetString("private-key"),
			"publicKey":  c.GetString("public-key"),
			"myBalance":  myBalance,
		})
	})

	u.r.GET("/block/:hash", func(c *gin.Context) {
		hash := c.Param("hash")
		var blocks map[string]host.Block
		if u.h == nil {
			c.String(500, "no host connected")
			return
		}
		blocks = u.h.AllBlocks()
		if b, has := blocks[hash]; !has {
			c.Status(404)
			return
		} else {
			block := UIBlock{
				Hash:       b.Hash(),
				ParentHash: b.ParentHash(),
				Level:      b.GetLevel(),
				Miner:      b.GetMiner(),
			}
			trs := b.GetTransactions()
			block.Transactions = make([]UITransaction, len(trs))
			for i, tr := range trs {
				block.Transactions[i] = UITransaction{
					Fee:       tr.GetFee(),
					TimeStamp: tr.GetTimestamp(),
				}
				block.Transactions[i].SenderAddreesses, block.Transactions[i].SenderAmounts = tr.GetSender()
				block.Transactions[i].ReceiverAddresses, block.Transactions[i].ReceiverAmounts = tr.GetReceiver()

			}
			c.HTML(200, "block.html", gin.H{
				"block":        block,
				"transactions": block.Transactions,
				"publicKey":    c.GetString("public-key"),
				"address":      c.GetString("address"),
			})
		}
	})

	u.r.GET("/", func(c *gin.Context) {
		var levelHashes map[int][]string
		var levels [][]UIBlock
		var blocks map[string]host.Block

		if u.h != nil {
			levelHashes = u.h.AllHashesInLevel()
			blocks = u.h.AllBlocks()
			u.debugLogger.Debug(blocks)
			if levelHashes == nil {
				levelHashes = make(map[int][]string)
			}
		}

		maxLevel := -1
		for l := range levelHashes {
			if l > maxLevel {
				maxLevel = l
			}
		}
		levels = make([][]UIBlock, maxLevel+1)
		for l, oneLevel := range levelHashes {
			levels[l] = make([]UIBlock, len(oneLevel))

			for i, h := range oneLevel {
				b, has := blocks[h]
				if !has {
					continue
				}
				parentHash := b.ParentHash()

				if len(b.ParentHashBytes()) == 0 {
					parentHash = "Genisus!"
				}
				levels[l][i] = UIBlock{
					Hash:       h,
					ParentHash: parentHash,
					Level:      b.GetLevel(),
					Miner:      b.GetMiner(),
				}
			}
		}

		balanceInterface, _ := c.Get("balance")
		balance := balanceInterface.(map[string]uint64)

		c.HTML(200, "index.html", gin.H{
			"levels":    levels,
			"publicKey": c.GetString("public-key"),
			"address":   c.GetString("address"),
			"balance":   balance,
		})
	})

}

// Main ...
func (u *UI) Main(port string) {
	go u.r.Run(":" + port)
}

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
	Transactions []host.Transaction
}

// New ...
func (u *UI) New(debugLogger, infoLogger *log.Logger, h host.Host) {
	u.debugLogger, u.infoLogger = debugLogger, infoLogger
	fmt.Println("ui new:", u.debugLogger, u.infoLogger)
	u.h = h
	u.r = gin.Default()
	u.r.LoadHTMLGlob("./templates/*")

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
				}
			}
		}

		c.HTML(200, "index.html", gin.H{
			"levels": levels,
		})
	})
}

// Main ...
func (u *UI) Main(port string) {
	go u.r.Run(":" + port)
}

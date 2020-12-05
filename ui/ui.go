package ui

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

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
	Hash          string
	ParentHash    string
	Transactions  []UITransaction
	Level         int
	Miner         string
	Time          string
	BalanceChange map[string]int64
	NumBytes      uint
}

// UITransaction ...
type UITransaction struct {
	SenderAddresses   []string
	SenderAmounts     []uint64
	ReceiverAddresses []string
	ReceiverAmounts   []uint64
	Fee               uint64
	Time              string
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

	u.r.POST("/transfer-post", func(c *gin.Context) {
		var (
			SenderAddresses   []string
			senderAmounts     []uint64
			senderKeys        []host.Key
			receiverAddresses []string
			receiverAmounts   []uint64
			// fee               uint64

			numSenders   = 0
			numReceivers = 0

			err error
		)
		c.MultipartForm()

		for key, value := range c.Request.PostForm {
			infoLogger.Info(key, value)
		}

		// First, iterate through sender amounts and receiver amounts to determine size.
		for key, value := range c.Request.PostForm {
			switch {
			case len(value) == 0 || value[0] == "0":
				continue
			case strings.Contains(key, "sender-amount-"):
				v, err := strconv.Atoi(key[len("sender-amount-"):])
				if err != nil {
					infoLogger.Error("transfer-post error:", err)
					c.String(http.StatusBadRequest, err.Error())
					return
				}
				if v > numSenders {
					numSenders = v
				}
			case strings.Contains(key, "receiver-amount-"):
				v, err := strconv.Atoi(key[len("receiver-amount-"):])
				if err != nil {
					infoLogger.Error("transfer-post error:", err)
					c.String(http.StatusBadRequest, err.Error())
					return
				}
				if v > numReceivers {
					numReceivers = v
				}
			}
		}

		numSenders++
		numReceivers++

		SenderAddresses = make([]string, 0)
		senderAmounts = make([]uint64, 0)
		senderKeys = make([]host.Key, 0)

		receiverAddresses = make([]string, 0)
		receiverAmounts = make([]uint64, 0)

		// feeStr := c.Request.FormValue("fee")
		// feeInt, _ := strconv.Atoi(feeStr)
		// fee = uint64(feeInt)

		for i := 0; i < numSenders; i++ {
			var key, value string
			key = fmt.Sprintf("sender-addr-%d", i)
			if value = c.Request.PostFormValue(key); value == "" {
				c.String(http.StatusBadRequest, "empty sender address.")
				return
			}
			SenderAddresses = append(SenderAddresses, value)

			key = fmt.Sprintf("sender-amount-%d", i)
			if value = c.Request.PostFormValue(key); value == "" {
				c.String(http.StatusBadRequest, "empty sender amount.")
				return
			}
			if amount, err := strconv.Atoi(value); err != nil {
				infoLogger.Error("transfer-post error:", err)
				c.String(http.StatusBadRequest, err.Error())
				return
			} else {
				senderAmounts = append(senderAmounts, uint64(amount))
			}

			key = fmt.Sprintf("sender-key-%d", i)
			if value = c.Request.PostFormValue(key); value == "" {
				c.String(http.StatusBadRequest, "empty sender key.")
				return
			}
			senderKey := host.Key{}
			if err = senderKey.LoadPrivateKeyString(value, h.GetCurve()); err != nil {
				infoLogger.Error("transfer-post error:", err)
				c.String(http.StatusBadRequest, err.Error())
				return
			}
			infoLogger.Info("sender key:", senderKey, senderKey.Key().Curve)
			senderKeys = append(senderKeys, senderKey)
		}

		for i := 0; i < numReceivers; i++ {
			var key, value string
			key = fmt.Sprintf("receiver-addr-%d", i)
			if value = c.Request.PostFormValue(key); value == "" {
				c.String(http.StatusBadRequest, "empty receiver address.")
				return
			}
			receiverAddresses = append(receiverAddresses, value)

			key = fmt.Sprintf("receiver-amount-%d", i)
			if value = c.Request.PostFormValue(key); value == "" {
				c.String(http.StatusBadRequest, "empty receiver amount")
				return
			}
			if amount, err := strconv.Atoi(value); err != nil {
				infoLogger.Error("transfer-post error:", err)
				c.String(http.StatusBadRequest, err.Error())
				return
			} else {
				receiverAmounts = append(receiverAmounts, uint64(amount))
			}
		}

		var newTransaction host.Transaction
		var ok bool
		newTransaction = new(host.HippoTransaction)
		newTransaction.New(h.GetHashFunction(), h.GetCurve())
		if ok = newTransaction.SetSender(SenderAddresses, senderAmounts); !ok {
			c.String(http.StatusBadRequest, "set senders failed.")
			infoLogger.Error("ui: transfer-post set senders failed.")
			return
		}
		if ok = newTransaction.SetReceiver(receiverAddresses, receiverAmounts); !ok {
			c.String(http.StatusBadRequest, "set receivers failed.")
			infoLogger.Error("ui: transfer-post set receivers failed.")
			return
		}
		if ok = newTransaction.UpdateFee(); !ok {
			c.String(http.StatusBadRequest, "Wrong fee!")
		}
		for _, key := range senderKeys {
			if ok = newTransaction.Sign(key); !ok {
				c.String(http.StatusBadRequest, "sign failed.")
				infoLogger.Error("ui: transfer-post sign failed.")
				return
			}
		}

		if ok = h.AddTransaction(newTransaction); !ok {
			c.String(http.StatusBadRequest, "host add transaction failed.")
			infoLogger.Error("ui: transfer-post transaction check failed.")
			return
		}

		infoLogger.Info("transfer-post success")
		c.String(200, "OK")
	})

	u.r.GET("/transfer", func(c *gin.Context) {
		var balance = make(map[string]uint64)
		balanceInterface, has := c.Get("balance")
		if has {
			balance = balanceInterface.(map[string]uint64)
		}

		myBalance, has := balance[c.GetString("public-key")]
		if !has {
			myBalance = 0
		}
		c.HTML(200, "transfer.html", gin.H{
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
				Hash:          b.Hash(),
				ParentHash:    b.ParentHash(),
				Level:         b.GetLevel(),
				Miner:         b.GetMiner(),
				BalanceChange: b.GetBalanceChange(),
				Time:          time.Unix(b.GetTimestamp(), 0).UTC().String(),
				NumBytes:      b.GetNumBytes(),
			}
			trs := b.GetTransactions()
			block.Transactions = make([]UITransaction, len(trs))
			for i, tr := range trs {
				block.Transactions[i] = UITransaction{
					Fee:  tr.GetFee(),
					Time: time.Unix(tr.GetTimestamp(), 0).UTC().String(),
				}
				block.Transactions[i].SenderAddresses, block.Transactions[i].SenderAmounts = tr.GetSender()
				block.Transactions[i].ReceiverAddresses, block.Transactions[i].ReceiverAmounts = tr.GetReceiver()
			}
			infoLogger.Info("ui-transaction:", block.Transactions)
			c.HTML(200, "block.html", gin.H{
				"block":        block,
				"transactions": block.Transactions,
				"publicKey":    c.GetString("public-key"),
				"address":      c.GetString("address"),
			})
		}
	})

	u.r.StaticFS("/show-log", http.Dir("log"))

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
		levelNumbers := make([]int, maxLevel+1)
		for l, oneLevel := range levelHashes {
			levels[l] = make([]UIBlock, len(oneLevel))
			levelNumbers[l] = maxLevel - l

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

		reverseAny(levels)
		c.HTML(200, "index.html", gin.H{
			"levels":      levels,
			"levelNumber": levelNumbers,
			"publicKey":   c.GetString("public-key"),
			"address":     c.GetString("address"),
			"balance":     balance,
		})
	})

}

// Main ...
func (u *UI) Main(port string) {
	go u.r.Run(":" + port)
}

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

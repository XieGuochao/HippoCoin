package host

import (
	"fmt"
	"testing"
)

func TestBlock(t *testing.T) {
	initTest(3)
	infoLogger.Debug("test block ===============================================")
	// balance
	balance := new(HippoBalance)
	balance.New()
	balance.Store(testKeys[0].ToAddress(), 20)

	tr := HippoTransaction{}
	tr.New(testHashfunction, testCurve)

	// transfer a 10 coins to b and 5 coins to c; 5 more for Fee.
	senders := make([]string, 1)
	senders[0] = testKeys[0].ToAddress()

	senderAmounts := make([]uint64, 1)
	senderAmounts[0] = 20
	tr.SetSender(senders, senderAmounts)

	receivers := make([]string, 2)
	receivers[0] = testKeys[1].ToAddress()
	receivers[1] = testKeys[2].ToAddress()
	receiverAmounts := make([]uint64, 2)
	receiverAmounts[0] = 10
	receiverAmounts[1] = 5
	tr.SetReceiver(receivers, receiverAmounts)
	tr.UpdateFee()

	// Sign
	tr.Sign(testKeys[0])
	debugLogger.Debug("check:", tr.Check(balance))

	// Get balance
	debugLogger.Debug("balances:", balance.Get(testKeys[0].ToAddress()),
		balance.Get(testKeys[1].ToAddress()), balance.Get(testKeys[2].ToAddress()))

	trs := make([]Transaction, 1)
	trs[0] = &tr
	// block
	block := new(HippoBlock)
	block.New([]byte{}, 250, testHashfunction, 0, balance, testCurve)
	block.SetTransactions(trs)
	debugLogger.Debug("check transactions:", block.CheckTransactions())

	debugLogger.Debug("check sign before:", block.CheckSignature())
	block.Sign(testKeys[0])
	debugLogger.Debug("check sign:", block.CheckSignature())

	debugLogger.Debug("check nonce:", block.CheckNonce())
	debugLogger.Debug("check:", block.Check())
}

func TestJsonBlock(t *testing.T) {
	initTest(3)
	block := new(HippoBlock)
	block.New([]byte{}, 250, testHashfunction, 0, nil, testCurve)

	var tr Transaction
	tr = new(HippoTransaction)
	tr.New(testHashfunction, testCurve)

	// transfer a 10 coins to b and 5 coins to c; 5 more for Fee.
	senders := make([]string, 1)
	senders[0] = testKeys[0].ToAddress()

	senderAmounts := make([]uint64, 1)
	senderAmounts[0] = 20
	tr.SetSender(senders, senderAmounts)

	receivers := make([]string, 2)
	receivers[0] = testKeys[1].ToAddress()
	receivers[1] = testKeys[2].ToAddress()
	receiverAmounts := make([]uint64, 2)
	receiverAmounts[0] = 10
	receiverAmounts[1] = 5
	tr.SetReceiver(receivers, receiverAmounts)
	tr.UpdateFee()

	// Sign
	tr.Sign(testKeys[0])

	trs := make([]Transaction, 1)
	trs[0] = tr
	block.SetTransactions(trs)
	block.Sign(testKeys[0])

	bb := BroadcastBlock{block: block}
	var (
		bytes []byte
		// err   error
	)
	bytes = bb.Encode()
	fmt.Printf("\n%+v\n", *block)
	debugLogger.Debug(string(bytes))

	var rb ReceiveBlock
	var bb2 BroadcastBlock
	rb.Data = bytes

	rb.Decode(&bb2)
	debugLogger.Debug(bb2.block, bb2.block.GetTransactions(), bb2.Level, bb2.Addresses)
}

func TestEncodeBlock(t *testing.T) {
	initTest(3)
	infoLogger.WithDebug()
	infoLogger.Info("test block ===============================================")
	// balance
	balance := new(HippoBalance)
	balance.New()
	balance.Store(testKeys[0].ToAddress(), 20)

	var tr Transaction
	tr = new(HippoTransaction)
	tr.New(testHashfunction, testCurve)

	// transfer a 10 coins to b and 5 coins to c; 5 more for Fee.
	senders := make([]string, 1)
	senders[0] = testKeys[0].ToAddress()

	senderAmounts := make([]uint64, 1)
	senderAmounts[0] = 20
	tr.SetSender(senders, senderAmounts)

	receivers := make([]string, 2)
	receivers[0] = testKeys[1].ToAddress()
	receivers[1] = testKeys[2].ToAddress()
	receiverAmounts := make([]uint64, 2)
	receiverAmounts[0] = 10
	receiverAmounts[1] = 5
	tr.SetReceiver(receivers, receiverAmounts)
	tr.UpdateFee()

	// Sign
	tr.Sign(testKeys[0])
	debugLogger.Debug("check:", tr.Check(balance))

	// Get balance
	debugLogger.Debug("balances:", balance.Get(testKeys[0].ToAddress()),
		balance.Get(testKeys[1].ToAddress()), balance.Get(testKeys[2].ToAddress()))

	trs := make([]Transaction, 1)
	trs[0] = tr
	// block
	block := new(HippoBlock)
	block.New([]byte{}, 250, testHashfunction, 0, balance, testCurve)
	block.SetTransactions(trs)
	block.Sign(testKeys[0])

	bytes := block.Encode()
	infoLogger.Warn(string(bytes))

	var b2 Block
	b2 = block.CloneConstants()
	infoLogger.Debug("template:", b2)

	b2 = DecodeBlock(bytes, b2)
	infoLogger.Warn("\n\nafter   decode:", b2)
	infoLogger.Warn("\n\ncompare decode:", block)

	infoLogger.Warn("\n\nafter   decode transaction:", b2.GetTransactions()[0])
	infoLogger.Warn("\n\ncompare decode transaction:", block.GetTransactions()[0])

}

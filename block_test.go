package main

import "testing"

func TestBlock(t *testing.T) {
	initTest(3)
	logger.Info("test block ===============================================")
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
	logger.Debug("check:", tr.Check(balance))

	// Get balance
	logger.Debug("balances:", balance.Get(testKeys[0].ToAddress()),
		balance.Get(testKeys[1].ToAddress()), balance.Get(testKeys[2].ToAddress()))

	trs := make([]Transaction, 1)
	trs[0] = &tr
	// block
	block := new(HippoBlock)
	block.New([]byte{}, 250, testHashfunction, 0, balance, testCurve)
	block.SetTransactions(trs)
	logger.Debug("check transactions:", block.CheckTransactions())

	logger.Debug("check sign before:", block.CheckSignature())
	block.Sign(testKeys[0])
	logger.Debug("check sign:", block.CheckSignature())

	logger.Debug("check nonce:", block.CheckNonce())
	logger.Debug("check:", block.Check())
}

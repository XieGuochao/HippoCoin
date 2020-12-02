package host

import "testing"

func TestTransaction(t *testing.T) {
	initTest(3)
	infoLogger.Debug("TestTransaction=====================================================")
	tr := HippoTransaction{}
	tr.New(testHashfunction, testCurve)

	hash0 := tr.Hash()
	debugLogger.Debug("hash0:", hash0)

	// transfer a 10 (+3) coins to b.
	senders := make([]string, 1)
	senders[0] = testKeys[0].ToAddress()

	revertedPublicKey := stringToPublicKey(senders[0], testCurve)

	debugLogger.Debug("revert public key from address:", revertedPublicKey.Equal(testKeys[0].publicKey))

	senderAmounts := make([]uint64, 1)
	senderAmounts[0] = 13
	tr.SetSender(senders, senderAmounts)

	receivers := make([]string, 1)
	receivers[0] = testKeys[1].ToAddress()
	receiverAmounts := make([]uint64, 1)
	receiverAmounts[0] = 10
	tr.SetReceiver(receivers, receiverAmounts)
	tr.UpdateFee()

	hash1 := tr.Hash()
	debugLogger.Debug("hash1:", hash1)
	assertT(hash0 != hash1, t)

	// Sign
	sigCheck := tr.CheckSignatures()
	debugLogger.Debug("signature check:", sigCheck)
	assertT(!sigCheck, t)

	result := tr.Sign(testKeys[0])
	debugLogger.Debug(result)
	sigCheck = tr.CheckSignatures()
	debugLogger.Debug("signature check:", sigCheck)
	assertT(sigCheck, t)
}

func TestTransactionWithBalance(t *testing.T) {
	initTest(3)
	infoLogger.Debug("TestTransactionWithoutBalance==============================================")

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
	debugLogger.Debug("signature check:", tr.CheckSignatures())
	assertT(!tr.CheckSignatures(), t)

	result := tr.Sign(testKeys[0])
	debugLogger.Debug(result)
	debugLogger.Debug("signature check:", tr.CheckSignatures())
	assertT(tr.CheckSignatures(), t)

	debugLogger.Debug("check fee:", tr.CheckFee())
	assertT(tr.CheckFee(), t)

	balance := new(HippoBalance)
	balance.New()
	debugLogger.Debug("balance check before:", tr.CheckBalance(balance))
	assertT(!tr.CheckBalance(balance), t)

	balance.Store(senders[0], 20)
	debugLogger.Debug("balance check after:", tr.CheckBalance(balance))
	assertT(tr.CheckBalance(balance), t)

	debugLogger.Debug("check:", tr.Check(balance))

	// Get balance
	debugLogger.Debug("balances:", balance.Get(testKeys[0].ToAddress()),
		balance.Get(testKeys[1].ToAddress()), balance.Get(testKeys[2].ToAddress()))
}

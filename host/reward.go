package host

import (
	"math"
)

// RewardFunc ...
type RewardFunc func(Block) int64

func basicReward(block Block) int64 {
	var power = 0
	if block.GetLevel()/1000 < 10 {
		power = 10 - int(block.GetLevel()/1000)
	}
	return int64(math.Pow10(power))
}

// Reward ...
var Reward = basicReward

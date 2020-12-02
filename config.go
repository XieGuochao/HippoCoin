package main

import (
	"crypto/elliptic"
	"fmt"
	"io/ioutil"

	"github.com/XieGuochao/HippoCoin/host"
	"gopkg.in/yaml.v2"
)

// HippoConfig ...
type HippoConfig struct {
	Curve             string `yaml:"curve"`
	curve             elliptic.Curve
	MiningThreads     int `yaml:"mining-threads"`
	miningFunction    host.MiningFunction
	BroadcastQueueLen int    `yaml:"broadcast-queue-len"`
	MiningCapacity    int    `yaml:"mining-capacity"`
	MiningInterval    int    `yaml:"mining-interval"`
	MiningTTL         int    `yaml:"mining-ttl"`
	Protocol          string `yaml:"protocol"`

	MaxNeighbors   int `yaml:"max-neighbors"`
	UpdateTimeBase int `yaml:"update-time-base"`
	UpdateTimeRand int `yaml:"update-time-rand"`

	RegisterAddress  string `yaml:"register-address"`
	RegisterProtocol string `yaml:"register-protocol"`

	DebugPath string `yaml:"debug-path"`
}

// Load ...
func (config *HippoConfig) Load(path string) *HippoConfig {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = yaml.Unmarshal(yamlFile, config)

	if err != nil {
		fmt.Println(err.Error())
	}
	switch config.Curve {
	case "P224":
		config.curve = elliptic.P224()
	case "P256":
		config.curve = elliptic.P256()
	default:
		config.curve = elliptic.P224()
	}

	if config.MiningThreads <= 1 {
		config.miningFunction = new(host.SingleMiningFunction)
	} else {
		config.miningFunction = new(host.MultipleMiningFunction)
	}
	return config
}

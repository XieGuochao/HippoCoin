package main

import (
	"crypto/elliptic"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// HippoConfig ...
type HippoConfig struct {
	Curve             string `yaml:"curve"`
	curve             elliptic.Curve
	MiningThreads     int    `yaml:"mining-threads"`
	BroadcastQueueLen int    `yaml:"broadcast-queue-len"`
	MiningCapacity    int    `yaml:"mining-capacity"`
	MiningInterval    int    `yaml:"mining-interval"`
	Protocol          string `yaml:"protocol"`

	MaxNeighbors   int `yaml:"max-neighbors"`
	UpdateTimeBase int `yaml:"update-time-base"`
	UpdateTimeRand int `yaml:"update-time-rand"`

	RegisterAddress  string `yaml:"register-address"`
	RegisterProtocol string `yaml:"register-protocol"`
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

	return config
}

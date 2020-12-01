package main

import (
	"testing"
)

func TestConfig(t *testing.T) {
	var config HippoConfig
	config.Load("./host.yml")
	initLogger()
	logger.Info("config:", config)
}

package main

import (
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	var config HippoConfig
	config.Load("./host.yml")
	tstr := time.Now().Format("2006-01-02-15-04-05")
	initLogger(tstr + "-debug.out")
	infoLogger.Debug("config:", config)
}

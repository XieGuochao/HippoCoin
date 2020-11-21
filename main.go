package main

import (
	"os"

	"github.com/withmandala/go-log"
)

var version = "1.0"

func initLogger() {
	logger = log.New(os.Stdout)
	logger.WithDebug()
	logger.WithColor()
}

func main() {
	initLogger()
	logger.Infof("Hello world, HippoCoin %s.", version)
}

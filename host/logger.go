package host

import (
	"fmt"
	"os"

	"github.com/withmandala/go-log"
)

var (
	debugLogger *log.Logger
	debugFile   *os.File
	infoLogger  *log.Logger
	infoFile    *os.File
)

func initLogger(debugPath string, infoPath string) {
	var err error
	fmt.Println("debug path:", debugFile)

	if debugPath == "" || debugPath == "STDOUT" {
		debugLogger = log.New(os.Stdout)
	} else {
		debugFile, err = os.Create(debugPath)
		if err != nil {
			_ = fmt.Errorf("error: %s", err)
			return
		}
		fmt.Println("create debug file:", debugFile)
		debugLogger = log.New(debugFile)
	}

	debugLogger.WithDebug()
	debugLogger.WithoutColor()

	if infoPath == "" || infoPath == "STDOUT" {
		infoLogger = log.New(os.Stdout)
		infoLogger.WithColor()
	} else {
		infoFile, err = os.Create(infoPath)
		if err != nil {
			_ = fmt.Errorf("error: %s", err)
			return
		}
		fmt.Println("create info file:", infoFile)
		infoLogger = log.New(infoFile)
		infoLogger.WithoutColor()
	}
	infoLogger.WithoutDebug()
}

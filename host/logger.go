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
)

func initLogger(debugPath string) {
	var err error
	fmt.Println("debug path:", debugFile)

	if debugPath == "" {
		debugLogger = log.New(os.Stdout)
	} else {
		debugFile, err = os.Create(debugPath)
		if err != nil {
			fmt.Errorf("error: %s", err)
			return
		}
		fmt.Println("create debug file:", debugFile)
		debugLogger = log.New(debugFile)
	}

	debugLogger.WithDebug()
	debugLogger.WithoutColor()

	infoLogger = log.New(os.Stdout)
	infoLogger.WithoutDebug()
	infoLogger.WithColor()
}

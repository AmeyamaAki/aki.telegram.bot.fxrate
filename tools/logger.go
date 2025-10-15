// logger.go
package tools

import (
	"fmt"
	"time"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

func logMessage(level, color, format string, v ...interface{}) {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, v...)
	fmt.Printf("%s[%s] [%s]: %s%s \n", color, currentTime, level, message, ColorReset)
}

func LogError(format string, v ...interface{}) {
	logMessage("Error", ColorRed, format, v...)
}

func LogInfo(format string, v ...interface{}) {
	logMessage("Info", ColorGreen, format, v...)
}

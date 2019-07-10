package cmd

import (
	"fmt"
)

var LogLevels = map[string]int{
	"DEBUG": 1,
	"INFO":  2,
	"WARN":  3,
	"ERROR": 4,
}

var LogLevel = LogLevels["WARN"]

func LogWarn(args ...interface{}) {
	log(LogLevels["WARN"], args...)
}
func LogError(args ...interface{}) {
	log(LogLevels["ERROR"], args...)
}
func LogInfo(args ...interface{}) {
	log(LogLevels["INFO"], args...)
}
func LogDebug(args ...interface{}) {
	log(LogLevels["DEBUG"], args...)
}

// Wrapper method that should be used to print output. Using this instead of fmt
// let's you implement verbosity levels or disable output completely.
func log(targetLogLevel int, args ...interface{}) {
	if targetLogLevel >= LogLevel {
		fmt.Println(args...)
	}
}

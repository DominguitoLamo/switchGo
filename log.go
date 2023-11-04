package switchgo

import (
	"log"
	"fmt"
)

func DebugLog(format string, s ...interface{}) {
	log.Printf("DEBUG: %s\n", fmt.Sprintf(format, s...))
}

func InfoLog(format string, s ...interface{}) {
	log.Printf("INFO: %s\n", fmt.Sprintf(format, s...))
}

func ErrorLog(format string, s ...interface{}) {
	log.Printf("ERROR: %s\n", fmt.Sprintf(format, s...))
}
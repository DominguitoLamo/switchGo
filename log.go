package switchgo

import "log"

func DebugLog(s string) {
	log.Printf("DEBUG: %s\n", s)
}

func InfoLog(s string) {
	log.Printf("INFO: %s\n", s)
}

func ErrorLog(s string) {
	log.Printf("ERROR: %s\n", s)
}
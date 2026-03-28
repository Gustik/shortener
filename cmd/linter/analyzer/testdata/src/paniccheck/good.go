package paniccheck

import (
	"log"
	"os"
)

func noViolations() {
	log.Println("info log is fine")
	log.Printf("formatted: %s", "value")
	_ = os.Getenv("HOME")
}

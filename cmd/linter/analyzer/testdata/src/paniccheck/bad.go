package paniccheck

import (
	"log"
	"os"
)

func triggerPanic() {
	panic("boom") // want `use of panic is forbidden`
}
func triggerLogFatal()   { log.Fatal("fatal") }        // want `log\.Fatal is forbidden outside main\(\) of main package`
func triggerLogFatalf()  { log.Fatalf("fatal %d", 1) } // want `log\.Fatalf is forbidden outside main\(\) of main package`
func triggerLogFatalln() { log.Fatalln("fatal") }      // want `log\.Fatalln is forbidden outside main\(\) of main package`
func triggerOsExit()     { os.Exit(1) }                // want `os\.Exit is forbidden outside main\(\) of main package`

package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("allowed in main")
	os.Exit(0)
}

func notMain() {
	log.Fatal("not in main func")     // want `log\.Fatal is forbidden outside main\(\) of main package`
	os.Exit(1)                        // want `os\.Exit is forbidden outside main\(\) of main package`
	panic("still forbidden anywhere") // want `use of panic is forbidden`
}

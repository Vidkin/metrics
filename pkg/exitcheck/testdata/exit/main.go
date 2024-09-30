package main

import (
	"os"
)

func main() { // want "exit in main func of main package"
	os.Exit(0)
}

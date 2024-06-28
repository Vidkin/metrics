package main

import (
	"github.com/Vidkin/metrics/app"
)

func main() {
	serverApp, err := app.NewServerApp()
	if err != nil {
		panic(err)
	}
	serverApp.Run()
}

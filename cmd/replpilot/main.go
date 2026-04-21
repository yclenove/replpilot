package main

import (
	"log"

	"github.com/yclenove/replpilot/internal/command"
)

func main() {
	if err := command.NewRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

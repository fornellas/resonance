package main

import (
	"log"

	"github.com/fornellas/resonance/cli"
)

func main() {
	log.SetFlags(0)
	if err := cli.Cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

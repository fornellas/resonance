package main

import (
	"log"

	_ "github.com/fornellas/resonance/resources"
)

func main() {
	log.SetFlags(0)
	if err := RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

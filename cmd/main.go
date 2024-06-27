package main

import (
	"log"

	_ "github.com/fornellas/resonance/resource/types"
)

func main() {
	log.SetFlags(0)
	if err := RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"log"
)

func main() {
	log.SetFlags(0)
	if err := RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

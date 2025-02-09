package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fornellas/resonance"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the program version.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s\n", resonance.Version)
	},
}

func init() {
	RootCmd.AddCommand(VersionCmd)
}

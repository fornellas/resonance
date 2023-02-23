package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/fornellas/resonance/cli/apply"
	"github.com/fornellas/resonance/log"

	"github.com/spf13/cobra"
)

var logLevel string

func cobraInit() {
	if err := log.Setup(logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
	}
}

var Cmd = &cobra.Command{
	Use:   "resonance",
	Short: "Resonance is a configuration management tool.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			logrus.Fatal(err)
		}

		logrus.Fatal(errors.New("missing command"))
	},
}

func init() {
	cobra.OnInitialize(cobraInit)

	Cmd.Flags().StringVarP(
		&logLevel, "log-level", "l", "info",
		"Logging level",
	)

	Cmd.AddCommand(apply.Cmd)
}

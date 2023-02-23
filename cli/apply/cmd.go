package apply

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/host"
)

var localhost bool
var hostname string

var Cmd = &cobra.Command{
	Use:   "apply [flags] yaml...",
	Short: "Applies configuration to a host.",
	Long:  "Applies configuration at yaml files to a host.\n\nA target host must be specified with either --localhost or --hostname.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var hst host.Host
		if localhost {
			hst = host.Local{}
		} else if hostname != "" {
			hst = host.Ssh{
				Hostname: hostname,
			}
		} else {
			logrus.Fatal(errors.New("must provide either --localhost or --hostname"))
		}
		logrus.Infof("Host: %v", hst)

		logrus.Fatal("TODO apply.Run")
	},
}

func init() {
	Cmd.Flags().BoolVarP(
		&localhost, "localhost", "", false,
		"Applies configuration to the same machine running the command",
	)

	Cmd.Flags().StringVarP(
		&hostname, "hostname", "", "",
		"Applies configuration to given hostname using SSH",
	)

}

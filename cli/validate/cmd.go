package validate

import (
	"github.com/spf13/cobra"

	"github.com/fornellas/resonance/cli/lib"
	"github.com/fornellas/resonance/log"
	"github.com/fornellas/resonance/resource"
)

var Cmd = &cobra.Command{
	Use:   "validate [flags] resources_root",
	Short: "Validates resource files.",
	Long:  "Loads all resoures from .yaml files at resources_root validating whether they are ok.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		logger := log.GetLogger(ctx)

		root := args[0]

		// Host
		hst, err := lib.GetHost(ctx)
		if err != nil {
			logger.Fatal(err)
		}
		defer hst.Close()

		// Load resources
		_, err = resource.LoadDir(ctx, hst, root)
		if err != nil {
			logger.Fatal(err)
		}

		logger.Info("🎆 Validation successful")
	},
}

func Reset() {

}

func init() {
	lib.AddHostFlags(Cmd)
	lib.AddPersistantStateFlags(Cmd)
}

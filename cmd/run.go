package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/resonance/host/types"
)

var RunCmd = &cobra.Command{
	Use:   "run [flags] -- CMD [ARGS]",
	Short: "Run command on host.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "🔎 Inspect")

		var retErr error
		defer func() {
			if retErr != nil {
				logger.Error("Failed", "err", retErr)
				Exit(1)
			}
		}()

		host, ctx, err := GetHost(ctx)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to get host: %w", err))
			return
		}
		defer func() {
			if err := host.Close(ctx); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to close host: %w", err))
			}
		}()
		ctx, _ = log.MustWithAttrs(ctx, "host", fmt.Sprintf("%s => %s", host.Type(), host.String()))

		waitStatus, err := host.Run(ctx, types.Cmd{
			Path:   args[0],
			Args:   args[1:],
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed run: %w", err))
			return
		}
		if waitStatus.Exited {
			Exit(int(waitStatus.ExitCode))
		} else {
			retErr = errors.Join(retErr, fmt.Errorf("command did not exit: %s", waitStatus.String()))
			return
		}
	},
}

func init() {
	AddHostFlags(RunCmd)

	RootCmd.AddCommand(RunCmd)
}

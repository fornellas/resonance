package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fornellas/slogxt/log"

	discoverPkg "github.com/fornellas/resonance/discover"
)

var ignorePatterns []string
var defaultIgnorePatterns = []string{
	"**/*.dpkg-*",
	"**/*.o",
	"**/*.pyc",
	"**/*.update-old",
	"**/*~",
	"**/__pycache__",
	"/boot/System.map*",
	"/boot/config-*",
	"/boot/device.map",
	"/boot/efi",
	"/boot/grub",
	"/boot/initrd.img-*",
	"/boot/vmlinuz-*",
	"/dev", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s06.html
	"/etc/*-",
	"/etc/.pwd.lock",
	"/etc/ld.so.cache",
	"/etc/ld.so.cache",
	"/etc/mtap",
	"/etc/resolv.conf",
	"/home",  // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s08.html
	"/media", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s11.html
	"/mnt",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s12.html
	"/proc",  // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#procKernelAndProcessInformationVir
	"/root",  // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s14.html
	"/run",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s15.html
	"/snap",
	"/srv",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s17.html
	"/sys",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#sysKernelAndSystemInformation
	"/tmp",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s18.html
	"/usr/src", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch04s12.html
	"/var",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05.html
	// "/usr/share/doc", TODO
	// "/usr/share/help", TODO
	// "/usr/share/icons", TODO
	// "/usr/share/locale", TODO
	// "/usr/share/man", TODO
	// "/backup",           // FIXME
	// "/opt",              // FIXME
	// "/steam",            // FIXME
	// "/usr/lib/firmware", // FIXME
	// "/usr/lib/modules",  // FIXME
	// "/usr/lib/python*",  // FIXME
	// "/windows",          // FIXME
}

var ignoreFsTypes []string
var defaultIgnoreFsTypes = []string{
	"devpts",
	"devtmpfs",
	"efivarfs",
	"proc",
	"sysfs",
	"tmpfs",
}

var resourcesPath string
var defaultResourcesPath = ""

var DiscoverCmd = &cobra.Command{
	Use:   "discover [flags]",
	Short: "Discover host state.",
	Long:  "Audit all files and packages at host to identify what requires automation.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, logger := log.MustWithGroupAttrs(cmd.Context(), "🔎 Discover")

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

		discover, err := discoverPkg.NewDiscover(
			ctx,
			discoverPkg.Options{
				IgnorePatterns: ignorePatterns,
				IgnoreFsTypes:  ignoreFsTypes,
			},
		)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to start discovery: %w", err))
			return
		}

		err = discover.Run(ctx, host, resourcesPath)
		if err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed run discover: %w", err))
			return
		}
	},
}

func init() {
	AddHostFlags(DiscoverCmd)

	DiscoverCmd.Flags().StringSliceVar(
		&ignorePatterns, "ignore-patterns", defaultIgnorePatterns,
		"file patterns to ignore",
	)
	if err := DiscoverCmd.MarkFlagFilename("ignore-patterns"); err != nil {
		panic(err)
	}

	DiscoverCmd.Flags().StringSliceVar(
		&ignoreFsTypes, "ignore-fstypes", defaultIgnoreFsTypes,
		"filesystem types to ignore",
	)

	DiscoverCmd.Flags().StringVar(
		&resourcesPath, "resources-path", defaultResourcesPath,
		"Path where to write generate resources.",
	)
	if err := DiscoverCmd.MarkFlagRequired("resources-path"); err != nil {
		panic(err)
	}
	if err := DiscoverCmd.MarkFlagDirname("resources-path"); err != nil {
		panic(err)
	}

	RootCmd.AddCommand(DiscoverCmd)
}

func init() {
	resetFlagsFns = append(resetFlagsFns, func() {
		ignorePatterns = defaultIgnorePatterns
		ignoreFsTypes = defaultIgnoreFsTypes
		resourcesPath = defaultResourcesPath
	})
}

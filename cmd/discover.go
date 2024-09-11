package main

import (
	"github.com/spf13/cobra"

	discoverPkg "github.com/fornellas/resonance/discover"
	"github.com/fornellas/resonance/log"
)

var ignorePatterns []string

var ignoreFsTypes []string

var resourcesPath string

var DiscoverCmd = &cobra.Command{
	Use:   "discover [flags]",
	Short: "Discover host state.",
	Long:  "Audit all files and packages at host to identify what requires automation.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		logger := log.MustLogger(ctx)

		logger.Info("🔎 Discovering")

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer host.Close(ctx)
		logger.Info("🖥️ Target", "host", host)

		discover, err := discoverPkg.NewDiscover(
			ctx,
			discoverPkg.Options{
				IgnorePatterns: ignorePatterns,
				IgnoreFsTypes:  ignoreFsTypes,
			},
		)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		err = discover.Run(ctx, host)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
	},
}

func init() {
	AddHostFlags(DiscoverCmd)

	DiscoverCmd.Flags().StringSliceVar(&ignorePatterns, "ignore-patterns", []string{
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
		"/home",  // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s08.html
		"/media", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s11.html
		"/mnt",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s12.html
		"/proc",  // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#procKernelAndProcessInformationVir
		"/root",  // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s14.html
		"/run",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s15.html
		"/snap",
		"/srv", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s17.html
		"/sys", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#sysKernelAndSystemInformation
		"/tmp", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s18.html
		// "/usr/share/doc",
		// "/usr/share/help",
		// "/usr/share/icons",
		// "/usr/share/locale",
		// "/usr/share/man",
		"/usr/src",          // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch04s12.html
		"/var",              // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05.html
		"/backup",           // FIXME
		"/home",             // FIXME
		"/opt",              // FIXME
		"/steam",            // FIXME
		"/usr/lib/firmware", // FIXME
		"/usr/lib/modules",  // FIXME
		"/usr/lib/python*",  // FIXME
		"/windows",          // FIXME
	}, "file patterns to ignore")
	if err := DiscoverCmd.MarkFlagFilename("ignore-patterns"); err != nil {
		panic(err)
	}

	DiscoverCmd.Flags().StringSliceVar(&ignoreFsTypes, "ignore-fstypes", []string{
		"devpts",
		"devtmpfs",
		"efivarfs",
		"proc",
		"sysfs",
		"tmpfs",
	}, "filesystem types to ignore")

	DiscoverCmd.Flags().StringVar(
		&resourcesPath, "resources-path", "",
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

package main

import (
	"github.com/spf13/cobra"

	auditPkg "github.com/fornellas/resonance/audit"
	"github.com/fornellas/resonance/log"
)

var excludePaths []string

var excludeFsTypes []string

var AuditCmd = &cobra.Command{
	Use:   "audit [flags]",
	Short: "Audit host state.",
	Long:  "Audit all files and packages at host to identify what requires automation.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		logger := log.MustLogger(ctx)

		logger.Info("🔎 Auditing")

		host, err := GetHost(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
		defer host.Close(ctx)
		logger.Info("🖥️ Target", "host", host)

		audit, err := auditPkg.NewAudit(
			ctx,
			host,
			excludePaths,
			excludeFsTypes,
		)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}

		err = audit.Run(ctx)
		if err != nil {
			logger.Error(err.Error())
			Exit(1)
		}
	},
}

func init() {
	AddHostFlags(AuditCmd)

	AuditCmd.Flags().StringArrayVarP(&excludePaths, "exclude-paths", "x", []string{
		// TODO "/**/__pycache__/*"
		"/boot/efi",
		"/dev",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s06.html
		"/home",        // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s08.html
		"/media",       // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s11.html
		"/mnt",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s12.html
		"/root",        // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s14.html
		"/run",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s15.html
		"/srv",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s17.html
		"/tmp",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch03s18.html
		"/var/account", // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s04.html
		"/var/cache",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s05.html
		"/var/crash",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s06.html
		"/var/lock",    // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s09.html
		"/var/log",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s10.html
		"/var/mail",    // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s11.html
		"/var/run",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s13.html
		"/var/spool",   // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s14.html
		"/var/tmp",     // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s15.html
		"/var/yp",      // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s16.html
		"/dev",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#devDevicesAndSpecialFiles
		"/proc",        // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#procKernelAndProcessInformationVir
		"/sys",         // https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch06.html#sysKernelAndSystemInformation
		"/snap",
		// FIXME
		"/backup",
		"/backup-windows",
		"/windows",
		"/var",
	}, "paths to exclude")

	AuditCmd.Flags().StringArrayVarP(&excludeFsTypes, "exclude-fstypes", "", []string{
		"devpts",
		"devtmpfs",
		"efivarfs",
		"proc",
		"sysfs",
		"tmpfs",
	}, "filesystem types to exclude")

	RootCmd.AddCommand(AuditCmd)
}

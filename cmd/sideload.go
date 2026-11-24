package cmd

import (
	"fmt"
	"os"

	"github.com/pgaskin/koboutils/v2/kobo"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "sideload",
		Short:   "enable sideloade mode (no account needed)",
		Aliases: []string{"sl"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSideload(kroot)
		},
	})
}

func runSideload(kroot string) error {
	kroot, err := resolveKroot(kroot)
	if err != nil {
		return err
	}

	_, version, _, err := kobo.ParseKoboVersion(kroot)
	if err != nil {
		return err
	}
	if kobo.VersionCompare(version, "4.31.19086") == -1 {
		return fmt.Errorf("sideload mode requires 4.31.19086+ (got %s), try '%s firmware' first", version, os.Args[0])
	}

	return editConf(kroot, func(cfg *ini.File) {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	})
}

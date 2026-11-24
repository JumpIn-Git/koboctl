package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumpin-git/koboctl/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "firmware",
		Short:   "install the latest Kobo firmware",
		Aliases: []string{"fw"},
		RunE: func(cmd *cobra.Command, args []string) error {
			kroot, err := resolveKroot(kroot)
			if err != nil {
				return err
			}
			return installFirmware(ctx, overwrite, kroot)
		},
	})
}

func installFirmware(ctx context.Context, overwrite bool, kroot string) error {
	if !overwrite {
		for _, f := range []string{
			"KoboRoot.tgz",
			"upgrade",
			"manifest.md5sum",
		} { // check if any of these exist to not overwrite
			_, err := os.Lstat(filepath.Join(kroot, ".kobo", f))
			if err == nil {
				return fmt.Errorf("upgrade file %s already exists", f)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("checking for %s: %w", f, err)
			}
		}
	}

	url, err := utils.UpgradeCheck(kroot)
	if err != nil {
		return err
	}
	if url == "" {
		fmt.Println("No update found!")
		return nil
	}

	return saveAndExtract(ctx, url, filepath.Join(kroot, ".kobo"), "firmware", "kobo-update-*.zip", utils.Zip, nil)
}

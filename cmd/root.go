package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jumpin-git/koboctl/utils"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var rootCmd = &cobra.Command{
	Use: "koboctl",
}

var kroot string
var nmconfig string
var overwrite bool
var ctx context.Context

func init() {
	rootCmd.PersistentFlags().StringVar(&kroot, "kobo", "", "path to Kobo root directory (for testing)")
	rootCmd.PersistentFlags().BoolVar(&overwrite, "overwrite", false, "overwrite existing files if applicable")
}

// Execute runs the koboctl CLI and returns an error on failure.
func Execute() error {
	ctx = context.Background()
	return rootCmd.Execute()
}

func editConf(kroot string, edit func(*ini.File)) error {
	confPath := filepath.Join(kroot, ".kobo", "Kobo", "Kobo eReader.conf")
	confBak := filepath.Join(kroot, ".kobo", "Kobo", "Kobo eReader.conf.bak")
	if err := func() error {
		src, err := os.Open(confPath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(confBak)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		return dst.Sync()
	}(); err != nil {
		return fmt.Errorf("creating configuration backup: %w", err)
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{SpaceBeforeInlineComment: true}, confPath)
	if err != nil {
		return fmt.Errorf("loading Kobo eReader.conf: %w", err)
	}
	edit(cfg)

	if err := cfg.SaveTo(confPath); err != nil {
		return fmt.Errorf("saving Kobo eReader.conf: %w", err)
	}
	return nil
}

func resolveKroot(kroot string) (string, error) {
	if kroot == "" {
		var err error
		kroot, err = utils.FindRoot()
		if err != nil {
			return "", err
		}
	}
	return kroot, nil
}

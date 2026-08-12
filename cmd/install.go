package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func init() {
	installCmd := &cobra.Command{
		Use:     "install [plato|koreader|nickelmenu ...]",
		Short:   "install mods onto the Kobo",
		Aliases: []string{"in"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(ctx, kroot, nmconfig, overwrite, args)
		},
	}
	installCmd.Flags().StringVar(&nmconfig, "nmconf", "", "NickelMenu config to optionally copy over.")
	rootCmd.AddCommand(installCmd)
}

func runInstall(ctx context.Context, kroot, nmconfig string, overwrite bool, args []string) error {
	kroot, err := resolveKroot(kroot)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if err := huh.NewMultiSelect[string]().
			Title("What to install? (space to toggle)").
			Options(
				huh.Option[string]{Key: "Plato", Value: "plato"}.Selected(true),
				huh.Option[string]{Key: "KOReader", Value: "koreader"}.Selected(true),
				huh.Option[string]{Key: "NickelMenu", Value: "nickelmenu"}.Selected(true),
			).
			Value(&args).Run(); err != nil {
			return fmt.Errorf("selection: %w", err)
		}
	}

	var plato, koreader, nickelmenu bool
	for _, i := range args {
		switch strings.ToLower(i) {
		case "plato":
			plato = true
		case "koreader":
			koreader = true
		case "nickelmenu":
			nickelmenu = true
		default:
			return fmt.Errorf("unknown option: %s", i)
		}
	}

	if nickelmenu {
		// other mods will write for their configs, make it here
		if err := os.MkdirAll(filepath.Join(kroot, ".adds", "nm"), 0755); err != nil {
			return err
		}
		if nmconfig != "" {
			data, err := os.ReadFile(nmconfig)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(kroot, ".adds", "nm", filepath.Base(nmconfig)), data, 0644); err != nil {
				return err
			}
		}
		if err := installNickelMenu(kroot, overwrite); err != nil {
			return err
		}
	}
	if plato {
		if err := installPlato(ctx, nickelmenu, kroot); err != nil {
			return err
		}
	}
	if koreader {
		if err := installKOReader(ctx, nickelmenu, kroot); err != nil {
			return err
		}
	}
	return nil
}

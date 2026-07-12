package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/jumpin-git/koboctl/utils"
	"github.com/pgaskin/koboutils/v2/kobo"
	"gopkg.in/ini.v1"
)

var subcommands = []string{"install", "firmware", "sideload"}

func main() {
	var kroot string
	var nmconfig string
	var overwrite bool

	installCmd := flag.NewFlagSet("install", flag.ExitOnError)
	installCmd.BoolVar(&overwrite, "overwrite", false, "overwrite KoboRoot.tgz if applicable")
	installCmd.StringVar(&kroot, "kobo", "", "path to Kobo root directory")
	installCmd.StringVar(&nmconfig, "nmconf", "", "NickelMenu config to optionally copy over.")

	firmwareCmd := flag.NewFlagSet("firmware", flag.ExitOnError)
	firmwareCmd.BoolVar(&overwrite, "overwrite", false, "overwrite firmware if applicable")
	firmwareCmd.StringVar(&kroot, "kobo", "", "path to Kobo root directory")

	sideloadCmd := flag.NewFlagSet("sideload", flag.ExitOnError)
	sideloadCmd.StringVar(&kroot, "kobo", "", "path to Kobo root directory")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "expected one of: %s subcommand\n", strings.Join(subcommands, ", "))
		installCmd.Usage()
		firmwareCmd.Usage()
		sideloadCmd.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "install":
		installCmd.Parse(os.Args[2:])
		err = runInstall(ctx, kroot, nmconfig, overwrite, installCmd.Args())
	case "firmware":
		firmwareCmd.Parse(os.Args[2:])
		err = runFirmware(ctx, kroot, overwrite)
	case "sideload":
		sideloadCmd.Parse(os.Args[2:])
		err = runSideload(kroot)
	default:
		err = fmt.Errorf("expected one of: %s subcommand", strings.Join(subcommands, ", "))
		installCmd.Usage()
		firmwareCmd.Usage()
		sideloadCmd.Usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Done.")
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
				huh.Option[string]{Key: "Cadmus", Value: "cadmus"}.Selected(true),
				huh.Option[string]{Key: "KOReader", Value: "koreader"}.Selected(true),
				huh.Option[string]{Key: "NickelMenu", Value: "nickelmenu"}.Selected(true),
			).
			Value(&args).Run(); err != nil {
			return fmt.Errorf("selection: %w", err)
		}
	}

	var cadmus, koreader, nickelmenu bool
	for _, i := range args {
		switch strings.ToLower(i) {
		case "cadmus":
			cadmus = true
		case "koreader":
			koreader = true
		case "nickelmenu":
			nickelmenu = true
		default:
			return fmt.Errorf("unknown option: %s", i)
		}
	}

	if nickelmenu {
		if err := os.MkdirAll(filepath.Join(kroot, ".adds", "nm"), 0755); err != nil {
			return err
		}
		if nmconfig != "" {
			nm, err := os.Open(nmconfig)
			if err != nil {
				return err
			}
			defer nm.Close()
			stat, err := os.Stat(nmconfig)
			if err != nil {
				return err
			}
			if stat.IsDir() {
				return fmt.Errorf("nmconfig must be a file, not a directory")
			}
			f, err := os.Create(filepath.Join(kroot, ".adds", "nm", stat.Name()))
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, nm)
			if err != nil {
				return err
			}
		}
		if err := installNickelMenu(kroot, overwrite); err != nil {
			return err
		}
	}
	if cadmus {
		if err := installCadmus(ctx, nickelmenu, kroot); err != nil {
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

func runFirmware(ctx context.Context, kroot string, overwrite bool) error {
	kroot, err := resolveKroot(kroot)
	if err != nil {
		return err
	}
	return installFirmware(ctx, overwrite, kroot)
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
	if kobo.VersionCompare(version, "4.31.19086") < 0 {
		return fmt.Errorf("sideload mode requires 4.31.19086+ (got %s), try 'koboctl firmware' first", version)
	}

	return editConf(kroot, func(cfg *ini.File) {
		cfg.Section("ApplicationPreferences").Key("SideloadedMode").SetValue("true")
	})
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

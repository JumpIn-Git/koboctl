package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumpin-git/koboctl/utils"
	"gopkg.in/ini.v1"
)

func installMod(repo, assetName string, save func(url, tagName string) error) error {
	release, err := utils.GetRelease(repo)
	if err != nil {
		return err
	}

	var assetUrl string
	if strings.Contains(assetName, "%s") {
		assetName = fmt.Sprintf(assetName, release.TagName)
	}
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			assetUrl = asset.Url
			break
		}
	}
	if assetUrl == "" {
		return fmt.Errorf("no %s asset found", assetName)
	}

	return save(assetUrl, release.TagName)
}

func saveAndExtract(ctx context.Context, url, dst, modName, pattern string, format utils.ExtractFormat) error {
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return err
	}
	defer tmp.Close()
	defer os.Remove(tmp.Name())

	if err := utils.Download(url, modName, tmp); err != nil {
		return err
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking %s: %w", modName, err)
	}
	stat, err := tmp.Stat()
	if err != nil {
		return fmt.Errorf("statting %s: %w", modName, err)
	}
	if err := utils.Extract(ctx, format, tmp, stat, dst, modName); err != nil {
		return fmt.Errorf("extracting %s: %w", modName, err)
	}
	return nil
}

func installCadmus(ctx context.Context, nickelmenu bool, kroot string) error {
	if err := installMod("ogkevin/cadmus", "cadmus-kobo.tar.gz", func(url, tagName string) error {
		return saveAndExtract(ctx, url, filepath.Join(kroot, ".adds", "cadmus"), "Cadmus", fmt.Sprintf("cadmus-kobo-%s-*.tar.gz", tagName), utils.Tgz)
	}); err != nil {
		return err
	}
	if nickelmenu {
		if err := utils.OptWrite(filepath.Join(kroot, ".adds", "nm", "cadmus"),
			`menu_item : main : Cadmus : cmd_spawn : quiet : exec /mnt/onboard/.adds/cadmus/cadmus.sh`); err != nil {
			return err
		}
	}
	return nil
}

func installKOReader(ctx context.Context, nickelmenu bool, kroot string) error {
	if err := installMod("koreader/koreader", "koreader-kobo-%s.zip",
		func(url, tagName string) error {
			return saveAndExtract(ctx, url, filepath.Join(kroot, ".adds", "koreader"), "KOReader", fmt.Sprintf("koreader-kobo-%s-*.zip", tagName), utils.Zip)
		}); err != nil {
		return err
	}
	if nickelmenu {
		if err := utils.OptWrite(filepath.Join(kroot, ".adds", "nm", "koreader"),
			`menu_item : main : KOReader : cmd_spawn : quiet : exec /mnt/onboard/.adds/koreader/koreader.sh`); err != nil {
			return err
		}
	}
	return nil
}

func installNickelMenu(kroot string, overwrite bool) error {
	if err := installMod("pgaskin/nickelmenu", "KoboRoot.tgz", func(url, tagName string) error {
		path := filepath.Join(kroot, ".kobo", "KoboRoot.tgz")
		var f *os.File
		var err error
		if overwrite {
			f, err = os.Create(path)
			if err != nil {
				return fmt.Errorf("creating %s: %w", path, err)
			}
		} else {
			f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
			if os.IsExist(err) { // file already exists (no truncate)
				return fmt.Errorf("KoboRoot.tgz already exists and overwrite is false")
			} else if err != nil {
				return fmt.Errorf("creating %s: %w", path, err)
			}
		}
		defer f.Close()
		return utils.Download(url, "NickelMenu", f)
	}); err != nil {
		return err
	}

	return editConf(kroot, func(cfg *ini.File) {
		cfg.Section("FeatureSettings").Key("ExcludeSyncFolders").SetValue(`(\\.(?!kobo|adobe).+|([^.][^/]*/)+\\..+)`)
	})
}

package cmd

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

func saveAndExtract(ctx context.Context, url, dst, modName, pattern string, format utils.ExtractFormat, prefixes utils.Prefixes) error {
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

	if prefixes != nil {
		err = utils.ExtractPrefix(ctx, format, tmp, stat, prefixes, modName)
	} else {
		err = utils.Extract(ctx, format, tmp, stat, dst, modName)
	}
	if err != nil {
		return fmt.Errorf("extracting %s: %w", modName, err)
	}
	return nil
}

type modDef struct {
	repo      string
	assetName string // can optionally contain %s (tag name)
	dir       string // .adds/{dir}
	pattern   string // tmp pattern, %s is replaced with tag name
	nmName    string // .adds/nm/{nmName}
	nmEntry   string
	format    utils.ExtractFormat
	prefixes  *[]string // optional, only extract paths with these prefixes to .adds/{dir}
}

func installModDef(ctx context.Context, m modDef, kroot string, nickelmenu bool) error {
	if err := installMod(m.repo, m.assetName, func(url, tagName string) error {
		var prefixes utils.Prefixes
		if m.prefixes != nil {
			prefixes = make(utils.Prefixes, len(*m.prefixes))
			for _, p := range *m.prefixes {
				prefixes[p] = filepath.Join(kroot, ".adds", m.dir)
			}
		}
		return saveAndExtract(ctx, url, filepath.Join(kroot, ".adds", m.dir), m.dir, fmt.Sprintf(m.pattern, tagName), m.format, prefixes)
	}); err != nil {
		return err
	}
	if nickelmenu {
		if err := utils.OptWrite(filepath.Join(kroot, ".adds", "nm", m.nmName), m.nmEntry); err != nil {
			return err
		}
	}
	return nil
}

func installPlato(ctx context.Context, nickelmenu bool, kroot string) error {
	return installModDef(ctx, modDef{
		repo: "baskerville/plato", assetName: "plato-%s.zip",
		dir: "plato", pattern: "plato-%s-*.zip",
		nmName:  "plato",
		nmEntry: "menu_item : main : Plato : cmd_spawn : quiet : exec /mnt/onboard/.adds/plato/plato.sh",
		format:  utils.Zip,
	}, kroot, nickelmenu)
}

func installCadmus(ctx context.Context, nickelmenu bool, kroot string) error {
	return installModDef(ctx, modDef{
		repo: "ogkevin/cadmus", assetName: "cadmus-kobo.tar.gz",
		dir: "cadmus", pattern: "cadmus-%s-*.zip",
		nmName:  "cadmus",
		nmEntry: "menu_item  :main  :Cadmus      :cmd_spawn  :quiet:/mnt/onboard/.adds/cadmus/cadmus.sh",
		format:  utils.Tgz,
	}, kroot, nickelmenu)
}

// TODO select correct zip (v5 for newer ereaders)
func installKOReader(ctx context.Context, nickelmenu bool, kroot string) error {
	return installModDef(ctx, modDef{
		repo: "koreader/koreader", assetName: "koreader-kobo-%s.zip",
		dir: "koreader", pattern: "koreader-kobo-%s-*.zip",
		nmName:   "koreader",
		nmEntry:  "menu_item : main : KOReader : cmd_spawn : quiet : exec /mnt/onboard/.adds/koreader/koreader.sh",
		format:   utils.Zip,
		prefixes: &[]string{"koreader/"},
	}, kroot, nickelmenu)
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

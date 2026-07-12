package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"charm.land/huh/v2"
	"github.com/pgaskin/koboutils/v2/kobo"
	"github.com/schollz/progressbar/v3"
)

var client = http.Client{Timeout: 1 * time.Minute}

func Get(url, name string) (*http.Response, error) {
	if name == "" {
		name = url
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading %s, %w", name, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s, status code: %d", name, resp.StatusCode)
	}
	return resp, nil
}

func Download(url, name string, dst *os.File) error {
	resp, err := Get(url, name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(resp.ContentLength, fmt.Sprintf("downloading %s", name))
	if _, err = io.Copy(io.MultiWriter(dst, bar), resp.Body); err != nil {
		return fmt.Errorf("saving %s: %w", name, err)
	}
	return nil
}

type Asset struct {
	Name string `json:"name"`
	Url  string `json:"browser_download_url"`
}
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func GetRelease(repo string) (*Release, error) {
	resp, err := Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo), repo)
	if err != nil {
		return nil, fmt.Errorf("%s: getting release: %w", repo, err)
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("%s: decoding release: %w", repo, err)
	}

	return &release, nil
}

func UpgradeCheck(kroot string) (string, error) {
	serial, version, id, err := kobo.ParseKoboVersion(kroot)
	if err != nil {
		return "", err
	}
	upgrade, err := kobo.CheckUpgrade(id, "kobo", version, serial)
	if err != nil {
		return "", err
	}
	if !upgrade.UpgradeType.IsUpdate() {
		return "", nil
	}

	proceed := true
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Update is %s, upgrade?", upgrade.UpgradeType)).
		Description(fmt.Sprintf("Upgrade from %s to %s", version, upgrade.ParseVersion())).
		Value(&proceed).Run(); err != nil {
		return "", fmt.Errorf("selection: %w", err)
	}
	if !proceed {
		return "", nil
	}
	return upgrade.UpgradeURL, nil
}

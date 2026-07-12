package utils

import (
	"errors"
	"fmt"

	"charm.land/huh/v2"
	"github.com/pgaskin/koboutils/v2/kobo"
)

func FindRoot() (root string, err error) {
	kobos, err := kobo.Find()
	if err != nil {
		return "", err
	}
	if len(kobos) == 0 {
		return "", errors.New("no Kobo's found, are any mounted?")
	} else if len(kobos) == 1 {
		return kobos[0], nil
	} else {
		if err := huh.NewSelect[string]().
			Title("Select Kobo").
			Options(huh.NewOptions(kobos...)...).
			Value(&root).Run(); err != nil {
			return "", fmt.Errorf("selection: %w", err)
		}
	}
	return root, nil
}

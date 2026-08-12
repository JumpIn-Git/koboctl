package main

import (
	"fmt"
	"os"

	"github.com/jumpin-git/koboctl/cli/koboctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Done.")
}

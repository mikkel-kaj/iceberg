package main

import (
	"fmt"
	"os"

	"github.com/mikkel-kaj/iceberg/internal/cli"
)

const version = "v0.1.0-dev"

func main() {
	root := cli.NewRootCmd(version)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

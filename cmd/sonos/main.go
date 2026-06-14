package main

import (
	"os"

	"github.com/shlomiuziel/sonosh/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

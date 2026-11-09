package main

import (
	"fmt"
	"os"

	"github.com/zpol/doki/internal/version"
)

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "doki: %v\n", err)
		os.Exit(1)
	}
}

// ldflags-friendly aliases referenced by Makefile / GoReleaser later.
var (
	_ = version.Version
)

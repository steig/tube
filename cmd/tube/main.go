package main

import (
	"os"

	"github.com/steig/tube/internal/cli"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func main() {
	cmd := cli.NewRootCmd(Version, Commit, Date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

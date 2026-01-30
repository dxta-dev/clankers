package main

import (
	"os"

	"github.com/dxta-dev/clankers-daemon/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

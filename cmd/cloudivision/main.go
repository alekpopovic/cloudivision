package main

import (
	"os"

	"github.com/cloudivision/cloudivision/internal/cli"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Commit = commit
	app.BuildDate = buildDate
	os.Exit(app.Run(os.Args[1:]))
}

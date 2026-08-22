package main

import (
	"os"

	"github.com/cloudivision/cloudivision/internal/cli"
)

var version = "dev"

func main() {
	app := cli.NewApp()
	app.Version = version
	os.Exit(app.Run(os.Args[1:]))
}

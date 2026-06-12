package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	buildRunName := os.Getenv("BUILD_RUN_NAME")
	if buildRunName == "" {
		buildRunName = "unknown"
	}

	logger.Info("starting cloudivision build runner", "buildRunName", buildRunName)
	logger.Info("runner execution is not implemented yet")
}

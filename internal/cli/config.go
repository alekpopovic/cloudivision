package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIURL    string `yaml:"apiURL" json:"apiURL"`
	Token     string `yaml:"token,omitempty" json:"-"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

type globalOptions struct {
	APIURL, Token, Namespace, Output, ConfigPath string
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read CLI config: %w", err)
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse CLI config: %w", err)
	}
	return config, nil
}

func saveConfig(path string, config Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode CLI config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create CLI config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write CLI config: %w", err)
	}
	return nil
}

func resolveConfig(options globalOptions, home string) (Config, string, error) {
	path := options.ConfigPath
	if path == "" {
		path = filepath.Join(home, ".cloudivision", "config.yaml")
	}
	config, err := loadConfig(path)
	if err != nil {
		return Config{}, path, err
	}
	config.APIURL = first(options.APIURL, os.Getenv("CLOU_DIVISION_API_URL"), config.APIURL, "http://localhost:8080")
	config.Token = first(options.Token, os.Getenv("CLOU_DIVISION_TOKEN"), config.Token)
	config.Namespace = first(options.Namespace, os.Getenv("CLOU_DIVISION_NAMESPACE"), config.Namespace, "default")
	return config, path, nil
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func extractGlobals(args []string) (globalOptions, []string, error) {
	var options globalOptions
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, hasValue := strings.Cut(arg, "=")
		var target *string
		switch name {
		case "--api-url":
			target = &options.APIURL
		case "--token":
			target = &options.Token
		case "--namespace", "-n":
			target = &options.Namespace
		case "--output", "-o":
			target = &options.Output
		case "--config":
			target = &options.ConfigPath
		default:
			remaining = append(remaining, arg)
			continue
		}
		if !hasValue {
			i++
			if i >= len(args) {
				return options, nil, fmt.Errorf("%s requires a value", name)
			}
			value = args[i]
		}
		*target = value
	}
	if options.Output == "" {
		options.Output = "table"
	}
	if options.Output != "table" && options.Output != "json" {
		return options, nil, fmt.Errorf("output must be table or json")
	}
	return options, remaining, nil
}

package config

import (
	_ "embed"
)

// Embedded configuration file (embedded at build time)
// This allows the binary to work anywhere without needing external config files
//
// The install.sh script copies config.yaml to this location before building,
// then cleans it up after the build completes.
//
// If embedded.yaml doesn't exist or is empty, the program will fall back to
// loading config from ~/.config/minitool/config.yaml or ./config.yaml
//
//go:embed embedded.yaml
var embeddedConfigYAML string

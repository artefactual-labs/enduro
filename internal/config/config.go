package config

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/artefactual-labs/enduro/internal/a3m"
	"github.com/artefactual-labs/enduro/internal/aipstore"
	"github.com/artefactual-labs/enduro/internal/api"
	"github.com/artefactual-labs/enduro/internal/db"
	"github.com/artefactual-labs/enduro/internal/search"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/version"
	"github.com/artefactual-labs/enduro/internal/watcher"
)

type Configuration struct {
	Debug       bool
	DebugListen string
	API         api.Config
	Database    db.Config
	Search      search.Config
	Temporal    temporal.Config
	Watcher     watcher.Config
	Validation  validation.Config

	AIPStore aipstore.Config
	A3m      a3m.Config
}

func (c Configuration) Validate() error {
	return nil
}

func Read(config *Configuration, configFile string) (found bool, configFileUsed string, err error) {
	v := viper.New()

	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/")
	v.AddConfigPath("/etc")
	v.SetConfigName("enduro")
	v.SetDefault("debugListen", "127.0.0.1:9001")
	v.SetDefault("api.listen", "127.0.0.1:9000")
	v.Set("api.appVersion", version.Version)

	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	err = v.ReadInConfig()
	_, ok := err.(viper.ConfigFileNotFoundError)
	if !ok {
		found = true
	}
	if found && err != nil {
		return found, configFileUsed, fmt.Errorf("failed to read configuration file: %w", err)
	}

	err = v.Unmarshal(config)
	if err != nil {
		return found, configFileUsed, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return found, configFileUsed, fmt.Errorf("failed to validate the provided config: %w", err)
	}

	configFileUsed = v.ConfigFileUsed()

	return found, configFileUsed, nil
}

package main

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	configFilename = ".withenv.yml"
)

func findConfig(curdir string) (string, error) {
	root, err := filepath.Abs("/")
	if err != nil {
		return "", err
	}

	curdir, err = filepath.Abs(curdir)
	if err != nil {
		return "", nil
	}

	log.Debugf("abs path: %q", curdir)

	config := filepath.Join(curdir, configFilename)
	log.Debugf("in findConfig: %q", config)
	if _, err := os.Stat(config); os.IsNotExist(err) {
		if curdir == root {
			return "", err
		} else {
			return findConfig(filepath.Dir(curdir))
		}
	}

	return filepath.Abs(config)
}

func InitConfig(path string) {
	log.Debug("initializing config")
	viper.SetConfigName(".withenv")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()

	if err != nil {
		log.Debugf("Error reading config: %q", err)
		return
	}

	if viper.GetBool("find_config") {
		config, err := findConfig(".")
		if err != nil {
			log.Fatalf("Error finding parent config: %q", err)
		}

		viper.AddConfigPath(config)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatalf("Error reading parent config: %q", err)
		}
		log.Debugf("Found and loaded config: %q", config)

		viper.Set("config_alias", config)
	}
}

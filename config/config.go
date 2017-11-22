package config

import (
	"fmt"
	"os"
)

type Config struct {
	Data map[string]string
}

func (c *Config) GetConfig(name string) string {
	if v, ok := c.Data[name]; ok {
		return v
	}

	return os.Getenv(name)
}

func (c *Config) Set(k, v string) {
	c.Data[k] = v
}

func (c *Config) Get(k string) (string, bool) {
	v, ok := c.Data[k]
	return v, ok
}

func (c *Config) ToEnv() []string {
	envlist := []string{}
	for key, val := range c.Data {
		if key == "" {
			continue
		}

		if val == "" && os.Getenv(key) != "" {
			val = os.Getenv(key)
		}

		envlist = append(envlist, fmt.Sprintf("%s=%s", key, val))
	}

	return envlist
}

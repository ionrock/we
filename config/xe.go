package config

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
)

type Service struct {
	Name string `json:"name"`
	Cmd  string `json:"cmd"`
	Dir  string `json:"dir"`
}

type Task struct {
	Name string `json:"name"`
	Cmd  string `json:"cmd"`
	Dir  string `json:"dir"`
}

type XeConfig struct {
	Service   *Service          `json:"service"`
	Env       map[string]string `json:"env"`
	EnvScript string            `json:"envscript"`
	Task      *Task             `json:"task"`
}

func NewXeConfig(path string) ([]XeConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := make([]XeConfig, 0)

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

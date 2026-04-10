package sandbox

import (
	"os"

	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// SandboxYAMLConfig represents the sandbox section in .withenv.yml.
type SandboxYAMLConfig struct {
	Deny        []string `yaml:"deny"`
	Allow       []string `yaml:"allow"`
	DenyNetwork bool     `yaml:"deny-network"`
	SkipPrefix  []string `yaml:"skip-prefix"`
}

// ParseSandboxConfig reads a .withenv.yml and extracts the sandbox configuration.
// Returns nil (no error) if no sandbox section exists.
func ParseSandboxConfig(aliasPath string) (*SandboxYAMLConfig, error) {
	b, err := os.ReadFile(aliasPath)
	if err != nil {
		return nil, err
	}

	// Use flexible unmarshaling since the sandbox entry has a nested map value
	// while other entries are simple string key-value pairs.
	var entries []map[string]interface{}
	if err := yaml.Unmarshal(b, &entries); err != nil {
		return nil, err
	}

	for _, entry := range entries {
		raw, ok := entry["sandbox"]
		if !ok {
			continue
		}

		// Re-marshal and unmarshal the sandbox value into our typed struct
		sandboxBytes, err := yaml.Marshal(raw)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal sandbox config")
			return nil, err
		}

		var cfg SandboxYAMLConfig
		if err := yaml.Unmarshal(sandboxBytes, &cfg); err != nil {
			return nil, err
		}

		return &cfg, nil
	}

	return nil, nil
}

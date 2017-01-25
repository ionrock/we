package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
)

const (
	JSON_FILETYPE = iota
	YAML_FILETYPE
)

type Flattener struct {
	path string
}

func (f Flattener) filetype() int {

	json := []string{"json"}
	yaml := []string{"yaml", "yml"}

	// Use the file extension first
	for _, sfx := range json {
		if strings.HasSuffix(f.path, sfx) {
			log.Debug("Using JSON parser")
			return JSON_FILETYPE
		}
	}

	for _, sfx := range yaml {
		if strings.HasSuffix(f.path, sfx) {
			log.Debug("Using YAML parser")
			return YAML_FILETYPE
		}
	}

	// peek at the first bytes for a {
	fh, err := os.Open(f.path)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(fh)
	start, err := reader.Peek(3)
	if err != nil {
		log.Fatal(err)
	}

	fh.Close()

	// if the { exists, we'll try json
	if bytes.HasPrefix(bytes.TrimLeft(start, " "), []byte("{")) {
		log.Debug("Using JSON parser")
		return JSON_FILETYPE
	}

	// if the [ exists, we'll try json
	if bytes.HasPrefix(bytes.TrimLeft(start, " "), []byte("[")) {
		log.Debug("Using JSON parser")
		return JSON_FILETYPE
	}

	log.Debug("Using YAML parser")
	return YAML_FILETYPE
}

func (f Flattener) unmarshal(b []byte, v interface{}) error {
	if f.filetype() == JSON_FILETYPE {
		return json.Unmarshal(b, v)
	}

	return yaml.Unmarshal(b, v)
}

func (f Flattener) loadMap(b []byte) ([]map[string]interface{}, error) {
	// try loading into a list of maps first
	m := make(map[string]interface{})

	err := f.unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return []map[string]interface{}{m}, nil
}

func (f Flattener) loadList(b []byte) ([]map[string]interface{}, error) {
	// try loading into a list of maps first
	m := make([]map[string]interface{}, 5)

	err := f.unmarshal(b, &m)
	if err != nil {
		log.Debug(err)
		return nil, err
	}

	return m, nil
}

func (f Flattener) load(path string) ([]map[string]interface{}, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	m, err := f.loadMap(b)
	if err == nil {
		return m, err
	}

	m, err = f.loadList(b)
	if err != nil {
		log.Fatal(err)
	}
	return m, nil
}

func flatKey(prefix []string, key string) string {
	return strings.Join(append(prefix, key), "_")
}

func compileValue(value string, path string) string {
	log.Debug("%#vs", value)
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") {
		parts := SplitCommand(os.ExpandEnv(strings.Trim(value, "`")))
		if parts != nil {
			cmd := exec.Command(parts[0], parts[1:]...)
			dirname, _ := filepath.Abs(path)
			cmd.Dir = filepath.Dir(dirname)
			out, err := cmd.Output()
			if err != nil {
				log.Fatalf("Error running command: '%s' %s", parts, err)
			}
			return string(bytes.TrimSpace(out))
		}
	}
	return value
}

func applyString(env map[string]string, prefix []string, key string, value string) {
	key = flatKey(prefix, key)
	env[key] = value
	os.Setenv(key, env[key])
}

// Take the default result from unmarshalling the file into an
// interface and return a map[string]string
func (f Flattener) flattenEnv(env []map[string]interface{}) map[string]string {
	fenv := make(map[string]string)
	for _, ev := range env {
		f.flattenMap(fenv, ev, []string{})
	}
	return fenv
}

func (f Flattener) flattenMap(env map[string]string, ev map[string]interface{}, prefix []string) map[string]string {
	for k, v := range ev {
		switch v.(type) {
		case string:
			applyString(env, prefix, k, compileValue(v.(string), f.path))

		case map[string]interface{}:
			f.flattenMap(env, v.(map[string]interface{}), append(prefix, k))

		case []map[string]interface{}:
			for _, submap := range v.([]map[string]interface{}) {
				f.flattenMap(env, submap, append(prefix, k))
			}

		case []interface{}: // handling a nested list of k/v maps
			for _, submap := range v.([]interface{}) {
				f.flattenMap(env, submap.(map[string]interface{}), append(prefix, k))
			}

		default:
			key := flatKey(prefix, k)
			env[key] = fmt.Sprintf("%#v", v)
			log.Debugf("NOT HANDLED: %s = %s", key, env[key])
			os.Setenv(key, env[key])
		}
	}

	return env
}

// Load a YAML/JSON file and flatten it to a list of KEY=VALUE pairs
// suitable for the environment.
//
// This will update this processes environment by calling os.Setenv
// for each entry.
func (f Flattener) Flatten() (map[string]string, error) {
	env, err := f.load(f.path)
	if err != nil {
		log.Error("Error loading JSON/YAML")
		return nil, err
	}

	return f.flattenEnv(env), nil
}

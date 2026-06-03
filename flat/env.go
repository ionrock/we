package flat

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ionrock/we/process"
	"github.com/ionrock/we/sopsutil"

	"github.com/rs/zerolog/log"
)

type FlatEnv struct {
	Path      string
	Env       map[string]string
	Order     []string
	Encrypted bool
}

func (env *FlatEnv) key(parts []string) (string, error) {
	if len(parts) == 0 {
		return "", errors.New("no prefix for key")
	}
	return strings.Join(parts, "_"), nil
}

func (env *FlatEnv) addString(prefix []string, value string) error {
	key, err := env.key(prefix)
	if err != nil || key == "" {
		return err
	}

	value, err = process.CompileValue(value, env.Path)
	if err != nil {
		return err
	}

	if _, ok := env.Env[key]; ok {
		value = fmt.Sprintf("%s %s", env.Env[key], value)
	} else {
		env.Order = append(env.Order, key)
	}
	env.Env[key] = value
	ApplyString(env.Env, key, value)
	return nil
}

func (env *FlatEnv) addBool(prefix []string, value bool) error {
	return env.addString(prefix, fmt.Sprintf("%t", value))
}

func (env *FlatEnv) addFloat64(prefix []string, value float64) error {
	return env.addString(prefix, strconv.FormatFloat(value, 'f', -1, 64))
}

func (env *FlatEnv) Load(v interface{}, prefix []string) error {
	iterMap := func(x map[string]interface{}, prefix []string) {
		for k, v := range x {
			env.Load(v, append(prefix, k))
		}
	}

	iterSlice := func(x []interface{}, prefix []string) {
		for _, v := range x {
			env.Load(v, prefix)
		}
	}

	switch vv := v.(type) {
	case string:
		if err := env.addString(prefix, v.(string)); err != nil {
			return err
		}
	case bool:
		if err := env.addBool(prefix, v.(bool)); err != nil {
			return err
		}
	case float64:
		if err := env.addFloat64(prefix, v.(float64)); err != nil {
			return err
		}
	case map[string]interface{}:
		iterMap(vv, prefix)
	case []interface{}:
		iterSlice(vv, prefix)
	default:
		return errors.New(fmt.Sprintf("Unknown type: %#v", vv))
	}

	return nil
}

func (env *FlatEnv) Decode() (interface{}, error) {
	b, err := os.ReadFile(env.Path)
	if err != nil {
		return nil, err
	}

	if sopsutil.IsSOPSYAML(b) {
		b, err = sopsutil.DecryptYAML(env.Path, b)
		if err != nil {
			return nil, err
		}
		env.Encrypted = true
	}

	var f interface{}

	err = yaml.Unmarshal(b, &f)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func NewFlatEnvWithOrderAndMetadata(path string) (map[string]string, []string, bool, error) {
	env := &FlatEnv{
		Path: path,
		Env:  make(map[string]string),
	}

	f, err := env.Decode()
	if err != nil {
		return nil, nil, false, err
	}

	err = env.Load(f, []string{})
	if err != nil {
		return nil, nil, false, err
	}
	return env.Env, env.Order, env.Encrypted, nil
}

func NewFlatEnvWithOrder(path string) (map[string]string, []string, error) {
	env, order, _, err := NewFlatEnvWithOrderAndMetadata(path)
	return env, order, err
}

func NewFlatEnv(path string) (map[string]string, error) {
	env, _, err := NewFlatEnvWithOrder(path)
	return env, err
}

func ApplyString(env map[string]string, key string, value string) {
	env[key] = value
	if strings.HasPrefix(env[key], "secret:") || strings.HasPrefix(env[key], "secret?:") {
		log.Debug().Msgf("setting %s to <redacted>", key)
	} else {
		log.Debug().Msgf("setting %s to %s", key, env[key])
	}
}

package envs

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const RedactedValue = "<redacted>"

type Value struct {
	Value    string
	Raw      string
	Secret   bool
	Resolved bool
	Source   string
	Provider string
}

type Env map[string]Value

func NewLiteralValue(value, source string) Value {
	return Value{Value: value, Raw: value, Resolved: true, Source: source, Provider: "literal"}
}

func EnvFromStringMap(values map[string]string, source string) Env {
	env := make(Env, len(values))
	for k, v := range values {
		value := NewLiteralValue(v, source)
		value.Secret = LooksSensitiveName(k)
		env[k] = value
	}
	return env
}

func (env Env) StringMap() map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = v.Value
	}
	return out
}

func (env Env) Merge(src Env) Env {
	if env == nil {
		env = make(Env)
	}
	for k, v := range src {
		env[k] = v
	}
	return env
}

func (env Env) Environ() []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		if k != "" {
			out = append(out, fmt.Sprintf("%s=%s", k, v.Value))
		}
	}
	return out
}

func (env Env) Lines(showSecrets bool) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		v := env[k]
		value := v.DisplayForKey(k, showSecrets)
		lines = append(lines, fmt.Sprintf("%s=%s", k, value))
	}
	return lines
}

func LooksSensitiveName(key string) bool {
	upper := strings.ToUpper(key)
	for _, marker := range []string{"SECRET", "PASSWORD", "TOKEN", "_KEY", "API_KEY", "ACCESS_KEY", "PRIVATE_KEY"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func (v Value) Display(showSecrets bool) string {
	if v.Secret && !showSecrets {
		return RedactedValue
	}
	return v.Value
}

func (v Value) DisplayForKey(key string, showSecrets bool) string {
	if (v.Secret || LooksSensitiveName(key)) && !showSecrets {
		return RedactedValue
	}
	return v.Value
}

var expansionPattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

func expandValue(value string, env Env) (string, bool) {
	usedSecret := false
	expanded := expansionPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := expansionPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		key := parts[1]
		if val, ok := env[key]; ok {
			if val.Secret {
				usedSecret = true
			}
			return val.Value
		}
		return os.Getenv(key)
	})
	return expanded, usedSecret
}

func isSecretRef(value string) (ref string, optional bool, ok bool) {
	switch {
	case strings.HasPrefix(value, "secret?:"):
		return strings.TrimPrefix(value, "secret?:"), true, true
	case strings.HasPrefix(value, "secret:"):
		return strings.TrimPrefix(value, "secret:"), false, true
	default:
		return "", false, false
	}
}

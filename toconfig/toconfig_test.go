package toconfig

import (
	"os"
	"testing"
)

func TestEnvMapParse(t *testing.T) {
	k := "TEST_ENV_MAP_PARSE"
	v := "it worked"
	os.Setenv(k, v)
	env := envMap()

	val, ok := env[k]
	if !ok {
		t.Fatalf("failed to parse envmap: %q no in %#v", k, env)
	}

	if val != v {
		t.Errorf("invalid value in envmap: %q != %q", val, v)
	}
}

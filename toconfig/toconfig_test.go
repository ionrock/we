package toconfig

import (
	"bytes"
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

func TestApplyConfig(t *testing.T) {
	tmpl := "testdata/my.cfg.tmpl"
	os.Setenv("LISTEN", "10.0.0.1:8900")
	os.Setenv("CLUSTER_HOSTS", "10.0.0.2 10.0.0.3 10.0.0.4")

	var b bytes.Buffer

	err := ApplyConfig(tmpl, &b)
	if err != nil {
		t.Fatalf("failed to write template: %q", err)
	}

	contents := b.String()
	expected := `[service]
listen = 10.0.0.1:8900
workers =
   - 10.0.0.2
   - 10.0.0.3
   - 10.0.0.4
`
	if contents != expected {
		t.Errorf("wrong content: \n%q\n !=\n%q", contents, expected)
	}
}

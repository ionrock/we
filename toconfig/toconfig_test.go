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

type tmplPathTest struct {
	path   string
	tmpl   string
	target string
}

func TestParseTemplatePath(t *testing.T) {
	tests := []tmplPathTest{
		{"foo.cfg.tmpl", "foo.cfg.tmpl", "foo.cfg"},
		{"foo.cfg.tmpl:foo.cfg", "foo.cfg.tmpl", "foo.cfg"},
		{"/path/to/tmpls/foo.tmpl:/etc/my.cfg", "/path/to/tmpls/foo.tmpl", "/etc/my.cfg"},
	}

	for _, test := range tests {
		tmpl, target, err := parseTemplatePath(test.path)
		if err != nil {
			t.Fatalf("failure parsing valid tmpl: %q %q", test.path, err)
		}

		if test.tmpl != tmpl {
			t.Errorf("failure finding tmpl: %q != %q", tmpl, test.tmpl)
		}

		if test.target != target {
			t.Errorf("failure finding target: %q != %q", target, test.target)
		}
	}
}

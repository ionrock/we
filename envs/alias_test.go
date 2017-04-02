package envs

import (
	"strings"
	"testing"
)

type testAliasEntry struct {
	k    string
	v    string
	flag string
	arg  string
}

var testAliasEntries = []testAliasEntry{
	{"file", "foo/bar.yml", "--env", "testdata/foo/bar.yml"},
	{"env", "foo/bar.yml", "--env", "testdata/foo/bar.yml"},
	{"envvar", "FOO=bar", "--envvar", "FOO=bar"},
	{"script", "cat foo.yml", "--script", "cat foo.yml"},
	{"dir", "foo", "--dir", "testdata/foo"},
}

func TestLoadEntry(t *testing.T) {
	a := Alias{path: "testdata/myalias.yml"}

	for _, test := range testAliasEntries {
		flag, arg := a.loadEntry(test.k, test.v)
		if test.flag != flag {
			t.Errorf("Error with expected flag: %q no %q", flag, test.flag)
		}
		if !strings.HasSuffix(arg, test.arg) {
			t.Errorf("Error with expected arg: %q no %q", arg, test.arg)
		}
	}
}

package envs

import (
	"testing"
)

func compareArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestFindCmds(t *testing.T) {
	cmd := "foo bar -baz | jq . foo | cat | tr"

	cmds := findCmds(cmd)

	expected := [][]string{
		{"foo", "bar", "-baz"},
		{"jq", ".", "foo"},
		{"cat"},
		{"tr"},
	}

	for i := range cmds {
		if compareArgs(cmds[i].Args, expected[i]) != true {
			t.Errorf("wrong cmd: %q != %q", cmds[i].Args, expected[i])
		}
	}
}

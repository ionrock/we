package process

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ionrock/we/utils"
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

func TestCompiledValueWithCmdAndExpansion(t *testing.T) {
	key, _ := utils.GenRandEnvVar()
	yml := "maps.yml"
	os.Setenv(key, yml)

	// This ensures we are expanding the env before executing the script.
	cmd := fmt.Sprintf("`cat $%s`", key)

	// We gave a relative path, so this ensures the path is relative
	// to the file being processed.
	path := "testdata/maps.yml"

	// this should return an error!
	result, err := CompileValue(cmd, path)
	if err != nil {
		t.Fatalf("Error running command: '%s' %s", cmd, err)
	}

	expected, err := ioutil.ReadFile("testdata/maps.yml")
	if err != nil {
		t.Fatalf("test data missing: %q", err)
	}

	if result != string(bytes.TrimSpace(expected)) {
		t.Errorf("compileValue failed: expected %q, got %q", expected, result)
	}
}

func TestCompileValueNoop(t *testing.T) {
	value, err := CompileValue("foo", "")
	if err != nil {
		t.Fatalf("failed compiling value: %q", err)
	}

	if value != "foo" {
		t.Errorf("compileValue didn't recognize there was no script")
	}
}

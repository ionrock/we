package we

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

var letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func genRandEnvVar() (string, string) {
	return randString(7), randString(5)
}

func TestLoadEnvFiles(t *testing.T) {
	paths := []string{
		"testdata/list_of_maps.json",
		"testdata/maps.json",
		"testdata/list_of_maps.yml",
		"testdata/maps.yml",
	}

	for _, path := range paths {
		env, err := NewFlatEnv(path)
		if err != nil {
			t.Fatal("failed to load %q: %q", path, err)
		}

		v, ok := env["FOO"]

		if !ok {
			t.Errorf("key missing FOO in %q", env)
		}

		if v != "bar" {
			t.Errorf("value is wrong: %q != bar; %q", env["FOO"])
		}
	}
}

func TestFlattenKey(t *testing.T) {
	fe := FlatEnv{}
	prefix := []string{"FOO", "BAR", "BAZ"}
	result, err := fe.key(prefix)
	if err != nil {
		t.Fatalf("error checking prefix size: %q", err)
	}

	expected := "FOO_BAR_BAZ"
	if result != expected {
		t.Errorf("flatKey failed: expected %q, got %q", expected, result)
	}
}

func TestCompiledValueWithCmdAndExpansion(t *testing.T) {
	key, _ := genRandEnvVar()
	yml := "maps.yml"
	os.Setenv(key, yml)

	// This ensures we are expanding the env before executing the script.
	cmd := fmt.Sprintf("`cat $%s`", key)

	// We gave a relative path, so this ensures the path is relative
	// to the file being processed.
	path := "testdata/maps.yml"

	// this should return an error!
	result := CompileValue(cmd, path)

	expected, err := ioutil.ReadFile("testdata/maps.yml")
	if err != nil {
		t.Fatalf("test data missing: %q", err)
	}

	if result != string(bytes.TrimSpace(expected)) {
		t.Errorf("compileValue failed: expected %q, got %q", expected, result)
	}
}

func TestCompileValueNoop(t *testing.T) {
	if CompileValue("foo", "") != "foo" {
		t.Errorf("compileValue didn't recognize there was no script")
	}
}

func TestNestedMaps(t *testing.T) {
	path := "testdata/nested_maps.yml"
	env, err := NewFlatEnv(path)
	if err != nil {
		t.Fatal("failed to load %q: %q", path, err)
	}

	if _, ok := env["FOO_BAR_BAZ"]; !ok {
		t.Fatalf("error getting key: %#v", env)
	}
}

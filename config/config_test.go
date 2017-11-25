package config_test

import (
	"os"
	"testing"

	"github.com/ionrock/we/config"
)

func TestGetConfig(t *testing.T) {
	data := map[string]string{
		"FOO": "bar",
	}
	c := config.Config{Data: data}

	if v := c.GetConfig("FOO"); v != "bar" {
		t.Errorf("config didn't contain expected value: %q != %q", v, "bar")
	}

	// Ensure we look up in the env when it doesn't exist in the
	// config.
	varName := "XE_CONFIG_TEST_VAR_FOO"
	os.Setenv(varName, "bar")
	defer os.Setenv(varName, "")
	if v := c.GetConfig(varName); v != "bar" {
		t.Errorf("config didn't contain expected value: %q != %q", v, "bar")
	}
}

func TestConfigGetAndSet(t *testing.T) {
	data := map[string]string{
		"FOO": "bar",
	}
	c := config.Config{Data: data}

	c.Set("FOO", "bar")

	if v, ok := c.Get("FOO"); v != "bar" || !ok {
		t.Errorf("config wasn't set correctly: %q != %q", v, "bar")
	}
}

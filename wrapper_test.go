package we

import (
	"testing"

	"github.com/urfave/cli"
)

func TestNewWrapper(t *testing.T) {
	before := ""
	after := false

	// wrap our ls command
	wrapper := cli.NewApp()
	wrapper.Before = func(c *cli.Context) error {
		before = c.String("foo")
		return nil
	}
	wrapper.After = func(c *cli.Context) error {
		after = true
		return nil
	}
	wrapper.Action = CommandAction
	wrapper.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "foo, f",
			Value: "hello",
		},
	}

	os_args := []string{"cmd", "--foo", "hi", "ls", "-la"}

	t.Log(len(os_args))

	wrapper.Run(os_args)

	if before != "hi" {
		t.Errorf("before: %s != hi", before)
	}

	if !after {
		t.Errorf("after: %s != true", after)
	}
}

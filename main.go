package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

var builddate = ""
var gitref = ""

func WeBefore(c *cli.Context) error {
	InitLogging(c.Bool("debug"))
	log.Debug("args: ", os.Args[1:])
	env, err := WithEnv(os.Args[1:])

	log.Debug("Computed Env")
	for k, v := range env {
		log.Debugf("export %s=%s", k, os.ExpandEnv(v))
	}
	return err
}

func main() {
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s-%s", gitref, builddate)

	app.Name = "we"
	app.Usage = "Add environment variables via YAML or scripts before running a command."
	app.ArgsUsage = "[COMMAND]"
	app.Before = WeBefore
	app.Action = CommandAction

	// NOTE: These flags are essentially ignored b/c we need ordered flags
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "A YAML/JSON file to include in the environment.",
		},

		cli.StringSliceFlag{
			Name:  "script, s",
			Usage: "Execute a script that outputs YAML/JSON.",
		},

		cli.StringFlag{
			Name:  "directory, d",
			Value: "",
			Usage: "A directory containing YAML/JSON files to recursively apply to the environment.",
		},

		cli.StringFlag{
			Name:  "alias, a",
			Value: "",
			Usage: "A YAML file containing a list of file/directory entries to apply to the environment.",
		},

		cli.StringSliceFlag{
			Name:  "envvar, E",
			Usage: "Override a single environment variable.",
		},

		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "Turn on debugging output",
		},
	}

	app.Run(os.Args)
}

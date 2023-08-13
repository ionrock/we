package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ionrock/we/envs"
	"github.com/ionrock/we/process"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var builddate = ""
var gitref = ""

func convertEnvForCmd(env map[string]string) []string {
	envlist := []string{}
	for key, _ := range env {
		if key != "" && os.Getenv(key) != "" {
			envlist = append(envlist, fmt.Sprintf("%s=%s", key, os.Getenv(key)))
		}
	}

	return envlist
}

func WeAction(c *cli.Context) error {
	InitLogging(c.Bool("debug"))

	log.Debug("initializing config")
	config, err := findConfig(".")
	if err != nil {
		log.Debugf("No config found: %q", err)
	}

	// weargs are the combined set of commmand line args after
	// considering automatic config like a ~/.withenv.yml.
	weargs := []string{}

	if config != "" {
		log.Debug("Adding config as alias: ", config)
		weargs = append(weargs, "--alias", config)
	}

	log.Debug("command args: ", os.Args[1:])
	weargs = append(weargs, os.Args[1:]...)

	log.Debug("all args: ", weargs)

	// We want to grab our working directory as it'll be used when
	// executing any commnds in scripts, or dynamic values (FOO=`cmd`)
	here, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %q", err)
		os.Exit(1)
	}

	// Create our env
	env, err := envs.WithEnv(weargs, here)

	log.Debug("Computed Env")
	for k, v := range env {
		log.Debugf("export %s=%s", k, v)
	}

	if err != nil {
		return err
	}

	// The args parsed after the flags from cli should be our actual
	// command.
	args := c.Args()

	parts := make([]string, args.Len())

	if len(parts) == 0 {
		parts = append(parts, "env")
	}

	for i, arg := range args.Slice() {
		parts[i] = os.ExpandEnv(arg)
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if c.Bool("clean") {
		log.Debugf("Cleaning env: %s", env)
		cmd.Env = convertEnvForCmd(env)
	}

	exitStatus, err := process.RunAndWait(cmd)
	if err != nil {
		os.Exit(exitStatus)
	}

	return err
}

func main() {
	app := cli.App{
		Name:      "we",
		Usage:     "Add environment variables via YAML or scripts before running a command.",
		Version:   fmt.Sprintf("%s-%s", gitref, builddate),
		ArgsUsage: "[COMMAND]",
		Flags: []cli.Flag{

			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"D"},
				Usage:   "Turn on debugging output",
			},

			&cli.StringSliceFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "A YAML/JSON file to include in the environment.",
			},

			&cli.StringSliceFlag{
				Name:    "script",
				Aliases: []string{"s"},
				Usage:   "Execute a script that outputs YAML/JSON.",
			},

			&cli.StringSliceFlag{
				Name:    "directory",
				Aliases: []string{"d"},
				Usage:   "A directory containing YAML/JSON files to recursively apply to the environment.",
			},

			&cli.StringSliceFlag{
				Name:    "alias",
				Aliases: []string{"a"},
				Usage:   "A YAML file containing a list of file/directory entries to apply to the environment.",
			},

			&cli.StringSliceFlag{
				Name:    "envvar",
				Aliases: []string{"E"},
				Usage:   "Override a single environment variable.",
			},

			&cli.BoolFlag{
				Name:    "clean",
				Aliases: []string{"c"},
				Usage:   "Only use variables defined by YAML",
			},

			&cli.StringSliceFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Apply a template.",
			},
		},
		Action: WeAction,
	}

	app.Run(os.Args)
}

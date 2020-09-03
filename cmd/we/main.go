package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ionrock/we/envs"
	"github.com/ionrock/we/process"
	"github.com/spf13/viper"

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
	InitConfig(".")

	// weargs are the combined set of commmand line args after
	// considering automatic config like a ~/.withenv.yml.
	weargs := []string{}

	if viper.IsSet("config_alias") {
		log.Debug("config alias: ", viper.GetString("config_alias"))
		weargs = append(weargs, "--alias", viper.GetString("config_alias"))
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

	if len(args) == 0 {
		args = []string{"env"}
	}
	parts := make([]string, len(args))

	for i, arg := range args {
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
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s-%s", gitref, builddate)

	app.Name = "we"
	app.Usage = "Add environment variables via YAML or scripts before running a command."
	app.ArgsUsage = "[COMMAND]"
	app.Action = WeAction

	// NOTE: These flags are essentially ignored b/c we need ordered flags
	app.Flags = []cli.Flag{

		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "Turn on debugging output",
		},

		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "A YAML/JSON file to include in the environment.",
		},

		cli.StringSliceFlag{
			Name:  "script, s",
			Usage: "Execute a script that outputs YAML/JSON.",
		},

		cli.StringSliceFlag{
			Name:  "directory, d",
			Usage: "A directory containing YAML/JSON files to recursively apply to the environment.",
		},

		cli.StringSliceFlag{
			Name:  "alias, a",
			Usage: "A YAML file containing a list of file/directory entries to apply to the environment.",
		},

		cli.StringSliceFlag{
			Name:  "envvar, E",
			Usage: "Override a single environment variable.",
		},

		cli.BoolFlag{
			Name:  "clean, c",
			Usage: "Only use variables defined by YAML",
		},

		cli.StringSliceFlag{
			Name:  "template, t",
			Usage: "Apply a template.",
		},
	}

	app.Run(os.Args)
}

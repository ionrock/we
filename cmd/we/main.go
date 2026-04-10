package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ionrock/we/envs"
	"github.com/ionrock/we/envscript"
	"github.com/ionrock/we/process"
	"github.com/ionrock/we/sandbox"

	"github.com/rs/zerolog/log"
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

	// Load .envrc-managed variables first so explicit we flags can override.
	here, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %q", err)
		os.Exit(1)
	}

	envrcEnv, err := envs.MaybeLoadEnvrc(here, c.Bool("no-direnv"))
	if err != nil {
		return err
	}

	log.Debug().Msg("initializing config")
	config, err := findConfig(".")
	if err != nil {
		log.Debug().Msgf("No config found: %q", err)
	}

	// weargs are the combined set of commmand line args after
	// considering automatic config like a ~/.withenv.yml.
	weargs := []string{}

	if config != "" {
		log.Debug().Msgf("Adding config as alias: %s", config)
		weargs = append(weargs, "--alias", config)
	}

	log.Debug().Msgf("command args: %v", os.Args[1:])
	weargs = append(weargs, os.Args[1:]...)

	log.Debug().Msgf("all args: %v", weargs)

	// Create our env
	env, err := envs.WithEnv(weargs, here)
	for k, v := range envrcEnv {
		if _, ok := env[k]; !ok {
			env[k] = v
		}
	}

	log.Debug().Msg("Computed Env")
	for k, v := range env {
		log.Debug().Msgf("export %s=%s", k, v)
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

	// Agent mode: set up sandbox and hooks
	if c.Bool("agent") {
		log.Debug().Msg("Agent mode enabled")

		// Collect sensitive paths from config
		var sensitivePaths []string
		if config != "" {
			paths, err := sandbox.CollectWithenvPaths(config)
			if err != nil {
				log.Debug().Err(err).Msg("error collecting withenv paths")
			} else {
				sensitivePaths = append(sensitivePaths, paths...)
			}
		}

		// Collect .envrc path
		envrcPath, _ := sandbox.CollectEnvrcPath(here)
		if envrcPath != "" {
			sensitivePaths = append(sensitivePaths, envrcPath)
		}

		// Build sandbox config
		sbCfg := sandbox.Config{
			Deny:    sandbox.DefaultDenyPaths(),
			DenyNet: c.Bool("sandbox-deny-network"),
		}

		// Add withenv-referenced paths to deny list
		for _, p := range sensitivePaths {
			info, err := os.Stat(p)
			if err != nil {
				continue
			}
			sbCfg.Deny = append(sbCfg.Deny, sandbox.DenyRule{
				Path: p,
				Dir:  info.IsDir(),
			})
		}

		// Add CLI deny overrides
		for _, p := range c.StringSlice("sandbox-deny") {
			resolved, err := sandbox.ResolvePath(p)
			if err != nil {
				log.Debug().Err(err).Str("path", p).Msg("skipping deny path")
				continue
			}
			info, statErr := os.Stat(resolved)
			sbCfg.Deny = append(sbCfg.Deny, sandbox.DenyRule{
				Path: resolved,
				Dir:  statErr == nil && info.IsDir(),
			})
		}

		// Add CLI allow overrides
		for _, p := range c.StringSlice("sandbox-allow") {
			resolved, err := sandbox.ResolvePath(p)
			if err != nil {
				log.Debug().Err(err).Str("path", p).Msg("skipping allow path")
				continue
			}
			info, statErr := os.Stat(resolved)
			sbCfg.Allow = append(sbCfg.Allow, sandbox.AllowRule{
				Path: resolved,
				Dir:  statErr == nil && info.IsDir(),
			})
		}

		// Parse sandbox config from .withenv.yml
		var skipPrefix []string
		if config != "" {
			userCfg, err := sandbox.ParseSandboxConfig(config)
			if err == nil && userCfg != nil {
				for _, p := range userCfg.Deny {
					resolved, err := sandbox.ResolvePath(p)
					if err != nil {
						continue
					}
					info, statErr := os.Stat(resolved)
					sbCfg.Deny = append(sbCfg.Deny, sandbox.DenyRule{
						Path: resolved,
						Dir:  statErr == nil && info.IsDir(),
					})
				}
				for _, p := range userCfg.Allow {
					resolved, err := sandbox.ResolvePath(p)
					if err != nil {
						continue
					}
					info, statErr := os.Stat(resolved)
					sbCfg.Allow = append(sbCfg.Allow, sandbox.AllowRule{
						Path: resolved,
						Dir:  statErr == nil && info.IsDir(),
					})
				}
				if userCfg.DenyNetwork {
					sbCfg.DenyNet = true
				}
				skipPrefix = userCfg.SkipPrefix
			}
		}

		// Set up Claude Code hooks for env re-evaluation
		weBin, _ := os.Executable()
		if weBin == "" {
			weBin = "we"
		}

		hooksCfg := sandbox.HooksConfig{
			WeBinary:   weBin,
			SkipPrefix: skipPrefix,
			ProjectDir: here,
		}

		cleanup, err := sandbox.WriteProjectSettings(hooksCfg)
		if err != nil {
			return fmt.Errorf("setting up agent hooks: %w", err)
		}

		// Register cleanup on exit and signals
		defer cleanup()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cleanup()
			os.Exit(1)
		}()

		// Create and apply filesystem sandbox
		sb, err := sandbox.New(sbCfg)
		if err != nil {
			cleanup()
			return fmt.Errorf("creating sandbox: %w", err)
		}

		wrappedCmd, wrappedArgs, err := sb.Wrap(parts[0], parts[1:])
		if err != nil {
			cleanup()
			return fmt.Errorf("wrapping command in sandbox: %w", err)
		}

		log.Debug().Str("cmd", wrappedCmd).Strs("args", wrappedArgs).Msg("launching sandboxed agent")

		cmd := exec.Command(wrappedCmd, wrappedArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if c.Bool("clean") {
			cmd.Env = convertEnvForCmd(env)
		}

		exitStatus, err := process.RunAndWait(cmd)
		cleanup()
		if err != nil {
			os.Exit(exitStatus)
		}

		return err
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if c.Bool("clean") {
		log.Debug().Msgf("Cleaning env: %v", env)
		cmd.Env = convertEnvForCmd(env)
	}

	exitStatus, err := process.RunAndWait(cmd)
	if err != nil {
		os.Exit(exitStatus)
	}

	return err
}

func ConvertAction(c *cli.Context) error {
	InitLogging(c.Bool("debug"))

	if c.NArg() == 0 {
		return fmt.Errorf("please provide a path to an env script file")
	}

	inputPath := c.Args().Get(0)
	outputPath := c.String("output")

	log.Debug().Msgf("Converting %s to YAML", inputPath)

	yamlData, err := envscript.ParseAndConvert(inputPath)
	if err != nil {
		return fmt.Errorf("failed to convert file: %w", err)
	}

	if outputPath != "" {
		err = os.WriteFile(outputPath, yamlData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		log.Debug().Msgf("Wrote YAML to %s", outputPath)
	} else {
		fmt.Print(string(yamlData))
	}

	return nil
}

func main() {
	app := cli.App{
		Name:      "we",
		Usage:     "Add environment variables via YAML or scripts before running a command.",
		Version:   fmt.Sprintf("%s-%s", gitref, builddate),
		ArgsUsage: "[COMMAND]",
		Commands: []*cli.Command{
			{
				Name:      "convert",
				Usage:     "Convert an env script file (.env, .envrc, etc.) to YAML",
				ArgsUsage: "<input-file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file path (defaults to stdout)",
					},
					&cli.BoolFlag{
						Name:    "debug",
						Aliases: []string{"D"},
						Usage:   "Turn on debugging output",
					},
				},
				Action: ConvertAction,
			},
		},
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

			&cli.BoolFlag{
				Name:  "no-direnv",
				Usage: "Disable automatic .envrc loading",
			},

			&cli.BoolFlag{
				Name:  "agent",
				Usage: "Run command in a sandboxed environment for AI agents",
			},

			&cli.StringSliceFlag{
				Name:  "sandbox-deny",
				Usage: "Additional paths to deny in agent sandbox",
			},

			&cli.StringSliceFlag{
				Name:  "sandbox-allow",
				Usage: "Paths to allow as exceptions to sandbox deny rules",
			},

			&cli.BoolFlag{
				Name:  "sandbox-deny-network",
				Usage: "Also restrict network access in agent sandbox",
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

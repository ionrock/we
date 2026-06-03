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
	"github.com/ionrock/we/sopsutil"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var builddate = ""
var gitref = ""

func versionString() string {
	if gitref == "" && builddate == "" {
		return "dev"
	}
	if gitref == "" {
		return builddate
	}
	if builddate == "" {
		return gitref
	}
	return fmt.Sprintf("%s-%s", gitref, builddate)
}

func convertEnvForCmd(env envs.Env) []string {
	return env.Environ()
}

func mergeEnv(dst, src envs.Env) envs.Env {
	if dst == nil {
		dst = make(envs.Env)
	}
	return dst.Merge(src)
}

func printEnv(env envs.Env, showSecrets bool) {
	for _, line := range env.Lines(showSecrets) {
		fmt.Println(line)
	}
}

func WeAction(c *cli.Context) error {
	InitLogging(c.Bool("debug"))
	if c.Bool("agent") && c.Bool("show-secrets") {
		return fmt.Errorf("--show-secrets cannot be used with --agent")
	}

	// Load .envrc-managed variables first so explicit we flags can override.
	here, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %q", err)
		os.Exit(1)
	}

	envrcVals, err := envs.MaybeLoadEnvrc(here, c.Bool("no-direnv"))
	if err != nil {
		return err
	}
	envrcEnv := envs.EnvFromStringMap(envrcVals, ".envrc")

	log.Debug().Msg("initializing config")
	config, err := findConfig(".")
	if err != nil {
		log.Debug().Msgf("No config found: %q", err)
	}

	// weargs are the combined set of command line args after
	// considering automatic config like ~/.withenv.yml.
	weargs := []string{}

	if config != "" {
		log.Debug().Msgf("Adding config as alias: %s", config)
		weargs = append(weargs, "--alias", config)
	}

	log.Debug().Msgf("command args: %v", os.Args[1:])
	weargs = append(weargs, os.Args[1:]...)

	log.Debug().Msgf("all args: %v", weargs)

	// Create our env with this precedence:
	// 1) ~/.withenv_global.yml (if present)
	// 2) .envrc values
	// 3) explicit withenv inputs (config alias + flags)
	env := envs.Env{}

	globalEnv, err := findGlobalEnv()
	if err != nil {
		log.Debug().Msgf("Unable to resolve global env: %q", err)
	}
	if globalEnv != "" {
		log.Debug().Msgf("Loading global env: %s", globalEnv)
		globalVals, err := envs.WithEnvTyped([]string{"--env", globalEnv}, here)
		if err != nil {
			return err
		}
		env = mergeEnv(env, globalVals)
	}

	env = mergeEnv(env, envrcEnv)

	explicitEnv, err := envs.WithEnvTypedFrom(weargs, here, env)
	if err != nil {
		return err
	}
	env = mergeEnv(env, explicitEnv)

	log.Debug().Msg("Computed Env")
	for k, v := range env {
		log.Debug().Msgf("export %s=%s", k, v.DisplayForKey(k, false))
	}

	if err != nil {
		return err
	}

	// The args parsed after the flags from cli should be our actual
	// command.
	args := c.Args()

	parts := make([]string, args.Len())

	if len(parts) == 0 {
		printEnv(env, c.Bool("show-secrets"))
		return nil
	}

	for i, arg := range args.Slice() {
		parts[i] = os.ExpandEnv(arg)
	}

	// Agent mode: set up sandbox and hooks
	if c.Bool("agent") {
		log.Debug().Msg("Agent mode enabled")

		// Collect sensitive environment source paths. The child command gets the
		// computed values, but should not be able to read the source files.
		var sensitivePaths []string
		if globalEnv != "" {
			if resolved, err := sandbox.ResolvePath(globalEnv); err == nil {
				sensitivePaths = append(sensitivePaths, resolved)
			}
		}
		if config != "" {
			paths, err := sandbox.CollectWithenvPaths(config)
			if err != nil {
				log.Debug().Err(err).Msg("error collecting withenv paths")
			} else {
				sensitivePaths = append(sensitivePaths, paths...)
			}
		}
		sensitivePaths = append(sensitivePaths, sandbox.CollectWithenvArgPaths(weargs, here)...)

		// Collect .envrc plus dotenv/source_env files referenced from it.
		envrcPaths, _ := sandbox.CollectEnvrcPaths(here)
		sensitivePaths = append(sensitivePaths, envrcPaths...)

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

func SOPSDecryptAction(c *cli.Context) error {
	InitLogging(c.Bool("debug"))
	if c.NArg() == 0 {
		return fmt.Errorf("please provide a SOPS-encrypted YAML file")
	}
	inputPath := c.Args().Get(0)
	cleartext, err := sopsutil.DecryptYAMLFile(inputPath)
	if err != nil {
		return err
	}
	if outputPath := c.String("output"); outputPath != "" {
		return os.WriteFile(outputPath, cleartext, 0600)
	}
	fmt.Print(string(cleartext))
	return nil
}

func SOPSEncryptAction(c *cli.Context) error {
	InitLogging(c.Bool("debug"))
	if c.NArg() == 0 {
		return fmt.Errorf("please provide a plaintext YAML file")
	}
	inputPath := c.Args().Get(0)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	encrypted, err := sopsutil.EncryptYAML(inputPath, data, sopsutil.EncryptOptions{
		AgeRecipients:    c.StringSlice("age"),
		EncryptedRegex:   c.String("encrypted-regex"),
		UnencryptedRegex: c.String("unencrypted-regex"),
	})
	if err != nil {
		return err
	}
	outputPath := c.String("output")
	if c.Bool("in-place") {
		outputPath = inputPath
	}
	if outputPath != "" {
		return os.WriteFile(outputPath, encrypted, 0600)
	}
	fmt.Print(string(encrypted))
	return nil
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
		Version:   versionString(),
		ArgsUsage: "[COMMAND]",
		Commands: []*cli.Command{
			{
				Name:  "sops",
				Usage: "Encrypt or decrypt YAML files with SOPS using the Go library",
				Subcommands: []*cli.Command{
					{
						Name:      "decrypt",
						Usage:     "Decrypt a SOPS-encrypted YAML file",
						ArgsUsage: "<input-file>",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Write plaintext YAML to `FILE` instead of stdout"},
							&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Usage: "Turn on debug logging"},
						},
						Action: SOPSDecryptAction,
					},
					{
						Name:      "encrypt",
						Usage:     "Encrypt a YAML file with SOPS and Age recipients",
						ArgsUsage: "<input-file>",
						Flags: []cli.Flag{
							&cli.StringSliceFlag{Name: "age", Usage: "Age `RECIPIENT` to encrypt to (repeatable); falls back to .sops.yaml when omitted"},
							&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Write encrypted YAML to `FILE` instead of stdout"},
							&cli.BoolFlag{Name: "in-place", Usage: "Replace the input file with encrypted YAML"},
							&cli.StringFlag{Name: "encrypted-regex", Usage: "Encrypt only keys matching `REGEX`"},
							&cli.StringFlag{Name: "unencrypted-regex", Usage: "Leave keys matching `REGEX` unencrypted"},
							&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Usage: "Turn on debug logging"},
						},
						Action: SOPSEncryptAction,
					},
				},
			},
			{
				Name:      "convert",
				Usage:     "Convert a dotenv-style env script (.env, .envrc, etc.) to withenv YAML",
				ArgsUsage: "<input-file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Write converted YAML to `FILE` instead of stdout",
					},
					&cli.BoolFlag{
						Name:    "debug",
						Aliases: []string{"D"},
						Usage:   "Turn on debug logging",
					},
				},
				Action: ConvertAction,
			},
		},
		Flags: []cli.Flag{

			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"D"},
				Usage:   "Turn on debug logging",
			},

			&cli.StringSliceFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Load environment variables from a YAML or JSON `FILE` (repeatable)",
			},

			&cli.StringSliceFlag{
				Name:    "script",
				Aliases: []string{"s"},
				Usage:   "Run `COMMAND` and load its YAML or JSON output (repeatable)",
			},

			&cli.StringSliceFlag{
				Name:    "directory",
				Aliases: []string{"d"},
				Usage:   "Recursively load YAML/JSON files from `DIR` (repeatable)",
			},

			&cli.StringSliceFlag{
				Name:    "alias",
				Aliases: []string{"a"},
				Usage:   "Load a withenv alias YAML `FILE` containing file, directory, script, envvar, template, or sandbox entries (repeatable)",
			},

			&cli.StringSliceFlag{
				Name:    "envvar",
				Aliases: []string{"E"},
				Usage:   "Set or override one environment variable as `KEY=VALUE` (repeatable)",
			},

			&cli.BoolFlag{
				Name:    "clean",
				Aliases: []string{"c"},
				Usage:   "Run command with only variables loaded by withenv sources",
			},

			&cli.BoolFlag{
				Name:  "show-secrets",
				Usage: "Show secret values in no-command inspection output (disabled with --agent)",
			},

			&cli.BoolFlag{
				Name:  "no-direnv",
				Usage: "Disable automatic upward search and loading of .envrc",
			},

			&cli.BoolFlag{
				Name:  "agent",
				Usage: "Run command with agent protections: filesystem sandbox plus Claude Code command re-evaluation hooks",
			},

			&cli.StringSliceFlag{
				Name:  "sandbox-deny",
				Usage: "Deny read access to additional `PATH` values in agent sandbox (repeatable)",
			},

			&cli.StringSliceFlag{
				Name:  "sandbox-allow",
				Usage: "Allow `PATH` values as exceptions to agent sandbox deny rules (repeatable)",
			},

			&cli.BoolFlag{
				Name:  "sandbox-deny-network",
				Usage: "Also restrict network access in agent sandbox",
			},

			&cli.StringSliceFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Render a Go `TEMPLATE[:TARGET]` file before running the command (repeatable)",
			},
		},
		Action: WeAction,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

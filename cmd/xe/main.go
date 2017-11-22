package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ionrock/we/config"
	"github.com/ionrock/we/process"
	"github.com/ionrock/we/process/forego"
	"github.com/urfave/cli"
)

var builddate = ""
var gitref = ""

type Environment struct {
	Services  *process.Manager
	Tasks     map[string]*exec.Cmd
	Config    *config.Config
	ConfigDir string
}

func (e *Environment) ConfigHandler(cfg XeConfig) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	switch {
	case cfg.Service != nil:
		svc := cfg.Service
		err := e.Services.Start(svc.Name, svc.Cmd, svc.Dir, e.Config.ToEnv(), e.Services.Output)
		if err != nil {
			return err
		}

	case cfg.Env != nil:
		for k, v := range cfg.Env {
			v = os.Expand(v, e.Config.GetConfig)
			val, err := process.CompileValue(v, e.ConfigDir)
			if err != nil {
				fmt.Printf("error getting value for env: %q %q\n", v, err)
				return err
			}
			fmt.Println(fmt.Sprintf("setting: %s=%s", k, val))
			e.Config.Set(k, val)
		}

	case cfg.EnvScript != "":
		s := config.Script{Cmd: cfg.EnvScript, Dir: wd}
		err := s.Apply(e.Config)
		if err != nil {
			return err
		}

	case cfg.Task != nil:
		t := cfg.Task
		proc := process.New(t.Cmd, t.Dir)
		fmt.Println("Running Task: " + t.Name)

		out, err := proc.Execute()
		fmt.Println(out.String())
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return err
		}

	}

	return nil
}

func (e *Environment) CleanUp() {
	for name, _ := range e.Services.Processes {
		err := e.Services.Stop(name)
		if err != nil {
			log.Printf("error killing service: %q\n", err)
		}
	}
}

func findLongestServiceName(cfgs []XeConfig) int {
	size := 0

	for _, cfg := range cfgs {
		if cfg.Service == nil {
			continue
		}

		if len(cfg.Service.Name) > size {
			size = len(cfg.Service.Name)
		}
	}

	return size
}

func XeAction(c *cli.Context) error {
	fmt.Println("loading " + c.String("config"))

	cfg, err := NewXeConfig(c.String("config"))
	if err != nil {
		fmt.Printf("error loading config: %s\n", err)
	}

	of := forego.NewOutletFactory()
	of.Padding = findLongestServiceName(cfg)

	env := &Environment{
		Services: &process.Manager{
			Processes: make(map[string]*exec.Cmd),
			Output:    of,
		},
		Tasks:     make(map[string]*exec.Cmd),
		Config:    &config.Config{make(map[string]string)},
		ConfigDir: filepath.Dir(c.String("config")),
	}

	for _, c := range cfg {
		if err := env.ConfigHandler(c); err != nil {
			return err
		}
	}

	parts := c.Args()

	fmt.Printf("Going to start: %s\n", parts)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.Env = env.Config.ToEnv()

	err = cmd.Run()
	env.CleanUp()

	if err != nil {
		return err
	}

	return nil
}

func main() {
	app := cli.NewApp()

	app.Version = fmt.Sprintf("%s-%s", gitref, builddate)

	app.Name = "xe"
	app.Usage = "Start and monitor processes creating an executable environment."
	app.ArgsUsage = "[COMMAND]"
	app.Action = XeAction

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Path to the xe config file, default is ./xe.yml",
			Value: "xe.yml",
		},
	}

	app.Run(os.Args)
}

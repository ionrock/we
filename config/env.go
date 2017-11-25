package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/ionrock/we/process"
	"github.com/ionrock/we/process/forego"
)

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

// Interfaces

// ProcessManager provides an interface to the process.Manager.
type ProcessManager interface {
	Processes() map[string]*exec.Cmd
	Start(name string, cmd string, dir string, env []string, of *forego.OutletFactory) error
	Stop(name string) error
}

type Environment struct {
	Services  ProcessManager
	Tasks     map[string]*exec.Cmd
	Config    *Config
	ConfigDir string
}

func NewEnvironment(cfgDir string, cfgs []XeConfig) *Environment {
	of := forego.NewOutletFactory()
	of.Padding = findLongestServiceName(cfgs)

	return &Environment{
		Services:  process.NewManager(of),
		Tasks:     make(map[string]*exec.Cmd),
		Config:    &Config{make(map[string]string)},
		ConfigDir: cfgDir,
	}
}

func (e *Environment) StartService(name, cmd, dir string) error {
	return e.Services.Start(name, cmd, dir, e.Config.ToEnv(), nil)
}

func (e *Environment) SetEnv(k, v string) error {
	v = os.Expand(v, e.Config.GetConfig)
	val, err := process.CompileValue(v, e.ConfigDir)
	if err != nil {
		fmt.Printf("error getting value for env: %q %q\n", v, err)
		return err
	}
	e.Config.Set(k, val)
	return nil
}

func (e *Environment) SetEnvFromScript(cmd, dir string) error {
	s := Script{Cmd: cmd, Dir: dir}
	err := s.Apply(e.Config)
	if err != nil {
		return err
	}

	return nil
}

func (e *Environment) RunTask(name, cmd, dir string) error {
	proc := process.New(cmd, dir)
	fmt.Println("Running Task: " + name)

	// TODO: Use some better logging here.
	out, err := proc.Execute()
	fmt.Println(out.String())
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}

	return nil
}

func (e *Environment) ConfigHandler(cfg XeConfig) error {
	switch {
	case cfg.Service != nil:
		err := e.StartService(cfg.Service.Name, cfg.Service.Cmd, cfg.Service.Dir)
		if err != nil {
			return err
		}

	case cfg.Env != nil:
		for k, v := range cfg.Env {
			err := e.SetEnv(k, v)
			return err
		}

	case cfg.EnvScript != "":
		err := e.SetEnvFromScript(cfg.EnvScript, e.ConfigDir)
		if err != nil {
			return err
		}

	case cfg.Task != nil:
		err := e.RunTask(cfg.Task.Name, cfg.Task.Cmd, cfg.Task.Dir)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environment) CleanUp() {
	for name, _ := range e.Services.Processes() {
		err := e.Services.Stop(name)
		if err != nil {
			log.Printf("error killing service: %q\n", err)
		}
	}
}

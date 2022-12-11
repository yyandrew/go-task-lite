package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

type Cmd struct {
	Cmd string
}

type Task struct {
	Task string
	Cmds []*Cmd
}

type Taskfile struct {
	Version string
	Tasks   map[string]*Task
}

const defaultTasksFile = `
version: "1"
tasks:
  hello:
    cmds:
      - echo 'hello go-task-lite'
`

func main() {
	var (
		init       bool
		entrypoint string
	)
	pflag.BoolVar(&init, "init", false, "create a new task.yaml")
	pflag.StringVar(&entrypoint, "taskfile", "tasks.yaml", `choose which Taskfile to run. Defaults to "Taskfile.yml"`)
	pflag.Parse()

	fmt.Printf("init: %v\n", init)
	if init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := InitTaskfile(os.Stdout, wd); err != nil {
			log.Fatal(err)
		}

		return
	}

	ctx := context.Background()
	f, err := os.Open(entrypoint)
	if err != nil {
		log.Fatal(err)
	}
	var taskfile Taskfile
	err = yaml.NewDecoder(f).Decode(&taskfile)
	if err != nil {
		log.Fatal(err)
	}
	environ := os.Environ()
	r, err := interp.New(
		interp.Env(expand.ListEnviron(environ...)),
		interp.StdIO(nil, os.Stdout, os.Stdout),
	)
	if err != nil {
		log.Fatal("Setup execute environ falied")
	}
	for name, task := range taskfile.Tasks {
		fmt.Printf("task: \"%v\" started\n", name)
		for _, cmd := range task.Cmds {
			opt, err := syntax.NewParser().Parse(strings.NewReader(cmd.Cmd), "")
			if err != nil {
				log.Fatal("Parse command failed")
			}
			r.Run(ctx, opt)
		}
	}

}

func (t *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tf struct {
		Version string
		Tasks   map[string]*Task
	}
	err := unmarshal(&tf)
	if err != nil {
		return err
	}
	t.Version = tf.Version
	t.Tasks = tf.Tasks

	return nil
}

func (c *Cmd) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string
	if err := unmarshal(&cmd); err != nil {
		return err
	}
	c.Cmd = cmd
	return nil
}

func InitTaskfile(w io.Writer, dir string) error {
	f := filepath.Join(dir, "tasks.yaml")
	if _, err := os.Stat(f); err == nil {
		return errors.New("task: A Taskfile already exists")
	}
	if err := os.WriteFile(f, []byte(defaultTasksFile), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(w, "tasks.yaml created in the current directory\n")
	return nil
}

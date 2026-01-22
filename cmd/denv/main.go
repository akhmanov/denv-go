package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

type EnvFile struct {
	Path     string
	Optional bool
}

type envFileFlag struct {
	files    *[]EnvFile
	optional bool
}

func (f *envFileFlag) String() string {
	return ""
}

func (f *envFileFlag) Set(value string) error {
	*f.files = append(*f.files, EnvFile{Path: value, Optional: f.optional})
	return nil
}

func main() {
	var files []EnvFile

	app := &cli.App{
		Name:  "denv",
		Usage: "A simple CLI utility to manage environment variables from .env files",
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "path to .env file",
				Value:   &envFileFlag{files: &files, optional: false},
			},
			&cli.GenericFlag{
				Name:    "file-optional",
				Aliases: []string{"fo"},
				Usage:   "path to .env file (optional, ignore if missing)",
				Value:   &envFileFlag{files: &files, optional: true},
			},
			&cli.BoolFlag{
				Name:    "isolate",
				Aliases: []string{"i"},
				Usage:   "ignore system environment variables (load only from .env files)",
			},
		},
		Before: func(c *cli.Context) error {
			if c.App.Metadata == nil {
				c.App.Metadata = make(map[string]interface{})
			}
			c.App.Metadata["files"] = &files
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:            "exec",
				Usage:           "Execute a command with the loaded environment variables",
				SkipFlagParsing: true,
				Action:          runExec,
			},
			{
				Name:      "get",
				Usage:     "Get the value of a specific environment variable",
				ArgsUsage: "<KEY>",
				Action:    runGet,
			},
			{
				Name:  "keys",
				Usage: "List all available environment variable keys",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "output format (text, json)",
						Value:   "text",
					},
				},
				Action: runKeys,
			},
			{
				Name:  "list",
				Usage: "List all environment variables in KEY=VALUE format",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "output format (text, json)",
						Value:   "text",
					},
				},
				Action: runList,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadEnv(c *cli.Context) (map[string]string, error) {
	envMap := make(map[string]string)

	if !c.Bool("isolate") {
		for _, e := range os.Environ() {
			pair := strings.SplitN(e, "=", 2)
			if len(pair) == 2 {
				envMap[pair[0]] = pair[1]
			}
		}
	}

	var files []EnvFile
	if v, ok := c.App.Metadata["files"]; ok {
		if f, ok := v.(*[]EnvFile); ok {
			files = *f
		}
	}

	for _, file := range files {
		loaded, err := godotenv.Read(file.Path)
		if err != nil {
			if file.Optional && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to read %s: %w", file.Path, err)
		}

		maps.Copy(envMap, loaded)
	}

	return envMap, nil
}

func runExec(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	envMap, err := loadEnv(c)
	if err != nil {
		return err
	}

	envSlice := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = envSlice
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	err = cmd.Wait()

	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}

	return err
}

func runGet(c *cli.Context) error {
	key := c.Args().First()
	if key == "" {
		return fmt.Errorf("key argument is required")
	}

	envMap, err := loadEnv(c)
	if err != nil {
		return err
	}

	val, ok := envMap[key]
	if !ok {
		return cli.Exit(fmt.Sprintf("key '%s' not found", key), 1)
	}

	fmt.Fprintln(c.App.Writer, val)
	return nil
}

func runKeys(c *cli.Context) error {
	envMap, err := loadEnv(c)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	output := c.String("output")

	if output == "json" {
		data, err := json.Marshal(keys)
		if err != nil {
			return err
		}
		fmt.Fprintln(c.App.Writer, string(data))
	} else {
		for _, k := range keys {
			fmt.Fprintln(c.App.Writer, k)
		}
	}

	return nil
}

func runList(c *cli.Context) error {
	envMap, err := loadEnv(c)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	output := c.String("output")

	if output == "json" {
		data, err := json.Marshal(envMap)
		if err != nil {
			return err
		}
		fmt.Fprintln(c.App.Writer, string(data))
	} else {
		for _, k := range keys {
			fmt.Fprintf(c.App.Writer, "%s=%s\n", k, envMap[k])
		}
	}

	return nil
}

package main

import (
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

func main() {
	app := &cli.App{
		Name:  "denv",
		Usage: "A simple CLI utility to manage environment variables from .env files",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "path to .env file",
			},
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
				Name:   "keys",
				Usage:  "List all available environment variable keys",
				Action: runKeys,
			},
			{
				Name:   "list",
				Usage:  "List all environment variables in KEY=VALUE format",
				Action: runList,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// loadEnv loads environment variables from files and merges them with the system environment.
// System environment variables are loaded first, then .env files override them in order.
func loadEnv(c *cli.Context) (map[string]string, error) {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}

	files := c.StringSlice("file")
	if len(files) == 0 {
		if _, err := os.Stat(".env"); err == nil {
			files = []string{".env"}
		}
	}

	for _, file := range files {
		loaded, err := godotenv.Read(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
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

	// Convert map back to []string environment
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

	fmt.Println(val)
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

	for _, k := range keys {
		fmt.Println(k)
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

	for _, k := range keys {
		fmt.Printf("%s=%s\n", k, envMap[k])
	}

	return nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestLoadEnv(t *testing.T) {
	tmpDir := t.TempDir()
	env1 := filepath.Join(tmpDir, ".env1")
	env2 := filepath.Join(tmpDir, ".env2")

	if err := os.WriteFile(env1, []byte("FOO=bar\nCOMMON=1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env2, []byte("BAZ=qux\nCOMMON=2"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "file"},
			&cli.BoolFlag{Name: "isolate"},
		},
		Action: func(c *cli.Context) error {
			envMap, err := loadEnv(c)
			if err != nil {
				return err
			}

			if envMap["COMMON"] != "2" {
				return fmt.Errorf("expected COMMON=2, got %s", envMap["COMMON"])
			}
			if envMap["FOO"] != "bar" {
				return fmt.Errorf("expected FOO=bar, got %s", envMap["FOO"])
			}
			if envMap["BAZ"] != "qux" {
				return fmt.Errorf("expected BAZ=qux, got %s", envMap["BAZ"])
			}
			return nil
		},
	}

	args := []string{"denv", "--file", env1, "--file", env2}
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultEnv(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	if err := os.WriteFile(".env", []byte("DEFAULT=true"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "file"},
			&cli.BoolFlag{Name: "isolate"},
		},
		Action: func(c *cli.Context) error {
			envMap, err := loadEnv(c)
			if err != nil {
				return err
			}
			if envMap["DEFAULT"] != "true" {
				return fmt.Errorf("expected DEFAULT=true, got %s", envMap["DEFAULT"])
			}
			return nil
		},
	}

	if err := app.Run([]string{"denv"}); err != nil {
		t.Fatal(err)
	}
}

func TestIsolate(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("MY_VAR=hello"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("SYSTEM_VAR", "system")
	defer os.Unsetenv("SYSTEM_VAR")

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "file"},
			&cli.BoolFlag{Name: "isolate"},
		},
		Action: func(c *cli.Context) error {
			envMap, err := loadEnv(c)
			if err != nil {
				return err
			}

			if envMap["MY_VAR"] != "hello" {
				return fmt.Errorf("expected MY_VAR=hello, got %s", envMap["MY_VAR"])
			}

			if _, ok := envMap["SYSTEM_VAR"]; ok {
				return fmt.Errorf("expected SYSTEM_VAR to be absent in isolate mode")
			}
			return nil
		},
	}

	args := []string{"denv", "--file", envFile, "--isolate"}
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
}

func TestKeysJSON(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("FOO=bar\nBAZ=qux"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "file"},
			&cli.BoolFlag{Name: "isolate"},
		},
		Commands: []*cli.Command{
			{
				Name: "keys",
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
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	args := []string{"denv", "--file", envFile, "--isolate", "keys", "--output", "json"}
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}

	var keys []string
	if err := json.Unmarshal(buf.Bytes(), &keys); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput was: %q", err, buf.String())
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestListJSON(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("FOO=bar\nBAZ=qux"), 0644); err != nil {
		t.Fatal(err)
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "file"},
			&cli.BoolFlag{Name: "isolate"},
		},
		Commands: []*cli.Command{
			{
				Name: "list",
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

	var buf bytes.Buffer
	app.Writer = &buf

	args := []string{"denv", "--file", envFile, "--isolate", "list", "--output", "json"}
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}

	var env map[string]string
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput was: %q", err, buf.String())
	}

	if env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %s", env["FOO"])
	}
	if env["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %s", env["BAZ"])
	}
}

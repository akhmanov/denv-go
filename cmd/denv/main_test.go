package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestLoadEnv(t *testing.T) {
	// Create temporary .env files
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

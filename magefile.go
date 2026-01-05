//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target when running mage without arguments.
var Default = Build

// Build builds the server binary.
func Build() error {
	mg.Deps(Generate)
	fmt.Println("Building server...")
	return sh.Run("go", "build", "-o", "bin/server", "./cmd/server")
}

// Generate runs all code generation (wire, etc.).
func Generate() error {
	mg.Deps(Wire)
	return nil
}

// Wire runs wire to generate dependency injection code.
func Wire() error {
	fmt.Println("Running wire...")

	// Find all directories containing wire.go files
	wireDirs, err := findWireDirs()
	if err != nil {
		return fmt.Errorf("finding wire directories: %w", err)
	}

	for _, dir := range wireDirs {
		fmt.Printf("  Generating wire code for %s\n", dir)
		if err := sh.Run("wire", dir); err != nil {
			return fmt.Errorf("wire %s: %w", dir, err)
		}
	}

	return nil
}

// findWireDirs finds all directories containing wire.go files.
func findWireDirs() ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Look for wire.go files
		if info.Name() == "wire.go" {
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, "./"+dir)
			}
		}

		return nil
	})

	return dirs, err
}

// Test runs all tests.
func Test() error {
	fmt.Println("Running tests...")
	return sh.Run("go", "test", "-v", "./...")
}

// TestCover runs tests with coverage.
func TestCover() error {
	fmt.Println("Running tests with coverage...")
	return sh.Run("go", "test", "-cover", "-coverprofile=coverage.out", "./...")
}

// Lint runs golangci-lint.
func Lint() error {
	fmt.Println("Running linter...")
	return sh.Run("golangci-lint", "run", "./...")
}

// Vet runs go vet.
func Vet() error {
	fmt.Println("Running go vet...")
	return sh.Run("go", "vet", "./...")
}

// Clean removes build artifacts.
func Clean() error {
	fmt.Println("Cleaning...")

	// Remove bin directory
	if err := os.RemoveAll("bin"); err != nil {
		return err
	}

	// Remove coverage files
	_ = os.Remove("coverage.out")

	// Remove generated wire files
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "wire_gen.go" {
			fmt.Printf("  Removing %s\n", path)
			return os.Remove(path)
		}
		return nil
	})
}

// Tidy runs go mod tidy.
func Tidy() error {
	fmt.Println("Running go mod tidy...")
	return sh.Run("go", "mod", "tidy")
}

// All runs tidy, generate, vet, lint, test, and build.
func All() error {
	mg.SerialDeps(Tidy, Generate, Vet, Lint, Test, Build)
	return nil
}

// Dev builds and runs the server for development.
func Dev() error {
	mg.Deps(Build)
	fmt.Println("Starting server...")
	cmd := exec.Command("./bin/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CI runs the CI pipeline (tidy, generate, vet, test with coverage).
func CI() error {
	mg.SerialDeps(Tidy, Generate, Vet, TestCover)
	return nil
}

// Install installs development tools.
func Install() error {
	fmt.Println("Installing development tools...")

	tools := []string{
		"github.com/google/wire/cmd/wire@latest",
		"github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
	}

	for _, tool := range tools {
		fmt.Printf("  Installing %s\n", tool)
		if err := sh.Run("go", "install", tool); err != nil {
			return fmt.Errorf("installing %s: %w", tool, err)
		}
	}

	return nil
}

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

// Generate runs all code generation (wire, swagger, etc.).
func Generate() error {
	mg.Deps(Wire, Swagger)
	return nil
}

// Swagger generates Swagger/OpenAPI documentation for all modules.
func Swagger() error {
	fmt.Println("Generating Swagger documentation...")
	return sh.Run("swag", "init", "-g", "cmd/server/docs.go", "-o", "cmd/server/docs", "--parseDependency", "--parseInternal")
}

// swaggerModules defines the available modules for swagger generation.
var swaggerModules = map[string]struct {
	Tag  string // Swagger tag name
	Desc string // Description for display
}{
	"user":          {Tag: "User", Desc: "User management APIs"},
	"auth":          {Tag: "Auth", Desc: "Authentication APIs"},
	"billing":       {Tag: "Billing", Desc: "Billing & subscription APIs"},
	"order":         {Tag: "Order", Desc: "Order management APIs"},
	"payment":       {Tag: "Payment", Desc: "Payment APIs"},
	"git":           {Tag: "Git", Desc: "Git hosting APIs"},
	"collaboration": {Tag: "Collaboration", Desc: "Team collaboration APIs"},
	"ai":            {Tag: "AI", Desc: "AI service APIs"},
}

// SwaggerModule generates Swagger documentation for a specific module.
// Usage: mage swaggermodule <module>
// Available modules: user, auth, billing, order, payment, git, collaboration, ai
func SwaggerModule(module string) error {
	mod, ok := swaggerModules[module]
	if !ok {
		fmt.Println("Available modules:")
		for name, m := range swaggerModules {
			fmt.Printf("  %-15s - %s\n", name, m.Desc)
		}
		return fmt.Errorf("unknown module: %s", module)
	}

	outputDir := fmt.Sprintf("cmd/server/docs/%s", module)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	fmt.Printf("Generating Swagger documentation for %s module (tag: %s)...\n", module, mod.Tag)

	// Generate swagger for specific module using tag filter
	return sh.Run("swag", "init",
		"--generalInfo", "cmd/server/docs.go",
		"--dir", ".",
		"--output", outputDir,
		"--parseDependency",
		"--parseInternal",
		"--instanceName", module,
		"--tags", mod.Tag,
	)
}

// SwaggerList lists all available modules for swagger generation.
func SwaggerList() error {
	fmt.Println("Available modules for swagger generation:")
	fmt.Println()
	for name, mod := range swaggerModules {
		fmt.Printf("  %-15s - %s (tag: %s)\n", name, mod.Desc, mod.Tag)
	}
	fmt.Println()
	fmt.Println("Usage: mage swaggermodule <module>")
	fmt.Println("Example: mage swaggermodule user")
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
		"google.golang.org/protobuf/cmd/protoc-gen-go@v1.30.0",
		"github.com/google/wire/cmd/wire@latest",
		"github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
		"github.com/swaggo/swag/cmd/swag@latest",
	}

	for _, tool := range tools {
		fmt.Printf("  Installing %s\n", tool)
		if err := sh.Run("go", "install", tool); err != nil {
			return fmt.Errorf("installing %s: %w", tool, err)
		}
	}

	return nil
}

// Proto builds protoc plugins and generates Go/HTTP (Gin) code from *.proto.
func Proto() error {
	mg.Deps(ProtoTools)
	fmt.Println("Generating proto REST stubs (Gin)...")

	files, err := filepath.Glob("api/*/protobuf_spec/*.proto")
	if err != nil {
		return fmt.Errorf("finding proto files: %w", err)
	}
	if len(files) == 0 {
		fmt.Println("  No proto files found under api/*/protobuf_spec/")
		return nil
	}

	env := map[string]string{
		"PATH": filepath.Join(".", "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	args := []string{
		"--proto_path=.",
		"--proto_path=third_party",
		"--go_out=paths=import,module=github.com/uniedit/server:.",
		"--go-gin_out=paths=import,module=github.com/uniedit/server:.",
	}
	args = append(args, files...)

	return sh.RunWith(env, "protoc", args...)
}

// ProtoTools builds local protoc plugins used by Proto().
func ProtoTools() error {
	fmt.Println("Building protoc plugins...")
	if err := sh.Run("go", "build", "-o", "bin/protoc-gen-go-gin", "./cmd/protoc-gen-go-gin"); err != nil {
		return fmt.Errorf("build protoc-gen-go-gin: %w", err)
	}
	return nil
}

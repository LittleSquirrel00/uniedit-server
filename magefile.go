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

// Generate runs all code generation (wire, proto, etc.).
func Generate() error {
	mg.Deps(Wire, Proto)
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

	const protoRoot = "api/protobuf_spec"
	const pbOutRoot = "api/pb"

	files, err := filepath.Glob(filepath.Join(protoRoot, "*", "*.proto"))
	if err != nil {
		return fmt.Errorf("finding proto files: %w", err)
	}
	if len(files) == 0 {
		fmt.Printf("  No proto files found under %s/*/*.proto\n", protoRoot)
		return nil
	}

	env := map[string]string{
		"PATH": filepath.Join(".", "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	if err := os.MkdirAll(pbOutRoot, 0755); err != nil {
		return fmt.Errorf("create pb output dir %s: %w", pbOutRoot, err)
	}

	relFiles := make([]string, 0, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(protoRoot, f)
		if err != nil {
			return fmt.Errorf("rel proto path for %s: %w", f, err)
		}
		relFiles = append(relFiles, filepath.ToSlash(rel))
	}

	args := []string{
		"--proto_path=" + protoRoot,
		"--proto_path=third_party",
		"--go_out=paths=source_relative:" + pbOutRoot,
		"--go-gin_out=paths=source_relative:" + pbOutRoot,
	}
	args = append(args, relFiles...)

	if err := sh.RunWith(env, "protoc", args...); err != nil {
		return err
	}
	return ProtoOpenAPI()
}

// ProtoTools builds local protoc plugins used by Proto().
func ProtoTools() error {
	fmt.Println("Building protoc plugins...")
	if err := sh.Run("go", "build", "-o", "bin/protoc-gen-go-gin", "./cmd/protoc-gen-go-gin"); err != nil {
		return fmt.Errorf("build protoc-gen-go-gin: %w", err)
	}

	binDir, err := filepath.Abs("bin")
	if err != nil {
		return fmt.Errorf("abs bin dir: %w", err)
	}
	env := map[string]string{
		"GOBIN": binDir,
	}
	if err := sh.RunWith(env, "go", "install", "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.4"); err != nil {
		return fmt.Errorf("install protoc-gen-openapiv2: %w", err)
	}
	return nil
}

// ProtoOpenAPI generates Swagger/OpenAPI specs from proto (google.api.http) mappings.
//
// Output: api/openapi_spec/<module>/<module>.swagger.yaml (merged per module).
func ProtoOpenAPI() error {
	mg.Deps(ProtoTools)
	fmt.Println("Generating OpenAPI from proto (google.api.http)...")

	const protoRoot = "api/protobuf_spec"
	const openapiOutRoot = "api/openapi_spec"

	protoDirs, err := filepath.Glob(filepath.Join(protoRoot, "*"))
	if err != nil {
		return fmt.Errorf("finding proto directories: %w", err)
	}
	if len(protoDirs) == 0 {
		fmt.Printf("  No proto directories found under %s/*\n", protoRoot)
		return nil
	}

	env := map[string]string{
		"PATH": filepath.Join(".", "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}

	generatedAny := false
	for _, protoDir := range protoDirs {
		st, err := os.Stat(protoDir)
		if err != nil {
			return fmt.Errorf("stat proto dir %s: %w", protoDir, err)
		}
		if !st.IsDir() {
			continue
		}

		module := filepath.Base(protoDir)
		outDir := filepath.Join(openapiOutRoot, module)

		protos, err := filepath.Glob(filepath.Join(protoDir, "*.proto"))
		if err != nil {
			return fmt.Errorf("finding proto files in %s: %w", protoDir, err)
		}
		if len(protos) == 0 {
			continue
		}
		generatedAny = true

		if err := os.RemoveAll(outDir); err != nil {
			return fmt.Errorf("clean openapi output dir %s: %w", outDir, err)
		}
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("create openapi output dir %s: %w", outDir, err)
		}

		args := []string{
			fmt.Sprintf("--proto_path=%s", protoDir),
			fmt.Sprintf("--proto_path=%s", protoRoot),
			"--proto_path=third_party",
			"--openapiv2_out", outDir,
			"--openapiv2_opt", "logtostderr=true",
			"--openapiv2_opt", "disable_default_errors=true",
			"--openapiv2_opt", "allow_merge=true",
			"--openapiv2_opt", "merge_file_name=" + module,
			"--openapiv2_opt", "output_format=yaml",
		}
		for _, p := range protos {
			args = append(args, filepath.Base(p))
		}

		fmt.Printf("  %s -> %s\n", module, outDir)
		if err := sh.RunWith(env, "protoc", args...); err != nil {
			return fmt.Errorf("protoc openapi for %s: %w", module, err)
		}
	}

	if !generatedAny {
		fmt.Printf("  No proto files found under %s/*/*.proto\n", protoRoot)
	}
	return nil
}

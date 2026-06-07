package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	exitCode := 0

	if err := test("go build", func() error {
		cmd := exec.Command("go", "build", "-o", os.DevNull, ".")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build failed: %w\n%s", err, out)
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		exitCode = 1
	}

	if err := test("go vet", func() error {
		cmd := exec.Command("go", "vet", "./...")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("vet failed: %w\n%s", err, out)
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		exitCode = 1
	}

	if err := test("unit tests", func() error {
		cmd := exec.Command("go", "test", "./...")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("tests failed: %w\n%s", err, out)
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		exitCode = 1
	}

	if exitCode == 0 {
		fmt.Println("\nAll smoke tests passed!")
	}
	os.Exit(exitCode)
}

func test(name string, fn func() error) error {
	fmt.Printf("=== %s ===\n", strings.ToUpper(name))
	if err := fn(); err != nil {
		return err
	}
	fmt.Printf("✓ %s passed\n\n", name)
	return nil
}

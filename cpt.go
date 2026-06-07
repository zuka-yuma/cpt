package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const template = `#include <iostream>
using namespace std;

int main() {
	return 0;
}
`

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		cmdNew(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: cpt <command> [args]")
	fmt.Fprintln(os.Stderr, "  new <filename>         create a new .cpp from template")
	fmt.Fprintln(os.Stderr, "  run <filename> [args]  compile and run without leaving a binary")
}

func cmdNew(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt new <filename>")
		os.Exit(1)
	}

	filename := args[0]
	if filepath.Ext(filename) != ".cpp" {
		filename += ".cpp"
	}

	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(os.Stderr, "error: %s already exists\n", filename)
		os.Exit(1)
	}

	if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created %s\n", filename)
}

func cmdRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt run <filename> [args]")
		os.Exit(1)
	}

	srcFile := args[0]
	progArgs := args[1:]

	binPath, cleanup, err := compile(srcFile)
	if err != nil {
		os.Exit(1)
	}
	defer cleanup()

	run := exec.Command(binPath, progArgs...)
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		os.Exit(1)
	}
}

func compile(src string) (bin string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp("", "cpt-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup = func() { os.RemoveAll(tmp) }
	bin = filepath.Join(tmp, "a.out")

	cxx := getenvOr("CPT_CXX", "g++-15")
	flags := strings.Fields(getenvOr("CPT_CXXFLAGS", "-std=gnu++23 -O2 -Wall"))
	args := append(flags, "-o", bin, src)

	cmd := exec.Command(cxx, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("compile failed: %w", err)
	}
	return bin, cleanup, nil
}

func getenvOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

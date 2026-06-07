package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		fmt.Fprintf(os.Stderr, "error: f\ni, errled to write file: %v\n", err)
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
	
	tmpDir, err := os.MkdirTemp("", "cpt-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)
	
	binPath := filepath.Join(tmpDir, "a.out")
	
	compile := exec.Command("g++", "-std=c++17", "-O2", "-o", binPath, srcFile)
	compile.Stdout = os.Stdout
	compile.Stderr = os.Stderr
	if err := compile.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "compile failed: %v\n", err)
		os.Exit(1)
	}

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
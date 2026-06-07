package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
)

var version = "dev"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

const template = `#include <iostream>
using namespace std;

// @snippet:global

int main() {
	// @snippet:local
	return 0;
}
`

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "-h", "--help":
		usage()
		os.Exit(0)
	case "-v", "--version":
		fmt.Println("cpt", version)
		os.Exit(0)
	case "new":
		cmdNew(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	case "ac", "atcoder":
		cmdAC(os.Args[2:])
	case "snippet", "sn":
		cmdSnippet(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: cpt <command> [args]")
	fmt.Fprintln(os.Stderr, "  new [filename]                   create a new .cpp from template (default: main.cpp)")
	fmt.Fprintln(os.Stderr, "  run [-i file] [-a arg]... <filename>  compile and run without leaving a binary")
	fmt.Fprintln(os.Stderr, "  ac test                          compile and run tests with oj")
	fmt.Fprintln(os.Stderr, "  snippet list                     list snippets")
	fmt.Fprintln(os.Stderr, "  snippet add [-scope] <name>      create a new snippet")
	fmt.Fprintln(os.Stderr, "  snippet show <name>              show snippet")
	fmt.Fprintln(os.Stderr, "  snippet edit <name>              edit snippet")
	fmt.Fprintln(os.Stderr, "  snippet insert <name> [file]     insert snippet into file")
	fmt.Fprintln(os.Stderr, "  snippet delete [-y] <name>       delete snippet")
}

func cmdNew(args []string) {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(os.Stderr, "usage: cpt new [filename]")
		os.Exit(0)
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt new [filename]")
		os.Exit(1)
	}

	filename := "main"
	if len(args) == 1 {
		filename = args[0]
	}
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

type arrayFlag []string

func (f *arrayFlag) String() string { return strings.Join(*f, " ") }
func (f *arrayFlag) Set(v string) error { *f = append(*f, v); return nil }

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: cpt run [-i file] [-a arg]... <filename>") }
	input := fs.String("i", "", "input file")
	var progArgs arrayFlag
	fs.Var(&progArgs, "a", "program argument (repeatable)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt run [-i file] [-a arg]... <filename>")
		os.Exit(1)
	}

	srcFile := fs.Arg(0)

	binPath, cleanup, err := compile(srcFile)
	if err != nil {
		os.Exit(1)
	}
	defer cleanup()

	stdin := os.Stdin
	if *input != "" {
		f, err := os.Open(*input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer f.Close()
		stdin = f
	}

	run := exec.Command(binPath, progArgs...)
	run.Stdin = stdin
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

func cmdAC(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "usage: cpt ac test")
		if len(args) >= 1 {
			os.Exit(0)
		}
		os.Exit(1)
	}
	switch args[0] {
	case "test":
		acTest(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "unknown: ac "+args[0])
		os.Exit(1)
	}
}

func runTests(src, testDir string) error {
	bin, cleanup, err := compile(src)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := exec.Command("oj", "test", "-c", bin, "-d", testDir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func acTest(args []string) {
	fs := flag.NewFlagSet("ac test", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: cpt ac test [-src file] [-d dir]") }
	src := fs.String("src", "main.cpp", "source file")
	dir := fs.String("d", "tests", "testcase dir")
	fs.Parse(args)
	if err := runTests(*src, *dir); err != nil {
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

func cmdSnippet(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet {list|add|show|edit|insert|delete}")
		if len(args) >= 1 {
			os.Exit(0)
		}
		os.Exit(1)
	}
	switch args[0] {
	case "list", "ls":
		snList()
	case "add":
		snAddCmd(args[1:])
	case "show", "cat":
		snShowCmd(args[1:])
	case "edit":
		snEditCmd(args[1:])
	case "insert", "i":
		snInsertCmd(args[1:])
	case "remove", "rm", "delete", "d":
		snDeleteCmd(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "unknown: snippet "+args[0])
		os.Exit(1)
	}
}

func snList() {
	entries, err := os.ReadDir(snippetDir())
	if err != nil {
		fmt.Println("(no snippets)")
		return
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".cpp" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".cpp")
		if s, err := loadSnippet(name); err == nil {
			fmt.Printf("%-18s [%-6s] %s\n", s.Name, s.Scope, s.Desc)
		}
	}
}

type snippet struct{ Name, Scope, Desc, Body string }

func loadSnippet(name string) (*snippet, error) {
	data, err := os.ReadFile(filepath.Join(snippetDir(), name+".cpp"))
	if err != nil {
		return nil, err
	}
	s := &snippet{Name: name, Scope: "global"}
	var body []string
	inHeader := true
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if inHeader && strings.HasPrefix(t, "//") {
			meta := strings.TrimSpace(strings.TrimPrefix(t, "//"))
			if k, v, ok := strings.Cut(meta, ":"); ok {
				switch strings.TrimSpace(k) {
				case "scope":
					s.Scope = strings.TrimSpace(v)
					continue
				case "desc":
					s.Desc = strings.TrimSpace(v)
					continue
				}
			}
		}
		inHeader = false
		body = append(body, line)
	}
	s.Body = strings.TrimRight(strings.Join(body, "\n"), "\n")
	return s, nil
}

func configHome() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return x
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}
func snippetDir() string { return filepath.Join(configHome(), "cpt", "snippets") }

func snAdd(name, scope string) {
	os.MkdirAll(snippetDir(), 0o755)
	path := filepath.Join(snippetDir(), name+".cpp")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(os.Stderr, "already exists: "+name)
		os.Exit(1)
	}
	skeleton := fmt.Sprintf("// scope: %s\n// desc: \n\n", scope)
	os.WriteFile(path, []byte(skeleton), 0o644)
	openEditor(path)
}

func openEditor(path string) error {
	ed := getenvOr("EDITOR", "vi")
	cmd := exec.Command("sh", "-c", ed+" "+path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func snInsert(name, target string, scopeOverride string) {
	s, err := loadSnippet(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if scopeOverride != "" {
		s.Scope = scopeOverride
	}

	data, err := os.ReadFile(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	content := string(data)

	guardOpen := "// >>> snippet:" + name
	if strings.Contains(content, guardOpen) {
		fmt.Printf("skip: %q is already inserted\n", name)
		return
	}

	marker := "// @snippet:" + s.Scope
	lines := strings.Split(content, "\n")
	idx := -1
	for i, l := range lines {
		if strings.Contains(l, marker) {
			idx = i
			break
		}
	}
	if idx < 0 {
		fmt.Fprintf(os.Stderr, "marker %q not found in %s\n", marker, target)
		os.Exit(1)
	}

	block := []string{guardOpen, s.Body, "// <<< snippet:" + name}
	out := append([]string{}, lines[:idx+1]...)
	out = append(out, block...)
	out = append(out, lines[idx+1:]...)

	os.WriteFile(target, []byte(strings.Join(out, "\n")), 0o644)
	fmt.Printf("inserted %q at %s\n", name, marker)
}

func snAddCmd(args []string) {
	fs := flag.NewFlagSet("snippet add", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: cpt snippet add [-scope global|local] <name>") }
	scope := fs.String("scope", "global", "global|local")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet add [-scope global|local] <name>")
		os.Exit(1)
	}
	snAdd(fs.Arg(0), *scope)
}

func snShowCmd(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet show <name>")
		if len(args) >= 1 {
			os.Exit(0)
		}
		os.Exit(1)
	}
	s, err := loadSnippet(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("// name : %s\n// scope: %s\n// desc : %s\n\n%s\n",
		s.Name, s.Scope, s.Desc, s.Body)
}

func snEditCmd(args []string) {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet edit <name>")
		if len(args) >= 1 {
			os.Exit(0)
		}
		os.Exit(1)
	}
	path := filepath.Join(snippetDir(), args[0]+".cpp")
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintln(os.Stderr, "no such snippet: "+args[0])
		os.Exit(1)
	}
	openEditor(path)
}

func snDeleteCmd(args []string) {
	fs := flag.NewFlagSet("snippet delete", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: cpt snippet delete [-y] <name>") }
	yes := fs.Bool("y", false, "skip confirmation")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet delete [-y] <name>")
		os.Exit(1)
	}
	name := fs.Arg(0)
	path := filepath.Join(snippetDir(), name+".cpp")
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintln(os.Stderr, "no such snippet: "+name)
		os.Exit(1)
	}
	if !*yes {
		fmt.Printf("delete %q? [y/N]: ", name)
		var ans string
		fmt.Scanln(&ans)
		if ans != "y" && ans != "Y" {
			fmt.Println("cancelled")
			return
		}
	}
	if err := os.Remove(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("deleted %q\n", name)
}

func snInsertCmd(args []string) {
	fs := flag.NewFlagSet("snippet insert", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet insert [-scope global|local] <name> [target.cpp]")
	}
	scope := fs.String("scope", "", "override snippet scope (global|local)")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cpt snippet insert [-scope ...] <name> [target.cpp]")
		os.Exit(1)
	}
	name := fs.Arg(0)
	target := "main.cpp"
	if fs.NArg() >= 2 {
		target = fs.Arg(1)
	}
	snInsert(name, target, *scope)
}

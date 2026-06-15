package main

import (
	"fmt"
	"os"
)

func mustHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		die("Cannot determine home directory: %v", err)
	}
	return home
}

var verbose bool

func red(s string) string    { return "\033[0;31m" + s + "\033[0m" }
func green(s string) string  { return "\033[0;32m" + s + "\033[0m" }
func yellow(s string) string { return "\033[0;33m" + s + "\033[0m" }

func bold(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[1m"+format+"\033[0m\n", args...)
}

func info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, green("●")+" "+format+"\n", args...)
}

func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, yellow("▲")+" "+format+"\n", args...)
}

func errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, red("✖")+" "+format+"\n", args...)
}

func debug(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "\033[1m⊙\033[0m "+format+"\n", args...)
	}
}

func die(format string, args ...any) {
	errorf(format, args...)
	os.Exit(1)
}

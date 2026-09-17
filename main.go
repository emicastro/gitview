package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

const version = "0.1.0"

type config struct {
	user         string
	top          int
	json         bool
	includeForks bool
	fresh        bool
	version      bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if err := ctx.Err(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	cfg, err := parseArgs(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 2
	}

	if cfg.version {
		fmt.Fprintf(stdout, "gitview %s\n", version)
		return 0
	}

	if getenv("GITHUB_TOKEN") == "" {
		fmt.Fprintln(stderr, "GITHUB_TOKEN required")
		return 2
	}

	fmt.Fprintf(
		stdout,
		"user:%s top:%d json:%v forks:%v fresh:%v\n",
		cfg.user, cfg.top, cfg.json, cfg.includeForks, cfg.fresh,
	)
	return 0
}

func parseArgs(args []string, stderr io.Writer) (config, error) {
	// create a FlagSet here; do not use the global flag set
	fs := flag.NewFlagSet("gitview", flag.ContinueOnError)
	fs.SetOutput(stderr)

	top := fs.Int("top", 8, "how many languages to show")
	jsonOut := fs.Bool("json", false, "print JSON instead of TUI")
	includeForks := fs.Bool("forks", false, "include forked repos")
	fresh := fs.Bool("fresh", false, "ignore cache and refetch")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	cfg := config{
		top:          *top,
		json:         *jsonOut,
		includeForks: *includeForks,
		fresh:        *fresh,
		version:      *showVersion,
	}

	if fs.NArg() >= 1 {
		cfg.user = fs.Arg(0)
	}

	if cfg.version {
		return cfg, nil
	}

	if cfg.user == "" {
		return config{}, errors.New("user required")
	}

	if cfg.top < 1 {
		return config{}, errors.New("top must be >= 1")
	}

	return cfg, nil
}

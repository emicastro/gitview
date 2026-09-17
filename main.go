package main

import (
	"flag"
	"fmt"
	"os"
)

type config struct {
	user         string
	top          int
	json         bool
	includeForks bool
}

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fmt.Printf(
		"user:%s top:%d, json:%v, forks:%v\n",
		cfg.user, cfg.top, cfg.json, cfg.includeForks,
	)
}

func parseArgs(args []string) (config, error) {
	// aca adentro creas un FlagSet, no uses el flag global
	fs := flag.NewFlagSet("gitview", flag.ContinueOnError)

	top := fs.Int("top", 8, "how many languages to show")
	jsonOut := fs.Bool("json", false, "print JSON instead of TUI")
	includeForks := fs.Bool("forks", false, "include forked repos")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "user required\n")
		os.Exit(2)
	}

	cfg := config{
		user:         fs.Arg(0),
		top:          *top,
		json:         *jsonOut,
		includeForks: *includeForks,
	}

	if cfg.top < 1 {
		fmt.Fprintf(os.Stderr, "top must be >= 1\n")
		os.Exit(2)
	}

	return cfg, nil
}

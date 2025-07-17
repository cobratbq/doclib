package main

import (
	"flag"

	"github.com/cobratbq/doclib/internal/repo"
	"github.com/cobratbq/goutils/std/log"
)

type config struct {
	location string
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.location, "repo", ".", "Location of the repository root directory.")
	flag.Parse()
	return cfg
}

func cmdCheck(cfg *config) {
	repo := repo.OpenRepo(cfg.location)
	log.Infoln("result:", repo.Check())
}

func main() {
	// TODO consider using spf13/cobra for command-line commands/parameters/shell-completions/...
	cfg := parseFlags()

	if flag.NArg() < 1 {
		return
	}

	switch flag.Arg(0) {
	case "check":
		cmdCheck(&cfg)
	default:
		flag.PrintDefaults()
	}
}

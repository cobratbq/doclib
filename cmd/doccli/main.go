package main

import (
	"flag"

	"github.com/cobratbq/doclib/internal/repo"
	"github.com/cobratbq/goutils/std/log"
)

type config struct {
	args     []string
	location string
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.location, "repo", ".", "Location of the repository root directory.")
	flag.Parse()
	cfg.args = flag.Args()
	return cfg
}

func cmdCheck(cfg *config) {
	repo := repo.OpenRepo(cfg.location)
	log.Infoln("Finished:", repo.Check())
}

// TODO eventually, may need to add lock if both UI and cli are used at same time, especially when performing checks/fixes.
func main() {
	// TODO consider using spf13/cobra for command-line commands/parameters/shell-completions/...
	cfg := parseFlags()

	if len(cfg.args) < 1 {
		flag.PrintDefaults()
		return
	}

	switch cfg.args[0] {
	case "check":
		cmdCheck(&cfg)
	default:
		flag.PrintDefaults()
	}
}

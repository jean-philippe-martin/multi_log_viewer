package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jpmartin/multi_log_viewer/internal/config"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
	"github.com/jpmartin/multi_log_viewer/internal/theme"
	"github.com/jpmartin/multi_log_viewer/internal/tui"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "./mlv.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	eng, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer eng.Close()

	th, err := theme.Load(cfg.Theme, cfg.ProjectRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := tui.Run(eng, th); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

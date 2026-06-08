package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: loggen PATH [--interval SECONDS] [--prefix TEXT] [--count N] [--timestamp-format NAME]")
		os.Exit(2)
	}
	path := os.Args[1]
	flagSet := flag.NewFlagSet("loggen", flag.ExitOnError)
	interval := flagSet.Float64("interval", 1.0, "seconds between lines")
	prefix := flagSet.String("prefix", "", "text after timestamp")
	count := flagSet.Int("count", 0, "number of lines (0 = forever)")
	format := flagSet.String("timestamp-format", "hms-ms", "timestamp format name")
	_ = flagSet.Parse(os.Args[2:])

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	n := 0
	ticker := time.NewTicker(time.Duration(*interval * float64(time.Second)))
	defer ticker.Stop()

	writeLine := func() {
		n++
		ts, err := formatTimestamp(*format)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		line := fmt.Sprintf("%s%s line %d\n", ts, *prefix, n)
		if _, err := f.WriteString(line); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	writeLine()
	if *count == 1 {
		return
	}

	remaining := -1 // count 0 = run until interrupted
	if *count > 1 {
		remaining = *count - 1
	}
	for {
		select {
		case <-sig:
			return
		case <-ticker.C:
			writeLine()
			if remaining > 0 {
				remaining--
				if remaining == 0 {
					return
				}
			}
		}
	}
}

func formatTimestamp(name string) (string, error) {
	now := time.Now()
	switch name {
	case "hms-ms", "utc-hms-ms", "YYYY-MM-DD HH:mm:ss.SSS":
		return now.UTC().Format("2006-01-02 15:04:05.000 ") , nil
	case "hms", "YYYY-MM-DD HH:mm:ss":
		return now.UTC().Format("2006-01-02 15:04:05 ") , nil
	case "iso8601", "TIMESTAMP_ISO8601":
		_, off := now.Zone()
		sign := "+"
		if off < 0 {
			sign = "-"
			off = -off
		}
		h := off / 3600
		m := (off % 3600) / 60
		return fmt.Sprintf("%s %s%02d:%02d ", now.Format("2006-01-02 15:04:05.000"), sign, h, m), nil
	case "iso8601-t":
		return now.Format(time.RFC3339Nano) + " ", nil
	default:
		return "", fmt.Errorf("unknown timestamp-format %q", name)
	}
}

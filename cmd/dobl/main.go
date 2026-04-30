package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lusingander/dobl"
)

const usage = "usage: dobl parse [--compact] [file]\n       dobl summary [--compact] [--events] [file]"

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) < 2 {
		return errors.New(usage)
	}

	switch args[1] {
	case "parse":
		return runParse(args[2:], stdin, stdout)
	case "summary":
		return runSummary(args[2:], stdin, stdout)
	default:
		return errors.New(usage)
	}
}

func runParse(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("parse", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	compact := flags.Bool("compact", false, "emit compact JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	log, err := parseInput(flags.Args(), stdin)
	if err != nil {
		return err
	}

	return encodeJSON(stdout, log, *compact)
}

func runSummary(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("summary", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	compact := flags.Bool("compact", false, "emit compact JSON")
	includeEvents := flags.Bool("events", false, "include step events")
	if err := flags.Parse(args); err != nil {
		return err
	}

	log, err := parseInput(flags.Args(), stdin)
	if err != nil {
		return err
	}

	steps := log.Steps()
	if !*includeEvents {
		for i := range steps {
			steps[i].Events = nil
		}
	}

	return encodeJSON(stdout, steps, *compact)
}

func parseInput(args []string, stdin io.Reader) (*dobl.BuildLog, error) {
	if len(args) > 1 {
		return nil, errors.New(usage)
	}

	var input io.Reader = stdin
	var file *os.File
	if len(args) == 1 && args[0] != "-" {
		var err error
		file, err = os.Open(args[0])
		if err != nil {
			return nil, err
		}
		defer file.Close()
		input = file
	}

	return dobl.Parse(input)
}

func encodeJSON(stdout io.Writer, output any, compact bool) error {
	encoder := json.NewEncoder(stdout)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(output)
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/lusingander/dobl"
)

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type cli struct {
	Parse   parseCmd   `cmd:"" help:"Parse a plain build log into event JSON."`
	Summary summaryCmd `cmd:"" help:"Summarize a plain build log by BuildKit step."`
}

type parseCmd struct {
	Compact bool   `help:"Emit compact JSON."`
	File    string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

type summaryCmd struct {
	Compact bool   `help:"Emit compact JSON."`
	Events  bool   `help:"Include source events in each step."`
	File    string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

type runContext struct {
	stdin  io.Reader
	stdout io.Writer
}

func (c *parseCmd) Run(ctx *runContext) error {
	log, err := parseInput(c.File, ctx.stdin)
	if err != nil {
		return err
	}

	return encodeJSON(ctx.stdout, log, c.Compact)
}

func (c *summaryCmd) Run(ctx *runContext) error {
	log, err := parseInput(c.File, ctx.stdin)
	if err != nil {
		return err
	}

	steps := log.Steps()
	if !c.Events {
		for i := range steps {
			steps[i].Events = nil
		}
	}

	return encodeJSON(ctx.stdout, steps, c.Compact)
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	var app cli
	parser, err := kong.New(
		&app,
		kong.Name("dobl"),
		kong.Description("Parse and summarize plain Docker BuildKit build logs."),
		kong.Writers(io.Discard, io.Discard),
		kong.Bind(&runContext{stdin: stdin, stdout: stdout}),
	)
	if err != nil {
		return err
	}

	ctx, err := parser.Parse(args[1:])
	if err != nil {
		return err
	}

	return ctx.Run()
}

func parseInput(fileName string, stdin io.Reader) (*dobl.BuildLog, error) {
	if fileName == "-" {
		fileName = ""
	}

	var input io.Reader = stdin
	var file *os.File
	if fileName != "" {
		var err error
		file, err = os.Open(fileName)
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

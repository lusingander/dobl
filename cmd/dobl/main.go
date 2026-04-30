package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
	"github.com/lusingander/dobl"
)

const (
	formatJSON  = "json"
	formatTable = "table"
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
	Format  string `default:"json" enum:"json" help:"Output format."`
	File    string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

type summaryCmd struct {
	Compact bool   `help:"Emit compact JSON."`
	Events  bool   `help:"Include source events in each step."`
	Failed  bool   `help:"Only include failed steps."`
	Format  string `default:"json" enum:"json,table" help:"Output format."`
	Status  string `help:"Only include steps with this status."`
	File    string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

type runContext struct {
	stdin  io.Reader
	stdout io.Writer
}

func (c *parseCmd) Run(ctx *runContext) error {
	if c.Format != formatJSON {
		return fmt.Errorf("parse format %q is not supported", c.Format)
	}

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
	if c.Failed && c.Status != "" {
		return fmt.Errorf("--failed and --status cannot be used together")
	}
	if c.Failed {
		steps = filterFailedSteps(steps)
	}
	if c.Status != "" {
		status := dobl.EventStatus(c.Status)
		if !isKnownStatus(status) {
			return fmt.Errorf("unknown status %q", c.Status)
		}
		steps = filterStepsByStatus(steps, status)
	}
	if !c.Events {
		for i := range steps {
			steps[i].Events = nil
		}
	}

	switch c.Format {
	case formatJSON:
		return encodeJSON(ctx.stdout, steps, c.Compact)
	case formatTable:
		if c.Events {
			return fmt.Errorf("--events is only supported with --format=json")
		}
		if c.Compact {
			return fmt.Errorf("--compact is only supported with --format=json")
		}
		return encodeSummaryTable(ctx.stdout, steps)
	default:
		return fmt.Errorf("summary format %q is not supported", c.Format)
	}
}

func filterFailedSteps(steps []dobl.Step) []dobl.Step {
	filtered := make([]dobl.Step, 0, len(steps))
	for _, step := range steps {
		if isFailedStatus(step.Status) {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func isFailedStatus(status dobl.EventStatus) bool {
	return status == dobl.EventStatusError || status == dobl.EventStatusCanceled
}

func filterStepsByStatus(steps []dobl.Step, status dobl.EventStatus) []dobl.Step {
	filtered := make([]dobl.Step, 0, len(steps))
	for _, step := range steps {
		if step.Status == status {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func isKnownStatus(status dobl.EventStatus) bool {
	switch status {
	case dobl.EventStatusDone,
		dobl.EventStatusCached,
		dobl.EventStatusError,
		dobl.EventStatusCanceled,
		dobl.EventStatusWarning,
		dobl.EventStatusProgress:
		return true
	default:
		return false
	}
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
			return nil, fmt.Errorf("open %s: %w", fileName, err)
		}
		defer file.Close()
		input = file
	}

	log, err := dobl.Parse(input)
	if err != nil {
		source := "stdin"
		if fileName != "" {
			source = fileName
		}
		return nil, fmt.Errorf("parse %s: %w", source, err)
	}
	return log, nil
}

func encodeJSON(stdout io.Writer, output any, compact bool) error {
	encoder := json.NewEncoder(stdout)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(output)
}

func encodeSummaryTable(stdout io.Writer, steps []dobl.Step) error {
	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tSTATUS\tDURATION\tSTEP\tINSTRUCTION\tOUTPUTS\tPROGRESS\tERROR"); err != nil {
		return err
	}
	for _, step := range steps {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			step.ID,
			step.Status,
			step.Duration,
			formatStepIndex(step),
			step.Instruction,
			step.OutputCount,
			step.ProgressCount,
			step.ErrorDetail,
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func formatStepIndex(step dobl.Step) string {
	if step.Index == 0 || step.Total == 0 {
		return ""
	}
	index := fmt.Sprintf("%d/%d", step.Index, step.Total)
	if step.Stage == "" {
		return index
	}
	return strings.Join([]string{step.Stage, index}, " ")
}

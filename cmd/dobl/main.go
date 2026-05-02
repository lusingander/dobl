package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
	"github.com/lusingander/dobl"
)

const (
	formatJSON  = "json"
	formatTable = "table"

	tableErrorWidth = 96
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
	Compact     bool   `help:"Emit compact JSON."`
	Events      bool   `help:"Include source events in each step."`
	Failed      bool   `help:"Only include failed steps."`
	Warnings    bool   `help:"Only include warning steps."`
	Format      string `default:"json" enum:"json,table" help:"Output format."`
	Status      string `placeholder:"STATUS" help:"Only include steps with this status. One of: DONE, CACHED, ERROR, CANCELED, WARNING, PROGRESS."`
	Stage       string `placeholder:"STAGE" help:"Only include Dockerfile steps from this stage."`
	Instruction string `placeholder:"INSTRUCTION" help:"Only include Dockerfile steps with this instruction."`
	Step        string `placeholder:"ID" help:"Only include a specific BuildKit step ID, such as #3 or 3."`
	Wide        bool   `help:"Do not truncate table error details."`
	File        string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

func (c *parseCmd) Help() string {
	return `Examples:
  dobl parse build.log
  docker buildx build --progress=plain . 2>&1 | dobl parse --compact`
}

func (c *summaryCmd) Help() string {
	return `Examples:
  dobl summary build.log
  dobl summary --format table build.log
  dobl summary --format table --wide build.log
  dobl summary --failed --format table build.log
  dobl summary --warnings --format table build.log
  dobl summary --status ERROR build.log
  dobl summary --stage build --instruction RUN build.log
  dobl summary --step '#3' build.log`
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
	if err := c.validate(); err != nil {
		return err
	}

	log, err := parseInput(c.File, ctx.stdin)
	if err != nil {
		return err
	}
	steps := log.Steps()
	if c.Failed {
		steps = filterFailedSteps(steps)
	}
	if c.Warnings {
		steps = filterWarningSteps(steps)
	}
	if c.Status != "" {
		status := dobl.EventStatus(c.Status)
		steps = filterStepsByStatus(steps, status)
	}
	if c.Stage != "" {
		steps = filterStepsByStage(steps, c.Stage)
	}
	if c.Instruction != "" {
		steps = filterStepsByInstruction(steps, c.Instruction)
	}
	if c.Step != "" {
		steps = filterStepsByID(steps, normalizeStepID(c.Step))
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
		return encodeSummaryTable(ctx.stdout, steps, c.Wide)
	default:
		return fmt.Errorf("summary format %q is not supported", c.Format)
	}
}

func (c *summaryCmd) validate() error {
	if c.Failed && c.Status != "" {
		return fmt.Errorf("--failed and --status cannot be used together")
	}
	if c.Warnings && c.Status != "" {
		return fmt.Errorf("--warnings and --status cannot be used together")
	}
	if c.Failed && c.Warnings {
		return fmt.Errorf("--failed and --warnings cannot be used together")
	}
	if c.Status != "" && !isKnownStatus(dobl.EventStatus(c.Status)) {
		return fmt.Errorf("unknown status %q", c.Status)
	}
	if c.Step != "" && !isValidStepID(c.Step) {
		return fmt.Errorf("invalid step id %q", c.Step)
	}
	if c.Format == formatTable {
		if c.Events {
			return fmt.Errorf("--events is only supported with --format=json")
		}
		if c.Compact {
			return fmt.Errorf("--compact is only supported with --format=json")
		}
	} else if c.Wide {
		return fmt.Errorf("--wide is only supported with --format=table")
	}
	return nil
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

func filterWarningSteps(steps []dobl.Step) []dobl.Step {
	filtered := make([]dobl.Step, 0, len(steps))
	for _, step := range steps {
		if step.Status == dobl.EventStatusWarning || step.WarningCount > 0 {
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

func filterStepsByStage(steps []dobl.Step, stage string) []dobl.Step {
	filtered := make([]dobl.Step, 0, len(steps))
	for _, step := range steps {
		if step.Stage == stage {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func filterStepsByInstruction(steps []dobl.Step, instruction string) []dobl.Step {
	filtered := make([]dobl.Step, 0, len(steps))
	for _, step := range steps {
		if strings.EqualFold(step.Instruction, instruction) {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func filterStepsByID(steps []dobl.Step, id string) []dobl.Step {
	filtered := make([]dobl.Step, 0, len(steps))
	for _, step := range steps {
		if step.ID == id {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func normalizeStepID(id string) string {
	if strings.HasPrefix(id, "#") {
		return id
	}
	return "#" + id
}

var stepIDRE = regexp.MustCompile(`^#?\d+$`)

func isValidStepID(id string) bool {
	return stepIDRE.MatchString(id)
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

type kongExit int

func run(args []string, stdin io.Reader, stdout io.Writer) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		exitCode, ok := recovered.(kongExit)
		if !ok {
			panic(recovered)
		}
		if exitCode != 0 {
			err = fmt.Errorf("exit %d", exitCode)
		}
	}()

	var app cli
	parser, err := kong.New(
		&app,
		kong.Name("dobl"),
		kong.Description("Parse and summarize plain Docker BuildKit build logs."),
		kong.Writers(stdout, io.Discard),
		kong.Exit(func(code int) {
			panic(kongExit(code))
		}),
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

func encodeSummaryTable(stdout io.Writer, steps []dobl.Step, wide bool) error {
	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tSTATUS\tDURATION\tSTEP\tINSTRUCTION\tNAME\tOUTPUTS\tPROGRESS\tERROR"); err != nil {
		return err
	}
	for _, step := range steps {
		errorDetail := step.ErrorDetail
		if !wide {
			errorDetail = truncateString(errorDetail, tableErrorWidth)
		}
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			step.ID,
			step.Status,
			step.Duration,
			formatStepIndex(step),
			step.Instruction,
			step.Name,
			step.OutputCount,
			step.ProgressCount,
			errorDetail,
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

func truncateString(value string, maxWidth int) string {
	if maxWidth <= 0 || len(value) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return value[:maxWidth]
	}
	return value[:maxWidth-3] + "..."
}

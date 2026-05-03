package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/lusingander/dobl"
	dobltui "github.com/lusingander/dobl/internal/tui"
)

const (
	formatJSON  = "json"
	formatTable = "table"
	formatText  = "text"
)

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
	Format      string `default:"json" enum:"json,table,text" help:"Output format."`
	Status      string `placeholder:"STATUS" help:"Only include steps with this status. One of: DONE, CACHED, ERROR, CANCELED, WARNING, PROGRESS."`
	Stage       string `placeholder:"STAGE" help:"Only include Dockerfile steps from this stage."`
	Instruction string `placeholder:"INSTRUCTION" help:"Only include Dockerfile steps with this instruction."`
	Step        string `placeholder:"ID" help:"Only include a specific BuildKit step ID, such as #3 or 3."`
	Sort        string `default:"order" placeholder:"KEY" enum:"order,duration,status,outputs,warnings" help:"Sort steps. One of: order, duration, status, outputs, warnings."`
	Top         string `placeholder:"KEY" help:"Include a top section in text output. One of: slow, warnings, outputs."`
	Details     string `placeholder:"MODE" help:"Set text detail section mode. One of: problems, all, none."`
	Wide        bool   `help:"Do not truncate table or text diagnostics."`
	File        string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

type reportCmd struct {
	Output string `short:"o" placeholder:"FILE" help:"Write the report to a file instead of stdout."`
	Title  string `placeholder:"TITLE" help:"Set the report title shown in the HTML viewer."`
	File   string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
}

type tuiCmd struct {
	Summary string `placeholder:"FILE" help:"Read summary JSON from this file instead of parsing a plain build log. Use '-' for stdin."`
	Filter  string `default:"all" enum:"all,problems,warnings,failed" placeholder:"MODE" help:"Initial filter. One of: all, problems, warnings, failed."`
	Search  string `placeholder:"QUERY" help:"Initial search query."`
	File    string `arg:"" optional:"" help:"Build log file. Reads stdin when omitted or set to '-'."`
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
  dobl summary --format text build.log
  dobl summary --format table --wide build.log
  dobl summary --format text --wide build.log
  dobl summary --failed --format table build.log
  dobl summary --warnings --format table build.log
  dobl summary --status ERROR build.log
  dobl summary --sort status --format text build.log
  dobl summary --top slow --format text build.log
  dobl summary --details all --format text build.log
  dobl summary --stage build --instruction RUN build.log
  dobl summary --step '#3' build.log`
}

func (c *reportCmd) Help() string {
	return `Examples:
  dobl report build.log > report.html
  dobl report --output report.html build.log
  dobl report --title "CI build" --output report.html build.log
  docker buildx build --progress=plain . 2>&1 | dobl report > report.html`
}

func (c *tuiCmd) Help() string {
	return `Examples:
  dobl tui build.log
  dobl tui --filter problems build.log
  dobl tui --search "missing dependency" build.log
  dobl tui --summary summary.json
  docker buildx build --progress=plain . 2>&1 | dobl tui`
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
	if c.Sort != "" {
		sortSteps(steps, c.Sort)
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
	case formatText:
		return encodeSummaryText(ctx.stdout, steps, textSummaryOptions{
			Source:  inputSource(c.File),
			Top:     c.Top,
			Details: c.Details,
			Wide:    c.Wide,
		})
	default:
		return fmt.Errorf("summary format %q is not supported", c.Format)
	}
}

func (c *reportCmd) Run(ctx *runContext) error {
	log, err := parseInput(c.File, ctx.stdin)
	if err != nil {
		return err
	}

	steps := log.Steps()
	for i := range steps {
		steps[i].Events = nil
	}

	source := c.File
	if c.Output == "" {
		return encodeHTMLReport(ctx.stdout, steps, inputSource(source), c.Title)
	}

	var output bytes.Buffer
	if err := encodeHTMLReport(&output, steps, inputSource(source), c.Title); err != nil {
		return err
	}
	return os.WriteFile(c.Output, output.Bytes(), 0o644)
}

func (c *tuiCmd) Run(ctx *runContext) error {
	filter, err := dobltui.ParseFilterMode(c.Filter)
	if err != nil {
		return err
	}

	steps, source, err := loadTUISteps(c.File, c.Summary, ctx.stdin)
	if err != nil {
		return err
	}

	return dobltui.Run(steps, dobltui.Options{
		Source:        source,
		InitialFilter: filter,
		InitialSearch: c.Search,
		Output:        ctx.stdout,
	})
}

func loadTUISteps(fileName string, summaryFileName string, stdin io.Reader) ([]dobl.Step, string, error) {
	if summaryFileName != "" {
		if fileName != "" {
			return nil, "", fmt.Errorf("--summary and build log file cannot be used together")
		}
		steps, err := readSummarySteps(summaryFileName, stdin)
		if err != nil {
			return nil, "", err
		}
		stripStepEvents(steps)
		return steps, inputSource(summaryFileName), nil
	}

	log, err := parseInput(fileName, stdin)
	if err != nil {
		return nil, "", err
	}
	steps := log.Steps()
	stripStepEvents(steps)
	return steps, inputSource(fileName), nil
}

func readSummarySteps(fileName string, stdin io.Reader) ([]dobl.Step, error) {
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

	var steps []dobl.Step
	if err := json.NewDecoder(input).Decode(&steps); err != nil {
		source := "stdin"
		if fileName != "" {
			source = fileName
		}
		return nil, fmt.Errorf("parse summary %s: %w", source, err)
	}
	return steps, nil
}

func stripStepEvents(steps []dobl.Step) {
	for i := range steps {
		steps[i].Events = nil
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
	if c.Format != formatJSON {
		if c.Events {
			return fmt.Errorf("--events is only supported with --format=json")
		}
		if c.Compact {
			return fmt.Errorf("--compact is only supported with --format=json")
		}
	}
	if c.Wide && c.Format != formatTable && c.Format != formatText {
		return fmt.Errorf("--wide is only supported with --format=table or --format=text")
	}
	if c.Top != "" && !isKnownTop(c.Top) {
		return fmt.Errorf("unknown top key %q", c.Top)
	}
	if c.Top != "" && c.Format != formatText {
		return fmt.Errorf("--top is only supported with --format=text")
	}
	if c.Details != "" && !isKnownDetailsMode(c.Details) {
		return fmt.Errorf("unknown details mode %q", c.Details)
	}
	if c.Details != "" && c.Format != formatText {
		return fmt.Errorf("--details is only supported with --format=text")
	}
	return nil
}

func inputSource(fileName string) string {
	if fileName == "" || fileName == "-" {
		return "stdin"
	}
	return fileName
}

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/lusingander/dobl"
)

const tableErrorWidth = 96

func encodeJSON(stdout io.Writer, output any, compact bool) error {
	encoder := json.NewEncoder(stdout)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(output)
}

func encodeHTMLReport(stdout io.Writer, steps []dobl.Step, source string) error {
	var summary bytes.Buffer
	encoder := json.NewEncoder(&summary)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(steps); err != nil {
		return err
	}

	payload := fmt.Sprintf(
		`<script id="embedded-summary" type="application/json" data-source="%s">%s</script>`,
		html.EscapeString(source),
		escapeClosingScriptTag(strings.TrimSpace(summary.String())),
	)
	report := strings.Replace(viewerHTML, "  <script>\n", payload+"\n  <script>\n", 1)
	_, err := io.WriteString(stdout, report)
	return err
}

func escapeClosingScriptTag(value string) string {
	return strings.ReplaceAll(value, "</script", "<\\/script")
}

func encodeSummaryTable(stdout io.Writer, steps []dobl.Step, wide bool) error {
	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tSTATUS\tDURATION\tSTEP\tINSTRUCTION\tNAME\tOUTPUTS\tPROGRESS\tDIAGNOSTIC"); err != nil {
		return err
	}
	for _, step := range steps {
		diagnostic := stepDiagnostic(step)
		if !wide {
			diagnostic = truncateString(diagnostic, tableErrorWidth)
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
			diagnostic,
		); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func stepDiagnostic(step dobl.Step) string {
	if step.ErrorDetail != "" {
		return step.ErrorDetail
	}
	return step.WarningDetail
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

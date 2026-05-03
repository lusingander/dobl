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
const textWidth = 100

type summaryStats struct {
	Total         int
	Done          int
	Cached        int
	Warnings      int
	Errors        int
	Canceled      int
	Progress      int
	Outputs       int
	ProgressLines int
	Unknowns      int
}

func encodeJSON(stdout io.Writer, output any, compact bool) error {
	encoder := json.NewEncoder(stdout)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(output)
}

func encodeHTMLReport(stdout io.Writer, steps []dobl.Step, source string, title string) error {
	var summary bytes.Buffer
	encoder := json.NewEncoder(&summary)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(steps); err != nil {
		return err
	}

	payload := fmt.Sprintf(
		`<script id="embedded-summary" type="application/json" data-source="%s" data-title="%s">%s</script>`,
		html.EscapeString(source),
		html.EscapeString(title),
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

func encodeSummaryText(stdout io.Writer, steps []dobl.Step, source string) error {
	stats := collectSummaryStats(steps)
	problems := problemSteps(steps)

	if _, err := fmt.Fprintln(stdout, "Dobl Summary"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Source: %s\n", source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		stdout,
		"Steps: %d  Done: %d  Cached: %d  Warnings: %d  Errors: %d  Canceled: %d  Outputs: %d\n\n",
		stats.Total,
		stats.Done,
		stats.Cached,
		stats.Warnings,
		stats.Errors,
		stats.Canceled,
		stats.Outputs,
	); err != nil {
		return err
	}

	if err := writeTimelineText(stdout, steps); err != nil {
		return err
	}
	if err := writeProblemsText(stdout, problems); err != nil {
		return err
	}
	if err := writeStepsText(stdout, steps); err != nil {
		return err
	}
	return writeProblemDetailsText(stdout, problems)
}

func collectSummaryStats(steps []dobl.Step) summaryStats {
	stats := summaryStats{Total: len(steps)}
	for _, step := range steps {
		switch step.Status {
		case dobl.EventStatusDone:
			stats.Done++
		case dobl.EventStatusCached:
			stats.Cached++
		case dobl.EventStatusWarning:
			stats.Warnings++
		case dobl.EventStatusError:
			stats.Errors++
		case dobl.EventStatusCanceled:
			stats.Canceled++
		case dobl.EventStatusProgress:
			stats.Progress++
		}
		stats.Outputs += step.OutputCount
		stats.ProgressLines += step.ProgressCount
		stats.Unknowns += step.UnknownCount
	}
	return stats
}

func problemSteps(steps []dobl.Step) []dobl.Step {
	var problems []dobl.Step
	for _, step := range steps {
		if isProblemStep(step) {
			problems = append(problems, step)
		}
	}
	return problems
}

func isProblemStep(step dobl.Step) bool {
	return step.Status == dobl.EventStatusError ||
		step.Status == dobl.EventStatusCanceled ||
		step.Status == dobl.EventStatusWarning ||
		step.ErrorDetail != "" ||
		step.WarningDetail != ""
}

func writeTimelineText(stdout io.Writer, steps []dobl.Step) error {
	if _, err := fmt.Fprintln(stdout, "Timeline:"); err != nil {
		return err
	}
	if len(steps) == 0 {
		if _, err := fmt.Fprintln(stdout, "(none)"); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout)
		return err
	}

	var line strings.Builder
	for _, step := range steps {
		item := timelineItem(step)
		nextWidth := line.Len() + len(item)
		if line.Len() > 0 {
			nextWidth += 3
		}
		if line.Len() > 0 && nextWidth > textWidth {
			if _, err := fmt.Fprintln(stdout, line.String()); err != nil {
				return err
			}
			line.Reset()
		}
		if line.Len() > 0 {
			line.WriteString(" | ")
		}
		line.WriteString(item)
	}
	if line.Len() > 0 {
		if _, err := fmt.Fprintln(stdout, line.String()); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(stdout)
	return err
}

func timelineItem(step dobl.Step) string {
	parts := []string{step.ID, statusShort(step.Status), stepLabel(step)}
	if step.Duration != "" {
		parts = append(parts, step.Duration)
	}
	return strings.Join(parts, " ")
}

func statusShort(status dobl.EventStatus) string {
	switch status {
	case dobl.EventStatusDone:
		return "D"
	case dobl.EventStatusCached:
		return "C"
	case dobl.EventStatusError:
		return "E"
	case dobl.EventStatusCanceled:
		return "X"
	case dobl.EventStatusWarning:
		return "W"
	case dobl.EventStatusProgress:
		return "P"
	default:
		return "?"
	}
}

func writeProblemsText(stdout io.Writer, problems []dobl.Step) error {
	if _, err := fmt.Fprintln(stdout, "Problems:"); err != nil {
		return err
	}
	if len(problems) == 0 {
		if _, err := fmt.Fprintln(stdout, "(none)"); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout)
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, step := range problems {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\n",
			problemMarker(step),
			step.ID,
			statusText(step.Status),
			stepLabel(step),
			truncateString(stepDiagnostic(step), tableErrorWidth),
		); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout)
	return err
}

func problemMarker(step dobl.Step) string {
	switch step.Status {
	case dobl.EventStatusError:
		return "x"
	case dobl.EventStatusCanceled:
		return "-"
	case dobl.EventStatusWarning:
		return "!"
	default:
		if step.ErrorDetail != "" {
			return "x"
		}
		if step.WarningDetail != "" {
			return "!"
		}
		return "?"
	}
}

func writeStepsText(stdout io.Writer, steps []dobl.Step) error {
	if _, err := fmt.Fprintln(stdout, "Steps:"); err != nil {
		return err
	}
	if len(steps) == 0 {
		if _, err := fmt.Fprintln(stdout, "(none)"); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout)
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, step := range steps {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\n",
			step.ID,
			statusText(step.Status),
			step.Duration,
			stepLabel(step),
			step.DisplayName,
		); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout)
	return err
}

func writeProblemDetailsText(stdout io.Writer, problems []dobl.Step) error {
	if len(problems) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(stdout, "Problem Details:"); err != nil {
		return err
	}
	for i, step := range problems {
		if i > 0 {
			if _, err := fmt.Fprintln(stdout); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(stdout, "%s %s %s\n", step.ID, statusText(step.Status), step.DisplayName); err != nil {
			return err
		}
		if lines := lineRange(step); lines != "" {
			if _, err := fmt.Fprintf(stdout, "  Lines: %s\n", lines); err != nil {
				return err
			}
		}
		if len(step.OutputTail) > 0 {
			if _, err := fmt.Fprintln(stdout, "  Outputs:"); err != nil {
				return err
			}
			for _, output := range step.OutputTail {
				if _, err := fmt.Fprintf(stdout, "    %s\n", output); err != nil {
					return err
				}
			}
		}
		if step.ErrorDetail != "" {
			if _, err := fmt.Fprintln(stdout, "  Error:"); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "    %s\n", step.ErrorDetail); err != nil {
				return err
			}
		}
		if step.WarningDetail != "" {
			if _, err := fmt.Fprintln(stdout, "  Warning:"); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "    %s\n", step.WarningDetail); err != nil {
				return err
			}
		}
	}
	return nil
}

func statusText(status dobl.EventStatus) string {
	if status == "" {
		return "UNKNOWN"
	}
	return string(status)
}

func stepLabel(step dobl.Step) string {
	if step.Instruction != "" {
		return step.Instruction
	}
	if step.Category != "" {
		return string(step.Category)
	}
	return "other"
}

func lineRange(step dobl.Step) string {
	if step.StartLine == 0 && step.EndLine == 0 {
		return ""
	}
	if step.StartLine == step.EndLine || step.EndLine == 0 {
		return fmt.Sprintf("%d", step.StartLine)
	}
	return fmt.Sprintf("%d-%d", step.StartLine, step.EndLine)
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

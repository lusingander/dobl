package tui

import (
	"fmt"
	"strings"

	"github.com/lusingander/dobl"
)

type stats struct {
	Total    int
	Done     int
	Cached   int
	Warnings int
	Errors   int
	Canceled int
	Outputs  int
}

func collectStats(steps []dobl.Step) stats {
	stats := stats{Total: len(steps)}
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
		}
		stats.Outputs += step.OutputCount
	}
	return stats
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

func lineRange(step dobl.Step) string {
	if step.StartLine == 0 && step.EndLine == 0 {
		return ""
	}
	if step.StartLine == step.EndLine || step.EndLine == 0 {
		return fmt.Sprintf("%d", step.StartLine)
	}
	return fmt.Sprintf("%d-%d", step.StartLine, step.EndLine)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

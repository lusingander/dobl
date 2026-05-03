package tui

import (
	"fmt"
	"strings"

	"github.com/lusingander/dobl"
)

func ParseFilterMode(value string) (FilterMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return FilterAll, nil
	case "problems":
		return FilterProblems, nil
	case "warnings":
		return FilterWarnings, nil
	case "failed":
		return FilterFailed, nil
	default:
		return FilterAll, fmt.Errorf("unknown TUI filter %q", value)
	}
}

type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterProblems
	FilterWarnings
	FilterFailed
)

func nextFilter(filter FilterMode) FilterMode {
	switch filter {
	case FilterAll:
		return FilterProblems
	case FilterProblems:
		return FilterWarnings
	case FilterWarnings:
		return FilterFailed
	default:
		return FilterAll
	}
}

func filterSteps(steps []dobl.Step, filter FilterMode, search string) []dobl.Step {
	var visible []dobl.Step
	for _, step := range steps {
		if !matchesFilter(step, filter) {
			continue
		}
		if !matchesSearch(step, search) {
			continue
		}
		visible = append(visible, step)
	}
	return visible
}

func matchesFilter(step dobl.Step, filter FilterMode) bool {
	switch filter {
	case FilterProblems:
		return isProblemStep(step)
	case FilterWarnings:
		return step.Status == dobl.EventStatusWarning || step.WarningDetail != "" || step.WarningCount > 0
	case FilterFailed:
		return step.Status == dobl.EventStatusError || step.Status == dobl.EventStatusCanceled || step.ErrorDetail != ""
	default:
		return true
	}
}

func isProblemStep(step dobl.Step) bool {
	return step.Status == dobl.EventStatusError ||
		step.Status == dobl.EventStatusCanceled ||
		step.Status == dobl.EventStatusWarning ||
		step.ErrorDetail != "" ||
		step.WarningDetail != ""
}

func matchesSearch(step dobl.Step, search string) bool {
	query := strings.ToLower(strings.TrimSpace(search))
	if query == "" {
		return true
	}
	fields := []string{
		step.ID,
		string(step.Status),
		string(step.Category),
		step.Name,
		step.DisplayName,
		step.Stage,
		step.Instruction,
		step.ErrorDetail,
		step.WarningDetail,
	}
	fields = append(fields, step.OutputTail...)
	return strings.Contains(strings.ToLower(strings.Join(fields, "\n")), query)
}

func (f FilterMode) String() string {
	switch f {
	case FilterProblems:
		return "problems"
	case FilterWarnings:
		return "warnings"
	case FilterFailed:
		return "failed"
	default:
		return "all"
	}
}

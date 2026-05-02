package cli

import (
	"regexp"
	"strings"

	"github.com/lusingander/dobl"
)

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

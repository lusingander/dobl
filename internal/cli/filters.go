package cli

import (
	"regexp"
	"sort"
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

func sortSteps(steps []dobl.Step, key string) {
	sort.SliceStable(steps, func(i, j int) bool {
		left := steps[i]
		right := steps[j]
		switch key {
		case "duration":
			if durationNanos(left) != durationNanos(right) {
				return durationNanos(left) > durationNanos(right)
			}
		case "status":
			if statusRank(left.Status) != statusRank(right.Status) {
				return statusRank(left.Status) < statusRank(right.Status)
			}
		case "outputs":
			if left.OutputCount != right.OutputCount {
				return left.OutputCount > right.OutputCount
			}
		case "warnings":
			if left.WarningCount != right.WarningCount {
				return left.WarningCount > right.WarningCount
			}
		}
		return left.Order < right.Order
	})
}

func durationNanos(step dobl.Step) int64 {
	if step.DurationNanos == nil {
		return -1
	}
	return *step.DurationNanos
}

func statusRank(status dobl.EventStatus) int {
	switch status {
	case dobl.EventStatusError:
		return 0
	case dobl.EventStatusCanceled:
		return 1
	case dobl.EventStatusWarning:
		return 2
	case dobl.EventStatusProgress:
		return 3
	case dobl.EventStatusDone:
		return 4
	case dobl.EventStatusCached:
		return 5
	default:
		return 6
	}
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

func isKnownTop(key string) bool {
	switch key {
	case "slow", "warnings", "outputs":
		return true
	default:
		return false
	}
}

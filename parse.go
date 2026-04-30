package dobl

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// EventKind identifies the parsed role of a plain BuildKit progress line.
type EventKind string

const (
	EventStepStart  EventKind = "step_start"
	EventStepStatus EventKind = "step_status"
	EventStepOutput EventKind = "step_output"
	EventUnknown    EventKind = "unknown"
)

// BuildLog is the intermediate representation for a parsed build log.
type BuildLog struct {
	Events []Event `json:"events"`
}

// Event is one parsed line from a docker build/buildx --progress=plain log.
type Event struct {
	Line     int       `json:"line"`
	Kind     EventKind `json:"kind"`
	Raw      string    `json:"raw"`
	StepID   string    `json:"step_id,omitempty"`
	Detail   string    `json:"detail,omitempty"`
	Status   string    `json:"status,omitempty"`
	Duration string    `json:"duration,omitempty"`
}

var (
	ansiEscapeRE   = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	stepLineRE     = regexp.MustCompile(`^#(\d+)\s*(.*)$`)
	stepStartRE    = regexp.MustCompile(`^\[[^\]]+\]\s+.+$`)
	stepStatusRE   = regexp.MustCompile(`^(DONE|CACHED|ERROR|CANCELED|WARNING)(?::\s*(.*)|\s+(.+))?$`)
	stepOutputRE   = regexp.MustCompile(`^\d+(?:\.\d+)?\s+.+$`)
	durationTailRE = regexp.MustCompile(`\b(\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h))\b(?:\s+done)?$`)
)

// Parse reads a complete docker build/buildx --progress=plain log and returns
// an event-oriented intermediate representation.
func Parse(r io.Reader) (*BuildLog, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	log := &BuildLog{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		log.Events = append(log.Events, ParseLine(scanner.Text(), lineNo))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return log, nil
}

// ParseLine parses a single plain progress line.
func ParseLine(raw string, lineNo int) Event {
	clean := strings.TrimRight(ansiEscapeRE.ReplaceAllString(raw, ""), "\r")
	event := Event{
		Line: lineNo,
		Kind: EventUnknown,
		Raw:  raw,
	}

	match := stepLineRE.FindStringSubmatch(clean)
	if match == nil {
		return event
	}

	event.StepID = "#" + match[1]
	detail := strings.TrimSpace(match[2])
	event.Detail = detail

	switch {
	case detail == "":
		event.Kind = EventStepStatus
	case isStepStartDetail(detail):
		event.Kind = EventStepStart
	case stepStatusRE.MatchString(detail):
		event.Kind = EventStepStatus
		setStatusFields(&event, detail)
	case stepOutputRE.MatchString(detail):
		event.Kind = EventStepOutput
	default:
		event.Kind = EventStepStatus
		event.Status = "PROGRESS"
	}

	if event.Duration == "" {
		event.Duration = extractDuration(detail)
	}

	return event
}

func setStatusFields(event *Event, detail string) {
	match := stepStatusRE.FindStringSubmatch(detail)
	if match == nil {
		return
	}

	event.Status = match[1]
	event.Detail = ""
	if match[2] != "" {
		event.Detail = strings.TrimSpace(match[2])
		return
	}
	if match[3] != "" {
		rest := strings.TrimSpace(match[3])
		if extractDuration(rest) != rest {
			event.Detail = rest
		}
	}
}

func isStepStartDetail(detail string) bool {
	if stepStartRE.MatchString(detail) {
		return true
	}

	for _, prefix := range []string{
		"exporting to ",
		"importing cache manifest from ",
		"resolving provenance for ",
	} {
		if strings.HasPrefix(detail, prefix) {
			return true
		}
	}

	return false
}

func extractDuration(detail string) string {
	match := durationTailRE.FindStringSubmatch(detail)
	if match == nil {
		return ""
	}
	return match[1]
}

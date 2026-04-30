package dobl

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const scannerMaxLineBytes = 1024 * 1024

// EventKind identifies the parsed role of a plain BuildKit progress line.
type EventKind string

const (
	EventStepStart  EventKind = "step_start"
	EventStepStatus EventKind = "step_status"
	EventStepOutput EventKind = "step_output"
	EventUnknown    EventKind = "unknown"
)

// EventStatus identifies a parsed BuildKit step status.
type EventStatus string

const (
	EventStatusDone     EventStatus = "DONE"
	EventStatusCached   EventStatus = "CACHED"
	EventStatusError    EventStatus = "ERROR"
	EventStatusCanceled EventStatus = "CANCELED"
	EventStatusWarning  EventStatus = "WARNING"
	EventStatusProgress EventStatus = "PROGRESS"
)

// BuildLog is the intermediate representation for a parsed build log.
type BuildLog struct {
	Events []Event `json:"events"`
}

// Steps returns build steps in first-seen order, grouped by step ID.
func (l *BuildLog) Steps() []Step {
	if l == nil {
		return nil
	}

	steps := make([]Step, 0)
	indexes := map[string]int{}
	for _, event := range l.Events {
		if event.StepID == "" {
			continue
		}

		index, ok := indexes[event.StepID]
		if !ok {
			index = len(steps)
			indexes[event.StepID] = index
			steps = append(steps, Step{
				ID:        event.StepID,
				StartLine: event.Line,
				EndLine:   event.Line,
			})
		}

		step := &steps[index]
		step.Events = append(step.Events, event)
		step.EndLine = event.Line

		step.applyEvent(event)
	}

	return steps
}

func (s *Step) applyEvent(event Event) {
	switch event.Kind {
	case EventStepStart:
		if s.Name == "" {
			s.Name = event.Detail
			s.applyNameMetadata(event.Detail)
		}
	case EventStepStatus:
		if event.Status == EventStatusProgress {
			s.ProgressCount++
		}
	case EventStepOutput:
		s.OutputCount++
	case EventUnknown:
		s.UnknownCount++
	}

	if event.Status != "" {
		s.Status = event.Status
	}
	if event.Status == EventStatusError && event.Detail != "" {
		s.ErrorDetail = event.Detail
	}
	if event.Duration != "" {
		s.Duration = event.Duration
	}
	if event.DurationNanos != nil {
		s.DurationNanos = event.DurationNanos
	}
}

func (s *Step) applyNameMetadata(name string) {
	match := dockerfileStepNameRE.FindStringSubmatch(name)
	if match == nil {
		return
	}

	index, err := strconv.Atoi(match[2])
	if err != nil {
		return
	}
	total, err := strconv.Atoi(match[3])
	if err != nil {
		return
	}

	s.Stage = match[1]
	s.Index = index
	s.Total = total
	s.Instruction = match[4]
}

// Step is an aggregate view of all events with the same BuildKit step ID.
type Step struct {
	ID            string      `json:"id"`
	Name          string      `json:"name,omitempty"`
	Status        EventStatus `json:"status,omitempty"`
	Duration      string      `json:"duration,omitempty"`
	DurationNanos *int64      `json:"duration_nanos,omitempty"`
	Stage         string      `json:"stage,omitempty"`
	Index         int         `json:"index,omitempty"`
	Total         int         `json:"total,omitempty"`
	Instruction   string      `json:"instruction,omitempty"`
	OutputCount   int         `json:"output_count"`
	ProgressCount int         `json:"progress_count"`
	UnknownCount  int         `json:"unknown_count"`
	ErrorDetail   string      `json:"error_detail,omitempty"`
	StartLine     int         `json:"start_line"`
	EndLine       int         `json:"end_line"`
	Events        []Event     `json:"events,omitempty"`
}

// Event is one parsed line from a docker build/buildx --progress=plain log.
//
// Detail keeps the meaningful text after the BuildKit step ID when it is not
// already represented by Status or Duration:
//   - EventStepStart: the step name, such as "[internal] load metadata".
//   - EventStepStatus with EventStatusProgress: the progress text.
//   - EventStepStatus with EventStatusError or EventStatusWarning: the message.
//   - EventStepOutput: the command output line after the step ID.
//   - EventUnknown: empty.
type Event struct {
	Line          int         `json:"line"`
	Kind          EventKind   `json:"kind"`
	Raw           string      `json:"raw"`
	StepID        string      `json:"step_id,omitempty"`
	Detail        string      `json:"detail,omitempty"`
	Status        EventStatus `json:"status,omitempty"`
	Duration      string      `json:"duration,omitempty"`
	DurationNanos *int64      `json:"duration_nanos,omitempty"`
}

var (
	ansiEscapeRE         = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	ciTimestampRE        = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z\s+`)
	dockerfileStepNameRE = regexp.MustCompile(`^\[(?:(.+)\s+)?(\d+)/(\d+)\]\s+(\S+)`)
	stepLineRE           = regexp.MustCompile(`^#(\d+)\s*(.*)$`)
	stepStartRE          = regexp.MustCompile(`^\[[^\]]+\]\s+.+$`)
	stepStatusRE         = regexp.MustCompile(`^(DONE|CACHED|ERROR|CANCELED|WARNING)(?::\s*(.*)|\s+(.+))?$`)
	stepOutputRE         = regexp.MustCompile(`^\d+(?:\.\d+)?\s+.+$`)
	durationTailRE       = regexp.MustCompile(`\b(\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h))\b(?:\s+done)?$`)
)

// Parse reads a complete docker build/buildx --progress=plain log and returns
// an event-oriented intermediate representation.
func Parse(r io.Reader) (*BuildLog, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxLineBytes)

	log := &BuildLog{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		log.Events = append(log.Events, ParseLine(scanner.Text(), lineNo))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse build log after line %d: %w", lineNo, err)
	}

	return log, nil
}

// ParseLine parses a single plain progress line.
func ParseLine(raw string, lineNo int) Event {
	clean := normalizeLine(raw)
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
	case isProgressDetail(detail):
		event.Kind = EventStepStatus
		event.Status = EventStatusProgress
	case stepOutputRE.MatchString(detail):
		event.Kind = EventStepOutput
	default:
		event.Kind = EventStepStatus
		event.Status = EventStatusProgress
	}

	if event.Duration == "" {
		event.Duration = extractDuration(detail)
	}
	if event.Duration != "" {
		event.DurationNanos = parseDurationNanos(event.Duration)
	}

	return event
}

func normalizeLine(raw string) string {
	clean := ansiEscapeRE.ReplaceAllString(raw, "")
	if index := strings.LastIndex(clean, "\r"); index >= 0 {
		clean = clean[index+1:]
	}
	return ciTimestampRE.ReplaceAllString(clean, "")
}

func setStatusFields(event *Event, detail string) {
	match := stepStatusRE.FindStringSubmatch(detail)
	if match == nil {
		return
	}

	event.Status = EventStatus(match[1])
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
		"exporting cache to ",
		"importing cache manifest from ",
		"resolving provenance for ",
	} {
		if strings.HasPrefix(detail, prefix) {
			return true
		}
	}

	return false
}

func isProgressDetail(detail string) bool {
	return strings.HasSuffix(detail, " done")
}

func extractDuration(detail string) string {
	match := durationTailRE.FindStringSubmatch(detail)
	if match == nil {
		return ""
	}
	return match[1]
}

func parseDurationNanos(raw string) *int64 {
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return nil
	}
	nanos := duration.Nanoseconds()
	return &nanos
}

package dobl

import (
	"os"
	"strings"
	"testing"
)

func TestParsePlainBuildLog(t *testing.T) {
	input := strings.Join([]string{
		"#1 [internal] load build definition from Dockerfile",
		"#1 transferring dockerfile: 227B 0.0s done",
		"#1 DONE 0.1s",
		"#2 [internal] load .dockerignore",
		"#2 CACHED",
		"#3 0.123 hello from run",
		"unexpected line",
	}, "\n")

	log, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(log.Events) != 7 {
		t.Fatalf("event count = %d, want 7", len(log.Events))
	}

	assertEvent(t, log.Events[0], EventStepStart, "#1", "", "[internal] load build definition from Dockerfile", "")
	assertEvent(t, log.Events[1], EventStepStatus, "#1", EventStatusProgress, "transferring dockerfile: 227B 0.0s done", "0.0s")
	assertEvent(t, log.Events[2], EventStepStatus, "#1", EventStatusDone, "", "0.1s")
	assertEvent(t, log.Events[4], EventStepStatus, "#2", EventStatusCached, "", "")
	assertEvent(t, log.Events[5], EventStepOutput, "#3", "", "0.123 hello from run", "")
	assertEvent(t, log.Events[6], EventUnknown, "", "", "", "")
}

func TestParseLineStripsANSIBeforeParsing(t *testing.T) {
	event := ParseLine("\x1b[32m#9 DONE 2.4s\x1b[0m", 1)

	assertEvent(t, event, EventStepStatus, "#9", EventStatusDone, "", "2.4s")
	if event.Raw != "\x1b[32m#9 DONE 2.4s\x1b[0m" {
		t.Fatalf("raw line was not preserved: %q", event.Raw)
	}
}

func TestParseLineErrorStatus(t *testing.T) {
	event := ParseLine(`#4 ERROR: process "/bin/sh -c exit 1" did not complete successfully`, 1)

	assertEvent(t, event, EventStepStatus, "#4", EventStatusError, `process "/bin/sh -c exit 1" did not complete successfully`, "")
}

func TestParseLineKnownStatuses(t *testing.T) {
	tests := []struct {
		line     string
		status   EventStatus
		detail   string
		duration string
	}{
		{line: "#1 DONE 0.1s", status: EventStatusDone, duration: "0.1s"},
		{line: "#2 CACHED", status: EventStatusCached},
		{line: "#3 ERROR: failed to solve", status: EventStatusError, detail: "failed to solve"},
		{line: "#4 CANCELED", status: EventStatusCanceled},
		{line: "#5 WARNING: cache import failed", status: EventStatusWarning, detail: "cache import failed"},
		{line: "#6 resolving docker.io/library/alpine:latest done", status: EventStatusProgress, detail: "resolving docker.io/library/alpine:latest done"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			event := ParseLine(tt.line, 1)

			assertEvent(t, event, EventStepStatus, strings.Fields(tt.line)[0], tt.status, tt.detail, tt.duration)
		})
	}
}

func TestBuildLogSteps(t *testing.T) {
	input, err := os.Open("testdata/success_plain.log")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer input.Close()

	log, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	steps := log.Steps()
	if len(steps) != 6 {
		t.Fatalf("step count = %d, want 6", len(steps))
	}

	first := steps[0]
	if first.ID != "#1" {
		t.Fatalf("first step id = %q, want #1", first.ID)
	}
	if first.Name != "[internal] load build definition from Dockerfile" {
		t.Fatalf("first step name = %q", first.Name)
	}
	if first.Status != EventStatusDone {
		t.Fatalf("first step status = %q, want DONE", first.Status)
	}
	if first.Duration != "0.0s" {
		t.Fatalf("first step duration = %q, want 0.0s", first.Duration)
	}
	if first.StartLine != 1 || first.EndLine != 3 {
		t.Fatalf("first step lines = %d-%d, want 1-3", first.StartLine, first.EndLine)
	}
	if len(first.Events) != 3 {
		t.Fatalf("first step event count = %d, want 3", len(first.Events))
	}

	last := steps[len(steps)-1]
	if last.ID != "#6" {
		t.Fatalf("last step id = %q, want #6", last.ID)
	}
	if last.Name != "exporting to image" {
		t.Fatalf("last step name = %q", last.Name)
	}
	if last.Status != EventStatusDone {
		t.Fatalf("last step status = %q, want DONE", last.Status)
	}
	if last.StartLine != 14 || last.EndLine != 16 {
		t.Fatalf("last step lines = %d-%d, want 14-16", last.StartLine, last.EndLine)
	}
}

func TestBuildLogStepsNil(t *testing.T) {
	var log *BuildLog
	if steps := log.Steps(); steps != nil {
		t.Fatalf("nil log steps = %#v, want nil", steps)
	}
}

func TestBuildLogStepsInterleavedFixture(t *testing.T) {
	input, err := os.Open("testdata/parallel_plain.log")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer input.Close()

	log, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	steps := log.Steps()
	assertStepIDs(t, steps, []string{"#1", "#2", "#3", "#4", "#5", "#6", "#7", "#8", "#9"})

	step3 := findStep(t, steps, "#3")
	if step3.Name != "[build 1/3] RUN go mod download" {
		t.Fatalf("step #3 name = %q", step3.Name)
	}
	if step3.Status != EventStatusDone {
		t.Fatalf("step #3 status = %q, want DONE", step3.Status)
	}
	if step3.StartLine != 5 || step3.EndLine != 13 {
		t.Fatalf("step #3 lines = %d-%d, want 5-13", step3.StartLine, step3.EndLine)
	}
	if len(step3.Events) != 4 {
		t.Fatalf("step #3 events = %d, want 4", len(step3.Events))
	}

	step4 := findStep(t, steps, "#4")
	if step4.Status != EventStatusDone {
		t.Fatalf("step #4 status = %q, want DONE", step4.Status)
	}
	if step4.StartLine != 6 || step4.EndLine != 11 {
		t.Fatalf("step #4 lines = %d-%d, want 6-11", step4.StartLine, step4.EndLine)
	}

	step5 := findStep(t, steps, "#5")
	if step5.Status != EventStatusCached {
		t.Fatalf("step #5 status = %q, want CACHED", step5.Status)
	}

	step8 := findStep(t, steps, "#8")
	if step8.Name != "exporting cache to local directory" {
		t.Fatalf("step #8 name = %q", step8.Name)
	}
	if step8.Status != EventStatusDone {
		t.Fatalf("step #8 status = %q, want DONE", step8.Status)
	}
}

func TestParseFixtures(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		events       int
		starts       int
		statuses     int
		outputs      int
		unknowns     int
		finalStepID  string
		finalStatus  EventStatus
		finalDuraton string
	}{
		{
			name:         "success",
			file:         "testdata/success_plain.log",
			events:       16,
			starts:       6,
			statuses:     9,
			outputs:      1,
			finalStepID:  "#6",
			finalStatus:  EventStatusDone,
			finalDuraton: "0.0s",
		},
		{
			name:         "cache",
			file:         "testdata/cache_plain.log",
			events:       8,
			starts:       4,
			statuses:     4,
			finalStepID:  "#4",
			finalStatus:  EventStatusDone,
			finalDuraton: "0.0s",
		},
		{
			name:        "error",
			file:        "testdata/error_plain.log",
			events:      12,
			starts:      3,
			statuses:    4,
			outputs:     2,
			unknowns:    3,
			finalStepID: "#3",
			finalStatus: EventStatusError,
		},
		{
			name:         "parallel",
			file:         "testdata/parallel_plain.log",
			events:       26,
			starts:       9,
			statuses:     13,
			outputs:      4,
			finalStepID:  "#9",
			finalStatus:  EventStatusDone,
			finalDuraton: "0.0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.Open(tt.file)
			if err != nil {
				t.Fatalf("open fixture: %v", err)
			}
			defer input.Close()

			log, err := Parse(input)
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}

			if len(log.Events) != tt.events {
				t.Fatalf("event count = %d, want %d", len(log.Events), tt.events)
			}

			counts := countKinds(log.Events)
			if counts[EventStepStart] != tt.starts {
				t.Fatalf("start count = %d, want %d", counts[EventStepStart], tt.starts)
			}
			if counts[EventStepStatus] != tt.statuses {
				t.Fatalf("status count = %d, want %d", counts[EventStepStatus], tt.statuses)
			}
			if counts[EventStepOutput] != tt.outputs {
				t.Fatalf("output count = %d, want %d", counts[EventStepOutput], tt.outputs)
			}
			if counts[EventUnknown] != tt.unknowns {
				t.Fatalf("unknown count = %d, want %d", counts[EventUnknown], tt.unknowns)
			}

			final := lastStatusEvent(log.Events)
			if final == nil {
				t.Fatal("no status event found")
			}
			if final.StepID != tt.finalStepID {
				t.Fatalf("final step id = %q, want %q", final.StepID, tt.finalStepID)
			}
			if final.Status != tt.finalStatus {
				t.Fatalf("final status = %q, want %q", final.Status, tt.finalStatus)
			}
			if final.Duration != tt.finalDuraton {
				t.Fatalf("final duration = %q, want %q", final.Duration, tt.finalDuraton)
			}
		})
	}
}

func assertStepIDs(t *testing.T, steps []Step, ids []string) {
	t.Helper()

	if len(steps) != len(ids) {
		t.Fatalf("step count = %d, want %d", len(steps), len(ids))
	}
	for i, id := range ids {
		if steps[i].ID != id {
			t.Fatalf("step[%d] id = %q, want %q", i, steps[i].ID, id)
		}
	}
}

func findStep(t *testing.T, steps []Step, id string) Step {
	t.Helper()

	for _, step := range steps {
		if step.ID == id {
			return step
		}
	}
	t.Fatalf("step %s not found", id)
	return Step{}
}

func countKinds(events []Event) map[EventKind]int {
	counts := map[EventKind]int{}
	for _, event := range events {
		counts[event.Kind]++
	}
	return counts
}

func lastStatusEvent(events []Event) *Event {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == EventStepStatus && events[i].Status != "" {
			return &events[i]
		}
	}
	return nil
}

func assertEvent(t *testing.T, event Event, kind EventKind, stepID string, status EventStatus, detail, duration string) {
	t.Helper()

	if event.Kind != kind {
		t.Fatalf("kind = %q, want %q", event.Kind, kind)
	}
	if event.StepID != stepID {
		t.Fatalf("step id = %q, want %q", event.StepID, stepID)
	}
	if event.Status != status {
		t.Fatalf("status = %q, want %q", event.Status, status)
	}
	if event.Detail != detail {
		t.Fatalf("detail = %q, want %q", event.Detail, detail)
	}
	if event.Duration != duration {
		t.Fatalf("duration = %q, want %q", event.Duration, duration)
	}
}

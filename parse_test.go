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
	assertEvent(t, log.Events[1], EventStepStatus, "#1", "PROGRESS", "transferring dockerfile: 227B 0.0s done", "0.0s")
	assertEvent(t, log.Events[2], EventStepStatus, "#1", "DONE", "", "0.1s")
	assertEvent(t, log.Events[4], EventStepStatus, "#2", "CACHED", "", "")
	assertEvent(t, log.Events[5], EventStepOutput, "#3", "", "0.123 hello from run", "")
	assertEvent(t, log.Events[6], EventUnknown, "", "", "", "")
}

func TestParseLineStripsANSIBeforeParsing(t *testing.T) {
	event := ParseLine("\x1b[32m#9 DONE 2.4s\x1b[0m", 1)

	assertEvent(t, event, EventStepStatus, "#9", "DONE", "", "2.4s")
	if event.Raw != "\x1b[32m#9 DONE 2.4s\x1b[0m" {
		t.Fatalf("raw line was not preserved: %q", event.Raw)
	}
}

func TestParseLineErrorStatus(t *testing.T) {
	event := ParseLine(`#4 ERROR: process "/bin/sh -c exit 1" did not complete successfully`, 1)

	assertEvent(t, event, EventStepStatus, "#4", "ERROR", `process "/bin/sh -c exit 1" did not complete successfully`, "")
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
		finalStatus  string
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
			finalStatus:  "DONE",
			finalDuraton: "0.0s",
		},
		{
			name:         "cache",
			file:         "testdata/cache_plain.log",
			events:       8,
			starts:       4,
			statuses:     4,
			finalStepID:  "#4",
			finalStatus:  "DONE",
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
			finalStatus: "ERROR",
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

func assertEvent(t *testing.T, event Event, kind EventKind, stepID, status, detail, duration string) {
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

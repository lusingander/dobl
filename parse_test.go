package dobl

import (
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
	assertEvent(t, log.Events[2], EventStepStatus, "#1", "DONE", "0.1s", "0.1s")
	assertEvent(t, log.Events[4], EventStepStatus, "#2", "CACHED", "CACHED", "")
	assertEvent(t, log.Events[5], EventStepOutput, "#3", "", "0.123 hello from run", "")
	assertEvent(t, log.Events[6], EventUnknown, "", "", "", "")
}

func TestParseLineStripsANSIBeforeParsing(t *testing.T) {
	event := ParseLine("\x1b[32m#9 DONE 2.4s\x1b[0m", 1)

	assertEvent(t, event, EventStepStatus, "#9", "DONE", "2.4s", "2.4s")
	if event.Raw != "\x1b[32m#9 DONE 2.4s\x1b[0m" {
		t.Fatalf("raw line was not preserved: %q", event.Raw)
	}
}

func TestParseLineErrorStatus(t *testing.T) {
	event := ParseLine(`#4 ERROR: process "/bin/sh -c exit 1" did not complete successfully`, 1)

	assertEvent(t, event, EventStepStatus, "#4", "ERROR", `process "/bin/sh -c exit 1" did not complete successfully`, "")
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

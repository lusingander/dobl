package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunParseFromStdin(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "parse"}, strings.NewReader("#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded struct {
		Events []struct {
			Kind          string `json:"kind"`
			StepID        string `json:"step_id"`
			Status        string `json:"status"`
			DurationNanos *int64 `json:"duration_nanos"`
		} `json:"events"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if len(decoded.Events) != 1 {
		t.Fatalf("event count = %d, want 1", len(decoded.Events))
	}
	if decoded.Events[0].Kind != "step_status" || decoded.Events[0].StepID != "#1" || decoded.Events[0].Status != "DONE" {
		t.Fatalf("unexpected event: %+v", decoded.Events[0])
	}
	if decoded.Events[0].DurationNanos == nil || *decoded.Events[0].DurationNanos != 100_000_000 {
		t.Fatalf("duration nanos = %v, want 100000000", decoded.Events[0].DurationNanos)
	}
}

func TestRunSummaryFromStdin(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary"}, strings.NewReader("#1 [internal] load build definition from Dockerfile\n#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Status        string `json:"status"`
		Duration      string `json:"duration"`
		DurationNanos *int64 `json:"duration_nanos"`
		Events        []any  `json:"events"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("step count = %d, want 1", len(decoded))
	}
	if decoded[0].ID != "#1" || decoded[0].Status != "DONE" || decoded[0].Duration != "0.1s" {
		t.Fatalf("unexpected step: %+v", decoded[0])
	}
	if decoded[0].DurationNanos == nil || *decoded[0].DurationNanos != 100_000_000 {
		t.Fatalf("duration nanos = %v, want 100000000", decoded[0].DurationNanos)
	}
	if decoded[0].Events != nil {
		t.Fatalf("events = %#v, want nil", decoded[0].Events)
	}
}

func TestRunSummaryWithEvents(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--events"}, strings.NewReader("#1 [internal] load build definition from Dockerfile\n#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		Events []struct {
			Kind string `json:"kind"`
		} `json:"events"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("step count = %d, want 1", len(decoded))
	}
	if len(decoded[0].Events) != 2 {
		t.Fatalf("event count = %d, want 2", len(decoded[0].Events))
	}
}

func TestRunParseCompact(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "parse", "--compact"}, strings.NewReader("#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if strings.Contains(out.String(), "\n  ") {
		t.Fatalf("compact output contains indentation: %q", out.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "unknown"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
}

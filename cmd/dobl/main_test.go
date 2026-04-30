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

func TestRunParseExplicitJSONFormat(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "parse", "--format", "json"}, strings.NewReader("#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded struct {
		Events []struct {
			Status string `json:"status"`
		} `json:"events"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Status != "DONE" {
		t.Fatalf("unexpected output: %+v", decoded)
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
		OutputCount   int    `json:"output_count"`
		ProgressCount int    `json:"progress_count"`
		UnknownCount  int    `json:"unknown_count"`
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
	if decoded[0].OutputCount != 0 || decoded[0].ProgressCount != 0 || decoded[0].UnknownCount != 0 {
		t.Fatalf("unexpected counts: %+v", decoded[0])
	}
	if decoded[0].Events != nil {
		t.Fatalf("events = %#v, want nil", decoded[0].Events)
	}
}

func TestRunSummaryExplicitJSONFormat(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--format", "json"}, strings.NewReader("#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Status != "DONE" {
		t.Fatalf("unexpected output: %+v", decoded)
	}
}

func TestRunSummaryTableFormat(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--format", "table"}, strings.NewReader("#1 [build 1/2] RUN echo hi\n#1 0.100 hi\n#1 ERROR: failed\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"ID",
		"STATUS",
		"DURATION",
		"STEP",
		"INSTRUCTION",
		"OUTPUTS",
		"#1",
		"ERROR",
		"build 1/2",
		"RUN",
		"failed",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output %q does not contain %q", output, want)
		}
	}
	if strings.Contains(output, "{") || strings.Contains(output, "}") {
		t.Fatalf("table output looks like json: %q", output)
	}
}

func TestRunSummaryTableRejectsJSONOnlyOptions(t *testing.T) {
	tests := [][]string{
		{"dobl", "summary", "--format", "table", "--events"},
		{"dobl", "summary", "--format", "table", "--compact"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out bytes.Buffer
			err := run(args, strings.NewReader("#1 DONE 0.1s\n"), &out)
			if err == nil {
				t.Fatal("run returned nil error")
			}
		})
	}
}

func TestRunSummaryIncludesStepMetadata(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary"}, strings.NewReader("#1 [1/1] RUN echo hi\n#1 0.100 hi\n#1 ERROR: failed\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		OutputCount int    `json:"output_count"`
		ErrorDetail string `json:"error_detail"`
		Index       int    `json:"index"`
		Total       int    `json:"total"`
		Instruction string `json:"instruction"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("step count = %d, want 1", len(decoded))
	}
	if decoded[0].OutputCount != 1 {
		t.Fatalf("output count = %d, want 1", decoded[0].OutputCount)
	}
	if decoded[0].ErrorDetail != "failed" {
		t.Fatalf("error detail = %q, want failed", decoded[0].ErrorDetail)
	}
	if decoded[0].Index != 1 || decoded[0].Total != 1 || decoded[0].Instruction != "RUN" {
		t.Fatalf("unexpected step metadata: %+v", decoded[0])
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

func TestRunRejectsUnknownFormat(t *testing.T) {
	tests := [][]string{
		{"dobl", "parse", "--format", "table"},
		{"dobl", "summary", "--format", "yaml"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var out bytes.Buffer
			err := run(args, strings.NewReader("#1 DONE 0.1s\n"), &out)
			if err == nil {
				t.Fatal("run returned nil error")
			}
		})
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "unknown"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
}

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
			Kind   string `json:"kind"`
			StepID string `json:"step_id"`
			Status string `json:"status"`
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
}

func TestRunSummaryFromStdin(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary"}, strings.NewReader("#1 [internal] load build definition from Dockerfile\n#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Status   string `json:"status"`
		Duration string `json:"duration"`
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
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "unknown"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
}

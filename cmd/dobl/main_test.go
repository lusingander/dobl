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

func TestRunHelp(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "--help"}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"Usage: dobl",
		"Parse and summarize plain Docker BuildKit build logs.",
		"parse",
		"summary",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output %q does not contain %q", output, want)
		}
	}
}

func TestRunCommandHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "parse",
			args: []string{"dobl", "parse", "--help"},
			want: []string{"Usage: dobl parse", "dobl parse build.log", "--compact"},
		},
		{
			name: "summary",
			args: []string{"dobl", "summary", "--help"},
			want: []string{"Usage: dobl summary", "dobl summary --format table build.log", "--status", "ERROR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(tt.args, strings.NewReader(""), &out)
			if err != nil {
				t.Fatalf("run returned error: %v", err)
			}

			output := out.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Fatalf("help output %q does not contain %q", output, want)
				}
			}
		})
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
		"NAME",
		"OUTPUTS",
		"#1",
		"ERROR",
		"build 1/2",
		"RUN",
		"[build 1/2] RUN echo hi",
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

func TestRunSummaryTableTruncatesLongErrors(t *testing.T) {
	longError := strings.Repeat("x", tableErrorWidth+20)
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--format", "table"}, strings.NewReader("#1 ERROR: "+longError+"\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	if strings.Contains(output, longError) {
		t.Fatalf("table output contains untruncated error: %q", output)
	}
	if !strings.Contains(output, strings.Repeat("x", tableErrorWidth-3)+"...") {
		t.Fatalf("table output missing truncated error: %q", output)
	}
}

func TestRunSummaryTableWideKeepsLongErrors(t *testing.T) {
	longError := strings.Repeat("x", tableErrorWidth+20)
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--format", "table", "--wide"}, strings.NewReader("#1 ERROR: "+longError+"\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if !strings.Contains(out.String(), longError) {
		t.Fatalf("wide table output missing full error: %q", out.String())
	}
}

func TestRunSummaryFailedJSON(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--failed"}, strings.NewReader(strings.Join([]string{
		"#1 DONE 0.1s",
		"#2 WARNING: cache import failed",
		"#3 ERROR: failed",
		"#4 CANCELED",
	}, "\n")), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("step count = %d, want 2", len(decoded))
	}
	if decoded[0].ID != "#3" || decoded[0].Status != "ERROR" {
		t.Fatalf("unexpected first failed step: %+v", decoded[0])
	}
	if decoded[1].ID != "#4" || decoded[1].Status != "CANCELED" {
		t.Fatalf("unexpected second failed step: %+v", decoded[1])
	}
}

func TestRunSummaryFailedTable(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--failed", "--format", "table"}, strings.NewReader("#1 DONE 0.1s\n#2 ERROR: failed\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "#1") {
		t.Fatalf("table output contains non-failed step: %q", output)
	}
	if !strings.Contains(output, "#2") || !strings.Contains(output, "ERROR") {
		t.Fatalf("table output missing failed step: %q", output)
	}
}

func TestRunSummaryStatusJSON(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--status", "WARNING"}, strings.NewReader(strings.Join([]string{
		"#1 DONE 0.1s",
		"#2 WARNING: cache import failed",
		"#3 ERROR: failed",
	}, "\n")), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("step count = %d, want 1", len(decoded))
	}
	if decoded[0].ID != "#2" || decoded[0].Status != "WARNING" {
		t.Fatalf("unexpected step: %+v", decoded[0])
	}
}

func TestRunSummaryStatusTable(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--status", "CACHED", "--format", "table"}, strings.NewReader("#1 DONE 0.1s\n#2 CACHED\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "#1") {
		t.Fatalf("table output contains filtered step: %q", output)
	}
	if !strings.Contains(output, "#2") || !strings.Contains(output, "CACHED") {
		t.Fatalf("table output missing cached step: %q", output)
	}
}

func TestRunSummaryStageFilterJSON(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--stage", "build"}, strings.NewReader(strings.Join([]string{
		"#1 [internal] load build definition from Dockerfile",
		"#1 DONE 0.0s",
		"#2 [build 1/2] RUN echo hi",
		"#2 DONE 0.1s",
		"#3 [stage-1 2/2] COPY --from=build /out /out",
		"#3 DONE 0.0s",
	}, "\n")), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		ID    string `json:"id"`
		Stage string `json:"stage"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if len(decoded) != 1 || decoded[0].ID != "#2" || decoded[0].Stage != "build" {
		t.Fatalf("unexpected filtered steps: %+v", decoded)
	}
}

func TestRunSummaryInstructionFilterJSON(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--instruction", "run"}, strings.NewReader(strings.Join([]string{
		"#1 [1/2] FROM alpine",
		"#1 DONE 0.0s",
		"#2 [2/2] RUN echo hi",
		"#2 DONE 0.1s",
	}, "\n")), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var decoded []struct {
		ID          string `json:"id"`
		Instruction string `json:"instruction"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if len(decoded) != 1 || decoded[0].ID != "#2" || decoded[0].Instruction != "RUN" {
		t.Fatalf("unexpected filtered steps: %+v", decoded)
	}
}

func TestRunSummaryStepFilterTable(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--step", "2", "--format", "table"}, strings.NewReader("#1 DONE 0.1s\n#2 ERROR: failed\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "#1") {
		t.Fatalf("table output contains filtered step: %q", output)
	}
	if !strings.Contains(output, "#2") || !strings.Contains(output, "ERROR") {
		t.Fatalf("table output missing selected step: %q", output)
	}
}

func TestRunSummaryRejectsFailedAndStatus(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--failed", "--status", "ERROR"}, strings.NewReader("#1 ERROR: failed\n"), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
}

func TestRunSummaryTableRejectsJSONOnlyOptions(t *testing.T) {
	tests := [][]string{
		{"dobl", "summary", "--format", "table", "--events"},
		{"dobl", "summary", "--format", "table", "--compact"},
		{"dobl", "summary", "--wide"},
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

func TestRunSummaryValidatesOptionsBeforeInput(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--format", "table", "--events", "missing.log"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
	if !strings.Contains(err.Error(), "--events is only supported with --format=json") {
		t.Fatalf("error = %q, want option validation error", err)
	}
	if strings.Contains(err.Error(), "missing.log") {
		t.Fatalf("error = %q, unexpectedly opened input before validating options", err)
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

func TestRunRejectsUnknownStatus(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "summary", "--status", "SKIPPED"}, strings.NewReader("#1 DONE 0.1s\n"), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
}

func TestRunParseReportsInputContext(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "parse"}, strings.NewReader(strings.Repeat("x", 1024*1024+1)), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
	if !strings.Contains(err.Error(), "parse stdin") {
		t.Fatalf("error = %q, want stdin context", err)
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("error = %q, want scanner cause", err)
	}
}

func TestRunParseReportsOpenContext(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "parse", "missing.log"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
	if !strings.Contains(err.Error(), "open missing.log") {
		t.Fatalf("error = %q, want file context", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"dobl", "unknown"}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("run returned nil error")
	}
}

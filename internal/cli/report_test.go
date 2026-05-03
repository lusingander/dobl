package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunReportEmbedsSummaryJSON(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"dobl", "report"}, strings.NewReader("#1 [1/1] RUN echo hi\n#1 DONE 0.1s\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"<!doctype html>",
		`id="embedded-summary"`,
		`data-source="stdin"`,
		`"id":"#1"`,
		"loadEmbeddedSummary();",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("report output missing %q", want)
		}
	}

	embeddedIndex := strings.Index(output, `id="embedded-summary"`)
	loaderIndex := strings.Index(output, "loadEmbeddedSummary();")
	if embeddedIndex < 0 || loaderIndex < 0 || embeddedIndex > loaderIndex {
		t.Fatal("embedded summary must appear before viewer initialization")
	}
}

func TestRunReportEscapesEmbeddedSummary(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"dobl", "report"}, strings.NewReader("#1 [1/1] RUN echo '</script>'\n#1 ERROR: failed </script>\n"), &out)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "failed </script>") {
		t.Fatalf("report output contains raw closing script tag")
	}
	if !strings.Contains(output, `failed <\/script>`) {
		t.Fatalf("report output missing escaped closing script tag")
	}
}

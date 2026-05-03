package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestViewerSampleMatchesVisualizationContractGolden(t *testing.T) {
	sample, err := os.ReadFile("../../examples/viewer/sample-summary.json")
	if err != nil {
		t.Fatalf("read viewer sample: %v", err)
	}
	golden, err := os.ReadFile("testdata/summary_visualization_contract.golden.json")
	if err != nil {
		t.Fatalf("read visualization golden: %v", err)
	}
	var compactSample bytes.Buffer
	if err := json.Compact(&compactSample, sample); err != nil {
		t.Fatalf("viewer sample is invalid json: %v", err)
	}
	var compactGolden bytes.Buffer
	if err := json.Compact(&compactGolden, golden); err != nil {
		t.Fatalf("visualization golden is invalid json: %v", err)
	}
	if compactSample.String() != compactGolden.String() {
		t.Fatal("viewer sample does not match visualization contract golden")
	}
}

func TestStaticViewerMatchesEmbeddedViewer(t *testing.T) {
	staticViewer, err := os.ReadFile("../../examples/viewer/index.html")
	if err != nil {
		t.Fatalf("read static viewer: %v", err)
	}
	if viewerHTML != string(staticViewer) {
		t.Fatal("examples/viewer/index.html does not match canonical internal/cli/viewer.html; run go generate ./internal/cli")
	}
}

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestAnalyzeSkipsNonKubernetesEntries(t *testing.T) {
	out := new(bytes.Buffer)
	err := analyze(out, "testdata/boltdb/db")
	if err != nil {
		t.Fatalf("analyze returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Largest objects (byte size sum of all revisions):") {
		t.Fatalf("expected largest objects section, got:\n%s", got)
	}
	if strings.Contains(got, "compact_rev_key") {
		t.Fatalf("expected non-kubernetes entries to be skipped from largest objects output, got:\n%s", got)
	}
}

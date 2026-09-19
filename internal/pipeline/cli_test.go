package pipeline

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// statusTargetDecode is the slice of RunStatus's JSON output these tests
// inspect. Decoding just the node identifier is enough to prove which node
// was resolved without coupling to the whole status shape.
type statusTargetDecode struct {
	Target struct {
		Node string `json:"node"`
	} `json:"target"`
}

func writeMarker(t *testing.T, projectRoot, contents string) {
	t.Helper()
	markerPath := filepath.Join(projectRoot, markerFileName)
	if err := os.WriteFile(markerPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing marker: %v", err)
	}
}

func TestRunStatusResolvesNodeFromMarkerWhenNodeFlagOmitted(t *testing.T) {
	projectRoot := t.TempDir()
	writeMarker(t, projectRoot, `{"mode":"in-project","node":"acme-hr/payroll"}`)

	var out, errOut bytes.Buffer
	code := RunStatus(nil, Environment{ProjectRoot: projectRoot}, &out, &errOut)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, ExitOK, errOut.String())
	}

	var decoded statusTargetDecode
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("decoding status: %v; stdout: %s", err, out.String())
	}
	if decoded.Target.Node != "acme-hr/payroll" {
		t.Errorf("resolved node = %q, want %q", decoded.Target.Node, "acme-hr/payroll")
	}
}

func TestRunStatusExplicitNodeFlagWinsOverMarker(t *testing.T) {
	projectRoot := t.TempDir()
	writeMarker(t, projectRoot, `{"mode":"in-project","node":"acme-hr/payroll"}`)

	var out, errOut bytes.Buffer
	code := RunStatus([]string{"--node", "acme-hr/other"}, Environment{ProjectRoot: projectRoot}, &out, &errOut)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, ExitOK, errOut.String())
	}

	var decoded statusTargetDecode
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("decoding status: %v; stdout: %s", err, out.String())
	}
	if decoded.Target.Node != "acme-hr/other" {
		t.Errorf("resolved node = %q, want %q (explicit --node must win)", decoded.Target.Node, "acme-hr/other")
	}
}

func TestRunStatusExplicitNodeFlagWorksWithNoMarkerAtAll(t *testing.T) {
	// Existing explicit --node behaviour, unchanged, with no marker on disk.
	// GlobalMode is set only so mode resolution (unrelated to this change)
	// succeeds without a vault base path; the point under test is the node.
	projectRoot := t.TempDir()

	var out, errOut bytes.Buffer
	code := RunStatus([]string{"--node", "acme-hr"},
		Environment{ProjectRoot: projectRoot, GlobalMode: ModeInProject}, &out, &errOut)

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, ExitOK, errOut.String())
	}

	var decoded statusTargetDecode
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("decoding status: %v; stdout: %s", err, out.String())
	}
	if decoded.Target.Node != "acme-hr" {
		t.Errorf("resolved node = %q, want %q", decoded.Target.Node, "acme-hr")
	}
}

func TestRunStatusFailsNamingTheFixWhenNeitherFlagNorMarkerHaveANode(t *testing.T) {
	projectRoot := t.TempDir()

	var out, errOut bytes.Buffer
	code := RunStatus(nil, Environment{ProjectRoot: projectRoot}, &out, &errOut)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, ExitUsage, errOut.String())
	}
	if !strings.Contains(errOut.String(), "--node") {
		t.Errorf("error does not mention --node: %s", errOut.String())
	}
	if !strings.Contains(errOut.String(), markerFileName) {
		t.Errorf("error does not name the fix (recording the node in %s): %s", markerFileName, errOut.String())
	}
}

func TestRunStatusMarkerWithModeButNoNodeStillFails(t *testing.T) {
	// A marker that only ever carried "mode" (the pre-T1 shape) must not be
	// silently treated as carrying a node.
	projectRoot := t.TempDir()
	writeMarker(t, projectRoot, `{"mode":"in-project"}`)

	var out, errOut bytes.Buffer
	code := RunStatus(nil, Environment{ProjectRoot: projectRoot}, &out, &errOut)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, ExitUsage, errOut.String())
	}
}

func TestRunStatusMalformedMarkerIsAnErrorNotASilentNoNode(t *testing.T) {
	projectRoot := t.TempDir()
	writeMarker(t, projectRoot, `{ not json`)

	var out, errOut bytes.Buffer
	code := RunStatus(nil, Environment{ProjectRoot: projectRoot}, &out, &errOut)

	if code != ExitUsage {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, ExitUsage, errOut.String())
	}
	// A malformed marker must not be reported through the "no node anywhere"
	// message: that would misrepresent a broken file as an absent one.
	if strings.Contains(errOut.String(), "status needs --node") {
		t.Errorf("malformed marker was reported as a missing node: %s", errOut.String())
	}
}

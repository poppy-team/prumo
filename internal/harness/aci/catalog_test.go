package aci

import "testing"

// TestFileOperationsAgreeWithKinds keeps the catalogue's two statements about a
// tool consistent: a tool cannot both be read-only and report a change.
//
// The mapping is data, and data drifts. This is the cheap check that catches the
// edit that updates one field and forgets the other — which would have a
// read-only tool claiming it modified a file on the run timeline.
func TestFileOperationsAgreeWithKinds(t *testing.T) {
	for _, tool := range Catalog() {
		if tool.FileOperation == "" {
			continue
		}
		if tool.Kind == "read-only" {
			t.Errorf("%s reports a file operation (%q) but is read-only", tool.Name, tool.FileOperation)
		}
		switch tool.FileOperation {
		case "created", "modified", "moved", "deleted":
		default:
			t.Errorf("%s reports an unknown file operation %q", tool.Name, tool.FileOperation)
		}
	}
}

// TestOperationOfReadsTheCatalogue pins the port the scheduler depends on: the
// answer comes from the tool's definition, and an unknown tool claims nothing.
func TestOperationOfReadsTheCatalogue(t *testing.T) {
	e := New(t.TempDir())
	if got := e.OperationOf("edit.patch"); got != "modified" {
		t.Errorf("edit.patch operation = %q, want modified", got)
	}
	if got := e.OperationOf("process.exec"); got != "" {
		t.Errorf("a command that can touch anything must claim nothing, got %q", got)
	}
	if got := e.OperationOf("not.a.tool"); got != "" {
		t.Errorf("an unknown tool must claim nothing, got %q", got)
	}
}

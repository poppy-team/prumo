package doccompile

import (
	"reflect"
	"testing"
)

const sectionDoc = "# T\n\nintro\n\n## A\n\na-body\n\n### A1\na1-body\n\n## B\n\nb-body\n"

func TestListSections(t *testing.T) {
	got := ListSections(sectionDoc)
	want := [][]string{{"T"}, {"T", "A"}, {"T", "A", "A1"}, {"T", "B"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestExtractReplaceDelete(t *testing.T) {
	sec, err := ExtractSection(sectionDoc, []string{"T", "A"})
	if err != nil || sec != "## A\n\na-body\n\n### A1\na1-body\n\n" {
		t.Fatalf("extract failed: %q %v", sec, err)
	}
	replaced, err := ReplaceSection(sectionDoc, []string{"T", "B"}, "new-b")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(replaced, "## B\nnew-b\n") || !contains(replaced, "### A1\na1-body") {
		t.Fatalf("replace broke siblings:\n%s", replaced)
	}
	deleted, err := DeleteSection(sectionDoc, []string{"T", "A"})
	if err != nil {
		t.Fatal(err)
	}
	if contains(deleted, "a-body") || !contains(deleted, "## B") {
		t.Fatalf("delete failed:\n%s", deleted)
	}
	if _, err := DeleteSection("# Only\n\nx\n", []string{"Only"}); err == nil {
		t.Fatal("last-section delete must be refused")
	}
	if _, err := ExtractSection(sectionDoc, []string{"T", "Nope"}); err == nil {
		t.Fatal("missing section must error")
	}
}

func TestFenceOpaque(t *testing.T) {
	doc := "# T\n\n```\n# not-a-heading\n```\n\n## Real\n\nx\n"
	got := ListSections(doc)
	if len(got) != 2 {
		t.Fatalf("fenced heading must not split: %v", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

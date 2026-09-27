package doccompile

import (
	"reflect"
	"testing"
)

func TestReplaceRegion(t *testing.T) {
	doc := "# T\n\n<!-- prumo:begin tbl -->\nOLD\n<!-- prumo:end tbl -->\n\ntail\n"
	got, err := ReplaceRegion(doc, "tbl", "NEW")
	if err != nil {
		t.Fatal(err)
	}
	want := "# T\n\n<!-- prumo:begin tbl -->\nNEW\n<!-- prumo:end tbl -->\n\ntail\n"
	if got != want {
		t.Fatalf("got:\n%s", got)
	}
	body, err := ExtractRegion(got, "tbl")
	if err != nil || body != "NEW" {
		t.Fatalf("extract failed: %q %v", body, err)
	}
	if _, err := ReplaceRegion(doc, "nope", "x"); err == nil {
		t.Fatal("missing region must error, never append")
	}
}

func TestRemoveRegion(t *testing.T) {
	doc := "# T\n\n<!-- prumo:begin tbl -->\nOLD\n<!-- prumo:end tbl -->\n\ntail\n"
	got, err := RemoveRegion(doc, "tbl")
	if err != nil {
		t.Fatal(err)
	}
	want := "# T\n\ntail\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
	// non-existing region is a no-op
	noop, err := RemoveRegion(doc, "missing")
	if err != nil || noop != doc {
		t.Fatalf("expected noop on missing region, got: %q, err: %v", noop, err)
	}
}

func TestUpsertRegion(t *testing.T) {
	// 1. In empty doc
	empty := UpsertRegion("", "reg", "HELLO")
	wantEmpty := "<!-- prumo:begin reg -->\nHELLO\n<!-- prumo:end reg -->\n"
	if empty != wantEmpty {
		t.Fatalf("empty upsert got:\n%q\nwant:\n%q", empty, wantEmpty)
	}

	// 2. Append to existing doc without region
	doc := "# Existing\n\nContent"
	appended := UpsertRegion(doc, "reg", "HELLO")
	wantAppended := "# Existing\n\nContent\n\n<!-- prumo:begin reg -->\nHELLO\n<!-- prumo:end reg -->\n"
	if appended != wantAppended {
		t.Fatalf("append upsert got:\n%q\nwant:\n%q", appended, wantAppended)
	}

	// 3. Replace existing region
	updated := UpsertRegion(appended, "reg", "UPDATED")
	wantUpdated := "# Existing\n\nContent\n\n<!-- prumo:begin reg -->\nUPDATED\n<!-- prumo:end reg -->\n"
	if updated != wantUpdated {
		t.Fatalf("replace upsert got:\n%q\nwant:\n%q", updated, wantUpdated)
	}
}

func TestApplyPatch(t *testing.T) {
	doc := map[string]any{"a": map[string]any{"b": 1.0}, "list": []any{"x", "y"}}
	got, err := ApplyPatch(doc, []PatchOp{
		{Op: "replace", Path: "/a/b", Value: 2.0},
		{Op: "add", Path: "/a/c", Value: "new"},
		{Op: "remove", Path: "/list/0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["a"].(map[string]any)["b"] != 2.0 || m["a"].(map[string]any)["c"] != "new" {
		t.Fatalf("map patch failed: %+v", m)
	}
	if !reflect.DeepEqual(m["list"], []any{"y"}) {
		t.Fatalf("list patch failed: %+v", m)
	}
	if _, err := ApplyPatch(m, []PatchOp{{Op: "move", Path: "/a"}}); err == nil {
		t.Fatal("unsupported op must error")
	}
	if _, err := ApplyPatch(m, []PatchOp{{Op: "replace", Path: "/missing"}}); err == nil {
		t.Fatal("missing key must error")
	}
}

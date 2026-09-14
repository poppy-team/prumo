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

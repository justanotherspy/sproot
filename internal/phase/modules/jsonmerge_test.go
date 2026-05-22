package modules

import "testing"

func TestDeepMerge(t *testing.T) {
	dst := map[string]any{
		"a": "original",
		"nested": map[string]any{
			"x": 1,
		},
	}
	src := map[string]any{
		"b": "new",
		"nested": map[string]any{
			"y": 2,
		},
	}
	deepMerge(dst, src)

	if dst["a"] != "original" {
		t.Error("existing key 'a' was overwritten")
	}
	if dst["b"] != "new" {
		t.Error("new key 'b' missing after merge")
	}
	nested, _ := dst["nested"].(map[string]any)
	if nested["x"] != 1 {
		t.Error("nested key 'x' lost after merge")
	}
	if nested["y"] != 2 {
		t.Error("nested key 'y' missing after merge")
	}
}

func TestDeepMerge_OverwritesScalarsAndSlices(t *testing.T) {
	dst := map[string]any{
		"scalar": "old",
		"list":   []any{"a", "b"},
	}
	src := map[string]any{
		"scalar": "new",
		"list":   []any{"c"},
	}
	deepMerge(dst, src)
	if dst["scalar"] != "new" {
		t.Errorf("scalar not overwritten: %v", dst["scalar"])
	}
	got, _ := dst["list"].([]any)
	if len(got) != 1 || got[0] != "c" {
		t.Errorf("list not replaced wholesale: %v", dst["list"])
	}
}

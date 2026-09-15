package main

import "testing"

func TestSplitIDs(t *testing.T) {
	values := splitIDs(" asset-a,asset-b, asset-a,,")
	if len(values) != 2 || values[0] != "asset-a" || values[1] != "asset-b" {
		t.Fatalf("split IDs = %#v", values)
	}
}

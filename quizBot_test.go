package main

import (
	"testing"
)

func TestParser(t *testing.T) {
	originalStr := "/copycat testing"
	outputStr := Parser(originalStr)

	if outputStr != "testing" {
		t.Error("Expected: " + "testing 123 but got: " + outputStr)
	}
}

package main

import (
	"testing"
)

func TestParser(t *testing.T) {
	originalStr := "/copycat testing 123"
	outputStr := Parser(originalStr)

	if outputStr != "testing" {
		t.Error("Expected: " + "testing but got: " + outputStr)
	}
}

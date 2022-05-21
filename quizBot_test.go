package main

import (
	"testing"
)

func TestParser(t *testing.T) {
	originalStr := "/copycat testing 123"
	outputStr := Parser(originalStr)

	if outputStr != "testing 123" {
		t.Error("Expected: " + "testing but got: " + outputStr)
	}
}

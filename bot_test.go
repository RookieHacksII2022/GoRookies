package main

import "testing"

func testParserCopycat(t *testing.T) {
	originalStr := "/copycat testing 123"
	outputStr := parserCopycat(originalStr)

	if outputStr != "testing 123" {
		t.Error("Expected: " + "testing 123 but got: " + outputStr)
	}
}

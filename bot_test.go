package bot

import (
	"testing"
)

func TestParserCopycat(t *testing.T) {
	originalStr := "/copycat testing 123"
	outputStr := ParserCopycat(originalStr)

	if outputStr != "testing 123" {
		t.Error("Expected: " + "testing 123 but got: " + outputStr)
	}
}

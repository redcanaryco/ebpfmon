package utils

import (
	"testing"
)

// Tests the RemoveStringColors function
func TestRemoveStringColors(t *testing.T) {
	// Create a string with colors
	str := "[blue]my string[-]"

	// Remove the colors
	result := RemoveStringColors(str)
	if result != "my string" {
		t.Errorf("Expected 'my string', got '%s'", result)
	}

	// Create an empty string with colors
	str = "[red][-]"
	result = RemoveStringColors(str)
	if result != "" {
		t.Errorf("Expected '', got '%s'", result)
	}

	// Test the empty string
	str = ""
	result = RemoveStringColors(str)
	if result != "" {
		t.Errorf("Expected '', got '%s'", result)
	}
}
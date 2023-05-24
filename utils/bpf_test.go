package utils

import (
	"strconv"
	"testing"
)

// Compares the values of two slices to determine if they are equal
func compareSlices(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for index, value := range a {
		if value != b[index] {
			return false
		}
	}

	return true
}

// Tests the ability for this function to convert a string slice to a byte slice
// The string is expected to be a hex string i.e. "0x00"
func TestConvertStringSliceToByteSlice(t *testing.T) {
	// Create a few test strings with hex values
	testStringEmpty := []string{}
	testStringOne := []string{"0x00"}
	testStringTwo := []string{"0x00", "0x01"}
	testStrings := []string{"0x00", "0x01", "0x02", "0x03", "0x04", "0x05"}

	// Test empty string
	result, err := convertStringSliceToByteSlice(testStringEmpty)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected length of 0, got %d", len(result))
	}

	// Test one string
	result, err = convertStringSliceToByteSlice(testStringOne)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !compareSlices(result, []byte{0x00}) {
		t.Errorf("Expected [0x00], got %v", result)
	}

	// Test two strings
	result, err = convertStringSliceToByteSlice(testStringTwo)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !compareSlices(result, []byte{0x00, 0x01}) {
		t.Errorf("Expected [0x00, 0x01], got %v", result)
	}

	// Convert the test strings to a byte slice
	result, err = convertStringSliceToByteSlice(testStrings)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !compareSlices(result, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}) {
		t.Errorf("Expected [0x00, 0x01, 0x02, 0x03, 0x04, 0x05], got %v", result)
	}

}

func TestGetMapEntries(t *testing.T) {
	// Create a new bpf map using bpftool
	_, _, err := RunCmd("sudo", "bpftool", "map", "create", "/sys/fs/bpf/mymap", "type", "hash", "key", "4", "value", "4", "entries", "1024", "name", "testmap")
	if err != nil {
		t.Errorf("Failed to create map, got %v", err)
	}

	// Get the id of the map
	maps, err := GetBpfMapInfo()
	if err != nil {
		t.Errorf("Failed to get bpf map info, got %v", err)
	}

	// Find the map id by matching the map name
	var mapId int
	for _, m := range maps {
		if m.Name == "testmap" {
			mapId = m.Id
		}
	}

	// Add entires to the map using bpftool map update
	_, _, err = RunCmd("sudo", "bpftool", "map", "update", "id", strconv.Itoa(mapId), "key", "0x00", "0x00", "0x00", "0x00", "value", "0x00", "0x00", "0x00", "0x00")

	// Get the map entries
	entries, err := GetBpfMapEntries(mapId)
	if err != nil {
		t.Errorf("Failed to get map entries, got %v", err)
	}

	// Check that the map entries are correct
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
	if !compareSlices(entries[0].Key, []byte{0x00, 0x00, 0x00, 0x00}) {
		t.Errorf("Expected key [0x00, 0x00, 0x00, 0x00], got %v", entries[0].Key)
	}
	if !compareSlices(entries[0].Value, []byte{0x00, 0x00, 0x00, 0x00}) {
		t.Errorf("Expected value [0x00, 0x00, 0x00, 0x00], got %v", entries[0].Value)
	}

	// Delete the map
	_, _, err = RunCmd("sudo", "bpftool", "map", "delete", "id", strconv.Itoa(mapId))
	if err != nil {
		t.Errorf("Failed to delete map, got %v", err)
	}
}
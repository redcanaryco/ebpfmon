package utils

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	log "github.com/sirupsen/logrus"
)

func init() {
	// Set the path to bpftool using the system path
	bpftoolEnvPath, exists := os.LookupEnv("BPFTOOL_PATH")
	if exists {
		_, err := os.Stat(bpftoolEnvPath)
		if err != nil {
			panic(err)
		}
		bpftoolEnvPath, err = filepath.Abs(bpftoolEnvPath)
		if err != nil {
			panic(err)
		}
		BpftoolPath = bpftoolEnvPath
	} else {
		path, err := Which("bpftool")
		if err != nil {
			panic(err)
		}
		BpftoolPath = path
	}

	// Set simple logging for tests
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

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

// Generate a random string of a given length
func generateRandomString(length int) string {
	var result string
	for i := 0; i < length; i++ {
		result += strconv.Itoa(rand.Intn(10))
	}
	return result
}

func TestGetMapEntries(t *testing.T) {
	sysfsPath := "/sys/fs/bpf"
	mapName := generateRandomString(10)
	mapPinPath := sysfsPath + "/" + mapName

	// Create a new bpf map using bpftool
	_, stderr, err := RunCmd("sudo", "bpftool", "map", "create", mapPinPath, "type", "hash", "key", "4", "value", "4", "entries", "1024", "name", mapName)
	if err != nil {
		t.Errorf("Failed to create map at %s, got %v - %s", mapPinPath, err, stderr)
		return
	}

	// Get the id of the map
	maps, err := GetBpfMapInfo()
	if err != nil {
		t.Errorf("Failed to get bpf map info, got %v", err)
		return
	}

	// Find the map id by matching the map name
	var mapId int = 0
	for _, m := range maps {
		if m.Name == mapName {
			mapId = m.Id
		}
	}

	if mapId == 0 {
		t.Errorf("Failed to find map id for map %s", mapName)
		return
	}

	// Add entires to the map using bpftool map update
	_, _, err = RunCmd("sudo", "bpftool", "map", "update", "id", strconv.Itoa(mapId), "key", "0x00", "0x00", "0x00", "0x00", "value", "0x00", "0x00", "0x00", "0x00")

	// Get the map entries
	entries, err := GetBpfMapEntries(mapId)
	if err != nil {
		t.Errorf("Failed to get map entries, got %v", err)
		return
	}

	// Check that the map entries are correct
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
		return
	}
	if !compareSlices(entries[0].Key, []byte{0x00, 0x00, 0x00, 0x00}) {
		t.Errorf("Expected key [0x00, 0x00, 0x00, 0x00], got %v", entries[0].Key)
		return
	}
	if !compareSlices(entries[0].Value, []byte{0x00, 0x00, 0x00, 0x00}) {
		t.Errorf("Expected value [0x00, 0x00, 0x00, 0x00], got %v", entries[0].Value)
		return
	}

	// Delete the map
	// _, _, err = RunCmd("sudo", "bpftool", "map", "delete", "id", strconv.Itoa(mapId))
	// if err != nil {
	// 	t.Errorf("Failed to delete map, got %v", err)
	// }

	// Delete the map pin
	_, _, err = RunCmd("sudo", "rm", "-rf", mapPinPath)
	if err != nil {
		t.Errorf("Failed to delete map pin, got %v", err)
		return
	}

}

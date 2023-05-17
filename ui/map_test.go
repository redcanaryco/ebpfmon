package ui

import (
	"strings"
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

func TestPadBytes(t *testing.T) {
	widths := []int{DataWidth8, DataWidth16, DataWidth32, DataWidth64}
	for _, width := range widths {
		// If byte slice is empty we should always get a byte slice back
		result := padBytes([]byte{}, width)
		if len(result) != 0 {
			t.Errorf("Expected length of %d, got %d", width, len(result))
		}

		result = padBytes([]byte("A"), width)
		if len(result)%width != 0 {
			t.Errorf("Expected length of %d, got %d", width, len(result))
		}

		if strings.Index(string(result), "A") != 0 {
			t.Errorf("Expected 'A' at index 0, got '%s'", string(result))
		}

		// If byte slice has 5 values we should various amounts of padding
		result = padBytes([]byte("AAAAA"), width)
		if len(result)%width != 0 {
			t.Errorf("Expected length of %d, got %d", width, len(result))
		}

		switch width {
		case DataWidth8:
		case DataWidth16:
		case DataWidth32:
			if compareSlices(result, []byte("AAAAA")) {
				t.Errorf("Expected 'A' at index 0, got '%s'", string(result))
			}
			break
		case DataWidth64:
			if !compareSlices(result, []byte("AAAAA\x00\x00\x00")) {
				t.Errorf("Expected %v, got %v", []byte("AAAAA\x00\x00\x00"), result)
			}
			break
		}

		// If byte slice has 9 values we should various amounts of padding
		result = padBytes([]byte("AAAAAAAAA"), width)
		if len(result)%width != 0 {
			t.Errorf("Expected length of %d, got %d", width, len(result))
		}

		switch width {
		case DataWidth8:
			if !compareSlices(result, []byte("AAAAAAAAA")) {
				t.Errorf("Expected %v, got '%v'", []byte("AAAAAAAAA"), result)
			}
			break
		case DataWidth16:
			if !compareSlices(result, []byte("AAAAAAAAA\x00")) {
				t.Errorf("Expected %v, got %v", []byte("AAAAAAAAA\x00"), result)
			}
			break
		case DataWidth32:
			if !compareSlices(result, []byte("AAAAAAAAA\x00\x00\x00")) {
				t.Errorf("Expected %v, got %v", []byte("AAAAAAAAA\x00\x00\x00"), result)
			}
			break
		case DataWidth64:
			if !compareSlices(result, []byte("AAAAAAAAA\x00\x00\x00\x00\x00\x00\x00")) {
				t.Errorf("Expected %v, got %v", []byte("AAAAAAAAA\x00\x00\x00\x00\x00\x00\x00"), result)
			}
			break
		}
	}
}

func validResultOne(result string, format int, width int, endianness int) (string, bool) {
	// Declare map of valid values
	m := map[int]map[int]map[int]string{
		Hex: {
			DataWidth8: map[int]string{
				Little: "0x41",
				Big:    "0x41",
			},
			DataWidth16: map[int]string{
				Little: "0x0041",
				Big:    "0x4100",
			},
			DataWidth32: map[int]string{
				Little: "0x00000041",
				Big:    "0x41000000",
			},
			DataWidth64: map[int]string{
				Little: "0x0000000000000041",
				Big:    "0x4100000000000000",
			},
		},
		Decimal: {
			DataWidth8: map[int]string{
				Little: "65",
				Big:    "65",
			},
			DataWidth16: map[int]string{
				Little: "65",
				Big:    "16640",
			},
			DataWidth32: map[int]string{
				Little: "65",
				Big:    "1090519040",
			},
			DataWidth64: map[int]string{
				Little: "65",
				Big:    "4683743612465315840",
			},
		},
		Raw: {
			DataWidth8: map[int]string{
				Little: "[65]",
				Big:    "[65]",
			},
			DataWidth16: map[int]string{
				Little: "[65 0]",
				Big:    "[65 0]",
			},
			DataWidth32: map[int]string{
				Little: "[65 0 0 0]",
				Big:    "[65 0 0 0]",
			},
			DataWidth64: map[int]string{
				Little: "[65 0 0 0 0 0 0 0]",
				Big:    "[65 0 0 0 0 0 0 0]",
			},
		},
	}
	if result != m[format][width][endianness] {
		return m[format][width][endianness], false
	}
	return "", true
}

// Write a test that can test the applyFormat function
// The test should handle all the variations of format,
// width, and endianness along with different values
func TestApplyFormat(t *testing.T) {
	formats := []int{Hex, Decimal, Raw}
	widths := []int{DataWidth8, DataWidth16, DataWidth32, DataWidth64}
	endiannesses := []int{Little, Big}

	// Test empty byte slice for all cases
	for _, format := range formats {
		for _, width := range widths {
			for _, endianness := range endiannesses {
				result := applyFormat(format, width, endianness, []byte{})
				if result != "" {
					t.Errorf("Expected empty string, got '%s'", result)
				}
			}
		}
	}

	for _, format := range formats {
		for _, width := range widths {
			for _, endianness := range endiannesses {
				result := applyFormat(format, width, endianness, []byte{0x41})
				expected, success := validResultOne(result, format, width, endianness)
				if !success {
					t.Errorf("(format, width, endian) %v, %v, %v, Expected %v, got %v", format, width, endianness, expected, result)
				}
			}
		}
	}
}

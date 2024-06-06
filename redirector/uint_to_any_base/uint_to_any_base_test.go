package uint_to_any_base

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestNewNumeralSystem(t *testing.T) {
	testCases := []struct {
		base    uint
		digits  string
		want    *NumeralSystem
		wantErr error
	}{
		{0, "", nil, fmt.Errorf("base cannot be 0 or 1")},
		{1, "", nil, fmt.Errorf("base cannot be 0 or 1")},
		{3, "011", nil, fmt.Errorf("all digits must be unique")},
		{2, "1", nil, fmt.Errorf("not enough digits for base 2")},
		{2, "012", nil, fmt.Errorf("too many digits for base 2")},
		{2, "01", &NumeralSystem{2, strings.Repeat("1", 64), "01", map[rune]uint{'0': 0, '1': 1}}, nil},
		{10, "0123456789", &NumeralSystem{10, fmt.Sprint(uint(math.MaxUint)), "0123456789", map[rune]uint{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9}}, nil},
		{16, "012345D6789ABCEF", &NumeralSystem{16, fmt.Sprintf("%X", uint(math.MaxUint)), "0123456789ABCDEF", map[rune]uint{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9, 'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14, 'F': 15}}, nil},
		{8, "01234567", &NumeralSystem{8, "1777777777777777777777", "01234567", map[rune]uint{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7}}, nil},
		{3, "210", &NumeralSystem{3, "11112220022122120101211020120210210211220", "012", map[rune]uint{'0': 0, '1': 1, '2': 2}}, nil},
		{3, "120", &NumeralSystem{3, "11112220022122120101211020120210210211220", "012", map[rune]uint{'0': 0, '1': 1, '2': 2}}, nil},
	}

	for _, tc := range testCases {
		got, err := NewNumeralSystem(tc.base, tc.digits)
		if err != nil && tc.wantErr == nil || err==nil && tc.wantErr != nil {
			t.Errorf("NewNumeralSystem(%v, %v) error = %v, wantErr %v", tc.base, tc.digits, err, tc.wantErr)
		}
		if err == nil && got != nil {
			if got.Base != tc.want.Base || got.largestNum != tc.want.largestNum || got.digitsList != tc.want.digitsList {
				t.Errorf("NewNumeralSystem(%v, %v) = %v, want %v", tc.base, tc.digits, got, tc.want)
			}
			for k, v := range got.digitsMap {
				if tc.want.digitsMap[k] != v {
					t.Errorf("NewNumeralSystem(%v, %v) map incorrect at key %v", tc.base, tc.digits, k)
				}
			}
		}
	}
}

func TestStringToIntegerAndIntegerToString(t *testing.T) {
	ns10, _ := NewNumeralSystem(10, "0123456789")
	ns3, _ := NewNumeralSystem(3, "012")
	testCases := []struct {
		number uint
		want   string
		ns *NumeralSystem
	}{
		{0, "0", ns10},
		{1, "1", ns10},
		{12345, "12345", ns10},
		{math.MaxUint, fmt.Sprint(uint(math.MaxUint)), ns10},
		{1234567890, "1234567890", ns10},
		{0, "0", ns3},
		{1, "1", ns3},
		{2, "2", ns3},
		{3, "10", ns3},
		{10, "101", ns3},
		{uint(math.MaxUint) - 1, "11112220022122120101211020120210210211212", ns3},
	}

	for _, tc := range testCases {
		// Test IntegerToString
		gotString, err := tc.ns.IntegerToString(tc.number)
		if err != nil {
			t.Errorf("IntegerToString(%v) error = %v", tc.number, err)
		}
		if gotString != tc.want {
			t.Errorf("IntegerToString(%v) = %v, want %v", tc.number, gotString, tc.want)
		}

		// Test StringToInteger
		gotUint, err := tc.ns.StringToInteger(tc.want)
		if err != nil {
			t.Errorf("StringToInteger(%v) error = %v", tc.want, err)
		}
		if gotUint != tc.number {
			t.Errorf("StringToInteger(%v) = %v, want %v", tc.want, gotUint, tc.number)
		}
	}

	_, err := testCases[0].ns.StringToInteger("A") // Invalid character
	if err == nil {
		t.Errorf("StringToInteger(\"A\") should have returned an error")
	}
}

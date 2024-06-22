package uint_to_any_base

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestNewNumeralSystem(t *testing.T) {
	testCases := []struct {
		base    uint32
		digits  string
		want    *NumeralSystem
		wantErr error
	}{
		{0, "", nil, fmt.Errorf("base cannot be 0 or 1")},
		{1, "", nil, fmt.Errorf("base cannot be 0 or 1")},
		{3, "011", nil, fmt.Errorf("all digits must be unique")},
		{2, "1", nil, fmt.Errorf("not enough digits for base 2")},
		{2, "012", nil, fmt.Errorf("too many digits for base 2")},
		{2, "01", &NumeralSystem{2, 4, strings.Repeat("1", 32), "0000", []rune("01"), map[rune]uint32{'0': 0, '1': 1}}, nil},
		{10, "0123456789", &NumeralSystem{10, 4, fmt.Sprint(uint32(math.MaxUint32)), "0000", []rune("0123456789"), map[rune]uint32{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9}}, nil},
		{16, "012345D6789ABCEF", &NumeralSystem{16, 4, fmt.Sprintf("%X", uint32(math.MaxUint32)), "0000", []rune("0123456789ABCDEF"), map[rune]uint32{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9, 'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14, 'F': 15}}, nil},
		{8, "01234567", &NumeralSystem{8, 4, "37777777777", "0000", []rune("01234567"), map[rune]uint32{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7}}, nil},
		{3, "210", &NumeralSystem{3, 4, "102002022201221111210", "0000", []rune("012"), map[rune]uint32{'0': 0, '1': 1, '2': 2}}, nil},
		{3, "120", &NumeralSystem{3, 4, "102002022201221111210", "0000", []rune("012"), map[rune]uint32{'0': 0, '1': 1, '2': 2}}, nil},
	}

	for _, tc := range testCases {
		var padding uint32 = 4
		if tc.want != nil {
			padding = tc.want.padding
		}
		got, err := NewNumeralSystem(tc.base, tc.digits, padding)
		if err != nil && tc.wantErr == nil || err==nil && tc.wantErr != nil {
			t.Errorf("NewNumeralSystem(%v, %v) error = %v, wantErr %v", tc.base, tc.digits, err, tc.wantErr)
		}
		if err == nil && got != nil {
			if got.Base != tc.want.Base || got.LargestNum != tc.want.LargestNum || string(got.digitsList) != string(tc.want.digitsList) {
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
	ns10, _ := NewNumeralSystem(10, "0123456789", 4)
	ns3, _ := NewNumeralSystem(3, "012", 4)
	testCases := []struct {
		number uint32
		want   string
		ns *NumeralSystem
	}{
		{0, "0000", ns10},
		{1, "0001", ns10},
		{12345, "12345", ns10},
		{math.MaxUint32, fmt.Sprint(uint(math.MaxUint32)), ns10},
		{1234567890, "1234567890", ns10},
		{0, "0000", ns3},
		{1, "0001", ns3},
		{2, "0002", ns3},
		{3, "0010", ns3},
		{10, "0101", ns3},
		{uint32(math.MaxUint32) - 1, "102002022201221111202", ns3},
	}

	for _, tc := range testCases {
		// Test IntegerToString
		gotString, err := tc.ns.IntegerToString(tc.number)
		if err != nil {
			t.Errorf("IntegerToString(%v) error = %v", tc.number, err)
		} else
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

func TestIncr(t *testing.T) {
	ns10, _ := NewNumeralSystem(10, "0123456789", 4)
	ns3, _ := NewNumeralSystem(3, "012", 4)
	ns8, _ := NewNumeralSystem(8, "01234567", 4)
	testCases := []struct {
		number string
		want   string
		ns *NumeralSystem
	}{
		{"0000", "0001", ns10},
		{"0001", "0002", ns10},
		{"12345", "12346", ns10},
		{"9999", "00000", ns10},
		{"102002022201221111200", "102002022201221111201", ns3},
		{"102002022201221111201", "102002022201221111202", ns3},
		{"102002022201221111202", "102002022201221111210", ns3},
		{"7777777777", "00000000000", ns8},
		{"0000000000", "0000000001", ns8},
		{"0000000001", "0000000002", ns8},
		{"0000000007", "0000000010", ns8},
		{"0000000010", "0000000011", ns8},
		{"0000000017", "0000000020", ns8},
	}

	for _, tc := range testCases {
		got, err := tc.ns.Incr(tc.number)
		if err != nil {
			t.Errorf("Incr(%v) error = %v", tc.number, err)
		} else
		if got != tc.want {
			t.Errorf("Incr(%v) = %v, want %v", tc.number, got, tc.want)
		}
	}
}
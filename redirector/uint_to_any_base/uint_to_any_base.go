package uint_to_any_base

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
)

type NumeralSystem struct {
	Base       uint
	largestNum string
	digitsList string
	digitsMap  map[rune]uint
}


func hasAllUniqueRunes(s string) bool {
	set := make(map[rune]struct{}, len(s))
	for _, r := range s {
		if _, ok := set[r]; ok {
			return false
		}
		set[r] = struct{}{}
	}
	return true
}

func NewNumeralSystem(base uint, digits string) (*NumeralSystem, error) {
	var err error
	switch {
	case base < 2:
		return nil, fmt.Errorf("base cannot be 0 or 1")
	case len(digits) < int(base):
		return nil, fmt.Errorf("not enough digits for base %v", base)
	case len(digits) > int(base):
		return nil, fmt.Errorf("too many digits for base %v", base)
	case !hasAllUniqueRunes(digits):
		return nil, fmt.Errorf("all digits must be unique")
	}

	tmpDigitsSlice := strings.Split(digits, "")
	slices.SortFunc(tmpDigitsSlice, func(a string, b string) int {
		return strings.Compare(a, b)
	})
	digits = strings.Join(tmpDigitsSlice, "")

	result := NumeralSystem{Base: base, largestNum: strings.Repeat(digits[len(digits)-1:], 64), digitsList: digits, digitsMap: make(map[rune]uint, len(digits))}
	for i, r := range digits {
		result.digitsMap[r] = uint(i)
	}

	result.largestNum, err = result.IntegerToString(math.MaxUint)
	return &result, err
}

func (system *NumeralSystem) StringToInteger(number string) (uint, error) {
	if len(number) > len(system.largestNum) || len(number) == len(system.largestNum) && number > system.largestNum {
		return 0, errors.New("Overflow: the number as a string is too large to be converted to uint")
	}
	var digitValue uint = 1
	var result uint = 0
	tmpslice := []rune(number)
	for i := len(tmpslice) - 1; i >= 0; i-- {
		val, ok := system.digitsMap[tmpslice[i]]
		if !ok {
			return result, fmt.Errorf("invalid numeral %v at position %v for the given base", tmpslice[i], i)
		}
		result +=  val * digitValue
		digitValue *= system.Base
	}
	return result, nil
}

func (system *NumeralSystem) IntegerToString(number uint) (string, error) {
	if number == 0 {
		return system.digitsList[:1], nil
	}
	var result strings.Builder
	for number > 0 {
		result.WriteByte(system.digitsList[number % system.Base])
		number /= system.Base
	}

	// Reverse the string as it was built backwards
	reversedResult := []rune(result.String())
	for i, j := 0, len(reversedResult)-1; i < j; i, j = i+1, j-1 {
		reversedResult[i], reversedResult[j] = reversedResult[j], reversedResult[i]
	}
	return string(reversedResult), nil
}

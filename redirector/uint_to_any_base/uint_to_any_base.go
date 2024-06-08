package uint_to_any_base

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
)

type NumeralSystem struct {
	Base       uint32
	padding   uint32
	LargestNum string
	Zero string
	digitsList []rune
	digitsMap  map[rune]uint32
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

func NewNumeralSystem(base uint32, digits string, padding uint32) (*NumeralSystem, error) {
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

	result := NumeralSystem{
		Base: base,
		padding: padding,
		LargestNum: strings.Repeat(digits[len(digits)-1:], 64),
		Zero: strings.Repeat(digits[:1], int(padding)),
		digitsList: []rune(digits),
		digitsMap: make(map[rune]uint32, len(digits)),
	}
	for i, r := range digits {
		result.digitsMap[r] = uint32(i)
	}

	result.LargestNum, err = result.IntegerToString(math.MaxUint32)
	return &result, err
}

func (system *NumeralSystem) StringToInteger(number string) (uint32, error) {
	if len(number) > len(system.LargestNum) || len(number) == len(system.LargestNum) && number > system.LargestNum {
		return 0, errors.New("overflow: the number as a string is too large to be converted to uint32")
	}
	var digitValue uint32 = 1
	var result uint32 = 0
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

func (system *NumeralSystem) IntegerToString(number uint32) (string, error) {
	if number == 0 {
		size := int(max(1, system.padding))
		return strings.Repeat(string(system.digitsList[:1]), size), nil
	}
	var resultBuilder strings.Builder
	for number > 0 {
		resultBuilder.WriteRune(system.digitsList[number % system.Base])
		number /= system.Base
	}

	// Reverse the string as it was built backwards
	reversedResult := []rune(resultBuilder.String())
	for i, j := 0, len(reversedResult)-1; i < j; i, j = i+1, j-1 {
		reversedResult[i], reversedResult[j] = reversedResult[j], reversedResult[i]
	}
	result := string(reversedResult)
	if uint32(len(result)) < system.padding {
		result = strings.Repeat(string(system.digitsList[:1]), int(system.padding) - len(result)) + result
	}
	return result, nil
}

func (system *NumeralSystem) incr(number []rune) []rune {
	if len(number) == 0 {
		return []rune{system.digitsList[0]}
	}
	digitValue := int(system.digitsMap[number[len(number) - 1]])
	if digitValue < len(system.digitsMap) - 1 {
		return append(number[:len(number) - 1], system.digitsList[digitValue+1])
	}
	return append(system.incr(number[:len(number) - 1]), system.digitsList[0])
} 

func (system *NumeralSystem) Incr(number string) (string, error) {
	return string(system.incr([]rune(number))), nil
}
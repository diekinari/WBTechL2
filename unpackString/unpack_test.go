package unpackString

import (
	"log"
	"testing"
)

type TestCase struct {
	input    string
	expected string
}

func TestUnpack(t *testing.T) {
	tCases := []TestCase{
		{
			input:    "a4bc2d5e",
			expected: "aaaabccddddde",
		},
		{
			input:    "abcd",
			expected: "abcd",
		},
		{
			input:    "45",
			expected: "",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "qwe\\4\\5",
			expected: "qwe45",
		},
		{
			input:    "qwe\\45",
			expected: "qwe44444",
		},
	}

	for caseNum, testCase := range tCases {
		actual, err := unpackString(testCase.input)
		if err != nil && testCase.input != "45" {
			t.Errorf("[%d] unexpected error: %v input: %v", caseNum, err, testCase.input)
		}
		if actual != testCase.expected {
			t.Errorf("[%d] wrong results: got %+v, expected %+v",
				caseNum, actual, testCase.expected)
		}
		log.Printf("[%d] right results: got %+v, expected %+v", caseNum, actual, testCase.expected)
	}
}

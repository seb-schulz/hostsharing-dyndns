package main

import (
	"testing"
)

func TestArgonPasswordValidator(t *testing.T) {
	for _, testCase := range []struct {
		v              passwordValidator
		password       []byte
		expectedResult bool
	}{
		{
			argonPasswordValidator([]byte{230, 104, 139, 86, 35, 176, 125, 179, 79, 26, 88, 17, 178, 50, 28, 214, 27, 165, 105, 84, 225, 141, 44, 123, 62, 196, 70, 127, 108, 203, 144, 225}, []byte("123"), 1, 1, 1, 32),
			[]byte(".test."), true,
		}, {
			argonPasswordValidator([]byte{110, 229, 10, 51, 153, 202, 41, 137, 248, 79, 231, 236, 127, 187, 80, 94, 249, 57, 166, 194, 156, 43, 72, 188, 139, 201, 240, 81, 164, 31, 152, 176}, []byte("abc"), 1, 1, 1, 32),
			[]byte(".test."), true,
		}, {
			argonPasswordValidator([]byte{230, 104, 139, 86, 35, 176, 125, 179, 79, 26, 88, 17, 178, 50, 28, 214, 27, 165, 105, 84, 225, 141, 44, 123, 62, 196, 70, 127, 108, 203, 144, 225}, []byte("123"), 1, 1, 1, 32),
			[]byte("..test.."), false,
		},
	} {
		if got := testCase.v(testCase.password); got != testCase.expectedResult {
			t.Errorf("test result not as expected: %v instead of %v", got, testCase.expectedResult)
		}
	}
}

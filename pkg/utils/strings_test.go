package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitNoEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: []string{},
		},
		{
			name:     "single value",
			input:    "foo",
			sep:      ",",
			expected: []string{"foo"},
		},
		{
			name:     "multiple values",
			input:    "foo,bar,baz",
			sep:      ",",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "leading/trailing separator",
			input:    ",foo,bar,",
			sep:      ",",
			expected: []string{"foo", "bar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := SplitNoEmpty(tc.input, tc.sep)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

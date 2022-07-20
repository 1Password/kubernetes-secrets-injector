package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeBuildVersion(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"test stable version": {
			input:    "1.5.6",
			expected: "1050601",
		},
		"test beta version": {
			input:    "0.1.0-beta.01",
			expected: "0010001",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, makeBuildVersion(tc.input))
		})
	}
}

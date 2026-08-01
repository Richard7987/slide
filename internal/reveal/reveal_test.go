package reveal_test

import (
	"reflect"
	"testing"

	"github.com/maaslalani/slides/internal/reveal"
)

func TestSteps(t *testing.T) {
	tt := []struct {
		name     string
		slide    string
		expected []string
	}{
		{
			name:     "no marker",
			slide:    "# Title\n\nSome content",
			expected: []string{"# Title\n\nSome content"},
		},
		{
			name:  "two markers",
			slide: "# Title\n<!-- pause -->\n- one\n<!-- pause -->\n- two",
			expected: []string{
				"# Title\n",
				"# Title\n\n- one\n",
				"# Title\n\n- one\n\n- two",
			},
		},
		{
			name:  "marker with surrounding spaces",
			slide: "intro\n   <!--   pause   -->  \nrest",
			expected: []string{
				"intro\n",
				"intro\n\nrest",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := reveal.Steps(tc.slide)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("Steps(%q) = %#v, want %#v", tc.slide, got, tc.expected)
			}
		})
	}
}

package bib

import "testing"

func TestSplitAuthors(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  []Author
	}{
		{"empty", "", nil},
		{"single last-first", "Smith, John", []Author{{First: "John", Last: "Smith"}}},
		{"single first-last", "John Smith", []Author{{First: "John", Last: "Smith"}}},
		{
			"two authors mixed forms",
			"Smith, John and Jane Doe",
			[]Author{{First: "John", Last: "Smith"}, {First: "Jane", Last: "Doe"}},
		},
		{
			"and others",
			"Smith, John and others",
			[]Author{{First: "John", Last: "Smith"}, {Others: true}},
		},
		{"single word", "Cher", []Author{{Last: "Cher"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAuthors(tt.field)
			if !authorsEqual(got, tt.want) {
				t.Errorf("splitAuthors(%q) = %+v, want %+v", tt.field, got, tt.want)
			}
		})
	}
}

func authorsEqual(a, b []Author) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFormatAuthors(t *testing.T) {
	tests := []struct {
		name    string
		authors []Author
		want    string
	}{
		{"none", nil, ""},
		{"one", []Author{{First: "John", Last: "Smith"}}, "J. Smith"},
		{
			"two",
			[]Author{{First: "John", Last: "Smith"}, {First: "Jane", Last: "Doe"}},
			"J. Smith and J. Doe",
		},
		{
			"three",
			[]Author{{First: "A", Last: "One"}, {First: "B", Last: "Two"}, {First: "C", Last: "Three"}},
			"A. One, B. Two, and C. Three",
		},
		{
			"and others",
			[]Author{{First: "John", Last: "Smith"}, {Others: true}},
			"J. Smith et al.",
		},
		{
			"more than six collapses to et al.",
			[]Author{
				{First: "A", Last: "One"}, {First: "B", Last: "Two"}, {First: "C", Last: "Three"},
				{First: "D", Last: "Four"}, {First: "E", Last: "Five"}, {First: "F", Last: "Six"},
				{First: "G", Last: "Seven"},
			},
			"A. One et al.",
		},
		{"multi-word first name", []Author{{First: "John Michael", Last: "Smith"}}, "J. M. Smith"},
		{"no first name", []Author{{Last: "Cher"}}, "Cher"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAuthors(tt.authors)
			if got != tt.want {
				t.Errorf("formatAuthors(%+v) = %q, want %q", tt.authors, got, tt.want)
			}
		})
	}
}

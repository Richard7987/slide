package bib

import "testing"

func TestFormatReference(t *testing.T) {
	tests := []struct {
		name  string
		entry Entry
		want  string
	}{
		{
			name: "article",
			entry: Entry{Type: "article", Key: "s2020", Fields: map[string]string{
				"author": "Smith, John", "title": "A Great Paper",
				"journal": "Journal of Things", "volume": "5", "number": "2",
				"pages": "100--110", "year": "2020",
			}},
			want: `[1] J. Smith, "A Great Paper," *Journal of Things*, vol. 5, no. 2, pp. 100–110, 2020.`,
		},
		{
			name: "book",
			entry: Entry{Type: "book", Key: "k1997", Fields: map[string]string{
				"author": "Knuth, Donald E.", "title": "The Art of Computer Programming",
				"address": "Boston", "publisher": "Addison-Wesley", "year": "1997",
			}},
			want: `[2] D. E. Knuth, *The Art of Computer Programming*, Boston: Addison-Wesley, 1997.`,
		},
		{
			name: "inproceedings",
			entry: Entry{Type: "inproceedings", Key: "p2019", Fields: map[string]string{
				"author": "Doe, Jane", "title": "Conference Paper",
				"booktitle": "Proc. Big Conf", "address": "NYC", "year": "2019", "pages": "1--5",
			}},
			want: `[3] J. Doe, "Conference Paper," in *Proc. Big Conf*, NYC, 2019, pp. 1–5.`,
		},
		{
			name: "misc",
			entry: Entry{Type: "misc", Key: "m2022", Fields: map[string]string{
				"author": "Roe, Sam", "title": "A Blog Post", "year": "2022",
				"url": "https://example.com",
			}},
			want: `[4] S. Roe, "A Blog Post," 2022, [Online]. Available: https://example.com.`,
		},
		{
			name: "misc with doi fallback",
			entry: Entry{Type: "misc", Key: "d2023", Fields: map[string]string{
				"author": "Roe, Sam", "title": "A Dataset", "year": "2023", "doi": "10.1/x",
			}},
			want: `[5] S. Roe, "A Dataset," 2023, [Online]. Available: https://doi.org/10.1/x.`,
		},
		{
			name: "missing fields are simply omitted",
			entry: Entry{Type: "article", Key: "bare", Fields: map[string]string{
				"title": "Untitled",
			}},
			want: `[6] "Untitled,".`,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatReference(i+1, tt.entry)
			if got != tt.want {
				t.Errorf("FormatReference(%d, %+v) =\n  %q\nwant\n  %q", i+1, tt.entry, got, tt.want)
			}
		})
	}
}

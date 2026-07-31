package bib_test

import (
	"strings"
	"testing"

	"github.com/maaslalani/slides/internal/bib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleBib = `
@article{smith2020,
  author = {Smith, John and Doe, Jane},
  title = {A Great Paper About {IEEE} Things},
  journal = {Journal of Things},
  year = {2020},
  volume = {5},
  number = {2},
  pages = {100--110}
}

@book{knuth1997,
  author = "Knuth, Donald E.",
  title = "The Art of Computer Programming",
  publisher = {Addison-Wesley},
  year = 1997
}

@misc{accented,
  author = {M{\"u}ller, Hans and G{\'o}mez, Ana},
  title = {Some {\'e}scaped Ch{\c c}ars},
  year = {2021}
}

@comment{ignore me}

@string{foo = "bar"}
`

func TestParse(t *testing.T) {
	entries, err := bib.Parse(strings.NewReader(sampleBib))
	require.NoError(t, err)
	require.Len(t, entries, 3)

	smith := entries["smith2020"]
	assert.Equal(t, "article", smith.Type)
	assert.Equal(t, "smith2020", smith.Key)
	assert.Equal(t, "Smith, John and Doe, Jane", smith.Fields["author"])
	assert.Equal(t, "A Great Paper About IEEE Things", smith.Fields["title"])
	assert.Equal(t, "Journal of Things", smith.Fields["journal"])
	assert.Equal(t, "2020", smith.Fields["year"])
	assert.Equal(t, "5", smith.Fields["volume"])
	assert.Equal(t, "100--110", smith.Fields["pages"])

	knuth := entries["knuth1997"]
	assert.Equal(t, "book", knuth.Type)
	assert.Equal(t, "Knuth, Donald E.", knuth.Fields["author"])
	assert.Equal(t, "The Art of Computer Programming", knuth.Fields["title"])
	assert.Equal(t, "1997", knuth.Fields["year"])

	accented := entries["accented"]
	assert.Equal(t, "Müller, Hans and Gómez, Ana", accented.Fields["author"])
	assert.Equal(t, "Some éscaped Chçars", accented.Fields["title"])

	_, hasComment := entries["ignore me"]
	assert.False(t, hasComment)
}

func TestParse_EmptyOrGarbage(t *testing.T) {
	entries, err := bib.Parse(strings.NewReader("not a bibtex file at all"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

package guard

import (
	"reflect"
	"strings"
	"testing"
)

// TestEachCommandAFindRunsIsRead: a word no line fixes before an -exec is read as
// one too, and the -exec after it is still read as its own.
func TestEachCommandAFindRunsIsRead(t *testing.T) {
	for line, want := range map[string][]string{
		`find . -exec tee x {} \;`:                      {"tee x \x00"},
		`find "$D" -exec tee x {} \;`:                   {"-exec tee x \x00", "tee x \x00"},
		`find "$D" -name "$N" -exec tee x {} \;`:        {"-name \x00 -exec tee x \x00", "-exec tee x \x00", "tee x \x00"},
		`find . -exec tee x {} + -exec tee y {} \;`:     {"tee x \x00", "tee y \x00"},
		`find "$D" -exec tee x {} \; -exec tee y {} \;`: {"-exec tee x \x00", "tee x \x00", "tee y \x00"},
	} {
		segs, err := tokenize(line)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, a := range findActions(segs[0], 1) {
			got = append(got, strings.Join(a.seg.tokens, " "))
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s runs %q, want %q", line, got, want)
		}
	}
}

// TestReadingAFindsCommandsCostsWhatItsWordsDo: many words no line fixes before
// one command are not each read as an action holding all the rest.
func TestReadingAFindsCommandsCostsWhatItsWordsDo(t *testing.T) {
	line := "find" + strings.Repeat(` "$D"`, 4000) + ` -exec tee x {} \;`
	segs, err := tokenize(line)
	if err != nil {
		t.Fatal(err)
	}
	words, last := 0, ""
	for _, a := range findActions(segs[0], 1) {
		words += len(a.seg.tokens)
		last = strings.Join(a.seg.tokens, " ")
	}
	if words > (maxUnknownSites+1)*len(segs[0].tokens) || last != "tee x \x00" {
		t.Errorf("the find's commands hold %d words of a line of %d, the last %q", words, len(segs[0].tokens), last)
	}
}

package recordid

import (
	"strings"
	"testing"
)

// TestHandleInTextFindsGenealogy pins what counts as a citation inside a
// principle's statement (spc-2609020626042471): any numbered record family or a
// condition identity as a whole token in any case, and never a placeholder, a
// or a longer word; a slug after the number still names the record.
func TestHandleInTextFindsGenealogy(t *testing.T) {
	for s, want := range map[string]string{
		"as adr-43 ruled":                    "adr-43",
		"(itd-79)":                           "itd-79",
		"ADR-6's concern":                    "ADR-6",
		"see rdi-2609020000000009.":          "rdi-2609020000000009",
		"while cond-2609020626047525 held":   "cond-2609020626047525",
		"rfm-1":                              "rfm-1",
		"the file itd-4-a-slug.md":           "itd-4",
		"the note mem-2610111200001234 says": "mem-2610111200001234",
	} {
		got, ok := HandleInText(s)
		if !ok || got != want {
			t.Errorf("HandleInText(%q) = %q, %v; want %q", s, got, ok, want)
		}
	}
	for _, s := range []string{
		"a placeholder itd-N is not a citation",
		"no handle here",
		"xadr-4 is a longer word",
		"iss-",
		"a memo-12 is a longer word",
	} {
		if got, ok := HandleInText(s); ok {
			t.Errorf("HandleInText(%q) found %q", s, got)
		}
	}
}

// TestIDNumberAtReadsOnlyARecordIDsDigits pins the digits that are a numbered
// id's own: all digits, directly after a family HandleInText finds or run, the
// family opening a word, nothing alphanumeric after them.
func TestIDNumberAtReadsOnlyARecordIDsDigits(t *testing.T) {
	at := func(text, num string) bool {
		i := strings.LastIndex(text, num)
		return IDNumberAt(text, i, i+len(num))
	}
	for text, num := range map[string]string{
		"fixes iss-2611060835372141.":    "2611060835372141",
		"run-2611100851356293":           "2611100851356293",
		"(ITD-79)":                       "79",
		"the file spc-12-a-slug.md":      "12",
		"by mem-2610111200001234, which": "2610111200001234",
		"iss-12345678901234567890":       "12345678901234567890",
	} {
		if !at(text, num) {
			t.Errorf("IDNumberAt(%q, %q) = false; want true", text, num)
		}
	}
	for text, num := range map[string]string{
		"a bare 2611060835372141":      "2611060835372141",
		"iss-2611060835372141abc":      "2611060835372141abc",
		"iss-12345678abcdef":           "12345678abcdef",
		"xiss-2611060835372141":        "2611060835372141",
		"a-iss-2611060835372141":       "2611060835372141",
		"lane-2611060835372141":        "2611060835372141",
		"iss-123456789012345678901":    "123456789012345678901",
		"iss-2611060835372141 runs on": "261106083537214",
	} {
		if at(text, num) {
			t.Errorf("IDNumberAt(%q, %q) = true; want false", text, num)
		}
	}
	if IDNumberAt("iss-1", 4, 9) || IDNumberAt("iss-1", -1, 2) {
		t.Error("IDNumberAt read outside its text")
	}
}

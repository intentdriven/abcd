package recordid

import "testing"

// TestHandleInTextFindsGenealogy pins what counts as a citation inside a
// principle's statement (spc-2609020626042471): any numbered record family or a
// condition identity as a whole token in any case, and never a placeholder, a
// or a longer word; a slug after the number still names the record.
func TestHandleInTextFindsGenealogy(t *testing.T) {
	for s, want := range map[string]string{
		"as adr-43 ruled":                  "adr-43",
		"(itd-79)":                         "itd-79",
		"ADR-6's concern":                  "ADR-6",
		"see rdi-2609020000000009.":        "rdi-2609020000000009",
		"while cond-2609020626047525 held": "cond-2609020626047525",
		"rfm-1":                            "rfm-1",
		"the file itd-4-a-slug.md":         "itd-4",
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
	} {
		if got, ok := HandleInText(s); ok {
			t.Errorf("HandleInText(%q) found %q", s, got)
		}
	}
}

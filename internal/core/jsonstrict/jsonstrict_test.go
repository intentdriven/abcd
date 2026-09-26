package jsonstrict

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode"
)

// TestNoDuplicateKeysRefusesEverySpellingEncodingJSONBindsAsOne pins the
// refusal against the way encoding/json matches a key to a struct field:
// case-insensitively, under Unicode simple folding, the last match winning. A
// key the decoder would bind to the same field as an earlier key is the same
// key, whatever its bytes (iss-2609252251311346).
func TestNoDuplicateKeysRefusesEverySpellingEncodingJSONBindsAsOne(t *testing.T) {
	cases := []struct{ name, doc string }{
		{"exact repeat", `{"a":1,"a":2}`},
		{"ASCII case twin", `{"verificationResult":"REJECT","VerificationResult":"PROMOTE"}`},
		{"kill-switch case twin", `{"disabled":false,"Disabled":true}`},
		{"all capitals", `{"disabled":false,"DISABLED":true}`},
		{"map keys differing in case", `{"domains":{"PII":{"rules":["a"]},"pii":{"rules":["b"]}}}`},
		{"unicode escape spelling the twin", `{"verificationResult":"REJECT","\u0056erificationResult":"PROMOTE"}`},
		{"unicode escape spelling the exact key", `{"state":"a","\u0073tate":"b"}`},
		{"escaped solidus spelling the same key", `{"a/b":1,"a\/b":2}`},
		{"long s folds to s", `{"disabled":false,"di\u017fabled":true}`},
		{"Kelvin sign folds to k", `{"kind":"a","\u212aind":"b"}`},
		{"non-ASCII case twin", `{"\u00e9t\u00e9":1,"\u00c9T\u00c9":2}`},
		{"nested object", `{"a":{"b":{"state":"active","State":"dormant"}}}`},
		{"object inside an array", `{"a":[1,{"k":1,"K":2}]}`},
		{"top-level array of objects", `[{"x":1},{"y":1,"Y":2}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !json.Valid([]byte(tc.doc)) {
				t.Fatalf("fixture is not valid JSON: %s", tc.doc)
			}
			err := NoDuplicateKeys([]byte(tc.doc))
			if err == nil {
				t.Fatalf("%s was admitted; encoding/json would read one of its keys last-wins", tc.doc)
			}
			var dk *DuplicateKeyError
			if !errors.As(err, &dk) {
				t.Fatalf("the refusal is not a *DuplicateKeyError: %T %v", err, err)
			}
			if !strings.Contains(err.Error(), "duplicate key") {
				t.Fatalf("the refusal does not say what it refused: %v", err)
			}
		})
	}
}

// TestNoDuplicateKeysAdmitsDistinctKeys guards the other side: keys encoding/json
// keeps apart stay apart, the same key in sibling or nested objects is not a
// repeat, and a malformed document is left for the decoder to report.
func TestNoDuplicateKeysAdmitsDistinctKeys(t *testing.T) {
	for _, doc := range []string{
		`{}`,
		`[]`,
		`"scalar"`,
		`{"a":1,"b":2}`,
		`{"a":{"k":1},"b":{"k":1}}`,
		`{"k":{"k":{"k":1}}}`,
		`[{"k":1},{"k":2}]`,
		`{"work_minutes":1,"workminutes":2,"work-minutes":3}`,
		`{"s":1,"t":2}`,
		`{"a":1,"a":2`, // malformed: the unmarshal reports it
		``,
	} {
		if err := NoDuplicateKeys([]byte(doc)); err != nil {
			t.Errorf("%q was refused: %v", doc, err)
		}
	}
}

// TestNoDuplicateKeysNamesTheRepeatAndWhereItSits: the refusal names the later
// spelling, the earlier one it collides with, and the enclosing path, so a
// reader can find both.
func TestNoDuplicateKeysNamesTheRepeatAndWhereItSits(t *testing.T) {
	err := NoDuplicateKeys([]byte(`{"domains":{"PII":{"rules":["a"]},"pii":{"rules":["b"]}}}`))
	var dk *DuplicateKeyError
	if !errors.As(err, &dk) {
		t.Fatalf("want a *DuplicateKeyError, got %v", err)
	}
	if dk.Key != "pii" || dk.First != "PII" || strings.Join(dk.Path, ".") != "domains" {
		t.Fatalf("got key %q first %q path %q", dk.Key, dk.First, dk.Path)
	}
	for _, want := range []string{`"pii"`, `"PII"`, "domains"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not name %s", err, want)
		}
	}

	err = NoDuplicateKeys([]byte(`{"a":[{"k":1,"k":2}]}`))
	if !errors.As(err, &dk) || strings.Join(dk.Path, ".") != "a.[]" || dk.Key != "k" || dk.First != "k" {
		t.Fatalf("an array element's path is not named: %+v", dk)
	}
}

// TestFoldingMatchesEncodingJSONAndEqualFold holds the fold to the decoder it
// guards: every case twin the refusal names is one encoding/json really binds to
// the same field, last-wins, and the fold agrees with strings.EqualFold, the
// Unicode simple folding the decoder's own foldName is documented against.
func TestFoldingMatchesEncodingJSONAndEqualFold(t *testing.T) {
	type target struct {
		Disabled bool   `json:"disabled"`
		Kind     string `json:"kind"`
	}
	for _, doc := range []string{
		`{"disabled":false,"Disabled":true}`,
		`{"disabled":false,"di\u017fabled":true}`,
	} {
		var got target
		if err := json.Unmarshal([]byte(doc), &got); err != nil {
			t.Fatal(err)
		}
		if !got.Disabled {
			t.Fatalf("premise: encoding/json no longer binds the twin in %s; revisit the fold", doc)
		}
	}
	var got target
	if err := json.Unmarshal([]byte(`{"kind":"a","\u212aind":"b"}`), &got); err != nil || got.Kind != "b" {
		t.Fatalf("premise: encoding/json no longer binds the Kelvin-sign twin (got %q, %v)", got.Kind, err)
	}

	runes := []rune{'a', 'A', 'k', 'K', '\u212a', 's', 'S', '\u017f', '\u00e9', '\u00c9',
		'\u03c3', '\u03a3', '\u03c2', '\u00df', '\u1e9e', 'i', 'I', '\u0130', '\u0131', '_', '-', '1'}
	for _, a := range runes {
		for _, b := range runes {
			x, y := string(a), string(b)
			if (fold(x) == fold(y)) != strings.EqualFold(x, y) {
				t.Errorf("fold(%q)==fold(%q) is %v, strings.EqualFold says %v",
					x, y, fold(x) == fold(y), strings.EqualFold(x, y))
			}
		}
	}
	// Every rune in a fold orbit shares one folded form.
	for r := rune(0); r <= unicode.MaxRune; r += 97 {
		for o := unicode.SimpleFold(r); o != r; o = unicode.SimpleFold(o) {
			if fold(string(o)) != fold(string(r)) {
				t.Fatalf("%U and %U share a fold orbit but fold apart", r, o)
			}
		}
	}
}

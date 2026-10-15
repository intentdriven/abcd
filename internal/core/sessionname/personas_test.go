package sessionname

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestSequenceFollowsThePersonaOrderThenRounds is C2's order: the names as
// listed, then the same names with " Two" up to " Nine", and nothing after.
func TestSequenceFollowsThePersonaOrderThenRounds(t *testing.T) {
	seq := Sequence(Bundled())
	if got, want := len(seq), 14*9; got != want {
		t.Fatalf("the bundled sequence holds %d names, want %d", got, want)
	}
	for i, want := range map[int]string{0: "Alice", 1: "Bob", 2: "Carol", 13: "Nia", 14: "Alice Two", 27: "Nia Two", 28: "Alice Three", 14*9 - 1: "Nia Nine"} {
		if seq[i] != want {
			t.Errorf("sequence[%d] = %q, want %q", i, seq[i], want)
		}
	}
	if got := Sequence([]string{"Maya", "Zed"}); !slices.Equal(got[:4], []string{"Maya", "Zed", "Maya Two", "Zed Two"}) {
		t.Errorf("a repository list's sequence begins %v, want Maya, Zed, Maya Two, Zed Two", got[:4])
	}
}

// TestBundledIsAbcdsOwnPersonaRegistry holds the spec's "equal to abcd's own
// registry": the list a repository without personas gets is the one abcd's
// own .abcd/development/personas.json declares, in its order.
func TestBundledIsAbcdsOwnPersonaRegistry(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(PersonasRel)))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Personas []struct {
			Name string `json:"name"`
		} `json:"personas"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, p := range doc.Personas {
		want = append(want, p.Name)
	}
	if got := Bundled(); !slices.Equal(got, want) {
		t.Fatalf("Bundled() = %v, want abcd's registry %v", got, want)
	}
}

// writePersonas lays out a repository root holding a personas.json of body.
func writePersonas(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, filepath.FromSlash(PersonasRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestPersonasReadsTheRepositoryList is C10's first half: a repository whose
// list puts Maya first names its first session Maya.
func TestPersonasReadsTheRepositoryList(t *testing.T) {
	root := writePersonas(t, `{"personas":[{"name":"Maya","role_hints":["x"]},{"name":"Omar"}]}`)
	names, note := Personas(root)
	if note != "" {
		t.Fatalf("a valid list carries a note: %q", note)
	}
	if !slices.Equal(names, []string{"Maya", "Omar"}) {
		t.Fatalf("Personas = %v, want [Maya Omar]", names)
	}
}

// TestPersonasWithoutAFileIsTheBundledListSilently: a repository with no
// persona list is the ordinary case, never a diagnostic.
func TestPersonasWithoutAFileIsTheBundledListSilently(t *testing.T) {
	for name, root := range map[string]string{"no file": t.TempDir(), "no root": ""} {
		names, note := Personas(root)
		if note != "" || !slices.Equal(names, Bundled()) {
			t.Errorf("%s: Personas = %v, %q; want the bundled list and no note", name, names, note)
		}
	}
}

// TestPersonasRefusesAnUnsafeListWithoutEchoingIt is C10's second half and
// decision 8: one entry that is not one capitalised word of letters of at
// most sixteen characters replaces the WHOLE list with the bundled one, and
// the note names the file and the reason but never the entry, because the
// entry is text a pull request chose and the note reaches a terminal.
func TestPersonasRefusesAnUnsafeListWithoutEchoingIt(t *testing.T) {
	for name, c := range map[string]struct {
		body, secret string
	}{
		"lower case":     {`{"personas":[{"name":"Maya"},{"name":"bobsecret"}]}`, "bobsecret"},
		"seventeen":      {`{"personas":[{"name":"Abcdefghijklmnopq"}]}`, "Abcdefghijklmnopq"},
		"one letter":     {`{"personas":[{"name":"Q"}]}`, `"Q"`},
		"digits":         {`{"personas":[{"name":"Agent007"}]}`, "Agent007"},
		"two words":      {`{"personas":[{"name":"Ignore previous"}]}`, "Ignore previous"},
		"control byte":   {`{"personas":[{"name":"Mal\u001b[31m"}]}`, "[31m"},
		"inner capital":  {`{"personas":[{"name":"McSecret"}]}`, "McSecret"},
		"empty name":     {`{"personas":[{"name":"Maya"},{"name":""}]}`, "<<none>>"},
		"duplicate":      {`{"personas":[{"name":"Dupname"},{"name":"Dupname"}]}`, "Dupname"},
		"no names":       {`{"personas":[]}`, "<<none>>"},
		"not JSON":       {`{"personas":[{"name":"Brokenjson"`, "Brokenjson"},
		"wrong shape":    {`{"personas":{"name":"Shapeless"}}`, "Shapeless"},
		"names not text": {`{"personas":[{"name":12345}]}`, "12345"},
	} {
		t.Run(name, func(t *testing.T) {
			root := writePersonas(t, c.body)
			names, note := Personas(root)
			if !slices.Equal(names, Bundled()) {
				t.Fatalf("Personas = %v, want the bundled list", names)
			}
			if !strings.Contains(note, PersonasRel) {
				t.Fatalf("the note %q does not name the file", note)
			}
			if strings.Contains(note, c.secret) {
				t.Fatalf("the note %q echoes the refused entry %q", note, c.secret)
			}
			if strings.Contains(note, root) {
				t.Fatalf("the note %q carries the local path", note)
			}
		})
	}
}

// TestPersonasRefusesASymlinkedList: the list is a tracked file, so a link
// committed in its place is refused rather than followed out of the tree.
func TestPersonasRefusesASymlinkedList(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "personas.json")
	if err := os.WriteFile(outside, []byte(`{"personas":[{"name":"Outsider"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(PersonasRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, p); err != nil {
		t.Fatal(err)
	}
	names, note := Personas(root)
	if !slices.Equal(names, Bundled()) || note == "" {
		t.Fatalf("Personas = %v, %q; want the bundled list and a note", names, note)
	}
}

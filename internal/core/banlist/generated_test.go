package banlist

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func needGrep(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("grep"); err != nil {
		t.Skip("grep unavailable: the enforcement engine cannot be driven")
	}
}

// phraseHits scans text with one phrase pattern through the enforcing engine.
func phraseHits(t *testing.T, pattern, text string) []Hit {
	t.Helper()
	hits, err := ScanText([]KeyedPattern{{Key: "k", Pattern: pattern}}, []byte(text))
	if err != nil {
		t.Fatalf("ScanText: %v", err)
	}
	return hits
}

// TestPhrasePatternMatchesThePhraseAndNotAWordContainingIt pins the projection a
// generated entry carries: the literal phrase, case-insensitive, whitespace-flexible,
// bounded by neighbours that are not letters or digits — and never `\b`, which the
// guard's POSIX engine does not define and RE2 reads as ASCII-only.
func TestPhrasePatternMatchesThePhraseAndNotAWordContainingIt(t *testing.T) {
	needGrep(t)
	p, err := PhrasePattern("Quiet Harbour Study")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p, `\b`) {
		t.Fatalf("pattern uses \\b: %q", p)
	}
	for _, tc := range []struct {
		text string
		want bool
	}{
		{"see the Quiet Harbour Study today", true},
		{"QUIET HARBOUR STUDY", true},
		{"quiet\tharbour   study.", true},
		{"(quiet harbour study)", true},
		{"quiet harbour studying", false},
		{"unquiet harbour study", false},
		{"quiet harbour", false},
	} {
		got := len(phraseHits(t, p, tc.text)) > 0
		if got != tc.want {
			t.Errorf("%q: matched=%v, want %v", tc.text, got, tc.want)
		}
	}
}

// TestPhrasePatternIsUnicodeAware pins the two ASCII hazards the spec names. A
// non-breaking space between the words must not evade the ban, a non-ASCII initial
// must still be found at a word start (where `\b` under RE2 finds no boundary), and
// case folds for non-ASCII letters, which the C-locale engine cannot fold itself.
func TestPhrasePatternIsUnicodeAware(t *testing.T) {
	needGrep(t)
	p, err := PhrasePattern("Élan Özgür")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		text string
		want bool
	}{
		{"Élan Özgür", true},
		{"about élan özgür here", true},
		{"ÉLAN ÖZGÜR", true},
		{"Élan Özgür", true},
		{"xÉlan Özgür", false},
	} {
		got := len(phraseHits(t, p, tc.text)) > 0
		if got != tc.want {
			t.Errorf("%q: matched=%v, want %v", tc.text, got, tc.want)
		}
	}
}

// TestPhrasePatternRefusesAFragment: a phrase with fewer than three letters or digits
// would ban ordinary text (iss-2609212142568782's hazard), so it is refused.
func TestPhrasePatternRefusesAFragment(t *testing.T) {
	for _, bad := range []string{"", "  ", "ab", "a b", "--"} {
		if _, err := PhrasePattern(bad); err == nil {
			t.Errorf("PhrasePattern(%q) accepted a fragment", bad)
		}
	}
}

// TestPhrasePatternEscapesMetacharacters: a title is a literal, never an expression.
func TestPhrasePatternEscapesMetacharacters(t *testing.T) {
	needGrep(t)
	p, err := PhrasePattern("C++ (draft) v1.2 [x] a|b $5^")
	if err != nil {
		t.Fatal(err)
	}
	if fault, _ := checkPattern(p); fault != faultNone {
		t.Fatalf("the guard's engine refuses the escaped pattern (fault %d)", fault)
	}
	if len(phraseHits(t, p, "C++ (draft) v1.2 [x] a|b $5^")) == 0 {
		t.Error("the literal phrase does not match itself")
	}
	if len(phraseHits(t, p, "Cxx (draft) v1x2 [x] a|b $5^")) != 0 {
		t.Error("a metacharacter was read as an expression")
	}
}

// TestScanTextReportsKeysAndOffsetsOnly: the scan names the entry and where it hit,
// never the text it matched.
func TestScanTextReportsKeysAndOffsetsOnly(t *testing.T) {
	needGrep(t)
	p, err := PhrasePattern("quiet harbour")
	if err != nil {
		t.Fatal(err)
	}
	hits, err := ScanText([]KeyedPattern{{Key: "sources/k1/title", Pattern: p}}, []byte("line one\nsee Quiet Harbour\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Key != "sources/k1/title" || hits[0].Line != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Offset < 9 || hits[0].Offset > 13 {
		t.Errorf("offset %d is not on line 2's match", hits[0].Offset)
	}
}

func readStore(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(PrivateRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestSyncGeneratedBlockKeepsHandLinesAndRegeneratesItsOwn pins the owned block:
// hand-written entries outside the markers survive every sync, the block is replaced
// whole, and an empty projection empties the block rather than leaving stale entries.
func TestSyncGeneratedBlockKeepsHandLinesAndRegeneratesItsOwn(t *testing.T) {
	needGrep(t)
	root := t.TempDir()
	writePrivate(t, root, privateFormatDecl+"\n# my own\nhand-key widgetworks\n")

	p1, _ := PhrasePattern("quiet harbour")
	p2, _ := PhrasePattern("second study")
	res, err := SyncGeneratedBlock(root, "sources", []KeyedPattern{{Key: "sources/a/title", Pattern: p1}, {Key: "sources/b/title", Pattern: p2}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entries != 2 || !res.Changed {
		t.Fatalf("result = %+v", res)
	}
	body := readStore(t, root)
	if !strings.Contains(body, "hand-key widgetworks") || !strings.Contains(body, "sources/a/title ") {
		t.Fatalf("store after sync:\n%s", body)
	}
	rep, err := ListPrivate(root)
	if err != nil || len(rep.Malformed) != 0 || len(rep.Entries) != 3 {
		t.Fatalf("store does not read back healthy: %+v %v", rep, err)
	}

	// Idempotent: the same projection writes nothing.
	res, err = SyncGeneratedBlock(root, "sources", []KeyedPattern{{Key: "sources/a/title", Pattern: p1}, {Key: "sources/b/title", Pattern: p2}})
	if err != nil || res.Changed {
		t.Fatalf("second sync changed the store: %+v %v", res, err)
	}

	// A shrunk projection drops the dropped key and keeps the hand line.
	if _, err := SyncGeneratedBlock(root, "sources", []KeyedPattern{{Key: "sources/b/title", Pattern: p2}}); err != nil {
		t.Fatal(err)
	}
	body = readStore(t, root)
	if strings.Contains(body, "sources/a/title") || !strings.Contains(body, "sources/b/title") || !strings.Contains(body, "hand-key widgetworks") {
		t.Fatalf("store after shrink:\n%s", body)
	}

	// An empty projection empties the block.
	if _, err := SyncGeneratedBlock(root, "sources", nil); err != nil {
		t.Fatal(err)
	}
	body = readStore(t, root)
	if strings.Contains(body, "sources/b/title") || !strings.Contains(body, "hand-key widgetworks") {
		t.Fatalf("store after empty sync:\n%s", body)
	}
	if strings.Count(body, generatedBeginPrefix) != 1 {
		t.Fatalf("block markers duplicated or lost:\n%s", body)
	}
}

// TestSyncGeneratedBlockCreatesTheStoreOnlyWhenThereIsSomethingToBan: an absent
// store and nothing to project is left absent; something to project creates a keyed
// store at 0600.
func TestSyncGeneratedBlockCreatesTheStoreOnlyWhenThereIsSomethingToBan(t *testing.T) {
	needGrep(t)
	root := t.TempDir()
	res, err := SyncGeneratedBlock(root, "sources", nil)
	if err != nil || res.Changed {
		t.Fatalf("empty projection on an absent store: %+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(PrivateRelPath))); !os.IsNotExist(err) {
		t.Fatalf("an empty projection created the store: %v", err)
	}
	p, _ := PhrasePattern("quiet harbour")
	res, err = SyncGeneratedBlock(root, "sources", []KeyedPattern{{Key: "sources/a/title", Pattern: p}})
	if err != nil || !res.Created {
		t.Fatalf("result = %+v %v", res, err)
	}
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(PrivateRelPath)))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("store mode: %v %v", fi, err)
	}
	if !strings.HasPrefix(readStore(t, root), privateFormatDecl+"\n") {
		t.Fatal("created store does not declare the keyed format on line 1")
	}
}

// TestSyncGeneratedBlockRefusals: the block's own integrity and the store's format
// are checked before anything is written, and no refusal quotes a pattern.
func TestSyncGeneratedBlockRefusals(t *testing.T) {
	needGrep(t)
	p, _ := PhrasePattern("quiet harbour")
	good := []KeyedPattern{{Key: "sources/a/title", Pattern: p}}
	for _, tc := range []struct {
		name  string
		store string
		in    []KeyedPattern
		want  error
	}{
		{"legacy store with entries", "somepattern\n", good, ErrLegacyStore},
		{"begin without end", privateFormatDecl + "\n" + generatedBeginPrefix + "sources >>>\n", good, ErrMalformedStore},
		{"two blocks", privateFormatDecl + "\n" + generatedBeginPrefix + "sources >>>\n" + generatedEndPrefix + "sources <<<\n" + generatedBeginPrefix + "sources >>>\n" + generatedEndPrefix + "sources <<<\n", good, ErrMalformedStore},
		{"key outside the namespace", privateFormatDecl + "\n", []KeyedPattern{{Key: "other/a", Pattern: p}}, ErrInvalidKey},
		{"hand key collides", privateFormatDecl + "\nsources/a/title x\n", good, ErrDuplicateKey},
		{"unusable pattern", privateFormatDecl + "\n", []KeyedPattern{{Key: "sources/a/title", Pattern: "[a-z-.]secretvalue"}}, ErrInvalidPattern},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writePrivate(t, root, tc.store)
			_, err := SyncGeneratedBlock(root, "sources", tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if strings.Contains(err.Error(), "secretvalue") || strings.Contains(err.Error(), "harbour") {
				t.Fatalf("refusal quotes a pattern: %v", err)
			}
			if readStore(t, root) != tc.store {
				t.Fatal("a refused sync wrote the store")
			}
		})
	}
}

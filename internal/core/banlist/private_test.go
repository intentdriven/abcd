package banlist

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// grepMatch is the agreement test's matcher, and it is the REAL enforcement
// engine: each parsed pattern goes to `grep -iE` on STDIN (never argv) against the
// candidate text, exactly as the committed hook drives it. A Go-regexp stand-in
// would make every RE2-versus-POSIX-ERE divergence invisible by construction —
// which is precisely the class the corpora exist to pin. Skips when grep is
// unavailable; an engine that refuses a pattern is an error, never a miss, so the
// fail-safe verdict is asserted rather than silently read as clean.
func grepMatch(t *testing.T, entries []rawEntry, text string) (key string, err error) {
	t.Helper()
	bin, lookErr := exec.LookPath("grep")
	if lookErr != nil {
		t.Skip("grep unavailable: the enforcement engine cannot be driven")
	}
	candidate := filepath.Join(t.TempDir(), "candidate")
	if werr := os.WriteFile(candidate, []byte(text), 0o600); werr != nil {
		t.Fatal(werr)
	}
	for _, e := range entries {
		if e.unparsed {
			return "", errors.New("banlist line does not parse")
		}
		cmd := exec.Command(bin, "-iqE", "-f", "-", "--", candidate)
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		cmd.Stdin = strings.NewReader(e.pattern + "\n")
		runErr := cmd.Run()
		if runErr == nil {
			return e.key, nil
		}
		var ee *exec.ExitError
		if errors.As(runErr, &ee) && ee.ExitCode() == 1 {
			continue
		}
		return "", errors.New("banlist line is not a usable pattern")
	}
	return "", nil
}

func writePrivate(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(PrivateRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// readCorpus reads one shared fixture banlist — the same bytes hook_test.go feeds
// the committed shell hook.
func readCorpus(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// keysOf lists the parsed keys in file order.
func keysOf(entries []rawEntry) []string {
	var keys []string
	for _, e := range entries {
		if e.unparsed {
			continue
		}
		keys = append(keys, e.key)
	}
	return keys
}

// TestParseAgreesWithTheHookOnTheKeyedCorpus is half the format-agreement test:
// the Go parser reads testdata/parse-corpus.txt — the same file hook_test.go feeds
// the committed shell hook — and must recognise the format declaration, derive the
// same keys, skip the same comment and blank lines, and reach the same verdict on
// both probe halves when the patterns are driven through the real grep.
func TestParseAgreesWithTheHookOnTheKeyedCorpus(t *testing.T) {
	entries, keyed, err := parse(readCorpus(t, "parse-corpus.txt"))
	if err != nil {
		t.Fatalf("the keyed corpus must parse: %v", err)
	}
	if !keyed {
		t.Fatal("the corpus's first line declares the keyed format; the parser did not recognise it")
	}
	for _, e := range entries {
		if e.unparsed {
			t.Fatalf("line %d does not parse; every line of the keyed corpus must", e.line)
		}
	}
	if got := strings.Join(keysOf(entries), ","); got != strings.Join(corpusKeys, ",") {
		t.Fatalf("keys = %v, want %v", keysOf(entries), corpusKeys)
	}

	for _, tc := range corpusMustBlock {
		hit, err := grepMatch(t, entries, tc.text)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if hit != tc.key {
			t.Errorf("%s: matched key %q, want %q (the hook names %q)", tc.name, hit, tc.key, tc.key)
		}
	}
	for _, tc := range corpusMustPass {
		hit, err := grepMatch(t, entries, tc.text)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if hit != "" {
			t.Errorf("%s: entry %q matched permitted content; the hook lets this commit through", tc.name, hit)
		}
	}
}

// TestParseReadsALegacyStoreAsWholeLines is the other half, and the detector for
// the key-splitting leak: without the format declaration NO line is split, so the
// keys are all synthetic and no part of a line's text is ever exposed as a key.
func TestParseReadsALegacyStoreAsWholeLines(t *testing.T) {
	data := readCorpus(t, "parse-corpus-legacy.txt")
	entries, keyed, err := parse(data)
	if err != nil {
		t.Fatalf("the legacy corpus must parse: %v", err)
	}
	if keyed {
		t.Fatal("a store with no format declaration was read as keyed")
	}
	if got := strings.Join(keysOf(entries), ","); got != strings.Join(legacyKeys, ",") {
		t.Fatalf("keys = %v, want %v (every legacy key is synthetic)", keysOf(entries), legacyKeys)
	}
	for _, e := range entries {
		for _, field := range legacyFirstFields {
			if e.key == field {
				t.Errorf("line %d is keyed by %q — on a legacy line the first field is part of the PATTERN, which on a real store is the secret", e.line, field)
			}
			if strings.HasPrefix(e.pattern, field) && !strings.Contains(e.pattern, " ") {
				t.Errorf("line %d's pattern lost its first field: %q", e.line, e.pattern)
			}
		}
	}

	for _, tc := range legacyMustBlock {
		hit, err := grepMatch(t, entries, tc.text)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if hit != tc.key {
			t.Errorf("%s: matched key %q, want %q", tc.name, hit, tc.key)
		}
	}
	for _, tc := range legacyMustPass {
		hit, err := grepMatch(t, entries, tc.text)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if hit != "" {
			t.Errorf("%s: entry %q matched; a legacy line is a whole-line pattern, never a key plus a remainder", tc.name, hit)
		}
	}
}

// TestParseReportsUnusableKeyedLinesByNumberOnly pins the unusable-line classes on
// the shared malformed corpus: a keyed line indented with U+00A0, one separated by
// U+000B, one with no separator, and one whose pattern the engine refuses. All four
// are reported by LINE NUMBER; the three that do not parse yield no key at all,
// because their first field may be the secret.
func TestParseReportsUnusableKeyedLinesByNumberOnly(t *testing.T) {
	root := t.TempDir()
	writePrivate(t, root, string(readCorpus(t, "parse-corpus-malformed.txt")))

	rep, err := ListPrivate(root)
	if err != nil {
		t.Fatalf("ListPrivate: %v", err)
	}
	if !rep.Keyed {
		t.Error("Keyed = false for a store whose first line declares the format")
	}
	var gotLines []int
	gotLines = append(gotLines, rep.Malformed...)
	if len(gotLines) != len(malformedUnusableLines) {
		t.Fatalf("Malformed = %v, want %v", rep.Malformed, malformedUnusableLines)
	}
	for i, want := range malformedUnusableLines {
		if gotLines[i] != want {
			t.Fatalf("Malformed = %v, want %v", rep.Malformed, malformedUnusableLines)
		}
	}
	var keys []string
	for _, e := range rep.Entries {
		keys = append(keys, e.Key)
	}
	if strings.Join(keys, ",") != strings.Join(malformedKeys, ",") {
		t.Errorf("Entries = %v, want %v (a line that does not parse has no key)", keys, malformedKeys)
	}
	blob, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"unclosed", "partnerco", "nbsp-key", "vt-key"} {
		if strings.Contains(string(blob), leak) {
			t.Errorf("the report echoes %q from an unusable line: %s", leak, blob)
		}
	}
}

// TestAddPrivateCreatesTheStoreAndListsKeysOnly pins AC6 for the private layer:
// the store is created on first add (0600, under the gitignored local tier, with
// the format declaration as its first line), and nothing a surface can render
// carries the pattern value. The whole report is marshalled and searched for the
// pattern — the assertion is about the DATA, so a later field addition cannot
// quietly reintroduce the leak.
func TestAddPrivateCreatesTheStoreAndListsKeysOnly(t *testing.T) {
	root := t.TempDir()
	const pattern = `widgetworks`
	res, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: pattern})
	if err != nil {
		t.Fatalf("AddPrivate: %v", err)
	}
	if res.Key != "widget-partner" {
		t.Errorf("Key = %q", res.Key)
	}

	path := filepath.Join(root, filepath.FromSlash(PrivateRelPath))
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("store not created: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("store mode = %04o, want 0600 (it holds the private patterns)", perm)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if first, _, _ := strings.Cut(string(body), "\n"); first != privateFormatDecl {
		t.Errorf("first line = %q, want the format declaration %q — a store abcd creates is never read as legacy", first, privateFormatDecl)
	}

	rep, err := ListPrivate(root)
	if err != nil {
		t.Fatalf("ListPrivate: %v", err)
	}
	if !rep.Present || !rep.Keyed || len(rep.Entries) != 1 || rep.Entries[0].Key != "widget-partner" {
		t.Fatalf("report = %+v", rep)
	}
	blob, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), pattern) {
		t.Errorf("the private report carries the pattern value: %s", blob)
	}
}

// TestListPrivateReportsAnAbsentStore pins AC4's Go-side counterpart: absent is a
// reported state, never an empty success that reads like "nothing is banned".
func TestListPrivateReportsAnAbsentStore(t *testing.T) {
	rep, err := ListPrivate(t.TempDir())
	if err != nil {
		t.Fatalf("ListPrivate: %v", err)
	}
	if rep.Present {
		t.Errorf("Present = true for an absent store")
	}
	if rep.Path != PrivateRelPath {
		t.Errorf("Path = %q, want %q", rep.Path, PrivateRelPath)
	}
	// The reach caveat is carried on the report unconditionally so a surface that
	// only summarises the JSON can relay it verbatim rather than paraphrase it.
	if rep.Reach != PrivateReachNote {
		t.Errorf("Reach = %q, want the canonical reach note", rep.Reach)
	}
}

// TestAddPrivateRefusals covers the input contract. Every refusal is checked for
// leakage too: an error message is output, so it may name the key and never the
// pattern.
func TestAddPrivateRefusals(t *testing.T) {
	root := t.TempDir()
	writePrivate(t, root, privateFormatDecl+"\nwidget-partner   widgetworks\n")

	cases := []struct {
		name    string
		key     string
		pattern string
		want    error
		secret  string
	}{
		{"duplicate key", "widget-partner", `otherthing`, ErrDuplicateKey, "otherthing"},
		{"key with a metacharacter", `widget*`, `otherthing`, ErrInvalidKey, "otherthing"},
		{"empty key", "", `otherthing`, ErrInvalidKey, "otherthing"},
		{"empty pattern", "empty", "", ErrInvalidPattern, ""},
		{"uncompilable pattern", "broken", `[unclosed`, ErrInvalidPattern, "unclosed"},
		{"newline in the pattern", "inject", "otherthing\nsecond-key  x", ErrInvalidPattern, "otherthing"},
		{"carriage return in the pattern", "inject", "otherthing\rmore", ErrInvalidPattern, "otherthing"},
		{"newline in the key", "two\nlines", `otherthing`, ErrInvalidKey, "otherthing"},
		{"perl escape the engine does not implement", "perlish", `\dwidgetworks`, ErrInvalidPattern, "widgetworks"},
		{"inline flag group the engine does not implement", "flagged", `(?i)widgetworks`, ErrInvalidPattern, "widgetworks"},
		{"ere-invalid bracket range", "badrange", `[a-z-.]widgetworks`, ErrInvalidPattern, "widgetworks"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: tc.key, Pattern: tc.pattern})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tc.secret != "" && strings.Contains(err.Error(), tc.secret) {
				t.Errorf("error message echoes the pattern value: %v", err)
			}
		})
	}
}

// TestAddPrivateRejectsNewlineInjectionAtTheStore is the bidirectional half of the
// newline refusal: the store is unchanged, so a rejected pattern cannot have
// written a second line (an entry the user never asked for, or a wedge-everything
// payload) on its way to being refused.
func TestAddPrivateRejectsNewlineInjectionAtTheStore(t *testing.T) {
	root := t.TempDir()
	body := privateFormatDecl + "\nwidget-partner   widgetworks\n"
	path := writePrivate(t, root, body)

	if _, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "injected", Pattern: "aaa\nwedge  ."}); !errors.Is(err, ErrInvalidPattern) {
		t.Fatalf("err = %v, want ErrInvalidPattern", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Errorf("store changed by a refused add:\n%q", got)
	}
}

// TestPrivateMutationsRefuseANonEmptyLegacyStore pins the one migration decision
// the verbs will not make for the user: a legacy store's lines are whole-line
// patterns, so writing a KEY<space>PATTERN line into it would silently change what
// every OTHER line means to a reader that then sees a declaration. Both mutations
// refuse and name the one line the user must add.
func TestPrivateMutationsRefuseANonEmptyLegacyStore(t *testing.T) {
	root := t.TempDir()
	body := string(readCorpus(t, "parse-corpus-legacy.txt"))
	path := writePrivate(t, root, body)

	_, addErr := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"})
	if !errors.Is(addErr, ErrLegacyStore) {
		t.Errorf("add: err = %v, want ErrLegacyStore", addErr)
	}
	if !strings.Contains(addErr.Error(), privateFormatDecl) {
		t.Errorf("add refusal does not name the declaration line to add: %v", addErr)
	}
	if _, err := RemovePrivate(root, "entry-8"); !errors.Is(err, ErrLegacyStore) {
		t.Errorf("remove: err = %v, want ErrLegacyStore", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Errorf("a refused mutation changed the store:\n%q", got)
	}
}

// TestAddPrivateAdoptsTheFormatOnAnEntrylessStore is the other side: a store with
// no entries has no whole-line pattern to reinterpret, so the add is safe and
// declares the format itself rather than refusing a user who has only a comment.
func TestAddPrivateAdoptsTheFormatOnAnEntrylessStore(t *testing.T) {
	root := t.TempDir()
	path := writePrivate(t, root, "# my local notes\n")

	if _, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"}); err != nil {
		t.Fatalf("AddPrivate: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := privateFormatDecl + "\n# my local notes\nwidget-partner widgetworks\n"
	if string(got) != want {
		t.Errorf("store =\n%q\nwant\n%q", got, want)
	}
}

// TestRemovePrivateDropsOneLineAndLeavesTheRestByteIdentical pins the edit
// contract: a removal is a line deletion, so hand-written comments, alignment,
// and ordering survive it.
func TestRemovePrivateDropsOneLineAndLeavesTheRestByteIdentical(t *testing.T) {
	root := t.TempDir()
	const body = "# abcd-banlist: keyed\n# local banlist\n\nwidget-partner   widgetworks\nlab-host         alice-laptop\\.example\\.com\n\nlab-ip           192\\.0\\.2\\.17\n"
	path := writePrivate(t, root, body)

	res, err := RemovePrivate(root, "lab-host")
	if err != nil {
		t.Fatalf("RemovePrivate: %v", err)
	}
	if res.Key != "lab-host" {
		t.Errorf("Key = %q", res.Key)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "# abcd-banlist: keyed\n# local banlist\n\nwidget-partner   widgetworks\n\nlab-ip           192\\.0\\.2\\.17\n"
	if string(got) != want {
		t.Errorf("store after removal:\n%q\nwant:\n%q", got, want)
	}

	if _, err := RemovePrivate(root, "lab-host"); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("removing a gone key: err = %v, want ErrUnknownKey", err)
	}
	if _, err := RemovePrivate(t.TempDir(), "lab-host"); !errors.Is(err, ErrNoStore) {
		t.Errorf("removing from an absent store: err = %v, want ErrNoStore", err)
	}
}

// TestListPrivateReportsInertPatternsSeparately pins the diagnostic verb's
// agreement with the enforcer (the one thing a list is FOR during an incident): a
// pattern the engine refuses is reported as unusable, because the guard refuses
// every commit until it is fixed; a pattern the engine ACCEPTS but reads
// differently (a Perl-style escape, an inline flag group) is reported as inert,
// because the guard does not refuse it — it just matches nothing. Conflating the
// two misdirects the response.
func TestListPrivateReportsInertPatternsSeparately(t *testing.T) {
	root := t.TempDir()
	writePrivate(t, root, privateFormatDecl+"\nusable       widgetworks\ninert        \\dwidgetworks\nunusable     [unclosed\n")

	rep, err := ListPrivate(root)
	if err != nil {
		t.Fatalf("ListPrivate: %v", err)
	}
	if len(rep.Malformed) != 1 || rep.Malformed[0] != 4 {
		t.Errorf("Malformed = %v, want [4] (the line the engine refuses)", rep.Malformed)
	}
	if len(rep.Inert) != 1 || rep.Inert[0] != 3 {
		t.Errorf("Inert = %v, want [3] (the line the engine accepts but reads differently)", rep.Inert)
	}
	blob, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"unclosed", "widgetworks"} {
		if strings.Contains(string(blob), leak) {
			t.Errorf("report echoes %q: %s", leak, blob)
		}
	}
}

// TestAddPrivateRefusesUnroundTrippablePatterns pins that a pattern which validates
// in isolation but does NOT round-trip through the composed `KEY<space>PATTERN` line
// is refused, writing nothing. A whitespace-only pattern composes to `key  ` and
// re-reads as malformed (wedging every commit, and `remove` cannot undo it); a
// pattern with leading/trailing whitespace is silently trimmed on re-read, so the
// enforced pattern differs from the validated one; a NUL byte the two readers
// disagree on is refused outright.
func TestAddPrivateRefusesUnroundTrippablePatterns(t *testing.T) {
	root := t.TempDir()
	body := privateFormatDecl + "\nwidget-partner   widgetworks\n"
	path := writePrivate(t, root, body)

	for _, tc := range []struct{ name, pattern string }{
		{"whitespace-only", " "},
		{"trailing whitespace", "widgetworks "},
		{"leading whitespace", " widgetworks"},
		{"NUL byte in the pattern", "widget\x00works"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "k", Pattern: tc.pattern})
			if !errors.Is(err, ErrInvalidPattern) {
				t.Fatalf("err = %v, want ErrInvalidPattern", err)
			}
			if got, _ := os.ReadFile(path); string(got) != body {
				t.Errorf("a refused add changed the store:\n%q", got)
			}
			// Nothing was written, so there is nothing to remove — the wedge the old
			// path created (`key  ` that `remove` reported as an unknown key) is gone.
			if _, err := RemovePrivate(root, "k"); !errors.Is(err, ErrUnknownKey) {
				t.Errorf("a refused add left a removable entry: err = %v", err)
			}
		})
	}
}

// TestRemovePrivateDeletesAllLinesForAKey pins that a duplicate key is fully
// removed. A hand-written store can hold a key twice; deleting only the first left
// the second still blocking under a key the report says is gone.
func TestRemovePrivateDeletesAllLinesForAKey(t *testing.T) {
	root := t.TempDir()
	body := privateFormatDecl + "\ndup   widgetworks\nother alice-laptop\\.example\\.com\ndup   partnerco\\.example\n"
	path := writePrivate(t, root, body)

	res, err := RemovePrivate(root, "dup")
	if err != nil {
		t.Fatalf("RemovePrivate: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "dup") {
		t.Errorf("a duplicate key survived removal:\n%q", got)
	}
	// One key remains (other); the count reports parsed entries, not lines.
	if res.Entries != 1 {
		t.Errorf("Entries = %d, want 1", res.Entries)
	}
	rep, err := ListPrivate(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range rep.Entries {
		if e.Key == "dup" {
			t.Errorf("dup still listed after removal: %+v", rep.Entries)
		}
	}
}

// TestAddPrivateCountAgreesWithList pins the count nit: a mutation's "N entries"
// must count only the entries `list` shows, not the raw line count. A store carrying
// an unparseable line (reported by list under Malformed, never as an entry) made the
// two disagree — add reported one more than list rendered.
func TestAddPrivateCountAgreesWithList(t *testing.T) {
	root := t.TempDir()
	writePrivate(t, root, privateFormatDecl+"\ngood   widgetworks\n[unparseable\n")

	res, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "added", Pattern: "partnerco"})
	if err != nil {
		t.Fatalf("AddPrivate: %v", err)
	}
	rep, err := ListPrivate(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Entries != len(rep.Entries) {
		t.Errorf("add reported %d entries, list shows %d — the count includes the unparseable line", res.Entries, len(rep.Entries))
	}
}

// TestListPrivateReportsANotIgnoredStore pins the read-path surfacing of the one
// precondition the whole layer rests on: a present store git would TRACK is one
// `git add -A` from committing its own patterns, and `list` must not report it as
// healthy. Bidirectional: NotIgnored with no gitignore entry, clear with one.
func TestListPrivateReportsANotIgnoredStore(t *testing.T) {
	t.Run("not ignored", func(t *testing.T) {
		root := t.TempDir()
		initRepo(t, root, "")
		writePrivate(t, root, privateFormatDecl+"\nwidget-partner   widgetworks\n")
		rep, err := ListPrivate(root)
		if err != nil {
			t.Fatalf("ListPrivate: %v", err)
		}
		if !rep.NotIgnored {
			t.Error("NotIgnored = false for a present store git would track")
		}
	})
	t.Run("ignored", func(t *testing.T) {
		root := t.TempDir()
		initRepo(t, root, ".abcd/.work.local/\n")
		writePrivate(t, root, privateFormatDecl+"\nwidget-partner   widgetworks\n")
		rep, err := ListPrivate(root)
		if err != nil {
			t.Fatalf("ListPrivate: %v", err)
		}
		if rep.NotIgnored {
			t.Error("NotIgnored = true for a store in the gitignored local tier")
		}
	})
}

// TestAPrivateStoreACommitCarriesIsRefused: a store git tracks came with a
// commit, not from this checkout's person, so it is refused as theirs rather
// than read as their layer.
func TestAPrivateStoreACommitCarriesIsRefused(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root, ".abcd/.work.local/\n")
	writePrivate(t, root, privateFormatDecl+"\nwidget-partner   widgetworks\n")
	cmd := exec.Command("git", "-C", root, "add", "-f", "--", PrivateRelPath)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add -f: %v\n%s", err, out)
	}
	_, err := ListPrivate(root)
	if !errors.Is(err, ErrMalformedStore) || !strings.Contains(err.Error(), "git tracks "+PrivateRelPath) {
		t.Fatalf("ListPrivate over a tracked store = %v; want it refused as tracked", err)
	}
	if strings.Contains(err.Error(), "widgetworks") {
		t.Fatalf("the refusal echoes the store's content: %v", err)
	}
}

// initRepo makes root a real git repo (with the isolated environment the tree's
// git-spawning tests share) so the ignored-path check has something to answer.
func initRepo(t *testing.T, root string, gitignore string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	cmd := exec.Command("git", "-C", root, "init")
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init: %v\n%s", err, out)
	}
	if gitignore != "" {
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestAddPrivateRefusesAnUnIgnoredStore pins the precondition the whole layer rests
// on: the store must be untracked. Writing the plaintext patterns to a path git
// would track hands them to the next `git add -A`, and the guard cannot catch that
// — the guard's patterns are exactly what the file holds. The refusal is
// bidirectional: refused with no .gitignore entry, accepted with one.
func TestAddPrivateRefusesAnUnIgnoredStore(t *testing.T) {
	t.Run("refused when git would track it", func(t *testing.T) {
		root := t.TempDir()
		initRepo(t, root, "")
		_, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"})
		if !errors.Is(err, ErrStoreNotIgnored) {
			t.Fatalf("err = %v, want ErrStoreNotIgnored", err)
		}
		if !strings.Contains(err.Error(), ".gitignore") {
			t.Errorf("refusal does not name the remediation: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(PrivateRelPath))); statErr == nil {
			t.Error("the store was written anyway; a refused add must leave no plaintext behind")
		}
	})
	t.Run("accepted when the local tier is ignored", func(t *testing.T) {
		root := t.TempDir()
		initRepo(t, root, ".abcd/.work.local/\n")
		if _, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"}); err != nil {
			t.Fatalf("AddPrivate: %v", err)
		}
	})
}

// TestAddPrivateWriteIsContainedInTheRepo pins containment: a symlinked local tier
// is the shape that lands the private patterns OUTSIDE the repo — in a world-
// readable directory, or a synced one — while every surface still reports the
// in-repo path. The write resolves inside an os.Root, so it is refused instead.
func TestAddPrivateWriteIsContainedInTheRepo(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".abcd", ".work.local")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"}); err == nil {
		t.Fatal("AddPrivate followed a symlinked local tier; want a refusal")
	}
	leaked, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaked) != 0 {
		t.Errorf("the private patterns landed outside the repo: %v", leaked)
	}
}

// TestParseRefusesTheDuplicateDeclarationCorpus is the Go half of the
// duplicate-declaration fixture. Its shell half is
// TestPreCommitHook_DuplicatedDeclarationRefuses: one file, both readers, so the
// divergence that made the status board contradict the guard is a test failure.
func TestParseRefusesTheDuplicateDeclarationCorpus(t *testing.T) {
	_, _, err := parse(readCorpus(t, "parse-corpus-duplicate-decl.txt"))
	if !errors.Is(err, ErrMalformedStore) {
		t.Fatalf("err = %v; want ErrMalformedStore", err)
	}
	if strings.Contains(err.Error(), "alice-laptop") {
		t.Errorf("the refusal echoes store content: %v", err)
	}
}

// TestPrivateStoreFailsClosedWhenGitCannotAnswer pins the third state the write
// gate and the report share with ahoy's storePathIsSafe: a plain directory is
// not a repository and keeps the skip, but a repo-shaped tree git will not
// answer for (here: a corrupt .git) can still commit — so the add refuses with
// nothing written, and the report carries the missing proof instead of silently
// dropping its warning.
func TestPrivateStoreFailsClosedWhenGitCannotAnswer(t *testing.T) {
	t.Run("plain directory keeps the skip", func(t *testing.T) {
		root := t.TempDir()
		if _, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"}); err != nil {
			t.Fatalf("AddPrivate outside any repository: %v", err)
		}
	})
	t.Run("repo-shaped but unanswerable refuses the write", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "widget-partner", Pattern: "widgetworks"})
		if !errors.Is(err, ErrStoreNotIgnored) {
			t.Fatalf("err = %v, want ErrStoreNotIgnored", err)
		}
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(PrivateRelPath))); statErr == nil {
			t.Error("the store was written anyway; a refused add must leave no plaintext behind")
		}
	})
	t.Run("the report says the proof is missing", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		writePrivate(t, root, privateFormatDecl+"\nwidget-partner   widgetworks\n")
		rep, err := ListPrivate(root)
		if err != nil {
			t.Fatalf("ListPrivate: %v", err)
		}
		if !rep.IgnoreUnanswerable {
			t.Error("IgnoreUnanswerable = false in a repo-shaped tree git cannot answer for — the warning silently vanished")
		}
		if rep.NotIgnored {
			t.Error("NotIgnored = true without git's answer; the two states carry different remedies")
		}
	})
}

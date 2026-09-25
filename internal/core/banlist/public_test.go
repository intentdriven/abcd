package banlist

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// realConfig copies this repo's own committed docs-lint config into a temp repo.
// Editing the REAL file's bytes is the point of these tests: a surgical editor
// proven only against a synthetic two-entry fixture is not proven at all.
func realConfig(t *testing.T) (root string, original []byte) {
	t.Helper()
	src := filepath.Join("..", "..", "..", ".abcd", "docs-lint.json")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("repo docs-lint config unavailable: %v", err)
	}
	root = t.TempDir()
	dst := filepath.Join(root, filepath.FromSlash(PublicConfigRelPath))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, data
}

func readConfig(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(PublicConfigRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestAddPublicIsAMinimalDiff is the HARD constraint on the public layer: adding
// an entry adds exactly one line and touches nothing else — unrelated keys,
// ordering, and formatting survive byte-for-byte — and the result still loads
// through the linter's own strict validation.
func TestAddPublicIsAMinimalDiff(t *testing.T) {
	root, original := realConfig(t)

	res, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `(?i)\bwidgetworks\b`})
	if err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	if res.Entry.ID != "names/widgetworks" {
		t.Errorf("ID = %q, want names/widgetworks", res.Entry.ID)
	}
	if res.Entry.Severity != "blocker" {
		t.Errorf("Severity = %q, want blocker by default", res.Entry.Severity)
	}

	// The minimal diff a JSON array admits: one inserted line, and the line that
	// held the previous last element gains its separating comma. Nothing else in
	// the file may move, reflow, or reorder.
	before := strings.Split(string(original), "\n")
	after := strings.Split(string(readConfig(t, root)), "\n")
	if len(after) != len(before)+1 {
		t.Fatalf("line count %d -> %d, want exactly one added line", len(before), len(after))
	}
	at := -1
	for i := range before {
		if before[i] != after[i] {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("no line changed; the entry was not written")
	}
	if after[at] != before[at]+"," {
		t.Errorf("line %d changed beyond its separator:\n got %q\nwant %q", at+1, after[at], before[at]+",")
	}
	if !strings.Contains(after[at+1], "names/widgetworks") {
		t.Errorf("the inserted line is not the new entry: %q", after[at+1])
	}
	if got, want := strings.Join(after[at+2:], "\n"), strings.Join(before[at+1:], "\n"); got != want {
		t.Errorf("the tail of the file changed:\n got %q\nwant %q", got, want)
	}

	cfg, err := lint.LoadConfig(filepath.Join(root, filepath.FromSlash(PublicConfigRelPath)))
	if err != nil {
		t.Fatalf("the edited config no longer loads: %v", err)
	}
	var found bool
	for _, tok := range cfg.BannedTokens {
		if tok.ID == "names/widgetworks" {
			found = true
			if len(tok.AllowContext) == 0 || tok.Successor == "" {
				t.Errorf("entry fails the strict banned_tokens schema: %+v", tok)
			}
		}
	}
	if !found {
		t.Error("the added entry is absent from the loaded banned_tokens family")
	}
}

// TestAddPublicEntryGatesUserFacingContent pins AC1 end to end: an entry the verb
// wrote is enforced by the same docs-lint family the hand-curated harness entries
// live in — blocker on a plain mention, silent on the escape line.
func TestAddPublicEntryGatesUserFacingContent(t *testing.T) {
	root, _ := realConfig(t)
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `(?i)\bwidgetworks\b`}); err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	cfg, err := lint.LoadConfig(filepath.Join(root, filepath.FromSlash(PublicConfigRelPath)))
	if err != nil {
		t.Fatal(err)
	}

	docs := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(docs, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The public config's roots are ["docs", "README.md"]; both must resolve now
	// that an unresolvable configured root fails loud (GitHub #360).
	write("README.md", "# readme\n")
	// Its name_roots must resolve too (iss-279).
	for _, r := range []string{".abcd/README.md", "AGENTS.md", "CONTRIBUTING.md", "scripts/README.md"} {
		write(r, "# t\n")
	}
	write("docs/named.md", "# t\n\nBuilt with widgetworks.\n")
	write("docs/allowed.md", "# t\n\n<!-- docs-lint: allow --> widgetworks is named deliberately.\n")
	write("docs/clean.md", "# t\n\nBuilt with a generic term.\n")

	findings, err := lint.Lint(cfg, docs)
	if err != nil {
		t.Fatal(err)
	}
	var hits []lint.Finding
	for _, f := range findings {
		if f.RuleID == "names/widgetworks" {
			hits = append(hits, f)
		}
	}
	if len(hits) != 1 {
		t.Fatalf("names/widgetworks findings = %+v, want exactly one (named.md only)", hits)
	}
	if hits[0].File != filepath.Join("docs", "named.md") || hits[0].Line != 3 || hits[0].Severity != "blocker" {
		t.Errorf("finding = %+v, want a blocker on docs/named.md:3", hits[0])
	}
}

// TestRemovePublicRestoresTheOriginalBytes is the strongest statement the
// surgical editor can make: add then remove is a round trip to the exact bytes
// that were there before.
func TestRemovePublicRestoresTheOriginalBytes(t *testing.T) {
	root, original := realConfig(t)
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `(?i)\bwidgetworks\b`}); err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	if _, err := RemovePublic(root, "widgetworks"); err != nil {
		t.Fatalf("RemovePublic: %v", err)
	}
	if got := readConfig(t, root); string(got) != string(original) {
		t.Errorf("round trip is not byte-identical:\n--- got\n%s\n--- want\n%s", got, original)
	}
}

// TestRemovePublicRefusesEntriesItDoesNotOwn keeps the hand-curated families
// (harness/*, present_tense/*) out of the verb's reach: the id namespace is the
// ownership boundary, and a refusal writes nothing.
func TestRemovePublicRefusesEntriesItDoesNotOwn(t *testing.T) {
	root, original := realConfig(t)

	if _, err := RemovePublic(root, "harness/gemini"); !errors.Is(err, ErrNotManaged) {
		t.Errorf("removing harness/gemini: err = %v, want ErrNotManaged", err)
	}
	if _, err := RemovePublic(root, "nosuchkey"); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("removing an unknown key: err = %v, want ErrUnknownKey", err)
	}
	if got := readConfig(t, root); string(got) != string(original) {
		t.Error("a refused removal wrote to the config")
	}
}

// TestRemovePublicDoesNotShadowAManagedTargetWithABareKey pins the fix for the
// namespace-check shadow: with both a bare hand-curated `widgetworks` and a managed
// `names/widgetworks` present, `remove --public widgetworks` must remove the managed
// entry it owns — not refuse with ErrNotManaged because it met the bare key first.
// The bare entry is placed FIRST, the order that triggered the early return.
func TestRemovePublicDoesNotShadowAManagedTargetWithABareKey(t *testing.T) {
	root := writeConfig(t, `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id":"widgetworks","pattern":"(?i)widgetworks","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"},
    {"id":"names/widgetworks","pattern":"(?i)widgetworks","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}
  ]
}
`)
	res, err := RemovePublic(root, "widgetworks")
	if err != nil {
		t.Fatalf("RemovePublic: %v (a bare key shadowed the managed target)", err)
	}
	if res.Entry.ID != "names/widgetworks" {
		t.Errorf("removed %q, want names/widgetworks", res.Entry.ID)
	}
	rep, err := ListPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]bool{}
	for _, e := range rep.Entries {
		byID[e.ID] = true
	}
	if byID["names/widgetworks"] {
		t.Error("the managed entry was not removed")
	}
	if !byID["widgetworks"] {
		t.Error("the hand-curated bare entry was removed; the verb must not touch it")
	}
}

// TestRemovePublicDeletesEveryDuplicateOfAManagedKey pins the removal of a
// hand-edited duplicate: AddPublic refuses to create two `names/<key>` entries, but a
// human edit can, and removing one while reporting success would leave a ban enforcing
// under a key the report calls gone. remove --public must delete them all.
func TestRemovePublicDeletesEveryDuplicateOfAManagedKey(t *testing.T) {
	root := writeConfig(t, `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id":"names/widgetworks","pattern":"(?i)widgetworks","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"},
    {"id":"harness/gemini","pattern":"(?i)gemini","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"},
    {"id":"names/widgetworks","pattern":"(?i)widgetworks","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}
  ]
}
`)
	if _, err := RemovePublic(root, "widgetworks"); err != nil {
		t.Fatalf("RemovePublic: %v", err)
	}
	rep, err := ListPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range rep.Entries {
		if e.ID == "names/widgetworks" {
			t.Error("a duplicate names/widgetworks entry survived removal")
		}
	}
	if got := countID(t, root, "names/widgetworks"); got != 0 {
		t.Errorf("names/widgetworks still present %d time(s) after removal", got)
	}
	// The unrelated hand-curated entry is untouched.
	if got := countID(t, root, "harness/gemini"); got != 1 {
		t.Errorf("harness/gemini count = %d, want 1 (removal disturbed an unrelated entry)", got)
	}
}

// TestListPublicRendersTheWholeFamilyInFull pins AC6's public half: public entries
// render in full (pattern included — they are committed and reviewable), with the
// managed flag marking which the verb owns.
func TestListPublicRendersTheWholeFamilyInFull(t *testing.T) {
	root, _ := realConfig(t)
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `(?i)\bwidgetworks\b`}); err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	rep, err := ListPublic(root)
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if !rep.Present {
		t.Fatal("Present = false for a config that exists")
	}
	byID := map[string]PublicEntry{}
	for _, e := range rep.Entries {
		byID[e.ID] = e
	}
	managed, ok := byID["names/widgetworks"]
	if !ok {
		t.Fatalf("the added entry is missing from the report: %+v", rep.Entries)
	}
	if !managed.Managed || managed.Key != "widgetworks" || managed.Pattern != `(?i)\bwidgetworks\b` {
		t.Errorf("managed entry = %+v", managed)
	}
	harness, ok := byID["harness/gemini"]
	if !ok {
		t.Fatalf("the hand-curated harness family is missing from the report: %+v", rep.Entries)
	}
	if harness.Managed {
		t.Error("harness/gemini reported as verb-managed")
	}
}

// TestAddPublicRefusals covers the input contract, including the duplicate that
// would otherwise land two entries under one id.
func TestAddPublicRefusals(t *testing.T) {
	root, original := realConfig(t)
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `(?i)\bwidgetworks\b`}); err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	after := readConfig(t, root)

	cases := []struct {
		name string
		req  AddPublicRequest
		want error
	}{
		{"duplicate key", AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `x`}, ErrDuplicateKey},
		{"key with a metacharacter", AddPublicRequest{RepoRoot: root, Key: "widget*", Pattern: `x`}, ErrInvalidKey},
		{"empty pattern", AddPublicRequest{RepoRoot: root, Key: "empty", Pattern: ""}, ErrInvalidPattern},
		{"uncompilable pattern", AddPublicRequest{RepoRoot: root, Key: "broken", Pattern: `[unclosed`}, ErrInvalidPattern},
		{"unknown severity", AddPublicRequest{RepoRoot: root, Key: "loud", Pattern: `x`, Severity: "shout"}, ErrInvalidSeverity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := AddPublic(tc.req); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if got := readConfig(t, root); string(got) != string(after) {
				t.Error("a refused add wrote to the config")
			}
		})
	}
	if _, err := AddPublic(AddPublicRequest{RepoRoot: t.TempDir(), Key: "k", Pattern: "x"}); !errors.Is(err, ErrNoStore) {
		t.Errorf("adding without a docs-lint config: err = %v, want ErrNoStore", err)
	}
	_ = original
}

// TestPublicSurgerySurvivesFormatting exercises the array-boundary cases a real
// config does not cover: a single-entry family and a pretty-printed one.
func TestPublicSurgerySurvivesFormatting(t *testing.T) {
	const single = `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id":"names/only","pattern":"only","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}
  ],
  "rules": {"links_resolve": {"enabled": true, "severity": "blocker"}}
}
`
	const pretty = `{
  "banned_tokens": [
    {
      "id": "names/first",
      "pattern": "first",
      "severity": "blocker",
      "successor": "s",
      "allow_context": ["(?i)<!--\\s*docs-lint:\\s*allow\\b"],
      "message": "m"
    }
  ]
}
`
	for _, tc := range []struct{ name, body string }{{"single entry", single}, {"pretty printed", pretty}} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(PublicConfigRelPath))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "second", Pattern: "second"}); err != nil {
				t.Fatalf("AddPublic: %v", err)
			}
			if _, err := lint.LoadConfig(path); err != nil {
				t.Fatalf("edited config does not load: %v", err)
			}
			if _, err := RemovePublic(root, "second"); err != nil {
				t.Fatalf("RemovePublic: %v", err)
			}
			if got := readConfig(t, root); string(got) != tc.body {
				t.Errorf("round trip is not byte-identical:\n--- got\n%s\n--- want\n%s", got, tc.body)
			}
			// Removing the last entry leaves a valid, loadable config.
			if _, err := RemovePublic(root, firstKey(t, root)); err != nil {
				t.Fatalf("RemovePublic (last entry): %v", err)
			}
			if _, err := lint.LoadConfig(path); err != nil {
				t.Fatalf("config with an emptied family does not load: %v", err)
			}
		})
	}
}

func firstKey(t *testing.T, root string) string {
	t.Helper()
	rep, err := ListPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) == 0 {
		t.Fatal("no entries to remove")
	}
	return rep.Entries[0].Key
}

// TestAddPublicIsCaseInsensitiveLikeTheCuratedEntries pins CI parity with the
// hand-written half of the family: every curated entry opens with (?i) and the docs
// promise the public layer matches case-insensitively, but the linter compiles the
// STORED pattern raw — so an entry written without the flag is case-SENSITIVE and a
// mixed-case spelling of the banned name walks straight through CI. The flag is
// added once, at the store, and the proof is the lint verdict rather than the string.
func TestAddPublicIsCaseInsensitiveLikeTheCuratedEntries(t *testing.T) {
	root, _ := realConfig(t)
	res, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks", Pattern: `\bwidgetworks\b`})
	if err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	if !strings.HasPrefix(res.Entry.Pattern, "(?i)") {
		t.Errorf("stored pattern = %q; it must carry (?i) like every hand-curated entry", res.Entry.Pattern)
	}

	cfg, err := lint.LoadConfig(filepath.Join(root, filepath.FromSlash(PublicConfigRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	docs := t.TempDir()
	p := filepath.Join(docs, "docs", "named.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# t\n\nBuilt with WidgetWorks.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The public config's roots are ["docs", "README.md"]; README.md must resolve
	// now that an unresolvable configured root fails loud (GitHub #360).
	if err := os.WriteFile(filepath.Join(docs, "README.md"), []byte("# readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Its name_roots must resolve too (iss-279).
	for _, r := range []string{".abcd/README.md", "AGENTS.md", "CONTRIBUTING.md", "scripts/README.md"} {
		p := filepath.Join(docs, filepath.FromSlash(r))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# t\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, err := lint.Lint(cfg, docs)
	if err != nil {
		t.Fatal(err)
	}
	hits := 0
	for _, f := range findings {
		if f.RuleID == "names/widgetworks" {
			hits++
		}
	}
	if hits != 1 {
		t.Errorf("mixed-case mention produced %d findings, want 1 — the entry is case-sensitive", hits)
	}
}

// countID counts banned_tokens elements carrying id by decoding the raw config,
// so duplicate ids a map-based reader would collapse are each counted.
func countID(t *testing.T, root, id string) int {
	t.Helper()
	var doc struct {
		BannedTokens []struct {
			ID string `json:"id"`
		} `json:"banned_tokens"`
	}
	if err := json.Unmarshal(readConfig(t, root), &doc); err != nil {
		t.Fatalf("countID unmarshal: %v", err)
	}
	n := 0
	for _, tok := range doc.BannedTokens {
		if tok.ID == id {
			n++
		}
	}
	return n
}

// writeConfig puts a docs-lint config body in a fresh temp repo.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, filepath.FromSlash(PublicConfigRelPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestPublicRefusesDuplicateBannedTokensKeys pins the one JSON shape where the
// editor and the linter read DIFFERENT arrays: with two top-level banned_tokens
// keys, encoding/json keeps the LAST and the byte editor finds the FIRST, so a ban
// the verb reports as written is enforced by nobody. Neither reading is safe to
// pick, so the config is refused until a human resolves it.
func TestPublicRefusesDuplicateBannedTokensKeys(t *testing.T) {
	root := writeConfig(t, `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id":"names/first","pattern":"(?i)first","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}
  ],
  "banned_tokens": [
    {"id":"names/second","pattern":"(?i)second","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}
  ]
}
`)
	if _, err := ListPublic(root); !errors.Is(err, ErrMalformedStore) {
		t.Errorf("ListPublic: err = %v, want ErrMalformedStore", err)
	}
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "third", Pattern: "third"}); !errors.Is(err, ErrMalformedStore) {
		t.Errorf("AddPublic: err = %v, want ErrMalformedStore", err)
	}
	if _, err := RemovePublic(root, "first"); !errors.Is(err, ErrMalformedStore) {
		t.Errorf("RemovePublic: err = %v, want ErrMalformedStore", err)
	}
}

// TestPublicLocatesTheKeyNotAStringValue pins that the locator reads object KEYS.
// A depth-1 string VALUE that happens to spell banned_tokens is ordinary config
// data, and mistaking it for the key aims the whole editor at the wrong offset.
func TestPublicLocatesTheKeyNotAStringValue(t *testing.T) {
	root := writeConfig(t, `{
  "note": "banned_tokens",
  "banned_tokens": [
    {"id":"names/first","pattern":"(?i)first","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}
  ]
}
`)
	rep, err := ListPublic(root)
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if len(rep.Entries) != 1 || rep.Entries[0].ID != "names/first" {
		t.Fatalf("entries = %+v, want the one real entry", rep.Entries)
	}
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "second", Pattern: "second"}); err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	if _, err := lint.LoadConfig(filepath.Join(root, filepath.FromSlash(PublicConfigRelPath))); err != nil {
		t.Fatalf("edited config does not load: %v", err)
	}
}

// TestAddPublicIndentsIntoACompactArray pins the insertion's shape when the last
// entry shares its line with the array's opening bracket: the "indent of that line"
// is the array's, not an element's, so an inserted entry landed flush against the
// margin (or at column zero for a one-line config) instead of one level in.
func TestAddPublicIndentsIntoACompactArray(t *testing.T) {
	root := writeConfig(t, `{"roots":["docs"],"banned_tokens":[{"id":"names/first","pattern":"(?i)first","severity":"blocker","successor":"s","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"],"message":"m"}]}
`)
	if _, err := AddPublic(AddPublicRequest{RepoRoot: root, Key: "second", Pattern: "second"}); err != nil {
		t.Fatalf("AddPublic: %v", err)
	}
	got := readConfig(t, root)
	if _, err := lint.LoadConfig(filepath.Join(root, filepath.FromSlash(PublicConfigRelPath))); err != nil {
		t.Fatalf("edited config does not load: %v", err)
	}
	var inserted string
	for _, line := range strings.Split(string(got), "\n") {
		if strings.Contains(line, "names/second") {
			inserted = line
		}
	}
	if inserted == "" {
		t.Fatalf("the entry was not inserted:\n%s", got)
	}
	if !strings.HasPrefix(inserted, "  ") {
		t.Errorf("inserted line is not indented into the array:\n%q", inserted)
	}
}

// TestConcurrentPrivateAddsAllLand pins the lock. Two verbs (or a verb and an
// out-of-band refresh) that each read the store, append one line, and write it back
// silently drop entries when they interleave: the last writer's body was derived
// from a read that predates the other's write. Every add here must survive.
func TestConcurrentPrivateAddsAllLand(t *testing.T) {
	root := t.TempDir()
	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = AddPrivate(AddPrivateRequest{
				RepoRoot: root,
				Key:      "widget-partner-" + strconv.Itoa(i),
				Pattern:  "widgetworks" + strconv.Itoa(i),
			})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	rep, err := ListPrivate(root)
	if err != nil {
		t.Fatalf("ListPrivate: %v", err)
	}
	if len(rep.Entries) != n {
		t.Errorf("entries = %d, want %d — a concurrent add was lost", len(rep.Entries), n)
	}
}

// TestConcurrentPublicAddsAllLand is the same statement for the committed config,
// where a lost entry is a ban that silently is not enforced.
func TestConcurrentPublicAddsAllLand(t *testing.T) {
	root, _ := realConfig(t)
	before, err := ListPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = AddPublic(AddPublicRequest{RepoRoot: root, Key: "widgetworks" + strconv.Itoa(i), Pattern: "widgetworks" + strconv.Itoa(i)})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	after, err := ListPublic(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Entries) != len(before.Entries)+n {
		t.Errorf("entries = %d, want %d — a concurrent add was lost", len(after.Entries), len(before.Entries)+n)
	}
}

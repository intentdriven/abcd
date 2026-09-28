package lint

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// glossaryTerm writes a minimal glossary term file with the given forbidden
// synonyms, so the GL002 tests read their forbidden set from the same source of
// truth the rule does — the glossary — rather than a hard-coded list.
func glossaryTerm(t *testing.T, root, term string, syns []string) {
	t.Helper()
	quoted := make([]string, len(syns))
	for i, s := range syns {
		quoted[i] = `"` + s + `"`
	}
	writeFile(t, root, "gloss/"+term+".md",
		"---\nterm: "+term+"\nforbidden_synonyms: ["+strings.Join(quoted, ", ")+"]\n---\n# "+term+"\n")
}

func fsCfg(enforce, exempt, allow []string) Config {
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"forbidden_synonyms": {
				Enabled: true, Severity: "blocker",
				GlossaryDir: "gloss", Enforce: enforce,
				ExemptPrefixes: exempt, AllowContext: allow,
			},
		},
	}
}

// TestForbiddenSynonymsGL002 is the core detector behaviour: an enforced synonym
// used as a standalone word in live prose fires GL002; frontmatter, code spans,
// exempt paths, the glossary itself, and allow_context lines do not.
func TestForbiddenSynonymsGL002(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic", "sprint"})

	// live prose noun -> fires
	writeFile(t, root, "rec/live.md", "# t\n\nNo epic currently owns the reviewer.\n")
	// frontmatter field -> not scanned (body only)
	writeFile(t, root, "rec/fm.md", "---\nglossary_terms_used: [core/epic]\n---\n# t\n\nclean body.\n")
	// fenced code -> masked
	writeFile(t, root, "rec/fenced.md", "# t\n\n```\nepic_id: spc-1\n```\n")
	// inline code span -> masked (a mention, not a substitution)
	writeFile(t, root, "rec/inline.md", "# t\n\nThe `epic` token is retired.\n")
	// allow_context line -> suppressed
	writeFile(t, root, "rec/allow.md", "# t\n\nThe external epic-review type is unchanged.\n")
	// clean
	writeFile(t, root, "rec/clean.md", "# t\n\nEverything is a spec now.\n")

	cfg := fsCfg([]string{"epic"}, nil, []string{`epic-review`})
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 1 {
		t.Fatalf("expected exactly 1 GL002 finding, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "live.md"), "GL002", 3) {
		t.Errorf("missing GL002 on live.md:3: %+v", fs)
	}
}

// TestForbiddenSynonymsBoundary proves the Unicode word-boundary behaviour: the
// bare word fires, but a longer word that merely contains it does not — including
// a unicode-letter neighbour that an ASCII-only \b would wrongly treat as a
// boundary (the RE2 ASCII-boundary hazard this rule guards against).
func TestForbiddenSynonymsBoundary(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic"})

	writeFile(t, root, "rec/plural.md", "# t\n\nThree epics remain.\n")      // suffix -> no match
	writeFile(t, root, "rec/center.md", "# t\n\nAt the epicenter of it.\n")  // substring -> no match
	writeFile(t, root, "rec/unicode.md", "# t\n\nThe wordépic is fused.\n")  // unicode-letter prefix -> no match
	writeFile(t, root, "rec/hit.md", "# t\n\nThat epic is done.\n")          // bare -> match
	writeFile(t, root, "rec/punct.md", "# t\n\nThe (epic) parenthetical.\n") // punctuation boundary -> match

	cfg := fsCfg([]string{"epic"}, nil, nil)
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 2 {
		t.Fatalf("expected 2 GL002 findings (hit, punct), got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "hit.md"), "GL002", 3) {
		t.Errorf("missing GL002 on hit.md:3: %+v", fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "punct.md"), "GL002", 3) {
		t.Errorf("missing GL002 on punct.md:3: %+v", fs)
	}
}

// TestForbiddenSynonymsFrontmatterWithComment proves a leading attribution
// comment above the `---` block does not expose the frontmatter to prose scanning:
// a `core/epic` term reference in frontmatter must not be flagged.
func TestForbiddenSynonymsFrontmatterWithComment(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic"})
	writeFile(t, root, "rec/commented.md",
		"<!-- adapted from a template -->\n---\nglossary_terms_used: [core/epic]\n---\n# t\n\nClean spec body.\n")

	fs, err := Lint(fsCfg([]string{"epic"}, nil, nil), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 0 {
		t.Fatalf("frontmatter with a leading comment must not be scanned; got %d: %+v", n, fs)
	}
}

// TestForbiddenSynonymsCaseInsensitive proves matching ignores case.
func TestForbiddenSynonymsCaseInsensitive(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic"})
	writeFile(t, root, "rec/caps.md", "# t\n\nEPIC work and Epic scope.\n")

	fs, err := Lint(fsCfg([]string{"epic"}, nil, nil), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 1 {
		t.Fatalf("expected 1 GL002 finding (one per line), got %d: %+v", n, fs)
	}
}

// TestForbiddenSynonymsExemptPrefix proves historical, git-tracked records are
// skipped by path prefix (itd-43 AC1 exemption).
func TestForbiddenSynonymsExemptPrefix(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic"})
	writeFile(t, root, "rec/decisions/adr.md", "# t\n\nThe old epic model.\n")
	writeFile(t, root, "rec/live.md", "# t\n\nThe new epic drift.\n")

	fs, err := Lint(fsCfg([]string{"epic"}, []string{"rec/decisions/"}, nil), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 1 {
		t.Fatalf("expected 1 GL002 finding (live only), got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "live.md"), "GL002", 3) {
		t.Errorf("expected GL002 on live.md, exempt path leaked: %+v", fs)
	}
}

// TestForbiddenSynonymsSourceOfTruth proves the glossary is authoritative: an
// enforced word the glossary does not forbid is a config error, not a silent pass.
func TestForbiddenSynonymsSourceOfTruth(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"sprint"}) // 'epic' NOT forbidden here
	writeFile(t, root, "rec/live.md", "# t\n\nAn epic here.\n")

	_, err := Lint(fsCfg([]string{"epic"}, nil, nil), root)
	if err == nil {
		t.Fatal("expected a config error for enforcing an undeclared synonym, got nil")
	}
	if !strings.Contains(err.Error(), "epic") {
		t.Errorf("error should name the offending word: %v", err)
	}
}

// TestStripInlineCodePairedSpanBeforeStrayBacktick is the iss-106 detector: a line
// with a CLOSED inline-code span containing an enforced synonym, followed by a
// single STRAY (unpaired) trailing backtick, must still have the closed span
// blanked. The old tail handling restored the whole original line on any unpaired
// backtick, un-masking the earlier span's token and firing a spurious GL002.
func TestStripInlineCodePairedSpanBeforeStrayBacktick(t *testing.T) {
	// `epic` is a closed span; the later lone backtick is unpaired.
	in := "The `epic` token is retired but the ` here is stray."
	got := stripInlineCode(in)
	if strings.Contains(got, "epic") {
		t.Errorf("closed span not blanked before a stray backtick: %q -> %q", in, got)
	}
}

// TestForbiddenSynonymsClosedSpanThenStrayBacktick is the higher-level iss-106
// guard: the full GL002 check must not flag a synonym that sits inside a closed
// inline-code span merely because a stray backtick follows later on the line.
func TestForbiddenSynonymsClosedSpanThenStrayBacktick(t *testing.T) {
	root := t.TempDir()
	glossaryTerm(t, root, "spec", []string{"epic"})
	writeFile(t, root, "rec/stray.md", "# t\n\nThe `epic` token is retired but the ` is stray.\n")

	fs, err := Lint(fsCfg([]string{"epic"}, nil, nil), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "GL002"); n != 0 {
		t.Fatalf("expected 0 GL002 findings (synonym is inside a closed code span), got %d: %+v", n, fs)
	}
}

// TestForbiddenSynonymsRealGlossary wires the real record-lint config against the
// real glossary and asserts the swept corpus is clean — deleting the rule, its
// enforcement of 'epic', or reintroducing an epic noun in live prose fails here.
func TestForbiddenSynonymsRealGlossary(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	cfg, err := LoadConfig(filepath.Join(repoRoot, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	rc, ok := cfg.Rules["forbidden_synonyms"]
	if !ok || !rc.Enabled {
		t.Fatal("record-lint.json must enable the forbidden_synonyms (GL002) rule")
	}
	found := false
	for _, w := range rc.Enforce {
		if w == "epic" {
			found = true
		}
	}
	if !found {
		t.Fatal("record-lint.json forbidden_synonyms.enforce must include 'epic' (itd-43)")
	}
	fs, err := Lint(cfg, repoRoot)
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if n := countRule(fs, "GL002"); n != 0 {
		t.Fatalf("live corpus has %d GL002 finding(s); the epic sweep is incomplete: %+v", n, fs)
	}
}

// TestStripInlineCodePairsRunsOfTheSameLength: stripInlineCode paired single
// backticks one at a time, so a span a double-backtick run opened was read as
// two empty spans with live prose between them: a synonym quoted in one was
// flagged, and a link quoted in one was checked. It pairs by termsafe's pairer,
// the rule every reader shares, and blanks rune for rune, so a column after a
// multi-byte rune stays where it was.
func TestStripInlineCodePairsRunsOfTheSameLength(t *testing.T) {
	for in, gone := range map[string]string{
		"the ``epic`` word":              "epic",
		"the `` `epic` `` word":          "epic",
		"a ``[x](gone.md)`` link":        "gone.md",
		"a ``b`epic`` word":              "epic",
		"é `epic` ü":                     "epic",
		"a `x`` epic `` y` word":         "epic",
		"\\`a ``epic`` b":                "epic",
		"ok `epic` then a stray ` here":  "epic",
		"``one`` and `epic` and ``two``": "epic",
	} {
		got := stripInlineCode(in)
		if strings.Contains(got, gone) {
			t.Errorf("stripInlineCode(%q) = %q, still holds %q", in, got, gone)
		}
		if utf8.RuneCountInString(got) != utf8.RuneCountInString(in) {
			t.Errorf("stripInlineCode(%q) = %q changed the rune count", in, got)
		}
	}
	for in, kept := range map[string]string{
		"a ``epic` word":     "epic",
		"\\`epic` word":      "epic",
		"a `epic`` word":     "epic",
		"the ```epic`` word": "epic",
	} {
		if got := stripInlineCode(in); !strings.Contains(got, kept) {
			t.Errorf("stripInlineCode(%q) = %q, blanked %q, which no span holds", in, got, kept)
		}
	}
}

package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// indexCfg builds a one-index index_drift config.
func indexCfg(spec IndexSpec) Config {
	return Config{Rules: map[string]RuleConfig{
		"index_drift": {Enabled: true, Severity: "blocker", Indexes: []IndexSpec{spec}},
	}}
}

// verbSpec is the shipped shape: a command README enumerating one file per verb.
func verbSpec() IndexSpec {
	return IndexSpec{
		ID: "commands", Doc: filepath.Join("commands", "README.md"),
		Dir: "commands/abcd", Entry: `^[a-z][a-z-]*$`, Suffix: ".md",
	}
}

// TestIndexDriftExactCatchesBothDirections reproduces the iss-38 corpus shape:
// a hand-written verb list that has fallen behind the directory (a verb added,
// a verb gone) while reading as if it were current.
func TestIndexDriftExactCatchesBothDirections(t *testing.T) {
	root := t.TempDir()
	for _, verb := range []string{"ahoy", "capture", "guard", "ideate"} {
		writeFile(t, root, filepath.Join("commands", "abcd", verb+".md"), "# "+verb+"\n")
	}
	writeFile(t, root, filepath.Join("commands", "abcd", "README.md"), "# index\n")

	// Lists a verb that has moved away (`consult`), omits two that exist
	// (`guard`, `ideate`), and carries prose the pattern must not read as an
	// entry (`--json`, `abcd/`).
	writeFile(t, root, filepath.Join("commands", "README.md"), strings.Join([]string{
		"# commands/",
		"",
		"Each file calls `abcd <verb> --json` and formats the result.",
		"",
		"<!-- index: commands -->",
		"`ahoy`, `capture`, `consult`.",
		"<!-- /index -->",
		"",
		"See [`abcd/`](abcd/).",
	}, "\n")+"\n")

	fs, err := Lint(indexCfg(verbSpec()), root)
	if err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join("commands", "README.md")
	if n := countRule(fs, "index_drift"); n != 3 {
		t.Fatalf("expected 3 index_drift findings (1 stale + 2 omitted), got %d: %+v", n, fs)
	}
	if !hasFinding(fs, doc, "index_drift", 6) {
		t.Errorf("expected a finding on the listed-but-absent verb's line 6; got %+v", fs)
	}
	for _, want := range []string{"`consult`", "`guard`", "`ideate`"} {
		if !messageContains(fs, want) {
			t.Errorf("expected a finding naming %s; got %+v", want, fs)
		}
	}
	// The omissions anchor to the region's opening line, not to a token line.
	if !hasFinding(fs, doc, "index_drift", 5) {
		t.Errorf("expected the omission findings on the region's opening line 5; got %+v", fs)
	}
}

// TestIndexDriftExactCleanIsSilent is the other half of the detector: a listing
// that agrees with its directory produces nothing, so the rule cannot pass by
// firing on everything.
func TestIndexDriftExactCleanIsSilent(t *testing.T) {
	root := t.TempDir()
	for _, verb := range []string{"ahoy", "capture", "guard"} {
		writeFile(t, root, filepath.Join("commands", "abcd", verb+".md"), "# "+verb+"\n")
	}
	writeFile(t, root, filepath.Join("commands", "abcd", "README.md"), "# index\n")
	writeFile(t, root, filepath.Join("commands", "README.md"),
		"# commands/\n\n<!-- index: commands -->\n`ahoy`, `capture`, `guard`.\n<!-- /index -->\n")

	fs, err := Lint(indexCfg(verbSpec()), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 0 {
		t.Fatalf("expected a clean index to be silent, got %d findings: %+v", n, fs)
	}
}

// TestIndexDriftAbsentCatchesShippedSeam is the planned-seams shape: a list of
// directories that do not exist yet, one of which has since shipped.
func TestIndexDriftAbsentCatchesShippedSeam(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("internal", "adapter", "scanner", "scanner.go"), "package scanner\n")
	writeFile(t, root, filepath.Join("internal", "README.md"), strings.Join([]string{
		"# internal/",
		"",
		"## Planned seams",
		"",
		"<!-- index: planned-seams -->",
		"- **`adapter/oracle/`** — LLM review.",
		"- **`adapter/scanner/`** — secret/PII.",
		"- **`registry/`** — declarative wiring; opt-in via `ccpm`.",
		"<!-- /index -->",
	}, "\n")+"\n")

	spec := IndexSpec{
		ID: "planned-seams", Doc: filepath.Join("internal", "README.md"),
		Dir: "internal", Entry: `^[a-z][a-z0-9-]*(/[a-z0-9-]+)*/$`, Mode: "absent",
	}
	fs, err := Lint(indexCfg(spec), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 1 {
		t.Fatalf("expected exactly the shipped seam to fire, got %d findings: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("internal", "README.md"), "index_drift", 7) {
		t.Errorf("expected the finding on the shipped seam's line 7; got %+v", fs)
	}
	if !messageContains(fs, "internal/adapter/scanner exists") {
		t.Errorf("expected the message to name the path that exists; got %+v", fs)
	}
}

// TestIndexDriftEntriesOnMarkerLines pins the region-boundary scan: an entry
// sharing a line with the opening or closing marker is still inside the
// region and must be parsed. A prior version of the parser returned as soon as
// it matched the closing marker (dropping any entries before it on that line)
// and skipped straight past the opening marker to the next line (dropping any
// entries after it on that line) — both silently produce wrong results, a
// spurious blocker on a correct document in `exact` mode or a disarmed check
// in `absent` mode.
func TestIndexDriftEntriesOnMarkerLines(t *testing.T) {
	root := t.TempDir()
	for _, verb := range []string{"alpha", "beta"} {
		writeFile(t, root, filepath.Join("commands", "abcd", verb+".md"), "# "+verb+"\n")
	}

	t.Run("entry on the opening marker line", func(t *testing.T) {
		writeFile(t, root, filepath.Join("commands", "README.md"), strings.Join([]string{
			"# commands/",
			"",
			"<!-- index: commands --> `alpha`,",
			"`beta`.",
			"<!-- /index -->",
		}, "\n")+"\n")

		fs, err := Lint(indexCfg(verbSpec()), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "index_drift"); n != 0 {
			t.Fatalf("expected the entry sharing the opening marker's line to be read as listed, got %d finding(s): %+v", n, fs)
		}
	})

	t.Run("entry on the closing marker line", func(t *testing.T) {
		writeFile(t, root, filepath.Join("commands", "README.md"), strings.Join([]string{
			"# commands/",
			"",
			"<!-- index: commands -->",
			"`alpha`,",
			"`beta`. <!-- /index -->",
		}, "\n")+"\n")

		fs, err := Lint(indexCfg(verbSpec()), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "index_drift"); n != 0 {
			t.Fatalf("expected the entry sharing the closing marker's line to be read as listed, got %d finding(s): %+v", n, fs)
		}
	})
}

// TestIndexDriftFailsClosed covers the two ways a gate could be silently
// disarmed: the region deleted from the document while the config still claims
// it, and a region that parses to no entries at all.
func TestIndexDriftFailsClosed(t *testing.T) {
	t.Run("region missing", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, filepath.Join("commands", "abcd", "ahoy.md"), "# ahoy\n")
		writeFile(t, root, filepath.Join("commands", "README.md"), "# commands/\n\nBrowse the directory.\n")

		fs, err := Lint(indexCfg(verbSpec()), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "index_drift"); n != 1 {
			t.Fatalf("expected 1 finding for the missing region, got %d: %+v", n, fs)
		}
		if !hasFinding(fs, filepath.Join("commands", "README.md"), "index_drift", 0) {
			t.Errorf("expected the missing-region finding at line 0; got %+v", fs)
		}
	})

	t.Run("region empty", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, filepath.Join("commands", "abcd", "ahoy.md"), "# ahoy\n")
		writeFile(t, root, filepath.Join("commands", "README.md"),
			"# commands/\n\n<!-- index: commands -->\nSee the directory.\n<!-- /index -->\n")

		fs, err := Lint(indexCfg(verbSpec()), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "index_drift"); n != 1 {
			t.Fatalf("expected 1 finding for the empty region, got %d: %+v", n, fs)
		}
	})

	t.Run("unclosed region still checks", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, filepath.Join("commands", "abcd", "ahoy.md"), "# ahoy\n")
		writeFile(t, root, filepath.Join("commands", "README.md"),
			"# commands/\n\n<!-- index: commands -->\n`ahoy`, `gone`.\n")

		fs, err := Lint(indexCfg(verbSpec()), root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "index_drift"); n != 1 {
			t.Fatalf("expected the stale entry to fire despite the lost closing marker, got %d: %+v", n, fs)
		}
	})
}

// TestIndexDriftDirEntryComparesByRecordID reproduces the iss-41 corpus shape:
// a brief section enumerating the uncommitted intent bench by id while the
// directory names each file by id AND slug. Without dir_entry the two sides can
// never agree; with it, a promoted intent left in the list and a fresh capture
// missing from it are both findings.
func TestIndexDriftDirEntryComparesByRecordID(t *testing.T) {
	root := t.TempDir()
	drafts := filepath.Join(".abcd", "development", "intents", "drafts")
	// itd-108.md carries no slug. The canonical intent-filename rule accepts it,
	// so an id-reduction pattern that does not is a gate that goes quiet on a
	// schema-legal record instead of firing on it.
	for _, f := range []string{"itd-8-with-code-bundling.md", "itd-60-doc-fidelity-anti-drift.md", "itd-76-source-provenance-ledger.md", "itd-108.md"} {
		writeFile(t, root, filepath.Join(drafts, f), "# fixture\n")
	}
	writeFile(t, root, filepath.Join(drafts, "README.md"), "# drafts\n")

	doc := filepath.Join(".abcd", "development", "brief", "06-delivery", "03-out-of-scope.md")
	// Lists itd-47 (since moved to superseded/), omits itd-76 and itd-108 (later
	// captures), and carries backticked prose the entry pattern must not read as
	// an id.
	writeFile(t, root, doc, strings.Join([]string{
		"# Out of Phase Scope",
		"",
		"<!-- index: later-phase-intents -->",
		"- `itd-8` — `--with-code` bundling",
		"- `itd-47` — oracle-backed gates",
		"- `itd-60` — doc-fidelity anti-drift",
		"<!-- /index -->",
	}, "\n")+"\n")

	spec := IndexSpec{
		ID: "later-phase-intents", Doc: doc, Dir: ".abcd/development/intents/drafts",
		Entry: `^itd-[0-9]+$`, Suffix: ".md", DirEntry: `^(itd-[0-9]+)`,
	}
	fs, err := Lint(indexCfg(spec), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "index_drift"); n != 3 {
		t.Fatalf("expected 3 index_drift findings (1 departed + 2 omitted), got %d: %+v", n, fs)
	}
	for _, want := range []string{"`itd-47`", "`itd-76`", "`itd-108`"} {
		if !messageContains(fs, want) {
			t.Errorf("expected a finding naming %s; got %+v", want, fs)
		}
	}
	if messageContains(fs, "with-code") {
		t.Errorf("expected the comparison to run on ids, not slugs; got %+v", fs)
	}
}

// TestTheDraftsBenchIsNotACommittedIndex: a committed list of drafts/ made
// every records pull request that added, planned or moved a draft rewrite
// neighbouring lines of one sorted region, so two such pull requests open at
// once conflicted as soon as the first merged, and the forge's merge runs no
// generator to settle it (iss-2610280107422462). The bench is read from
// drafts/ itself, so the shipped config declares no index over it and the
// out-of-scope chapter carries no region for one.
func TestTheDraftsBenchIsNotACommittedIndex(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	cfg, err := LoadConfig(filepath.Join(repoRoot, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	drafts := filepath.Join(".abcd", "development", "intents", "drafts")
	for _, s := range cfg.Rules["index_drift"].Indexes {
		if filepath.Clean(s.Dir) == drafts {
			t.Errorf("record-lint.json declares index %q over %s; the bench is read from the directory, never committed as a list", s.ID, drafts)
		}
	}
	doc := filepath.Join(repoRoot, ".abcd", "development", "brief", "06-delivery", "03-out-of-scope.md")
	b, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "<!-- index: later-phase-intents -->") {
		t.Errorf("%s carries the later-phase-intents region; the bench is read from drafts/, never committed as a list", doc)
	}
}

// TestIndexDriftConfigGuards holds a malformed index to a loud error rather
// than a quiet pass — a rule that no-ops on a typo gates nothing.
func TestIndexDriftConfigGuards(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join("commands", "README.md"), "# commands/\n")

	cases := []struct {
		name string
		spec IndexSpec
	}{
		{"no doc", IndexSpec{ID: "x", Dir: "commands/abcd", Entry: `^[a-z]+$`}},
		{"no dir", IndexSpec{ID: "x", Doc: filepath.Join("commands", "README.md"), Entry: `^[a-z]+$`}},
		{"no entry pattern", IndexSpec{ID: "x", Doc: filepath.Join("commands", "README.md"), Dir: "commands/abcd"}},
		{"bad entry pattern", IndexSpec{ID: "x", Doc: filepath.Join("commands", "README.md"), Dir: "commands/abcd", Entry: `^[a-z`}},
		{"unknown mode", IndexSpec{ID: "x", Doc: filepath.Join("commands", "README.md"), Dir: "commands/abcd", Entry: `^[a-z]+$`, Mode: "maybe"}},
		{"bad dir_entry pattern", IndexSpec{ID: "x", Doc: filepath.Join("commands", "README.md"), Dir: "commands/abcd", Entry: `^[a-z]+$`, DirEntry: `^(a`}},
		{"dir_entry in absent mode", IndexSpec{ID: "x", Doc: filepath.Join("commands", "README.md"), Dir: "commands/abcd", Entry: `^[a-z]+$`, Mode: "absent", DirEntry: `^(a)`}},
		{"missing doc file", IndexSpec{ID: "x", Doc: filepath.Join("commands", "absent.md"), Dir: "commands/abcd", Entry: `^[a-z]+$`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Lint(indexCfg(tc.spec), root); err == nil {
				t.Fatalf("expected a configuration error for %s", tc.name)
			}
		})
	}
}

// messageContains reports whether any finding's message holds the substring.
func messageContains(fs []Finding, sub string) bool {
	for _, f := range fs {
		if strings.Contains(f.Message, sub) {
			return true
		}
	}
	return false
}

package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes content to root/rel, creating parent directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLintWalkContainsRootsAndLeaves pins that the reporting walk — not only the
// citation collector three functions away — refuses a cloned repo's committed
// config that points its roots or a leaf symlink outside the repository. A
// `roots` of "../secret" read and linted a file the repo does not own, and a
// committed `docs/leak.md -> outside` was followed by the raw os.ReadFile; both
// now fail closed the way CollectCitedURLs already does over the same cfg.Roots.
func TestLintWalkContainsRootsAndLeaves(t *testing.T) {
	banCfg := Config{
		Roots: []string{"docs"},
		BannedTokens: []BannedToken{{
			ID: "t1", Pattern: "SEKRET", Message: "no", Severity: "blocker",
			Successor: "x", AllowContext: []string{"never"},
		}},
	}

	t.Run("escaping root is refused", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "docs/ok.md", "# ok\n")
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "notes.md"), []byte("SEKRET\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg := banCfg
		cfg.Roots = []string{"../" + filepath.Base(outside)}
		if _, err := Lint(cfg, root); err == nil {
			t.Fatal("Lint accepted a root outside the repository")
		}
	})

	t.Run("a symlinked leaf pointing outside is refused", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "docs/ok.md", "# ok\n")
		outside := t.TempDir()
		secret := filepath.Join(outside, "notes.md")
		if err := os.WriteFile(secret, []byte("SEKRET\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(secret, filepath.Join(root, "docs", "leak.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		fs, err := Lint(banCfg, root)
		if err == nil {
			for _, f := range fs {
				if strings.Contains(f.File, "leak.md") || strings.Contains(f.File, "notes.md") {
					t.Fatal("Lint read and reported a leaf that resolves outside the repository")
				}
			}
			t.Fatal("Lint read a leaf that resolves outside the repository")
		}
	})

	t.Run("an in-repo symlink bridge is allowed", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "docs/real.md", "# real\n")
		if err := os.Symlink(filepath.Join(root, "docs", "real.md"), filepath.Join(root, "docs", "bridge.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := Lint(banCfg, root); err != nil {
			t.Fatalf("Lint refused a legitimate in-repo symlink bridge: %v", err)
		}
	})
}

// TestForbiddenSynonymsGlossaryDirIsContained pins that the GL002 glossary walk —
// over the cloned-repo-controlled glossary_dir config field — is contained like
// the main walk, so a "../secret" glossary_dir cannot read a file outside the
// repository. This is the sibling read path the main-walk containment shares.
func TestForbiddenSynonymsGlossaryDirIsContained(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/ok.md", "# ok\n")
	writeFile(t, root, "glossary/term.md", "---\nterm: widget\nforbidden_synonyms: [gadget]\n---\n")

	contained := Config{
		Roots: []string{"docs"},
		Rules: map[string]RuleConfig{
			"forbidden_synonyms": {Enabled: true, Severity: "blocker", GlossaryDir: "glossary"},
		},
	}
	if _, err := Lint(contained, root); err != nil {
		t.Fatalf("Lint refused a contained glossary_dir: %v", err)
	}

	escaping := Config{
		Roots: []string{"docs"},
		Rules: map[string]RuleConfig{
			"forbidden_synonyms": {Enabled: true, Severity: "blocker", GlossaryDir: "../secret"},
		},
	}
	if _, err := Lint(escaping, root); err == nil {
		t.Fatal("Lint accepted a glossary_dir outside the repository")
	}
}

// TestGitMetadataSeesCommentLedFrontmatter pins that the no_git_metadata blocker
// reaches a record whose `---` block is preceded by a leading attribution comment
// (the mattpocock glossary template). frontmatter.Fields requires `---` on line 0,
// so such a file previously yielded an empty field map and slipped the blocker
// entirely; the lint-local frontmatterFields now slices past the comment.
func TestGitMetadataSeesCommentLedFrontmatter(t *testing.T) {
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"no_git_metadata": {Enabled: true, Severity: "blocker", Fields: []string{"author"}},
		},
	}
	root := t.TempDir()
	// A plain ---led record: the blocker already caught this one.
	writeFile(t, root, "rec/plain.md", "---\nauthor: someone\n---\n\n# Plain\n")
	// A comment-led record carrying the same banned key.
	writeFile(t, root, "rec/commented.md",
		"<!-- Adapted from mattpocock/skills (MIT). -->\n---\nauthor: someone\n---\n\n# Commented\n")
	// A record whose `---` sits behind a UTF-8 BOM, invisible to TrimSpace.
	writeFile(t, root, "rec/bom.md", "\ufeff---\nauthor: someone\n---\n\n# BOM\n")
	// A record led by a multi-line HTML comment.
	writeFile(t, root, "rec/multiline.md",
		"<!--\nAdapted from mattpocock/skills (MIT).\n-->\n---\nauthor: someone\n---\n\n# Multiline\n")

	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	var plain, commented int
	var commentedLine int
	found := map[string]int{}
	lineOf := map[string]int{}
	for _, f := range fs {
		if f.RuleID != "no_git_metadata" {
			continue
		}
		switch {
		case strings.HasSuffix(f.File, "plain.md"):
			plain++
		case strings.HasSuffix(f.File, "commented.md"):
			commented++
			commentedLine = f.Line
		case strings.HasSuffix(f.File, "bom.md"):
			found["bom"]++
			lineOf["bom"] = f.Line
		case strings.HasSuffix(f.File, "multiline.md"):
			found["multiline"]++
			lineOf["multiline"] = f.Line
		}
	}
	if plain != 1 {
		t.Errorf("no_git_metadata on the ---led record = %d, want 1", plain)
	}
	if commented != 1 {
		t.Fatalf("no_git_metadata on the comment-led record = %d, want 1 (the blocker was bypassed)", commented)
	}
	if commentedLine != 3 {
		t.Errorf("comment-led finding line = %d, want 3 (absolute, not offset by the sliced comment)", commentedLine)
	}
	// A BOM or a multi-line comment ahead of the `---` must not slip the blocker.
	if found["bom"] != 1 {
		t.Errorf("no_git_metadata on the BOM-led record = %d, want 1 (a BOM bypassed the blocker)", found["bom"])
	}
	if lineOf["bom"] != 2 {
		t.Errorf("BOM-led finding line = %d, want 2", lineOf["bom"])
	}
	if found["multiline"] != 1 {
		t.Errorf("no_git_metadata on the multi-line-comment-led record = %d, want 1 (the comment bypassed the blocker)", found["multiline"])
	}
	if lineOf["multiline"] != 5 {
		t.Errorf("multi-line-comment-led finding line = %d, want 5 (absolute)", lineOf["multiline"])
	}
}

// countRule returns how many findings carry the given rule id.
func countRule(fs []Finding, ruleID string) int {
	n := 0
	for _, f := range fs {
		if f.RuleID == ruleID {
			n++
		}
	}
	return n
}

func hasFinding(fs []Finding, file, ruleID string, line int) bool {
	for _, f := range fs {
		if f.File == file && f.RuleID == ruleID && f.Line == line {
			return true
		}
	}
	return false
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record-lint.json")
	body := `{
      "roots": ["rec"],
      "banned_tokens": [{"id":"t1","pattern":"foo","message":"no foo","severity":"blocker","successor":"bar","allow_context":["ok"]}],
      "rules": {"no_git_metadata": {"enabled": true, "severity": "blocker", "fields": ["created"]}}
    }`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Roots) != 1 || cfg.Roots[0] != "rec" {
		t.Errorf("roots = %v", cfg.Roots)
	}
	if len(cfg.BannedTokens) != 1 || cfg.BannedTokens[0].ID != "t1" {
		t.Errorf("banned_tokens = %v", cfg.BannedTokens)
	}
	if cfg.BannedTokens[0].Successor != "bar" {
		t.Errorf("banned_tokens[0].Successor = %q, want bar", cfg.BannedTokens[0].Successor)
	}
	if r := cfg.Rules["no_git_metadata"]; !r.Enabled || r.Fields[0] != "created" {
		t.Errorf("no_git_metadata rule = %+v", r)
	}
}

func TestBannedTokens(t *testing.T) {
	root := t.TempDir()
	// bad: matches with no allow context
	writeFile(t, root, "rec/bad.md", "the tool intent_lint.py is gone\nplain line\n")
	// allow context suppresses
	writeFile(t, root, "rec/allowed.md", "historical note: intent_lint.py existed\n")
	// inside a code fence -> skipped by default
	writeFile(t, root, "rec/fenced.md", "text\n```\nintent_lint.py\n```\nmore\n")
	// clean file
	writeFile(t, root, "rec/clean.md", "nothing to see here\n")

	cfg := Config{
		Roots: []string{"rec"},
		BannedTokens: []BannedToken{
			{ID: "py", Pattern: `intent_lint\.py`, Message: "no python name", Severity: "blocker", AllowContext: []string{"historical"}},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "py"); n != 1 {
		t.Fatalf("expected 1 banned-token finding, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "bad.md"), "py", 1) {
		t.Errorf("missing expected finding on bad.md:1: %+v", fs)
	}
}

// TestDocsLintHarnessNameGate guards the real .abcd/docs-lint.json harness-name
// family (the prevention gate): a specific agent-harness name in user-facing
// content is a blocker, and the docs-lint:allow comment on the same line
// suppresses it. Loading the actual config means deleting the family (or dropping
// its blocker severity) fails this test.
func TestDocsLintHarnessNameGate(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "docs-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	root := t.TempDir()
	// The real docs-lint.json roots are ["docs", "README.md"]; both must resolve
	// now that an unresolvable configured root fails loud (GitHub #360).
	writeFile(t, root, "README.md", "# readme\n")
	// Its name_roots must resolve too (iss-279).
	for _, r := range []string{".abcd/README.md", "AGENTS.md", "CONTRIBUTING.md", "scripts/README.md"} {
		writeFile(t, root, r, "# t\n")
	}
	writeFile(t, root, "docs/named.md", "# t\n\nRun this in Claude Code.\n")
	writeFile(t, root, "docs/allowed.md", "# t\n\n<!-- docs-lint: allow --> Claude Code is named deliberately.\n")
	writeFile(t, root, "docs/clean.md", "# t\n\nUse the agent harness.\n")

	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(fs, filepath.Join("docs", "named.md"), "harness/claude-code", 3) {
		t.Fatalf("expected harness/claude-code blocker on docs/named.md:3: %+v", fs)
	}
	for _, f := range fs {
		if f.RuleID == "harness/claude-code" {
			if f.Severity != "blocker" {
				t.Errorf("harness/claude-code severity = %q, want blocker", f.Severity)
			}
			if f.File != filepath.Join("docs", "named.md") {
				t.Errorf("harness gate fired outside named.md (allow-context/clean leaked): %+v", f)
			}
		}
	}
}

func TestNoGitMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/bad.md", "---\nid: x\nupdated: 2026-01-01\nauthor: someone\n---\n# Title\n")
	writeFile(t, root, "rec/clean.md", "---\nid: x\nkind: standalone\n---\n# Title\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"no_git_metadata": {Enabled: true, Severity: "blocker", Fields: []string{"created", "updated", "author"}},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "no_git_metadata"); n != 2 {
		t.Fatalf("expected 2 metadata findings, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "bad.md"), "no_git_metadata", 3) {
		t.Errorf("expected finding on updated (line 3): %+v", fs)
	}
}

func TestLinksResolve(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/target.md", "# Target\n")
	writeFile(t, root, "rec/doc.md",
		"good: [t](target.md)\n"+
			"anchor ok: [t](target.md#section)\n"+
			"external: [x](https://example.com)\n"+
			"broken: [t](missing.md)\n"+
			"escape: [t](../../../etc/passwd)\n"+
			"fenced skip:\n```\n[t](also-missing.md)\n```\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"links_resolve": {Enabled: true, Severity: "blocker"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	// missing.md and the escape -> 2 findings; the fenced link and externals ignored.
	if n := countRule(fs, "links_resolve"); n != 2 {
		t.Fatalf("expected 2 link findings, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 4) {
		t.Errorf("expected broken-link finding on line 4: %+v", fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 5) {
		t.Errorf("expected escape finding on line 5: %+v", fs)
	}
}

// TestLinksResolveSkipsInlineCodeSpans: a code span takes precedence over link
// syntax (CommonMark), so `Get[T](s, key)` in a record is code, not a link to a
// file called "s, key" (iss-2609250915305413). A real link beside a span, and a
// link whose text is a span, are still checked.
func TestLinksResolveSkipsInlineCodeSpans(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/target.md", "# Target\n")
	writeFile(t, root, "rec/doc.md",
		"api: `Get[T](s, key, bundled, check)` and `Decode[T](raw)`\n"+
			"beside: `x[i](j)` then [t](missing.md)\n"+
			"code text: [`target`](target.md) and [`gone`](gone.md)\n")
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{"links_resolve": {Enabled: true, Severity: "blocker"}},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "links_resolve"); n != 2 {
		t.Fatalf("expected 2 link findings (missing.md, gone.md), got %d: %+v", n, fs)
	}
	for _, line := range []int{2, 3} {
		if !hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", line) {
			t.Errorf("expected a finding on line %d: %+v", line, fs)
		}
	}
}

func TestBrittleLineRefs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/doc.md", "see configuration.md:171 for detail\nno ref here\nalso other.md:9 inline\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"no_brittle_line_refs": {Enabled: true, Severity: "warn"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "no_brittle_line_refs"); n != 2 {
		t.Fatalf("expected 2 brittle-ref findings, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "doc.md"), "no_brittle_line_refs", 1) {
		t.Errorf("expected finding on line 1: %+v", fs)
	}
}

func TestBrittleLineRefsSkipsFencedBlocks(t *testing.T) {
	root := t.TempDir()
	// A fenced block quoting tool output must not trip the rule — it is a
	// verbatim example, not a live cross-reference, mirroring the fence-mask
	// house convention every sibling content rule honours.
	writeFile(t, root, "rec/doc.md",
		"quoted example below\n```\nconfiguration.md:171 stale anchor\n```\nno ref here\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"no_brittle_line_refs": {Enabled: true, Severity: "warn"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "no_brittle_line_refs"); n != 0 {
		t.Fatalf("fenced brittle ref should be masked, got %d findings: %+v", n, fs)
	}
}

func TestDirectoryCoverage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/README.md", "# rec\n")
	writeFile(t, root, "rec/covered/README.md", "# covered\n")
	writeFile(t, root, "rec/covered/note.md", "hi\n")
	writeFile(t, root, "rec/bare/note.md", "no readme here\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"directory_coverage": {Enabled: true, Severity: "warn", Exempt: []string{}},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "directory_coverage"); n != 1 {
		t.Fatalf("expected 1 coverage finding, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("rec", "bare"), "directory_coverage", 0) {
		t.Errorf("expected coverage finding on rec/bare: %+v", fs)
	}

	// exempting the bare dir clears the finding.
	cfg.Rules["directory_coverage"] = RuleConfig{Enabled: true, Severity: "warn", Exempt: []string{filepath.Join("rec", "bare")}}
	fs, err = Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "directory_coverage"); n != 0 {
		t.Fatalf("exempt glob should clear finding, got %d: %+v", n, fs)
	}
}

func TestIntentLifecycle(t *testing.T) {
	root := t.TempDir()
	base := "rec/intents"

	// Well-formed intents across buckets (the reference targets for superseded).
	writeFile(t, root, base+"/shipped/itd-48-good.md", "---\nid: itd-48\nkind: standalone\nspec_id: spc-10-thing\n---\n# ok\n")
	writeFile(t, root, base+"/planned/itd-20-good.md", "---\nid: itd-20\nkind: bundle-member\nspec_id: spc-83-thing\n---\n# ok\n")
	writeFile(t, root, base+"/planned/itd-23-good.md", "---\nid: itd-23\nkind: standalone\nspec_id: null\n---\n# ok\n") // re-baseline: planned may be unscheduled (null spec_id)
	writeFile(t, root, base+"/drafts/itd-10-good.md", "---\nid: itd-10\nkind: null\nspec_id: null\n---\n# ok\n")
	writeFile(t, root, base+"/disciplines/itd-1-good.md", "---\nid: itd-1\nkind: discipline\nspec_id: null\n---\n# ok\n")
	writeFile(t, root, base+"/superseded/itd-31-good.md", "---\nid: itd-31\nkind: standalone\nsuperseded_by: itd-48\n---\n# ok\n")

	// Violations, one family per file.
	writeFile(t, root, base+"/drafts/itd-11-bad.md", "---\nid: itd-11\nkind: null\nspec_id: spc-99-nope\n---\n# bad\n")                        // drafts spec_id must be null
	writeFile(t, root, base+"/planned/itd-21-bad.md", "---\nid: itd-21\nkind: null\nspec_id: spc-1-x\n---\n# bad\n")                           // planned kind must be non-null
	writeFile(t, root, base+"/planned/itd-22-bad.md", "---\nid: itd-22\nkind: standalone\nspec_id: itd-5-wrong\n---\n# bad\n")                 // planned spec_id must be ^spc-
	writeFile(t, root, base+"/shipped/itd-49-bad.md", "---\nid: itd-49\nkind: standalone\nspec_id: null\n---\n# bad\n")                        // shipped spec_id must be non-null
	writeFile(t, root, base+"/disciplines/itd-2-bad.md", "---\nid: itd-2\nkind: standalone\nspec_id: null\n---\n# bad\n")                      // disciplines kind must be discipline
	writeFile(t, root, base+"/superseded/itd-32-bad.md", "---\nid: itd-32\nkind: standalone\nsuperseded_by: itd-999\n---\n# bad\n")            // target missing
	writeFile(t, root, base+"/shipped/itd-50-status.md", "---\nid: itd-50\nkind: standalone\nspec_id: spc-2-x\nstatus: shipped\n---\n# bad\n") // status: forbidden

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		file string
		line int
	}{
		{filepath.Join(base, "drafts", "itd-11-bad.md"), 4},  // spec_id line
		{filepath.Join(base, "planned", "itd-21-bad.md"), 3}, // kind line
		{filepath.Join(base, "planned", "itd-22-bad.md"), 4}, // spec_id line
		{filepath.Join(base, "shipped", "itd-49-bad.md"), 4}, // spec_id line
		{filepath.Join(base, "disciplines", "itd-2-bad.md"), 3},
		{filepath.Join(base, "superseded", "itd-32-bad.md"), 4},
		{filepath.Join(base, "shipped", "itd-50-status.md"), 5}, // status line
	}
	for _, c := range checks {
		if !hasFinding(fs, c.file, "intent_lifecycle", c.line) {
			t.Errorf("expected intent_lifecycle finding on %s:%d; got %+v", c.file, c.line, fs)
		}
	}
	// The good intents must produce no lifecycle findings.
	good := []string{"itd-48-good.md", "itd-20-good.md", "itd-23-good.md", "itd-10-good.md", "itd-1-good.md", "itd-31-good.md"}
	for _, f := range fs {
		for _, g := range good {
			if filepath.Base(f.File) == g {
				t.Errorf("unexpected finding on clean intent %s: %+v", g, f)
			}
		}
	}
}

// Two parallel branches each allocate "the next free" intent id, and both land.
// The id is the intent's identity across the whole record, so a duplicate is a
// blocker on BOTH files -- neither is authoritative, and a reader cannot tell
// which itd-N a cross-reference means.
func TestIntentLifecycleDuplicateID(t *testing.T) {
	root := t.TempDir()
	base := "rec/intents"

	// Same id, different slugs, different buckets -- the real collision shape.
	writeFile(t, root, base+"/drafts/itd-82-first-one.md", "---\nid: itd-82\nkind: null\nspec_id: null\n---\n# one\n")
	writeFile(t, root, base+"/drafts/itd-82-second-one.md", "---\nid: itd-82\nkind: null\nspec_id: null\n---\n# two\n")
	// A collision across buckets must be caught too, not just within one.
	writeFile(t, root, base+"/planned/itd-90-here.md", "---\nid: itd-90\nkind: standalone\nspec_id: spc-3-x\n---\n# a\n")
	writeFile(t, root, base+"/drafts/itd-90-there.md", "---\nid: itd-90\nkind: null\nspec_id: null\n---\n# b\n")
	// A unique id must stay clean.
	writeFile(t, root, base+"/drafts/itd-91-unique.md", "---\nid: itd-91\nkind: null\nspec_id: null\n---\n# ok\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"intent_lifecycle": {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	// Every file in a colliding set is flagged -- flagging only one would imply
	// the other is the "right" one, which the linter cannot know.
	dupes := []string{
		filepath.Join(base, "drafts", "itd-82-first-one.md"),
		filepath.Join(base, "drafts", "itd-82-second-one.md"),
		filepath.Join(base, "planned", "itd-90-here.md"),
		filepath.Join(base, "drafts", "itd-90-there.md"),
	}
	for _, f := range dupes {
		if !hasFinding(fs, f, "intent_lifecycle", 2) { // the id: line
			t.Errorf("expected duplicate-id finding on %s:2; got %+v", f, fs)
		}
	}
	for _, f := range fs {
		if filepath.Base(f.File) == "itd-91-unique.md" {
			t.Errorf("unexpected finding on unique intent: %+v", f)
		}
	}
}

// A hand-added issue file that bypassed the capture allocator can land an iss-N
// id another file already claims (how a past iss-56 collision arose — the
// allocator only rejects duplicates on the reservation path). The id is the
// issue's identity across the ledger, so a duplicate is a blocker on BOTH files:
// neither is authoritative, and a cross-reference cannot tell which iss-N it
// means. The rule scans all three status dirs (open/, resolved/, wontfix/).
func TestIssueIDUniqueDuplicate(t *testing.T) {
	root := t.TempDir()
	base := ".abcd/work/issues"

	// Same id, different slugs, different status dirs -- the real collision shape.
	writeFile(t, root, base+"/open/iss-56-first-one.md", "---\nid: \"iss-56\"\n---\n# one\n")
	writeFile(t, root, base+"/resolved/iss-56-second-one.md", "---\nid: \"iss-56\"\n---\n# two\n")
	// A collision within a single status dir must be caught too.
	writeFile(t, root, base+"/open/iss-70-here.md", "---\nid: \"iss-70\"\n---\n# a\n")
	writeFile(t, root, base+"/wontfix/iss-70-there.md", "---\nid: \"iss-70\"\n---\n# b\n")
	// A unique id must stay clean.
	writeFile(t, root, base+"/open/iss-71-unique.md", "---\nid: \"iss-71\"\n---\n# ok\n")

	cfg := Config{
		Rules: map[string]RuleConfig{
			"issue_id_unique": {Enabled: true, Severity: "blocker", IssuesDir: ".abcd/work/issues"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	// Every file in a colliding set is flagged -- flagging only one would imply
	// the other is the "right" one, which the linter cannot know.
	dupes := []string{
		filepath.Join(base, "open", "iss-56-first-one.md"),
		filepath.Join(base, "resolved", "iss-56-second-one.md"),
		filepath.Join(base, "open", "iss-70-here.md"),
		filepath.Join(base, "wontfix", "iss-70-there.md"),
	}
	for _, f := range dupes {
		if !hasFinding(fs, f, "issue_id_unique", 2) { // the id: line
			t.Errorf("expected duplicate-id finding on %s:2; got %+v", f, fs)
		}
	}
	for _, f := range fs {
		if filepath.Base(f.File) == "iss-71-unique.md" {
			t.Errorf("unexpected finding on unique issue: %+v", f)
		}
	}
}

// TestIssueIDUniqueZeroPadded is iss-392: a zero-padded filename spells the same
// canonical id as its un-padded twin (iss-0100 == iss-100), so the uniqueness
// backstop must key on the canonical id and catch the collision rather than
// treating the two spellings as distinct ids.
func TestIssueIDUniqueZeroPadded(t *testing.T) {
	root := t.TempDir()
	base := ".abcd/work/issues"
	writeFile(t, root, base+"/open/iss-100-canonical.md", "---\nid: \"iss-100\"\n---\n# one\n")
	writeFile(t, root, base+"/resolved/iss-0100-padded.md", "---\nid: \"iss-100\"\n---\n# two\n")

	cfg := Config{Rules: map[string]RuleConfig{
		"issue_id_unique": {Enabled: true, Severity: "blocker", IssuesDir: ".abcd/work/issues"},
	}}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		filepath.Join(base, "open", "iss-100-canonical.md"),
		filepath.Join(base, "resolved", "iss-0100-padded.md"),
	} {
		if !hasFinding(fs, f, "issue_id_unique", 2) {
			t.Errorf("zero-padded collision not caught on %s; got %+v", f, fs)
		}
	}
}

func TestSpecIDUniqueDuplicate(t *testing.T) {
	root := t.TempDir()
	base := "rec/specs"

	// The real collision shape: two branches each minted spc-10 with a different
	// slug and different linked intent, then merged. Both files claim spc-10.
	writeFile(t, root, base+"/open/spc-10-alpha.md", "---\nid: spc-10\nslug: alpha\nintent: itd-70\n---\n# a\n")
	writeFile(t, root, base+"/closed/spc-10-beta.md", "---\nid: spc-10\nslug: beta\nintent: itd-80\n---\n# b\n")
	// A unique id must stay clean under this rule.
	writeFile(t, root, base+"/open/spc-11-solo.md", "---\nid: spc-11\nslug: solo\nintent: itd-90\n---\n# c\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"spec_id_unique": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	// Every file in the colliding set is flagged — flagging only one would imply
	// the other is authoritative, which the linter cannot know.
	dupes := []string{
		filepath.Join(base, "open", "spc-10-alpha.md"),
		filepath.Join(base, "closed", "spc-10-beta.md"),
	}
	for _, f := range dupes {
		if !hasFinding(fs, f, "spec_id_unique", 2) { // the id: line
			t.Errorf("expected duplicate-id finding on %s:2; got %+v", f, fs)
		}
	}
	for _, f := range fs {
		if filepath.Base(f.File) == "spc-11-solo.md" && f.RuleID == "spec_id_unique" {
			t.Errorf("unexpected spec_id_unique finding on unique spec: %+v", f)
		}
	}
}

// TestSpecIDUniqueZeroPadded is the spec-family sibling of TestIssueIDUniqueZeroPadded
// (iss-392): spc-005 and spc-5 are one canonical id, so the uniqueness backstop
// must key on the canonical spelling and catch the collision. Before the
// canonRecordID keyspace fix the spec backstop keyed on the raw frontmatter
// value and failed open on a padded twin.
func TestSpecIDUniqueZeroPadded(t *testing.T) {
	root := t.TempDir()
	base := "rec/specs"
	writeFile(t, root, base+"/open/spc-5-alpha.md", "---\nid: spc-5\nslug: alpha\nintent: itd-70\n---\n# a\n")
	writeFile(t, root, base+"/closed/spc-005-beta.md", "---\nid: spc-005\nslug: beta\nintent: itd-80\n---\n# b\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"spec_id_unique": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		filepath.Join(base, "open", "spc-5-alpha.md"),
		filepath.Join(base, "closed", "spc-005-beta.md"),
	} {
		if !hasFinding(fs, f, "spec_id_unique", 2) {
			t.Errorf("zero-padded spec collision not caught on %s; got %+v", f, fs)
		}
	}
}

// TestSpecLifecyclePaddedIntentResolves pins the lookup side of the canonical-id
// keyspace: a spec whose intent link and a padded intent file spell the same id
// differently must resolve, not false-block. Before the fix the KnownIntents map
// keyed on the canonical spelling while the lookup used the raw value (or vice
// versa), so a padded intent produced a phantom "does not exist in any bucket".
func TestSpecLifecyclePaddedIntentResolves(t *testing.T) {
	root := t.TempDir()
	// Intent id spelled zero-padded (itd-047); the spec links it canonically
	// (itd-47). The two are one handle, so the link must resolve.
	writeFile(t, root, "rec/intents/planned/itd-047-thing.md",
		"---\nid: itd-047\nspec_id: spc-9\n---\n# intent\n")
	writeFile(t, root, "rec/specs/open/spc-9-thing.md",
		"---\nid: spc-9\nslug: thing\nintent: itd-47\n---\n# spec\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"spec_lifecycle": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == "spec_lifecycle" && strings.Contains(f.Message, "does not exist in any bucket") {
			t.Errorf("padded intent falsely reported missing: %+v", f)
		}
	}
}

func TestExemptions(t *testing.T) {
	root := t.TempDir()
	// A banned token in an exempt_paths file → no finding.
	writeFile(t, root, "rec/research/note.md", "the old intent_lint.py ran here\n")
	// The same token in a non-exempt file → finding.
	writeFile(t, root, "rec/live/doc.md", "the tool intent_lint.py is referenced\n")
	// A status:-exempt file: banned token is excused, but a broken link in it
	// STILL fires — structural checks stay universal.
	writeFile(t, root, "rec/live/old.md",
		"---\nid: x\nstatus: superseded\n---\nintent_lint.py mentioned\n[dead](missing.md)\n")

	cfg := Config{
		Roots: []string{"rec"},
		BannedTokens: []BannedToken{
			{ID: "py", Pattern: `intent_lint\.py`, Message: "no python name", Severity: "blocker"},
		},
		Rules: map[string]RuleConfig{
			"links_resolve": {Enabled: true, Severity: "blocker"},
		},
		ExemptPaths:    []string{filepath.Join("rec", "research") + string(filepath.Separator)},
		ExemptIfStatus: []string{"superseded"},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	// research note: exempt from banned tokens.
	if hasFinding(fs, filepath.Join("rec", "research", "note.md"), "py", 1) {
		t.Errorf("exempt_paths file should not fire banned token: %+v", fs)
	}
	// live doc: not exempt.
	if !hasFinding(fs, filepath.Join("rec", "live", "doc.md"), "py", 1) {
		t.Errorf("non-exempt file should fire banned token: %+v", fs)
	}
	// status:superseded file: banned token exempt...
	if hasFinding(fs, filepath.Join("rec", "live", "old.md"), "py", 5) {
		t.Errorf("status-exempt file should not fire banned token: %+v", fs)
	}
	// ...but its broken link STILL fires (structural, universal).
	if !hasFinding(fs, filepath.Join("rec", "live", "old.md"), "links_resolve", 6) {
		t.Errorf("structural link check must stay universal in exempt file: %+v", fs)
	}
	if n := countRule(fs, "py"); n != 1 {
		t.Fatalf("expected exactly 1 banned-token finding, got %d: %+v", n, fs)
	}
}

func TestSurfaceCoverage(t *testing.T) {
	root := t.TempDir()

	// Real plugin surfaces: three commands and one skill. The directory is flat —
	// a subdirectory would be an extra harness namespace segment, so anything
	// nested is a different command name than the registry describes.
	writeFile(t, root, "commands/ahoy.md", "# ahoy\n")
	writeFile(t, root, "commands/capture.md", "# capture\n")
	writeFile(t, root, "commands/orphan.md", "# orphan\n") // no registry row → coverage fires
	writeFile(t, root, "commands/README.md", "# index\n")  // README is not a surface, skipped
	writeFile(t, root, "commands/abcd.md", "# board\n")    // the bare top-level, skipped via BareCommand
	writeFile(t, root, "commands/nested/deep.md", "# deep\n")
	writeFile(t, root, "skills/review/SKILL.md", "# review skill\n")

	// The brief surface registry. Column order deliberately differs from the
	// checked columns to prove the parser keys off the header, not fixed offsets.
	registry := "# Surfaces\n\n" +
		"| # | Command | Status | Purpose | File |\n" +
		"|---|---|---|---|---|\n" +
		"| 1 | `/abcd:ahoy` | shipped | Install | [`01-ahoy.md`](01-ahoy.md) |\n" + // ok: shipped + file
		"| 2 | `/abcd:disembark` | staged | Pack | [`02.md`](02.md) |\n" + // ok: staged, no file
		"| 3 | `/abcd:capture` | staged | Ledger | [`06.md`](06.md) |\n" + // staged but file exists → fires
		"| 4 | `/abcd:launch` | shipped | Release | [`04.md`](04.md) |\n" + // shipped but no file → fires
		"| 5 | `/abcd:review` | shipped | Review | [`05.md`](05.md) |\n" + // ok: shipped + skill dir
		"| 6 | `/abcd` | shipped | Board | [`08.md`](08.md) |\n" + // bare top-level: file check skipped
		"| 7 | `/abcd:weird` | bogus | Bad | [`07.md`](07.md) |\n" // unknown status → fires
	writeFile(t, root, "rec/registry.md", registry)

	cfg := Config{
		Rules: map[string]RuleConfig{
			"surface_coverage": {
				Enabled:     true,
				Severity:    "blocker",
				CommandsDir: "commands",
				BareCommand: "abcd",
				SkillsDir:   "skills",
				Registry:    filepath.Join("rec", "registry.md"),
			},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	reg := filepath.Join("rec", "registry.md")
	want := []struct {
		file string
		line int
		desc string
	}{
		{filepath.Join("commands", "orphan.md"), 0, "command with no registry row"},
		{reg, 7, "staged row with a backing surface"},
		{reg, 8, "shipped row with no backing surface"},
		{reg, 11, "row with an unknown status"},
	}
	for _, w := range want {
		if !hasFinding(fs, w.file, "surface_coverage", w.line) {
			t.Errorf("expected surface_coverage finding (%s) on %s:%d; got %+v", w.desc, w.file, w.line, fs)
		}
	}
	if n := countRule(fs, "surface_coverage"); n != len(want) {
		t.Fatalf("expected exactly %d surface_coverage findings, got %d: %+v", len(want), n, fs)
	}
	// itd-147 ac-7: every finding names the rule as the row-level presence check
	// it is, and disclaims chapter prose, so a green run cannot be read as a
	// statement that the chapters are correct.
	for _, f := range fs {
		if f.RuleID == "surface_coverage" && !strings.HasPrefix(f.Message, SurfaceCoverageLabel) {
			t.Errorf("surface_coverage finding does not carry its row-level label: %q", f.Message)
		}
	}
	// The label names both grains the rule checks: the surfaces index rows and
	// each chapter's `## Sub-verbs` table rows, since the sub-verb findings carry
	// the same label.
	if !strings.Contains(SurfaceCoverageLabel, "row-level presence check over the surfaces index and each chapter's sub-verb table") || !strings.Contains(SurfaceCoverageLabel, "not whether a chapter's prose is correct") {
		t.Errorf("SurfaceCoverageLabel = %q; it must name the row-level presence check over the index and each chapter's sub-verb table, and disclaim chapter prose", SurfaceCoverageLabel)
	}

	// A well-formed registry over the real surfaces produces zero findings.
	clean := "# Surfaces\n\n" +
		"| # | Command | Status | File |\n" +
		"|---|---|---|---|\n" +
		"| 1 | `/abcd:ahoy` | shipped | [`a.md`](a.md) |\n" +
		"| 2 | `/abcd:capture` | shipped | [`c.md`](c.md) |\n" +
		"| 3 | `/abcd:orphan` | shipped | [`o.md`](o.md) |\n" +
		"| 4 | `/abcd:review` | shipped | [`r.md`](r.md) |\n" +
		"| 5 | `/abcd:disembark` | staged | [`d.md`](d.md) |\n" +
		"| 6 | `/abcd` | shipped | [`x.md`](x.md) |\n"
	writeFile(t, root, "rec/clean.md", clean)
	cfg.Rules["surface_coverage"] = RuleConfig{
		Enabled: true, Severity: "blocker",
		CommandsDir: "commands", BareCommand: "abcd", SkillsDir: "skills",
		Registry: filepath.Join("rec", "clean.md"),
	}
	fs, err = Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "surface_coverage"); n != 0 {
		t.Fatalf("clean registry should yield no findings, got %d: %+v", n, fs)
	}

	// A missing registry file is not an error (nothing to cross-check).
	cfg.Rules["surface_coverage"] = RuleConfig{
		Enabled: true, Severity: "blocker",
		CommandsDir: "commands", BareCommand: "abcd", Registry: filepath.Join("rec", "absent.md"),
	}
	if _, err := Lint(cfg, root); err != nil {
		t.Fatalf("missing registry must not error: %v", err)
	}

	// A fenced example table before the real one must be ignored (fence mask):
	// the parser keys off the real table, so the clean surfaces stay clean. Were
	// the fenced header latched onto, every real surface would fire a bogus
	// coverage finding.
	fenced := "# Surfaces\n\n" +
		"Example of the table shape:\n\n" +
		"```\n" +
		"| # | Command | Status | File |\n" +
		"|---|---|---|---|\n" +
		"| 1 | `/abcd:example` | shipped | [`x.md`](x.md) |\n" +
		"```\n\n" +
		"The real registry:\n\n" +
		"| # | Command | Status | File |\n" +
		"|---|---|---|---|\n" +
		"| 1 | `/abcd:ahoy` | shipped | [`a.md`](a.md) |\n" +
		"| 2 | `/abcd:capture` | shipped | [`c.md`](c.md) |\n" +
		"| 3 | `/abcd:orphan` | shipped | [`o.md`](o.md) |\n" +
		"| 4 | `/abcd:review` | shipped | [`r.md`](r.md) |\n"
	writeFile(t, root, "rec/fenced.md", fenced)
	cfg.Rules["surface_coverage"] = RuleConfig{
		Enabled: true, Severity: "blocker",
		CommandsDir: "commands", BareCommand: "abcd", SkillsDir: "skills",
		Registry: filepath.Join("rec", "fenced.md"),
	}
	fs, err = Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "surface_coverage"); n != 0 {
		t.Fatalf("fenced example table must be ignored, got %d findings: %+v", n, fs)
	}
}

func TestReceiptGate(t *testing.T) {
	root := t.TempDir()
	const sha = "0123456789abcdef0123456789abcdef01234567"
	const other = "ffffffffffffffffffffffffffffffffffffffff"
	reviews := filepath.Join(".abcd", "work", "reviews")

	// A valid receipt binds its policy.detector to the gate it attests. Callers
	// pass the detector explicitly so a receipt can be built for the WRONG gate.
	receipt := func(commit, result, model, detector string) string {
		return `{
  "subject": {"digest": {"gitCommit": "` + commit + `"}},
  "verifier": {"id": "maintainer"},
  "timeVerified": "2026-07-11T00:00:00Z",
  "verificationResult": "` + result + `",
  "judgeModel": "` + model + `",
  "policy": {"detector": "` + detector + `", "version": "1"}
}`
	}
	gates := []string{"docs-currency-reviewer", "iss35-brief-surface-crosscheck"}
	baseCfg := func() RuleConfig {
		return RuleConfig{
			Enabled: true, Severity: "blocker",
			ReceiptsDir: reviews, Commit: sha, RequiredGates: gates,
		}
	}
	run := func(cfg RuleConfig) []Finding {
		fs, err := Lint(Config{Rules: map[string]RuleConfig{"receipt_gate": cfg}}, root)
		if err != nil {
			t.Fatal(err)
		}
		return fs
	}
	put := func(commit, gate, body string) {
		writeFile(t, root, filepath.Join(reviews, commit, gate+".json"), body)
	}

	// 1. Every required gate has a PROMOTE receipt for the target sha whose
	// policy.detector names that gate → clean.
	put(sha, "docs-currency-reviewer", receipt(sha, "PROMOTE", "claude-opus-4-8", "docs-currency-reviewer"))
	put(sha, "iss35-brief-surface-crosscheck", receipt(sha, "PROMOTE", "claude-opus-4-8", "iss35-brief-surface-crosscheck"))
	if n := countRule(run(baseCfg()), "receipt_gate"); n != 0 {
		t.Fatalf("all-PROMOTE receipts should be clean, got %d", n)
	}

	// 2. Disabled → never fires, even pointed at a sha with no receipts.
	cfg := baseCfg()
	cfg.Enabled, cfg.Commit = false, other
	if n := countRule(run(cfg), "receipt_gate"); n != 0 {
		t.Fatalf("disabled rule must not fire, got %d", n)
	}

	// 3. Missing receipts for the target sha → fail-closed BLOCK, one per gate.
	cfg = baseCfg()
	cfg.Commit = other
	if n := countRule(run(cfg), "receipt_gate"); n != len(gates) {
		t.Fatalf("missing receipts should fire once per gate (%d), got %d", len(gates), n)
	}

	// 4. A HOLD verdict blocks (only the HOLD gate fires).
	put(other, "docs-currency-reviewer", receipt(other, "HOLD", "claude-opus-4-8", "docs-currency-reviewer"))
	put(other, "iss35-brief-surface-crosscheck", receipt(other, "PROMOTE", "claude-opus-4-8", "iss35-brief-surface-crosscheck"))
	if n := countRule(run(cfg), "receipt_gate"); n != 1 {
		t.Fatalf("one HOLD should fire once, got %d", n)
	}

	// 5. subject digest ≠ target commit → BLOCK (receipt not for this release).
	put(sha, "iss35-brief-surface-crosscheck", receipt("deadbeef", "PROMOTE", "claude-opus-4-8", "iss35-brief-surface-crosscheck"))
	if n := countRule(run(baseCfg()), "receipt_gate"); n != 1 {
		t.Fatalf("subject mismatch should fire, got %d", n)
	}
	put(sha, "iss35-brief-surface-crosscheck", receipt(sha, "PROMOTE", "claude-opus-4-8", "iss35-brief-surface-crosscheck"))

	// 6. A blank (floating) judge model is not auditable → BLOCK.
	put(sha, "docs-currency-reviewer", receipt(sha, "PROMOTE", "", "docs-currency-reviewer"))
	if n := countRule(run(baseCfg()), "receipt_gate"); n != 1 {
		t.Fatalf("blank judge model should fire, got %d", n)
	}
	put(sha, "docs-currency-reviewer", receipt(sha, "PROMOTE", "claude-opus-4-8", "docs-currency-reviewer"))

	// 7. Malformed receipt JSON → BLOCK (never a silent pass).
	put(sha, "iss35-brief-surface-crosscheck", "{ not json")
	if n := countRule(run(baseCfg()), "receipt_gate"); n != 1 {
		t.Fatalf("malformed receipt should fire, got %d", n)
	}
	put(sha, "iss35-brief-surface-crosscheck", receipt(sha, "PROMOTE", "claude-opus-4-8", "iss35-brief-surface-crosscheck"))

	// 8. Enabled with no target commit configured → fail-closed config error.
	cfg = baseCfg()
	cfg.Commit = ""
	if n := countRule(run(cfg), "receipt_gate"); n == 0 {
		t.Fatal("enabled rule with no target commit must fail closed")
	}

	// 9. Enabled with an empty required-gates list → fail closed (verifies nothing
	// otherwise). Symmetric with the empty-commit guard.
	cfg = baseCfg()
	cfg.RequiredGates = nil
	if n := countRule(run(cfg), "receipt_gate"); n == 0 {
		t.Fatal("enabled rule with no required gates must fail closed, not pass vacuously")
	}

	// 10. A target commit that is not a valid sha (a path-traversal attempt) →
	// fail closed, never a filesystem escape.
	cfg = baseCfg()
	cfg.Commit = "../../etc"
	if n := countRule(run(cfg), "receipt_gate"); n == 0 {
		t.Fatal("a non-sha target commit must fail closed")
	}

	// 11. An unsafe gate name (path component) → fail closed for that gate.
	cfg = baseCfg()
	cfg.RequiredGates = []string{"../evil"}
	if n := countRule(run(cfg), "receipt_gate"); n == 0 {
		t.Fatal("an unsafe gate name must fail closed")
	}

	// 12. A genuine PROMOTE receipt for one detector, copied to another gate's
	// path, must NOT satisfy that gate — the receipt is bound to its detector, not
	// its filename. This is the one-receipt-satisfies-every-gate hole (C16).
	put(sha, "docs-currency-reviewer", receipt(sha, "PROMOTE", "claude-opus-4-8", "docs-currency-reviewer"))
	put(sha, "iss35-brief-surface-crosscheck", receipt(sha, "PROMOTE", "claude-opus-4-8", "docs-currency-reviewer"))
	if n := countRule(run(baseCfg()), "receipt_gate"); n != 1 {
		t.Fatalf("a receipt whose detector names a different gate must fire, got %d", n)
	}

	// 13. A receipt with a blank/absent policy.detector cannot be bound to a gate
	// → BLOCK (never a silent pass on an unbound attestation).
	put(sha, "iss35-brief-surface-crosscheck", receipt(sha, "PROMOTE", "claude-opus-4-8", ""))
	if n := countRule(run(baseCfg()), "receipt_gate"); n != 1 {
		t.Fatalf("a receipt with no detector must fire, got %d", n)
	}
	put(sha, "iss35-brief-surface-crosscheck", receipt(sha, "PROMOTE", "claude-opus-4-8", "iss35-brief-surface-crosscheck"))
}

func TestGateLockstep(t *testing.T) {
	root := t.TempDir()

	workflow := "name: release\n" +
		"on:\n" +
		"  push:\n" +
		"    tags: ['v*']\n" +
		"jobs:\n" +
		"  verify:\n" +
		"    runs-on: ubuntu-latest\n" +
		"    steps:\n" +
		"      - name: Check out the pushed commit\n" +
		"      - name: Set up Go\n" +
		"      - name: Format (gofmt)\n" +
		"      - name: Build\n" +
		"      - name: Vet\n" +
		"  release:\n" +
		"    steps:\n" +
		"      - name: Publish\n" // a step in another job — must NOT be counted
	writeFile(t, root, ".github/workflows/release.yml", workflow)

	runbook := func(gates string) string {
		return "# Release gate\n\n## Deterministic gates (CI-enforced)\n\n" + gates +
			"\n> a staging note, not a gate\n\n## Semantic gates\n\n1. Not a deterministic gate\n"
	}
	// In-lockstep: exactly the verify gate steps (setup steps ignored).
	writeFile(t, root, "runbook.md", runbook("1. Format (gofmt)\n2. Build\n3. Vet\n"))

	cfg := func() RuleConfig {
		return RuleConfig{
			Enabled: true, Severity: "blocker",
			Runbook: "runbook.md", Workflow: ".github/workflows/release.yml",
			Job: "verify", IgnoreSteps: []string{"Check out the pushed commit", "Set up Go"},
		}
	}
	run := func() []Finding {
		fs, err := Lint(Config{Rules: map[string]RuleConfig{"gate_lockstep": cfg()}}, root)
		if err != nil {
			t.Fatal(err)
		}
		return fs
	}

	// 1. Lists agree → clean.
	if n := countRule(run(), "gate_lockstep"); n != 0 {
		t.Fatalf("matching lists should be clean, got %d: %+v", n, run())
	}

	// 2. A verify step missing from the runbook → BLOCK.
	writeFile(t, root, "runbook.md", runbook("1. Format (gofmt)\n2. Build\n")) // drop Vet
	if n := countRule(run(), "gate_lockstep"); n != 1 {
		t.Fatalf("a workflow gate missing from the runbook should fire once, got %d: %+v", n, run())
	}

	// 3. A runbook gate not in the workflow → BLOCK.
	writeFile(t, root, "runbook.md", runbook("1. Format (gofmt)\n2. Build\n3. Vet\n4. Phantom Gate\n"))
	if n := countRule(run(), "gate_lockstep"); n != 1 {
		t.Fatalf("a phantom runbook gate should fire once, got %d: %+v", n, run())
	}

	// 4. Setup steps in the ignore list are not required in the runbook (already
	// clean in case 1 proves this); dropping an ignore entry surfaces them.
	c := cfg()
	c.IgnoreSteps = nil
	writeFile(t, root, "runbook.md", runbook("1. Format (gofmt)\n2. Build\n3. Vet\n"))
	fs, err := Lint(Config{Rules: map[string]RuleConfig{"gate_lockstep": c}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "gate_lockstep"); n != 2 { // Check out + Set up Go now unlisted
		t.Fatalf("un-ignored setup steps should each fire, got %d: %+v", n, fs)
	}

	lint := func(c RuleConfig) []Finding {
		fs, err := Lint(Config{Rules: map[string]RuleConfig{"gate_lockstep": c}}, root)
		if err != nil {
			t.Fatal(err)
		}
		return fs
	}

	// 5. Non-empty floor: a renamed runbook heading parses to zero gates → fail
	// closed (the double-empty / rename silent-pass hole).
	writeFile(t, root, "renamed.md", "# R\n\n## CI checks\n\n1. Format (gofmt)\n2. Build\n3. Vet\n")
	c5 := cfg()
	c5.Runbook, c5.MinGates = "renamed.md", 3
	if n := countRule(lint(c5), "gate_lockstep"); n == 0 {
		t.Fatal("a renamed heading (zero parsed gates) must trip the non-empty floor")
	}

	// 6. The alternate step form (`- uses:` then `name:` on the next line) is seen,
	// not invisible — matching runbook ⇒ clean.
	writeFile(t, root, ".github/workflows/alt.yml", "name: r\non:\n  push:\njobs:\n  verify:\n    steps:\n"+
		"      - name: Format (gofmt)\n      - uses: some/scanner@v1\n        name: Secret Scan\n      - name: Build\n")
	writeFile(t, root, "alt-runbook.md", runbook("1. Format (gofmt)\n2. Secret Scan\n3. Build\n"))
	c6 := cfg()
	c6.Workflow, c6.Runbook, c6.MinGates, c6.IgnoreSteps = ".github/workflows/alt.yml", "alt-runbook.md", 3, nil
	if n := countRule(lint(c6), "gate_lockstep"); n != 0 {
		t.Fatalf("alternate `- uses:`/`name:` step form must be captured, got %d: %+v", n, lint(c6))
	}

	// 7. A 2-space comment inside the job does not close it early (steps after it
	// stay in scope).
	writeFile(t, root, ".github/workflows/cmt.yml", "name: r\non:\n  push:\njobs:\n  verify:\n    steps:\n"+
		"      - name: Format (gofmt)\n  # a stray 2-space comment: not a job\n      - name: Build\n      - name: Vet\n")
	writeFile(t, root, "clean-runbook.md", runbook("1. Format (gofmt)\n2. Build\n3. Vet\n"))
	c7 := cfg()
	c7.Workflow, c7.Runbook, c7.MinGates, c7.IgnoreSteps = ".github/workflows/cmt.yml", "clean-runbook.md", 3, nil
	if n := countRule(lint(c7), "gate_lockstep"); n != 0 {
		t.Fatalf("a 2-space comment must not drop steps after it, got %d: %+v", n, lint(c7))
	}

	// 7b. A step with no step-level name but a nested `with: name:` (deeper indent)
	// contributes NO gate name — the nested key is not the step's name (P4). Here
	// the upload-artifact step is unnamed, so only Format + Build are gates.
	writeFile(t, root, ".github/workflows/nested.yml", "name: r\non:\n  push:\njobs:\n  verify:\n    steps:\n"+
		"      - name: Format (gofmt)\n      - uses: actions/upload-artifact@v4\n        with:\n          name: build-logs\n      - name: Build\n")
	writeFile(t, root, "nested-runbook.md", runbook("1. Format (gofmt)\n2. Build\n"))
	c7b := cfg()
	c7b.Workflow, c7b.Runbook, c7b.MinGates, c7b.IgnoreSteps = ".github/workflows/nested.yml", "nested-runbook.md", 2, nil
	if n := countRule(lint(c7b), "gate_lockstep"); n != 0 {
		t.Fatalf("a nested with:name: must not be captured as a step name, got %d: %+v", n, lint(c7b))
	}

	// 8. A configured file that does not exist → fail loud, never silent.
	c8 := cfg()
	c8.Runbook = "does-not-exist.md"
	if n := countRule(lint(c8), "gate_lockstep"); n == 0 {
		t.Fatal("a missing configured runbook must fail closed")
	}

	// 9. Quote normalization is symmetric — a quoted workflow name equals an
	// unquoted runbook item.
	writeFile(t, root, ".github/workflows/q.yml", "name: r\non:\n  push:\njobs:\n  verify:\n    steps:\n"+
		"      - name: \"Format (gofmt)\"\n      - name: Build\n      - name: Vet\n")
	c9 := cfg()
	c9.Workflow, c9.Runbook, c9.MinGates, c9.IgnoreSteps = ".github/workflows/q.yml", "clean-runbook.md", 3, nil
	if n := countRule(lint(c9), "gate_lockstep"); n != 0 {
		t.Fatalf("quoted vs unquoted gate names must normalize equal, got %d: %+v", n, lint(c9))
	}
}

func TestArmReceiptGate(t *testing.T) {
	base := Config{Rules: map[string]RuleConfig{
		"receipt_gate": {Enabled: false, Severity: "blocker", ReceiptsDir: ".abcd/work/reviews", RequiredGates: []string{"config-gate"}},
	}}

	// Arming with a commit + explicit gates overrides enable/commit/gates.
	armed := ArmReceiptGate(base, "abc123", []string{"cli-gate-a", "cli-gate-b"})
	rc := armed.Rules["receipt_gate"]
	if !rc.Enabled || rc.Commit != "abc123" {
		t.Fatalf("arming must enable + set commit, got %+v", rc)
	}
	if len(rc.RequiredGates) != 2 || rc.RequiredGates[0] != "cli-gate-a" {
		t.Fatalf("supplied gates must override the config list, got %+v", rc.RequiredGates)
	}

	// The input config is not mutated (its map is copied).
	if base.Rules["receipt_gate"].Enabled || base.Rules["receipt_gate"].Commit != "" {
		t.Fatal("ArmReceiptGate must not mutate the input config")
	}

	// No supplied gates → the committer-editable config list is NOT inherited;
	// arming is trust-rooted to the caller, so an empty caller list clears the
	// gates and the rule fails closed at check time (P9). Never silently fall back
	// to a config a committer could have shrunk.
	cleared := ArmReceiptGate(base, "def456", nil)
	if got := cleared.Rules["receipt_gate"].RequiredGates; len(got) != 0 {
		t.Fatalf("nil gates must clear the config list (fail-closed), got %+v", got)
	}

	// A committer-downgraded severity in the config must NOT defang the armed
	// gate — arming forces blocker so the teeth are trust-rooted to the caller.
	downgraded := Config{Rules: map[string]RuleConfig{
		"receipt_gate": {Enabled: false, Severity: "warning", ReceiptsDir: ".abcd/work/reviews", RequiredGates: []string{"g"}},
	}}
	if sev := ArmReceiptGate(downgraded, "abc123", nil).Rules["receipt_gate"].Severity; sev != "blocker" {
		t.Fatalf("arming must force blocker severity, got %q", sev)
	}

	// End to end: an armed config with no receipt on disk fails closed.
	root := t.TempDir()
	fs, err := Lint(armed, root)
	if err != nil {
		t.Fatal(err)
	}
	if countRule(fs, "receipt_gate") == 0 {
		t.Fatal("an armed gate with no receipts must fire (fail-closed)")
	}
}

// presentTenseTokens mirrors the change-narration phrase list shipped in
// .abcd/docs-lint.json (word-boundary, case-insensitive, inline-escape).
func presentTenseTokens() []BannedToken {
	allow := []string{`(?i)<!--\s*docs-lint:\s*allow\b`}
	msg := "change-narration in a doc body; docs state present reality only."
	mk := func(id, pat, sev string) BannedToken {
		return BannedToken{ID: id, Pattern: pat, Severity: sev, AllowContext: allow, Message: msg}
	}
	return []BannedToken{
		mk("present_tense/previously", `(?i)\bpreviously\b`, "blocker"),
		mk("present_tense/formerly", `(?i)\bformerly\b`, "blocker"),
		mk("present_tense/to-be-implemented", `(?i)\bto be implemented\b`, "blocker"),
		mk("present_tense/has-been-replaced", `(?i)\b(has|have|had)\s+been\s+replaced\b`, "blocker"),
		mk("present_tense/we-switched", `(?i)\bwe switched\b`, "blocker"),
		mk("present_tense/renamed-from", `(?i)\brenamed from\b`, "blocker"),
		// Ambiguous with legitimate present-state prose -> advisory (warn), never block.
		mk("present_tense/no-longer", `(?i)\bno longer\b`, "warn"),
		mk("present_tense/deprecated", `(?i)\bdeprecated\b`, "warn"),
		mk("present_tense/migrated-from", `(?i)\bmigrated from\b`, "warn"),
		// NB: "used to" is intentionally absent — RE2 has no lookbehind to tell
		// change-narration ("used to be X") from passive present ("is used to X").
	}
}

func TestPresentTense(t *testing.T) {
	root := t.TempDir()
	// Unambiguous change-narration -> each phrase is a BLOCKER finding.
	writeFile(t, root, "docs/history.md",
		"The flag was previously named --old.\n"+ // previously
			"It was formerly a shell script.\n"+ // formerly
			"The engine has been replaced by a Go port.\n"+ // has been replaced
			"We switched to TOML for config.\n"+ // we switched
			"The type was renamed from Foo.\n"+ // renamed from
			"The MCP surface is to be implemented.\n") // to be implemented
	// Ambiguous phrasing that also states present reality -> WARN, never blocks.
	writeFile(t, root, "docs/ambiguous.md",
		"The upstream token is deprecated.\n"+ // deprecated (present-state)
			"Records migrated from the legacy store are validated on read.\n") // migrated from (provenance)
	// Legitimate present tense, incl. passive "is used to" -> NO finding at all
	// (this is the regression that broke a blocking gate before "used to" was dropped).
	writeFile(t, root, "docs/clean.md",
		"The --config flag is used to override defaults.\n"+
			"This directory is used to store build output.\n"+
			"Run the command now.\nFirst do X, then do Y.\nThe engine currently reads config.json.\n")
	// The inline escape suppresses an otherwise-flagged line.
	writeFile(t, root, "docs/escaped.md",
		"The API was previously named --old. <!-- docs-lint: allow -->\n")
	// Inside a fenced code block -> skipped by default.
	writeFile(t, root, "docs/fenced.md",
		"prose\n```\nthis was previously broken\n```\nmore\n")

	cfg := Config{Roots: []string{"docs"}, BannedTokens: presentTenseTokens()}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	sevOf := func(base, rule string, line int) (string, bool) {
		for _, f := range fs {
			if filepath.Base(f.File) == base && f.RuleID == rule && f.Line == line {
				return f.Severity, true
			}
		}
		return "", false
	}

	// history.md: the six unambiguous phrases each fire as a BLOCKER, one per line.
	for _, w := range []struct {
		rule string
		line int
	}{
		{"present_tense/previously", 1},
		{"present_tense/formerly", 2},
		{"present_tense/has-been-replaced", 3},
		{"present_tense/we-switched", 4},
		{"present_tense/renamed-from", 5},
		{"present_tense/to-be-implemented", 6},
	} {
		if sev, ok := sevOf("history.md", w.rule, w.line); !ok || sev != "blocker" {
			t.Errorf("expected blocker %s on history.md:%d; got sev=%q ok=%v; all=%+v", w.rule, w.line, sev, ok, fs)
		}
	}

	// ambiguous.md: deprecated + migrated-from fire, but only as WARN (gate stays satisfiable).
	for _, w := range []struct {
		rule string
		line int
	}{
		{"present_tense/deprecated", 1},
		{"present_tense/migrated-from", 2},
	} {
		if sev, ok := sevOf("ambiguous.md", w.rule, w.line); !ok || sev != "warn" {
			t.Errorf("expected warn %s on ambiguous.md:%d; got sev=%q ok=%v", w.rule, w.line, sev, ok)
		}
	}

	// clean/escaped/fenced docs produce NO finding; nothing in ambiguous.md may block.
	for _, f := range fs {
		switch filepath.Base(f.File) {
		case "clean.md", "escaped.md", "fenced.md":
			t.Errorf("unexpected present-tense finding on %s: %+v", f.File, f)
		case "ambiguous.md":
			if f.Severity == "blocker" {
				t.Errorf("ambiguous present-state prose must not block the gate: %+v", f)
			}
		}
	}
}

func TestStrayRootDocs(t *testing.T) {
	root := t.TempDir()
	// Allowlisted regular files at root -> exempt.
	writeFile(t, root, "README.md", "# readme\n")
	writeFile(t, root, "AGENTS.md", "# agents\n")
	// A stray, non-allowlisted root markdown -> finding.
	writeFile(t, root, "NOTES.md", "# stray\n")
	// Markdown under a subdirectory is never touched (non-recursive).
	writeFile(t, root, "docs/guide.md", "# guide\n")
	// A symlink whose target stem is allowlisted -> exempt (the CLAUDE.md ->
	// AGENTS.md case). Judged by the resolved target, not the link name.
	if err := os.Symlink("AGENTS.md", filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	// A symlink with a missing target -> finding.
	if err := os.Symlink("GONE.md", filepath.Join(root, "BROKEN.md")); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Rules: map[string]RuleConfig{
			"stray_root_docs": {Enabled: true, Severity: "blocker",
				Allowlist: []string{"README", "AGENTS", "CHANGELOG", "CONTRIBUTING", "SECURITY", "LICENSE"}},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	if !hasFinding(fs, "NOTES.md", "stray_root_docs", 0) {
		t.Errorf("expected stray finding on NOTES.md: %+v", fs)
	}
	if !hasFinding(fs, "BROKEN.md", "stray_root_docs", 0) {
		t.Errorf("expected broken-symlink finding on BROKEN.md: %+v", fs)
	}
	// README, AGENTS, the CLAUDE.md symlink, and docs/guide.md must not fire.
	for _, f := range fs {
		switch f.File {
		case "README.md", "AGENTS.md", "CLAUDE.md", filepath.Join("docs", "guide.md"):
			t.Errorf("unexpected stray finding on %s: %+v", f.File, f)
		}
	}
	if n := countRule(fs, "stray_root_docs"); n != 2 {
		t.Fatalf("expected exactly 2 stray_root_docs findings, got %d: %+v", n, fs)
	}
}

func TestDocsLinkResolveInDocs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/target.md", "# target\n")
	writeFile(t, root, "docs/doc.md",
		"good: [t](target.md)\n"+
			"dir ok: [d](../docs/)\n"+
			"broken: [t](missing.md)\n")

	cfg := Config{
		Roots: []string{"docs"},
		Rules: map[string]RuleConfig{
			"links_resolve": {Enabled: true, Severity: "blocker"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "links_resolve"); n != 1 {
		t.Fatalf("expected 1 link finding, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("docs", "doc.md"), "links_resolve", 3) {
		t.Errorf("expected broken-link finding on docs/doc.md:3: %+v", fs)
	}
}

func TestCleanRecordProducesNoFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/README.md", "# rec\n")
	writeFile(t, root, "rec/doc.md", "---\nid: x\nkind: standalone\n---\n# Clean\n\nA [link](README.md) and prose. No banned words.\n")

	cfg := Config{
		Roots: []string{"rec"},
		BannedTokens: []BannedToken{
			{ID: "py", Pattern: `intent_lint\.py`, Message: "no", Severity: "blocker"},
		},
		Rules: map[string]RuleConfig{
			"no_git_metadata":      {Enabled: true, Severity: "blocker", Fields: []string{"created", "updated"}},
			"links_resolve":        {Enabled: true, Severity: "blocker"},
			"no_brittle_line_refs": {Enabled: true, Severity: "warn"},
			"directory_coverage":   {Enabled: true, Severity: "warn"},
			"intent_lifecycle":     {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected no findings on clean record, got %d: %+v", len(fs), fs)
	}
}

func TestContextStatusFree(t *testing.T) {
	target := filepath.Join(".abcd", "work", "CONTEXT.md")
	patterns := []string{
		`(?i)^#+\s*current (phase|status)`,
		`(?i)\*\*current phase`,
		`(?i)^\*\*next:`,
		`(?i)\bphase [0-9]+(\.[0-9]+)? — `,
		`(?i)^status:`,
	}
	msg := "CONTEXT.md is status-free (DECISIONS.md 2026-07-10): status lives in the live surfaces (CLI, ledger), not in orientation docs"

	// (1) A CONTEXT.md carrying status claims fires exactly those lines as blockers.
	t.Run("flags status claims", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target,
			"# Working context\n"+ // 1
				"\n"+ // 2
				"## Current phase\n"+ // 3 - heading
				"\n"+ // 4
				"Doing work.\n"+ // 5
				"\n"+ // 6
				"**Next:** Phase 0.5 wire the detector\n") // 7 - next:
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target, Patterns: patterns},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "context_status_free"); n != 2 {
			t.Fatalf("expected exactly 2 context_status_free findings, got %d: %+v", n, fs)
		}
		if !hasFinding(fs, target, "context_status_free", 3) {
			t.Errorf("expected finding on Current phase heading (line 3): %+v", fs)
		}
		if !hasFinding(fs, target, "context_status_free", 7) {
			t.Errorf("expected finding on **Next:** line (line 7): %+v", fs)
		}
		for _, f := range fs {
			if f.RuleID == "context_status_free" {
				if f.Severity != "blocker" {
					t.Errorf("severity = %q, want blocker: %+v", f.Severity, f)
				}
				if f.Message != msg {
					t.Errorf("message = %q, want %q", f.Message, msg)
				}
			}
		}
	})

	// (2) A status-free CONTEXT.md yields nothing.
	t.Run("clean file is silent", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target,
			"# Working context\n\nOrientation prose with no status claims.\nSee the ledger for state.\n")
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target, Patterns: patterns},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "context_status_free"); n != 0 {
			t.Fatalf("clean CONTEXT.md must be silent, got %d: %+v", n, fs)
		}
	})

	// (3) Disabled rule never fires, even with status claims present.
	t.Run("disabled rule is silent", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target, "## Current phase\n**Next:** Phase 1\n")
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: false, Severity: "blocker", Target: target, Patterns: patterns},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "context_status_free"); n != 0 {
			t.Fatalf("disabled rule produced findings: %+v", fs)
		}
	})

	// (4) A missing target is not an error and yields nothing.
	t.Run("missing target is silent", func(t *testing.T) {
		root := t.TempDir()
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target, Patterns: patterns},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatalf("missing target must not error: %v", err)
		}
		if n := countRule(fs, "context_status_free"); n != 0 {
			t.Fatalf("missing target produced findings: %+v", fs)
		}
	})

	// (5) A phase mention inside a fenced code block is masked, not flagged.
	t.Run("fenced code block is masked", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target,
			"# Working context\n"+ // 1
				"\n"+ // 2
				"Example ledger row:\n"+ // 3
				"\n"+ // 4
				"```\n"+ // 5
				"Phase 3 — Intent\n"+ // 6 - would match, but fenced
				"```\n") // 7
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target, Patterns: patterns},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if n := countRule(fs, "context_status_free"); n != 0 {
			t.Fatalf("fenced phase mention must be masked, got %d: %+v", n, fs)
		}
	})

	// Custom patterns REPLACE the defaults: a distinctive pattern fires on a
	// line the defaults ignore, and a default-only idiom no longer fires.
	t.Run("configured patterns override defaults", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target,
			"# Working context\n"+ // 1
				"\n"+ // 2
				"blocked: waiting on review\n"+ // 3 - custom pattern only
				"\n"+ // 4
				"## Current phase\n") // 5 - default idiom, must NOT fire
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target,
					Patterns: []string{`(?i)^blocked:`}},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if !hasFinding(fs, target, "context_status_free", 3) {
			t.Errorf("custom pattern should flag 'blocked:' on line 3: %+v", fs)
		}
		if hasFinding(fs, target, "context_status_free", 5) {
			t.Errorf("default idiom must not fire when custom patterns are supplied: %+v", fs)
		}
		if n := countRule(fs, "context_status_free"); n != 1 {
			t.Fatalf("expected exactly 1 finding with custom patterns, got %d: %+v", n, fs)
		}
	})

	// A malformed pattern is a config error: Lint errors, no findings.
	t.Run("invalid pattern errors", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target, "## Current phase\n")
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target,
					Patterns: []string{"("}},
			},
		}
		fs, err := Lint(cfg, root)
		if err == nil {
			t.Fatal("expected error for uncompilable pattern")
		}
		if len(fs) != 0 {
			t.Fatalf("errored Lint must return zero findings, got %+v", fs)
		}
	})

	// Default patterns apply when the config supplies none.
	t.Run("default patterns when none configured", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, target, "# c\n\n## Current status\n")
		cfg := Config{
			Rules: map[string]RuleConfig{
				"context_status_free": {Enabled: true, Severity: "blocker", Target: target},
			},
		}
		fs, err := Lint(cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		if !hasFinding(fs, target, "context_status_free", 3) {
			t.Errorf("default patterns should flag '## Current status' on line 3: %+v", fs)
		}
	})
}

func TestPersonaRegistry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/development/personas.json",
		`{"schema_version":2,"personas":[{"name":"Alice","role_hints":["solo founder"]},{"name":"Bob","role_hints":["staff engineer"]}]}`)
	writeFile(t, root, "rec/ok.md", "# ok\n\n> \"Fine,\" said Alice, solo founder.\n")
	writeFile(t, root, "rec/bad.md", "# bad\n\n> \"Nope,\" said Zorro, pirate captain.\n")
	writeFile(t, root, "rec/fenced.md", "# fenced\n\n```\nsaid Zorro, pirate captain.\n```\n")
	writeFile(t, root, "rec/hist/old.md", "# old\n\n> \"Old,\" said Zorro, pirate captain.\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"persona_registry": {Enabled: true, Severity: "blocker", Registry: ".abcd/development/personas.json"},
		},
		ExemptPaths: []string{"rec/hist/"},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "persona_registry"); n != 1 {
		t.Fatalf("expected exactly 1 persona finding, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, "rec/bad.md", "persona_registry", 3) {
		t.Errorf("expected finding on rec/bad.md:3: %+v", fs)
	}
}

func TestPersonaRegistryMissingFileErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/ok.md", "# ok\n")
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"persona_registry": {Enabled: true, Severity: "blocker", Registry: "nope/personas.json"},
		},
	}
	if _, err := Lint(cfg, root); err == nil {
		t.Fatal("expected error for enabled rule with missing registry")
	}
}

func TestPersonaRegistryEdgeCases(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "reg.json",
		`{"personas":[{"name":"Alice"},{"name":"Zoë"}]}`)
	writeFile(t, root, "rec/multi.md",
		"# m\n\n> \"A,\" said Alice, founder. \"B,\" said Zorro, pirate. And said Anne-Marie, sailor.\n")
	writeFile(t, root, "rec/unicode.md", "# u\n\n> \"Fine,\" said Zoë, designer.\n")
	writeFile(t, root, "rec/nocomma.md", "# n\n\n> as Zorro said before, nothing here attributes a quote.\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"persona_registry": {Enabled: true, Severity: "blocker", Registry: "reg.json"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "persona_registry"); n != 2 {
		t.Fatalf("expected 2 findings (Zorro + Anne-Marie on one line), got %d: %+v", n, fs)
	}
	if !hasFinding(fs, "rec/multi.md", "persona_registry", 3) {
		t.Errorf("expected findings on rec/multi.md:3: %+v", fs)
	}
}

func TestPersonaRegistryConfigGuards(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/ok.md", "# ok\n")
	writeFile(t, root, "empty.json", `{"personas":[]}`)

	// Enabled with empty registry path: loud error, not a directory read.
	cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{
		"persona_registry": {Enabled: true, Severity: "blocker"},
	}}
	if _, err := Lint(cfg, root); err == nil {
		t.Fatal("expected error for enabled rule with empty registry path")
	}

	// Enabled with zero-persona roster: loud error, not whole-record flagging.
	cfg.Rules["persona_registry"] = RuleConfig{Enabled: true, Severity: "blocker", Registry: "empty.json"}
	if _, err := Lint(cfg, root); err == nil {
		t.Fatal("expected error for zero-persona roster")
	}

	// Disabled rule with missing registry: no error, no findings.
	cfg.Rules["persona_registry"] = RuleConfig{Enabled: false, Registry: "nope.json"}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatalf("disabled rule must not touch the registry: %v", err)
	}
	if n := countRule(fs, "persona_registry"); n != 0 {
		t.Fatalf("disabled rule produced findings: %+v", fs)
	}
}

func TestSpecLifecycle(t *testing.T) {
	root := t.TempDir()
	base := "rec/specs"
	ibase := "rec/intents"

	// The intent corpus the specs cross-check against.
	writeFile(t, root, ibase+"/shipped/itd-10-alpha.md", "---\nid: itd-10\nkind: standalone\nspec_id: spc-1\n---\n# ok\n")
	writeFile(t, root, ibase+"/shipped/itd-20-beta.md", "---\nid: itd-20\nkind: standalone\nspec_id: spc-9\n---\n# ok\n") // points at a DIFFERENT spec (drift target)
	writeFile(t, root, ibase+"/planned/itd-30-gamma.md", "---\nid: itd-30\nkind: standalone\nspec_id: null\n---\n# ok\n") // never linked back

	// Good spec: names itd-10, which points back at spc-1. Agreement passes.
	writeFile(t, root, base+"/closed/spc-1-alpha.md", "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# ok\n")

	// Bad: names an intent that does not exist in the corpus.
	writeFile(t, root, base+"/open/spc-2-nope.md", "---\nid: spc-2\nslug: nope\nintent: itd-99\n---\n# bad\n")
	// Bad: bidirectional drift — names itd-20, but itd-20's spec_id is spc-9, not spc-3.
	writeFile(t, root, base+"/open/spc-3-drift.md", "---\nid: spc-3\nslug: drift\nintent: itd-20\n---\n# bad\n")
	// Bad: malformed spec id.
	writeFile(t, root, base+"/open/spc-4-badid.md", "---\nid: spec-4\nslug: badid\nintent: itd-10\n---\n# bad\n")
	// Bad: names itd-30, whose spec_id is null (drift: intent points at nothing).
	writeFile(t, root, base+"/open/spc-5-null.md", "---\nid: spc-5\nslug: null\nintent: itd-30\n---\n# bad\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"spec_lifecycle": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}

	badFiles := []string{
		filepath.Join(base, "open", "spc-2-nope.md"),
		filepath.Join(base, "open", "spc-3-drift.md"),
		filepath.Join(base, "open", "spc-4-badid.md"),
		filepath.Join(base, "open", "spc-5-null.md"),
	}
	for _, bf := range badFiles {
		found := false
		for _, f := range fs {
			if f.File == bf && f.RuleID == "spec_lifecycle" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected spec_lifecycle finding on %s; got %+v", bf, fs)
		}
	}
	// The good spec produces no finding.
	for _, f := range fs {
		if filepath.Base(f.File) == "spc-1-alpha.md" && f.RuleID == "spec_lifecycle" {
			t.Errorf("unexpected finding on clean spec: %+v", f)
		}
	}
}

func TestSpecLifecycleMissingDirIsSoft(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/README.md", "# rec\n")
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"spec_lifecycle": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, "spec_lifecycle"); n != 0 {
		t.Fatalf("missing specs/ dir must be soft; got %+v", fs)
	}
}

// TestLinksResolveReadsALinkBetweenEscapedBackticks: a backslash-escaped
// backtick is a literal character in CommonMark and never a code-span
// delimiter, so a link between two of them is read and a broken one reported
// (iss-2609251004336500). An escaped backtick beside a real span leaves the
// span blanked.
func TestLinksResolveReadsALinkBetweenEscapedBackticks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/doc.md",
		"literal \\` text [x](gone.md) \\` end\n"+
			"escaped \\` then `code [y](gone2.md)` end\n"+
			"double \\\\` then [z](gone3.md) ` end\n")
	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{"links_resolve": {Enabled: true, Severity: "blocker"}},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 1) {
		t.Errorf("a broken link between escaped backticks is not reported: %+v", fs)
	}
	if hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 2) {
		t.Errorf("a link inside a real code span beside an escaped backtick is reported: %+v", fs)
	}
	if hasFinding(fs, filepath.Join("rec", "doc.md"), "links_resolve", 3) {
		t.Errorf("an escaped backslash does not escape the backtick after it, so the span is code: %+v", fs)
	}
}

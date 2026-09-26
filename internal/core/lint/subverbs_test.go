package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// subverbFixture builds a minimal repo: a surface registry dir with files, a
// snapshot, and a config arming the sub-verb pass.
type subverbFixture struct {
	repo string
	cfg  RuleConfig
}

func newSubverbFixture(t *testing.T, commands []map[string]any) *subverbFixture {
	t.Helper()
	repo := t.TempDir()
	regDir := ".abcd/development/brief/04-surfaces"
	if err := os.MkdirAll(filepath.Join(repo, regDir), 0o755); err != nil {
		t.Fatal(err)
	}
	// The surface-grain registry file exists but carries no table — the
	// sub-verb pass is what these tests exercise.
	if err := os.WriteFile(filepath.Join(repo, regDir, "README.md"), []byte("# registry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap := map[string]any{"schema_version": 1, "commands": commands}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	snapRel := ".abcd/development/release/surface.json"
	if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(snapRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, snapRel), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return &subverbFixture{
		repo: repo,
		cfg: RuleConfig{
			Enabled: true, Severity: "blocker",
			Registry:         regDir + "/README.md",
			BareCommand:      "abcd",
			Snapshot:         snapRel,
			HostDelegated:    []string{"consult"},
			OperatorInternal: []string{"spec", "rules", "hook", "completion"},
		},
	}
}

func (f *subverbFixture) writeSurface(t *testing.T, name, content string) {
	t.Helper()
	p := filepath.Join(f.repo, ".abcd/development/brief/04-surfaces", name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cmd(path string, hidden bool) map[string]any {
	return map[string]any{"path": path, "hidden": hidden}
}

func runSubverbCheck(t *testing.T, f *subverbFixture) []Finding {
	t.Helper()
	out, err := checkSubVerbCoverage(f.repo, f.cfg)
	if err != nil {
		t.Fatalf("checkSubVerbCoverage: %v", err)
	}
	return out
}

func messages(fs []Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.File + ": " + f.Message + "\n")
	}
	return b.String()
}

const cleanCaptureTable = `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| ` + "`list`" + ` | — | shipped |
| ` + "`promote`" + ` | — | shipped |
`

func TestSubVerbCleanTablePasses(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false),
		cmd("abcd capture list", false), cmd("abcd capture promote", false),
	})
	f.writeSurface(t, "06-capture.md", cleanCaptureTable)
	if out := runSubverbCheck(t, f); len(out) != 0 {
		t.Fatalf("clean table must pass, got:\n%s", messages(out))
	}
}

func TestSubVerbBothDirections(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false),
		cmd("abcd capture list", false), cmd("abcd capture resolve", false),
	})
	// promote: shipped row, NOT registered. resolve: registered, no row.
	// wontfix: staged row and not registered (fine).
	f.writeSurface(t, "06-capture.md", `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`list`"+` | — | shipped |
| `+"`promote`"+` | — | shipped |
| `+"`wontfix`"+` | — | staged |
`)
	out := runSubverbCheck(t, f)
	msgs := messages(out)
	if !strings.Contains(msgs, "promote") || !strings.Contains(msgs, "resolve") {
		t.Fatalf("want findings for shipped-unregistered promote AND registered-rowless resolve, got:\n%s", msgs)
	}
	if strings.Contains(msgs, "wontfix") {
		t.Fatalf("staged+unregistered wontfix must not be flagged:\n%s", msgs)
	}
	if len(out) != 2 {
		t.Fatalf("want exactly 2 findings, got %d:\n%s", len(out), msgs)
	}
}

func TestSubVerbStagedButRegisteredFails(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false), cmd("abcd capture promote", false),
	})
	f.writeSurface(t, "06-capture.md", `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`promote`"+` | — | staged |
`)
	out := runSubverbCheck(t, f)
	if len(out) != 1 || !strings.Contains(out[0].Message, "staged") {
		t.Fatalf("staged-but-registered must fail once, got:\n%s", messages(out))
	}
}

// TestSubVerbMissingTableFailsOnEverySurfaceFile: itd-122 ac-1 and ac-5 put a
// table on every surface file, so the check demands one whatever the verb
// registers: a verb with no sub-commands, the bare command's own file, a
// host-delegated surface and a staged verb the tree does not register at all
// each fail without one. Only the cobra comparison is ever exempted.
func TestSubVerbMissingTableFailsOnEverySurfaceFile(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false),
		cmd("abcd capture", false), cmd("abcd capture list", false),
		cmd("abcd version", false),
	})
	f.writeSurface(t, "06-capture.md", "# capture — no table here\n")
	f.writeSurface(t, "08-abcd.md", "# the bare board — no table\n")
	f.writeSurface(t, "09-reflect.md", "# reflect — staged, no verb, no table\n")
	f.writeSurface(t, "12-version.md", "# version — no table, no subs\n")
	f.writeSurface(t, "13-consult.md", "# consult — host-delegated, no table\n")
	out := runSubverbCheck(t, f)
	msgs := messages(out)
	for _, file := range []string{"06-capture.md", "08-abcd.md", "09-reflect.md", "12-version.md", "13-consult.md"} {
		if !strings.Contains(msgs, file) {
			t.Errorf("a surface file without a '## Sub-verbs' table must fail: %s is not named in:\n%s", file, msgs)
		}
	}
	if len(out) != 5 {
		t.Fatalf("want exactly one finding per tableless file (5), got %d:\n%s", len(out), msgs)
	}
}

// TestSubVerbEmptyTablePassesForAVerbWithNoSubs: a verb that registers no
// sub-command records that with a header-only table; the table is present and
// every registered sub-command (none) has a row, so it passes. The same empty
// table on a sub-command-bearing verb still fails row by row.
func TestSubVerbEmptyTablePassesForAVerbWithNoSubs(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd version", false),
		cmd("abcd capture", false), cmd("abcd capture list", false),
	})
	empty := func(title string) string {
		return "# " + title + "\n\n## Sub-verbs\n\n| Verb | Bucket | Status |\n|---|---|---|\n\nNo sub-verb.\n"
	}
	f.writeSurface(t, "12-version.md", empty("version"))
	f.writeSurface(t, "13-consult.md", empty("consult"))
	f.writeSurface(t, "09-reflect.md", empty("reflect"))
	f.writeSurface(t, "06-capture.md", "# capture\n\n## Sub-verbs\n\n| Verb | Bucket | Status |\n|---|---|---|\n| `list` | — | shipped |\n")
	if out := runSubverbCheck(t, f); len(out) != 0 {
		t.Fatalf("an empty table on a verb with no sub-commands must pass:\n%s", messages(out))
	}
	f.writeSurface(t, "06-capture.md", empty("capture"))
	out := runSubverbCheck(t, f)
	if len(out) != 1 || !strings.Contains(out[0].Message, "capture list") {
		t.Fatalf("an empty table on a sub-command-bearing verb must fail on the missing row:\n%s", messages(out))
	}
}

func TestSubVerbVocabularyEnforced(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false), cmd("abcd capture list", false),
	})
	f.writeSurface(t, "06-capture.md", `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`list`"+` | verify | live |
`)
	out := runSubverbCheck(t, f)
	msgs := messages(out)
	if !strings.Contains(msgs, "verify") || !strings.Contains(msgs, "live") {
		t.Fatalf("unknown bucket AND unknown status must each be flagged:\n%s", msgs)
	}
}

func TestSubVerbHostDelegatedFormatOnly(t *testing.T) {
	// consult is host-delegated: shipped rows with no cobra backing pass; a
	// bad bucket is still flagged (format check applies).
	f := newSubverbFixture(t, []map[string]any{cmd("abcd", false)})
	f.writeSurface(t, "13-consult.md", `# consult

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`sources`"+` | — | shipped |
`)
	if out := runSubverbCheck(t, f); len(out) != 0 {
		t.Fatalf("host-delegated shipped rows must pass without cobra backing:\n%s", messages(out))
	}
	f.writeSurface(t, "13-consult.md", `# consult

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`sources`"+` | vibes | shipped |
`)
	if out := runSubverbCheck(t, f); len(out) != 1 {
		t.Fatalf("host-delegated tables are still format-checked:\n%s", messages(out))
	}
}

func TestSubVerbReverseSweepDemandsSurfaceFile(t *testing.T) {
	// guard has registered sub-commands but no surface file at all.
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd guard", false), cmd("abcd guard check", false),
	})
	out := runSubverbCheck(t, f)
	if len(out) != 1 || !strings.Contains(out[0].Message, "guard") {
		t.Fatalf("a sub-command-bearing verb with no surface file must fail:\n%s", messages(out))
	}
}

func TestSubVerbExclusions(t *testing.T) {
	// hook is hidden (subtree excluded structurally); spec is operator-internal
	// (excluded by config); the bare command's own file is exempt from the
	// cobra comparison.
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false),
		cmd("abcd hook", true), cmd("abcd hook session-start", false),
		cmd("abcd spec", false), cmd("abcd spec close", false),
	})
	f.writeSurface(t, "08-abcd.md", "# the bare board\n\n## Sub-verbs\n\n| Verb | Bucket | Status |\n|---|---|---|\n| `mode` | — | shipped |\n")
	if out := runSubverbCheck(t, f); len(out) != 0 {
		t.Fatalf("hidden subtree, operator-internal, and bare must all be excluded:\n%s", messages(out))
	}
}

func TestSubVerbNestedPathsMatch(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd intent", false),
		cmd("abcd intent audit", false), cmd("abcd intent audit ingest", false),
	})
	f.writeSurface(t, "05-intent.md", `# intent

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`audit`"+` | audit | shipped |
| `+"`audit ingest`"+` | audit | shipped |
`)
	if out := runSubverbCheck(t, f); len(out) != 0 {
		t.Fatalf("nested sub-verb rows must match snapshot paths:\n%s", messages(out))
	}
}

func TestSubVerbMissingSnapshotFailsLoudly(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{cmd("abcd", false)})
	if err := os.Remove(filepath.Join(f.repo, f.cfg.Snapshot)); err != nil {
		t.Fatal(err)
	}
	out, err := checkSubVerbCoverage(f.repo, f.cfg)
	if err != nil {
		t.Fatalf("a missing snapshot must be a finding, not a hard error: %v", err)
	}
	if len(out) != 1 || !strings.Contains(out[0].Message, "snapshot") {
		t.Fatalf("an armed check with no snapshot must fail loudly:\n%s", messages(out))
	}
	// The finding must not leak the absolute repo root (iss-29/iss-76): the
	// PathError is stripped to its bare cause, File carries the relative path.
	if strings.Contains(out[0].Message, f.repo) || strings.Contains(out[0].Message, "/tmp/") {
		t.Fatalf("snapshot finding leaks an absolute path: %q", out[0].Message)
	}
}

func TestSubVerbUnarmedIsInert(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false), cmd("abcd capture list", false),
	})
	f.cfg.Snapshot = "" // not armed: the sub-verb grain is off, no silent partiality
	out, err := checkSubVerbCoverage(f.repo, f.cfg)
	if err != nil || len(out) != 0 {
		t.Fatalf("unarmed sub-verb pass must be inert, got err=%v findings:\n%s", err, messages(out))
	}
}

// TestSubVerbDuplicateHeadingIsFlagged: only the first '## Sub-verbs' table is
// parsed, so a second heading could carry an unchecked lying table — it must
// be a finding (review regression: a shipped claim for an unregistered verb
// under a duplicate heading previously passed clean).
func TestSubVerbDuplicateHeadingIsFlagged(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false), cmd("abcd capture list", false),
	})
	f.writeSurface(t, "06-capture.md", `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`list`"+` | — | shipped |

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`bogus`"+` | — | shipped |
`)
	out := runSubverbCheck(t, f)
	if len(out) != 1 || !strings.Contains(out[0].Message, "duplicate") {
		t.Fatalf("a duplicate Sub-verbs heading must be exactly one finding:\n%s", messages(out))
	}
	// A fenced second heading is an example, not a duplicate.
	f.writeSurface(t, "06-capture.md", `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`list`"+` | — | shipped |

`+"```markdown"+`
## Sub-verbs
`+"```"+`
`)
	if out := runSubverbCheck(t, f); len(out) != 0 {
		t.Fatalf("a fenced heading is not a duplicate:\n%s", messages(out))
	}
}

// TestSubVerbShortRowIsFlagged: a data row with fewer than three cells may not
// silently drop its fact (review regression: a two-cell row vanished clean).
func TestSubVerbShortRowIsFlagged(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false), cmd("abcd capture list", false),
	})
	f.writeSurface(t, "06-capture.md", `# capture

## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `+"`list`"+` | — | shipped |
| `+"`scan`"+` | — |
`)
	out := runSubverbCheck(t, f)
	if len(out) != 1 || !strings.Contains(out[0].Message, "fewer than three cells") {
		t.Fatalf("a short row must be exactly one finding:\n%s", messages(out))
	}
}

// itd-147 ac-7: a sub-verb pass finding reached through Lint carries the same
// row-level label as the registry pass, so neither half reads as a check of
// chapter prose.
func TestSubVerbFindingsCarryTheRowLevelLabel(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{
		cmd("abcd", false), cmd("abcd capture", false),
		cmd("abcd capture list", false), cmd("abcd capture resolve", false),
	})
	f.writeSurface(t, "06-capture.md", cleanCaptureTable)
	fs, err := Lint(Config{Rules: map[string]RuleConfig{"surface_coverage": f.cfg}}, f.repo)
	if err != nil {
		t.Fatal(err)
	}
	if countRule(fs, "surface_coverage") == 0 {
		t.Fatalf("fixture produced no sub-verb finding; the label assertion would pass vacuously")
	}
	for _, x := range fs {
		if x.RuleID == "surface_coverage" && !strings.HasPrefix(x.Message, SurfaceCoverageLabel) {
			t.Errorf("sub-verb finding lacks the row-level label: %q", x.Message)
		}
	}
}

// movedCmd is a snapshot command recording the successor of a moved spelling
// (itd-2609212130136102).
func movedCmd(path, to string) map[string]any {
	return map[string]any{"path": path, "hidden": false, "moved_to": to}
}

// TestSubVerbMovedSpellingsNeedNoRowAndMayHaveNone is itd-2609212130136102
// criterion 4 on the brief: a moved spelling stays in the tree for one release
// as a stub, but the chapters say the new forms only, so it needs no row, and a
// row that still names it is a finding that names its successor. A parent whose
// bare form moved keeps its live sub-verbs, which still need rows.
func TestSubVerbMovedSpellingsNeedNoRowAndMayHaveNone(t *testing.T) {
	commands := []map[string]any{
		cmd("abcd", false), cmd("abcd ahoy", false), cmd("abcd ahoy doctor", false),
		movedCmd("abcd ahoy dry-run", "abcd ahoy --dry-run"),
		movedCmd("abcd ahoy remote", "abcd ahoy --remote"),
		cmd("abcd ahoy remote apply", false),
	}
	clean := newSubverbFixture(t, commands)
	clean.writeSurface(t, "01-ahoy.md", "# ahoy\n\n## Sub-verbs\n\n| Verb | Bucket | Status |\n|---|---|---|\n"+
		"| `doctor` | — | shipped |\n| `remote apply` | gate | shipped |\n")
	if out := runSubverbCheck(t, clean); len(out) != 0 {
		t.Fatalf("a table naming the current forms only must pass, got:\n%s", messages(out))
	}

	stale := newSubverbFixture(t, commands)
	stale.writeSurface(t, "01-ahoy.md", "# ahoy\n\n## Sub-verbs\n\n| Verb | Bucket | Status |\n|---|---|---|\n"+
		"| `doctor` | — | shipped |\n| `dry-run` | — | shipped |\n| `remote apply` | gate | shipped |\n")
	got := messages(runSubverbCheck(t, stale))
	if !strings.Contains(got, "'dry-run'") || !strings.Contains(got, "abcd ahoy --dry-run") {
		t.Fatalf("a row naming a moved spelling must be a finding naming its successor, got:\n%s", got)
	}
}

// TestSubVerbHeadingWithoutTableFails: finding the `## Sub-verbs` heading is not
// finding the table. A heading followed by prose and no header row carries no
// table, so a verb with no sub-command waves the grain through in prose exactly
// as a file with no heading does, and is the same finding (iss-2609250937494009).
func TestSubVerbHeadingWithoutTableFails(t *testing.T) {
	f := newSubverbFixture(t, []map[string]any{cmd("abcd", false), cmd("abcd version", false)})
	f.writeSurface(t, "12-version.md", "# version\n\n## Sub-verbs\n\nNone: version has no sub-verb.\n")
	out := runSubverbCheck(t, f)
	if len(out) != 1 || !strings.Contains(out[0].Message, "no '## Sub-verbs' table") {
		t.Fatalf("a heading with no table under it must be the missing-table finding:\n%s", messages(out))
	}
}

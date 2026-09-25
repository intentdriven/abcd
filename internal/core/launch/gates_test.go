package launch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// gateRow returns the named row of a gate list, failing the test when it is absent.
func gateRow(t *testing.T, gates []GateSummary, name string) GateSummary {
	t.Helper()
	for _, g := range gates {
		if g.Name == name {
			return g
		}
	}
	t.Fatalf("no %q row in the gates list; got %v", name, gateNames(gates))
	return GateSummary{}
}

// anyContains reports whether some reason contains every one of the fragments.
func anyContains(reasons []string, fragments ...string) bool {
	for _, r := range reasons {
		all := true
		for _, f := range fragments {
			if !strings.Contains(r, f) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// docsFixture is a non-git payload that ships docs/, README.md and commands/,
// with manifests that pass the smoke. The dirty-tree gate is out of its reach
// (there is no repository to read), so callers pass DirtySkip or read only the
// rows they plant findings in.
func docsFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "docs", "commands", "README.md"]}`)
	writeFile(t, root, "README.md", "# readme\n\nThe tool reads the record.\n")
	writeLockstepTree(t, root, "", "", "")
	return root
}

// TestMarkerBlockGateRefusesAMalformedBlock is itd-65 AC2's marker half: an
// unclosed BEGIN, an orphan END and a nested pair in shipped Markdown are each a
// hard-fail naming the file and line, on the preview and on the render path alike.
func TestMarkerBlockGateRefusesAMalformedBlock(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, "docs/unclosed.md", "# a\n\n<!-- BEGIN ABCD -->\nrules\n")
	writeFile(t, root, "docs/orphan.md", "# b\n\nrules\n<!-- END ABCD -->\n")
	writeFile(t, root, "commands/nested.md", "# c\n<!-- BEGIN ABCD -->\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n<!-- END ABCD -->\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	row := gateRow(t, report.Gates, "marker-block")
	if row.Status != "ran" || row.Tier != TierHardFail {
		t.Fatalf("marker-block row = %+v, want ran/hard-fail", row)
	}
	for _, want := range [][]string{
		{"docs/unclosed.md:3", "never closed"},
		{"docs/orphan.md:4", "no open"},
		{"commands/nested.md:3", "nested"},
	} {
		if !anyContains(report.WouldRefuseOn, want...) {
			t.Errorf("would_refuse_on does not name %v:\n%s", want, strings.Join(report.WouldRefuseOn, "\n"))
		}
	}

	_, err = PrecheckPayload(root, filepath.Join(t.TempDir(), "out"), PrecheckOptions{Dirty: DirtySkip})
	if !errors.Is(err, ErrPayloadGateRefused) {
		t.Fatalf("the render path must refuse a malformed marker block, got %v", err)
	}
	if !strings.Contains(err.Error(), "docs/unclosed.md:3") {
		t.Errorf("the render refusal must name the file and line, got %v", err)
	}
}

// TestMarkerBlockGatePassesBalancedAndQuotedMarkers keeps the gate precise: a
// balanced block, two sequential blocks, and a marker quoted in a code fence or
// an inline code span are all well formed.
func TestMarkerBlockGatePassesBalancedAndQuotedMarkers(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, "docs/ok.md", "# a\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n\n<!-- BEGIN ABCD -->\ny\n<!-- END ABCD -->\n")
	writeFile(t, root, "docs/quoted.md", "# b\n\nThe block opens with `<!-- BEGIN ABCD -->`.\n\n```\n<!-- END ABCD -->\n```\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	row := gateRow(t, report.Gates, "marker-block")
	if len(row.Findings) != 0 {
		t.Fatalf("well-formed markers were flagged: %+v", row.Findings)
	}
}

// narrationSpecification is the change-narration gate's specification: every
// sentence the hard tier must refuse, and every sentence it must pass. It is
// one table so the two sets are read against each other — a rule widened for
// recall is checked against the passes, and one narrowed for precision against
// the refusals. itd-65 AC3 and AC10 are the promise: "previously X, now Y" and
// a genuine "changed from X to Y" / "no longer" hard-fail, and bare
// present-tense "now"/"previously" does not. The passes include the four
// present-state sentences the first review found refused
// (iss-2609251827286563); the refusals include the genuine narration the
// second review found passed (iss-2609251940304726). Where a pair cannot be
// told apart lexically the gate refuses (the escape marker is one comment
// away and every finding names it); the residual trade is recorded in
// .abcd/work/DECISIONS.md.
var narrationSpecification = []struct {
	sentence string
	refuse   bool
}{
	// Must refuse.
	{"The default changed from JSON to YAML in this release.", true},
	{"The store migrated from SQLite to flat files.", true},
	{"Previously, the ledger was a flat file; now it is a folder.", true},
	{"Previously the gate read JSON, now it reads YAML.", true},
	{"Reports previously went to the log, and now they go to the record.", true},
	{"The default was previously JSON; it is now YAML.", true},
	{"The verb no longer writes a receipt.", true},
	{"abcd no longer writes a receipt.", true},
	{"The registry can no longer be edited by hand.", true},
	{"The scanner no longer skips fenced blocks.", true},
	{"The record no longer carries a grounds scalar.", true},
	{"The scanner, which no longer skips fenced blocks, reads every line.", true},
	{"The flag was renamed from --out to --dest.", true},
	{"We renamed the flag to --dest.", true},
	{"It was renamed to `abcd lint`.", true},
	{"The loader was renamed to abcd rules.", true},
	{"The tool used to print a banner.", true},
	{"It used to print a banner, which was noisy.", true},
	{"The gate skips the classes that used to drift.", true},
	{"Until v0.6 the dry-run used to skip the tags.", true},
	{"In earlier releases the archive used to include the record.", true},
	// Must pass.
	{"The command now accepts a path.", false},
	{"Run the previously saved query with `--replay`.", false},
	{"Now, as previously noted, the report lists every gate.", false},
	{"A passive key is used to sign the archive.", false},
	{"The token used to authenticate the request is read from the environment.", false},
	{"Set the token used to authenticate the request.", false},
	{"In the config used to sign releases, set the key path.", false},
	{"Files that are no longer present in the tree are skipped.", false},
	{"Records which are no longer open move to resolved/.", false},
	{"The gate is no longer than one screen.", false},
	{"Keep each summary no longer than one line.", false},
	{"The output is renamed to match the tag.", false},
	{"Keep the names that must not be renamed, and link each to the glossary.", false},
	{"The words \"now\" and \"previously\" are fine on their own.", false},
}

// TestNarrationGateHardFailsOnAChangeConstruct is itd-65 AC3 and the second half
// of AC10: a shipped doc body narrating a change hard-fails, naming the
// sentence and the escape — every refusal in narrationSpecification.
func TestNarrationGateHardFailsOnAChangeConstruct(t *testing.T) {
	for _, c := range narrationSpecification {
		if !c.refuse {
			continue
		}
		t.Run(c.sentence, func(t *testing.T) {
			root := docsFixture(t)
			writeFile(t, root, "docs/guide.md", "# Guide\n\nIntro line.\n"+c.sentence+"\n")

			report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
			if err != nil {
				t.Fatalf("DryRun: %v", err)
			}
			row := gateRow(t, report.Gates, "change-narration")
			if row.Tier != TierHardFail || len(row.Findings) != 1 {
				t.Fatalf("change-narration row = %+v, want one hard-fail finding", row)
			}
			// Inline code is blanked before the gate reads a line.
			shown := strings.TrimSuffix(inlineCodeRe.ReplaceAllString(c.sentence, ""), ".")
			if !anyContains(report.WouldRefuseOn, "docs/guide.md:4", shown) {
				t.Errorf("would_refuse_on does not name the file, line and sentence:\n%s", strings.Join(report.WouldRefuseOn, "\n"))
			}
			if !anyContains(report.WouldRefuseOn, "docs/guide.md:4", "<!-- docs-lint: allow -->") {
				t.Errorf("the finding must name the escape marker:\n%s", strings.Join(report.WouldRefuseOn, "\n"))
			}
		})
	}
}

// TestNarrationGatePassesPresentTense is AC10's first half and the gate's scope:
// every pass in narrationSpecification — bare present-tense "now" and
// "previously", apart and together with no change construct, and the
// present-state readings of "used to", "no longer" and "renamed to" — does
// not hard-fail; nor does a construct inside code, a line carrying the
// docs-lint escape, the release records (a changelog is narration by
// definition), or a file outside the doc bodies (commands/ is plugin surface,
// not a doc body).
func TestNarrationGatePassesPresentTense(t *testing.T) {
	root := docsFixture(t)
	var doc strings.Builder
	doc.WriteString("# Present\n\n")
	for _, c := range narrationSpecification {
		if !c.refuse {
			doc.WriteString(c.sentence + "\n\n")
		}
	}
	doc.WriteString("```\nthe verb no longer writes\n```\n\n" +
		"Use `no longer` sparingly.\n" +
		"The page lists what changed from one release to the next. <!-- docs-lint: allow -->\n")
	writeFile(t, root, "docs/present.md", doc.String())
	writeFile(t, root, "docs/CHANGELOG.md", "# Changelog\n\nThe verb no longer writes a receipt.\n")
	writeFile(t, root, "commands/x.md", "The verb no longer writes a receipt.\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	row := gateRow(t, report.Gates, "change-narration")
	for _, f := range row.Findings {
		t.Errorf("present-tense prose was flagged: %s:%d %s", f.File, f.Line, f.Detail)
	}
}

// dirtyRepo is a committed plugin payload with one uncommitted doc edit, one
// untracked file, and an untracked report in the local tier.
func dirtyRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "docs", "README.md"]}`)
	r.Write(".abcd/config/version-location.json", `{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}`)
	r.Write(".claude-plugin/plugin.json", `{"name": "abcd"}`)
	r.Write(".claude-plugin/marketplace.json", `{"plugins": [{"name": "abcd", "source": "./"}]}`)
	r.Write("README.md", "# readme\n")
	r.Write("docs/a.md", "# a\n")
	r.Commit("the payload")
	return r
}

// TestDirtyTreeGateRefusesUnlessAllowed is itd-65 AC4: a dirty tree refuses the
// cut, naming what is uncommitted; --allow-dirty lets it proceed and the report
// records the override and the paths it carried. The local tier is never dirt.
func TestDirtyTreeGateRefusesUnlessAllowed(t *testing.T) {
	r := dirtyRepo(t)
	r.Write("docs/a.md", "# a, edited\n")
	r.Write("notes.txt", "scratch\n")
	r.Write(".abcd/.work.local/logs/launch/x/preflight.json", "{}\n")

	dirty, err := DirtyTreeFiles(r.Root())
	if err != nil {
		t.Fatalf("DirtyTreeFiles: %v", err)
	}
	if strings.Join(dirty, ",") != "docs/a.md,notes.txt" {
		t.Fatalf("dirty = %v, want the edit and the untracked file and nothing in the local tier", dirty)
	}

	ship, err := Ship(ShipRequest{RepoRoot: r.Root(), Version: "1.2.3", ExistingTags: []Semver{}})
	if !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("a dirty tree must block the ship, got %v (%v)", err, ship.BlockReasons)
	}
	if !anyContains(ship.BlockReasons, "uncommitted", "docs/a.md", "notes.txt") {
		t.Errorf("the refusal must name the uncommitted paths: %v", ship.BlockReasons)
	}

	allowed, err := Ship(ShipRequest{RepoRoot: r.Root(), Version: "1.2.3", ExistingTags: []Semver{}, AllowDirty: true})
	if err != nil {
		t.Fatalf("--allow-dirty must let the ship proceed: %v (%v)", err, allowed.BlockReasons)
	}
	if strings.Join(allowed.AllowedDirty, ",") != "docs/a.md,notes.txt" {
		t.Errorf("the override must record what it carried, got %v", allowed.AllowedDirty)
	}

	dest := filepath.Join(t.TempDir(), "out")
	if _, err := PrecheckPayload(r.Root(), dest, PrecheckOptions{}); !errors.Is(err, ErrDirtyTree) {
		t.Fatalf("the render path must refuse a dirty tree by default, got %v", err)
	}
	pre, err := PrecheckPayload(r.Root(), dest, PrecheckOptions{Dirty: DirtyAllow})
	if err != nil {
		t.Fatalf("an allowed dirty tree must pass the precheck: %v", err)
	}
	rep := pre.PreflightReport(time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), "1.2.3")
	if !rep.AllowDirty || strings.Join(rep.Dirty, ",") != "docs/a.md,notes.txt" {
		t.Errorf("the cut report must record the override and its paths: %+v", rep)
	}

	r.Commit("everything")
	if _, err := PrecheckPayload(r.Root(), dest, PrecheckOptions{}); err != nil {
		t.Fatalf("a clean tree must pass: %v", err)
	}
}

// TestDirtyTreeGateFailsClosedOffAGitTree keeps the gate honest outside a
// repository: a state it cannot read is a refusal, never a clean pass.
func TestDirtyTreeGateFailsClosedOffAGitTree(t *testing.T) {
	root := docsFixture(t)
	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !anyContains(report.WouldRefuseOn, "working tree", "could not be read") {
		t.Errorf("an unreadable tree must be a would-refuse, got %v", report.WouldRefuseOn)
	}
}

// TestWarnRowsSurfaceWithoutBlocking is itd-65 AC5: a documentation-audit
// finding and a hook-compliance concern are surfaced as warnings and refuse
// nothing, unless the repository configures the suite strict.
func TestWarnRowsSurfaceWithoutBlocking(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, ".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "docs", "hooks", "scripts", "README.md"]}`)
	writeFile(t, root, "hooks/hooks.json",
		`{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "$CLAUDE_PLUGIN_ROOT/scripts/go.sh"}, {"type": "command"}]}]}}`)
	writeFile(t, root, "scripts/go.sh", "#!/bin/sh\nexit 0\n") // 0644: the hook cannot run it
	audit := &DocAuditPreflight{Findings: []GateFinding{{File: "docs/a.md", Line: 3, Detail: "broken link ./gone.md"}}}

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3", DocAudit: audit})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	doc := gateRow(t, report.Gates, "documentation-auditor")
	if doc.Status != "ran" || doc.Tier != TierWarn || len(doc.Findings) != 1 {
		t.Errorf("documentation-auditor row = %+v, want ran/warn with the finding", doc)
	}
	hook := gateRow(t, report.Gates, "hook-compliance")
	if hook.Status != "ran" || hook.Tier != TierWarn || len(hook.Findings) != 2 {
		t.Errorf("hook-compliance row = %+v, want ran/warn with two findings", hook)
	}
	if !anyContains(report.Warnings, "scripts/go.sh", "not executable") ||
		!anyContains(report.Warnings, "hooks/hooks.json", "no command") ||
		!anyContains(report.Warnings, "docs/a.md:3", "broken link") {
		t.Errorf("warnings do not carry every concern:\n%s", strings.Join(report.Warnings, "\n"))
	}
	for _, r := range report.WouldRefuseOn {
		if strings.Contains(r, "go.sh") || strings.Contains(r, "broken link") || strings.Contains(r, "no command") {
			t.Errorf("a warn-tier concern refused the release: %q", r)
		}
	}

	writeFile(t, root, ".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "docs", "hooks", "scripts", "README.md"], "strict_warnings": true}`)
	strict, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3", DocAudit: audit})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !anyContains(strict.WouldRefuseOn, "strict", "scripts/go.sh") || !anyContains(strict.WouldRefuseOn, "strict", "broken link") {
		t.Errorf("a strict suite must refuse on its warnings, got %v", strict.WouldRefuseOn)
	}

	writeFile(t, root, ".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "docs", "README.md"], "strict_warnings": "yes"}`)
	if _, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"}); err == nil {
		t.Error("a strict_warnings that is not a boolean must be a configuration fault")
	}
}

// TestDocAuditRowSaysWhenItIsNotArmed keeps an unmeasured row from reading as a
// pass: without a docs-lint configuration the row says what it would have run.
func TestDocAuditRowSaysWhenItIsNotArmed(t *testing.T) {
	root := docsFixture(t)
	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	doc := gateRow(t, report.Gates, "documentation-auditor")
	if doc.Status != "not_armed" || !strings.Contains(doc.Detail, "docs-lint") {
		t.Errorf("documentation-auditor row = %+v, want not_armed naming the docs-lint config", doc)
	}
	for _, g := range report.Gates {
		if g.Status == "not_implemented" && (g.Name == "marker-block" || g.Name == "documentation-auditor") {
			t.Errorf("the %s row still reports not_implemented", g.Name)
		}
	}
}

// TestSuiteReportsEveryGateInOnePass is itd-65 AC8 (spec piece 6): findings
// planted in four gates at once — the secret scan, the marker block, the
// change narration and the dirty tree — are ALL reported by the preview, the
// render path's refusal and the written pre-flight report, never just the first.
func TestSuiteReportsEveryGateInOnePass(t *testing.T) {
	r := dirtyRepo(t)
	r.Write("docs/secret.md", "# s\ntoken = "+fakeSecret+"\n")
	r.Write("docs/marker.md", "# m\n<!-- BEGIN ABCD -->\n")
	r.Write("docs/story.md", "# t\n\nThe verb no longer writes a receipt.\n")
	r.Commit("three planted findings")
	r.Write("docs/a.md", "# a, uncommitted\n")

	report, err := DryRun(DryRunRequest{RepoRoot: r.Root(), Version: "1.2.3", ExistingTags: []Semver{}})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	fragments := [][]string{
		{"secret/PII hard-fail"},
		{"docs/marker.md:2"},
		{"docs/story.md:3", "no longer"},
		{"uncommitted", "docs/a.md"},
	}
	for _, want := range fragments {
		if !anyContains(report.WouldRefuseOn, want...) {
			t.Errorf("the preview dropped %v:\n%s", want, strings.Join(report.WouldRefuseOn, "\n"))
		}
	}

	pre, err := PrecheckPayload(r.Root(), filepath.Join(t.TempDir(), "out"), PrecheckOptions{})
	for _, sentinel := range []error{ErrPayloadScanRefused, ErrPayloadGateRefused, ErrDirtyTree} {
		if !errors.Is(err, sentinel) {
			t.Errorf("the render refusal does not carry %v: %v", sentinel, err)
		}
	}
	for _, want := range fragments {
		if !anyContains([]string{err.Error()}, want...) {
			t.Errorf("the render refusal dropped %v: %v", want, err)
		}
	}

	at := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	rel, err := WritePreflightReport(r.Root(), pre.PreflightReport(at, "1.2.3"))
	if err != nil {
		t.Fatalf("WritePreflightReport: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(rel), "preflight.json"))
	if err != nil {
		t.Fatalf("the report was not written: %v", err)
	}
	var written PreflightReport
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("the report is not JSON: %v", err)
	}
	if written.Verdict != VerdictRefused {
		t.Errorf("verdict = %q, want %q", written.Verdict, VerdictRefused)
	}
	for _, want := range fragments {
		if !anyContains(written.Refusals, want...) {
			t.Errorf("the written report dropped %v:\n%s", want, strings.Join(written.Refusals, "\n"))
		}
	}
}

// TestWritePreflightReportLandsInTheLocalTier is itd-65 AC7's report half and
// spec piece 5: the report is a JSON and a Markdown file in a per-run directory
// under the gitignored logs tier, and two runs in one second never share one.
func TestWritePreflightReportLandsInTheLocalTier(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	rep := PreflightReport{Mode: ModePreview, At: at, Version: "1.2.3", Verdict: VerdictClear,
		Gates: []GateSummary{{Name: "marker-block", Status: "ran", Tier: TierHardFail, Detail: "0 finding(s)"}}}

	first, err := WritePreflightReport(root, rep)
	if err != nil {
		t.Fatalf("WritePreflightReport: %v", err)
	}
	second, err := WritePreflightReport(root, rep)
	if err != nil {
		t.Fatalf("WritePreflightReport: %v", err)
	}
	if first == second {
		t.Fatalf("two runs share one report directory: %s", first)
	}
	if !strings.HasPrefix(first, ".abcd/.work.local/logs/launch/20260925T120000Z") {
		t.Errorf("report dir = %s, want a per-run directory under the local logs tier", first)
	}
	for _, name := range []string{"preflight.json", "preflight.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(first), name))
		if err != nil {
			t.Fatalf("%s not written: %v", name, err)
		}
		if !strings.Contains(string(data), "marker-block") {
			t.Errorf("%s does not carry the gate rows:\n%s", name, data)
		}
	}
}

// TestHookRowFindsAnUnparseableHooksConfig is iss-2609251827104081: a hooks
// config the host cannot parse registers no hook on any install, so it is a
// finding of the hook-compliance row (AC5: the concern is surfaced) and the
// installability smoke refuses it, as it refuses an unparseable plugin
// manifest. Neither may read it as a clean pass.
func TestHookRowFindsAnUnparseableHooksConfig(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, ".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "docs", "hooks", "README.md"]}`)
	writeFile(t, root, "hooks/hooks.json", "{not json at all")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	hook := gateRow(t, report.Gates, "hook-compliance")
	if len(hook.Findings) == 0 || !anyContains(report.Warnings, "hook-compliance", "hooks/hooks.json") {
		t.Errorf("an unparseable hooks config must be a finding of the hook row naming it; row = %+v, warnings = %v", hook, report.Warnings)
	}

	if report.Smoke.OK || !anyContains(report.WouldRefuseOn, "hooks/hooks.json", "does not parse") {
		t.Errorf("the smoke must refuse an unparseable hooks config, naming it; smoke = %+v, would_refuse_on = %v",
			report.Smoke, report.WouldRefuseOn)
	}
}

// TestRenderPayloadRefusesADirtyTreeByDefault is iss-2609251827294854: the
// dirty-tree policy is the render caller's to state, and a caller that states
// none fails closed. A render request with no policy refuses a dirty tree; the
// cut's post-write render, which states DirtySkip, renders it.
func TestRenderPayloadRefusesADirtyTreeByDefault(t *testing.T) {
	r := dirtyRepo(t)
	r.Write("notes.txt", "scratch\n")
	req := PayloadRenderRequest{
		RepoRoot: r.Root(), Dest: filepath.Join(t.TempDir(), "payload"),
		Version: "0.4.0", Entry: sampleEntry(),
	}
	if _, err := RenderPayload(req); !errors.Is(err, ErrDirtyTree) {
		t.Fatalf("a render that states no dirty-tree policy must refuse a dirty tree, got %v", err)
	}

	req.Dest = filepath.Join(t.TempDir(), "payload")
	req.Dirty = DirtySkip
	if _, err := RenderPayload(req); err != nil {
		t.Fatalf("a render that states DirtySkip must render: %v", err)
	}
}

// TestRenderPathDocAuditRowSaysItWasNotMeasured keeps the render path honest
// (iss-2609251827294854): it does not measure the documentation audit, so its
// row says so rather than claiming the repository has no docs-lint config.
func TestRenderPathDocAuditRowSaysItWasNotMeasured(t *testing.T) {
	root := renderFixture(t)
	writeFile(t, root, ".abcd/docs-lint.json", `{"roots": ["docs"], "banned_tokens": [], "rules": {}}`+"\n")
	pre, err := PrecheckPayload(root, filepath.Join(t.TempDir(), "payload"),
		PrecheckOptions{Dirty: DirtySkip, DocAudit: renderPathDocAudit})
	if err != nil {
		t.Fatalf("PrecheckPayload: %v", err)
	}
	row := gateRow(t, pre.Gates, "documentation-auditor")
	if row.Status != "not_measured" || strings.Contains(row.Detail, "no .abcd/docs-lint.json") {
		t.Errorf("the render path's doc-auditor row = %+v, want not_measured and no claim about the config", row)
	}
}

// TestProseLinesReadsAnUnclosedOpeningRuleAsProse is iss-2609251827296447: a
// document that opens with a "---" rule and never closes it has no frontmatter,
// so both content gates read all of it; closed frontmatter is still metadata.
func TestProseLinesReadsAnUnclosedOpeningRuleAsProse(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, "docs/ruled.md", "---\n\nThe verb no longer writes a receipt.\n\n<!-- BEGIN ABCD -->\n")
	writeFile(t, root, "docs/front.md", "---\ntitle: The verb no longer writes a receipt.\n---\n\n# Front\n\nThe tool reads the record.\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	for _, want := range [][]string{
		{"change-narration", "docs/ruled.md:3", "no longer"},
		{"marker-block", "docs/ruled.md:5", "never closed"},
	} {
		if !anyContains(report.WouldRefuseOn, want...) {
			t.Errorf("a document opening with an unclosed rule was not read; missing %v:\n%s", want, strings.Join(report.WouldRefuseOn, "\n"))
		}
	}
	if anyContains(report.WouldRefuseOn, "docs/front.md") {
		t.Errorf("closed frontmatter must stay metadata:\n%s", strings.Join(report.WouldRefuseOn, "\n"))
	}
}

// TestProseLinesReadsARuledDocumentWhole: a document that opens with a "---"
// rule and repeats one later has no frontmatter unless the block between the
// two reads as YAML (iss-2609251940383450), so its prose is read, not dropped
// up to the later rule. Frontmatter with list items, continuations, comments
// and a blank line stays metadata.
func TestProseLinesReadsARuledDocumentWhole(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, "docs/ruled.md", "---\n\nThe verb no longer writes a receipt.\n\n<!-- BEGIN ABCD -->\n\n---\n\nAfter the rule.\n")
	writeFile(t, root, "docs/front.md", "---\n# metadata\ntitle: Front\ntags:\n  - one\n- two\n\ndescription: >\n  The verb no longer writes a receipt.\n---\n\n# Front\n\nThe tool reads the record.\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	for _, want := range [][]string{
		{"change-narration", "docs/ruled.md:3", "no longer"},
		{"marker-block", "docs/ruled.md:5", "never closed"},
	} {
		if !anyContains(report.WouldRefuseOn, want...) {
			t.Errorf("a ruled document was dropped up to its later rule; missing %v:\n%s", want, strings.Join(report.WouldRefuseOn, "\n"))
		}
	}
	if anyContains(report.WouldRefuseOn, "docs/front.md") {
		t.Errorf("YAML frontmatter must stay metadata:\n%s", strings.Join(report.WouldRefuseOn, "\n"))
	}
}

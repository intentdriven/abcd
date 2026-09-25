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

// TestNarrationGateHardFailsOnAChangeConstruct is itd-65 AC3 and the second half
// of AC10: a shipped doc body narrating a change hard-fails, naming the sentence.
func TestNarrationGateHardFailsOnAChangeConstruct(t *testing.T) {
	cases := map[string]string{
		"changed from": "The default changed from JSON to YAML in this release.",
		"no longer":    "The verb no longer writes a receipt.",
		"migrated":     "The store migrated from SQLite to flat files.",
		"previously":   "Reports previously went to the log, and now they go to the record.",
		"renamed":      "The flag was renamed from --out to --dest.",
		"used to":      "The tool used to print a banner.",
		// The narrowed constructs keep their true positives
		// (iss-2609251827286563): a pronoun subject, a subject naming abcd,
		// an active rename, and a past-tense copula beside "previously".
		"used to, pronoun":         "It used to print a banner, which was noisy.",
		"no longer, abcd":          "abcd no longer writes a receipt.",
		"renamed, active":          "We renamed the flag to --dest.",
		"previously, same clause":  "The default was previously JSON; it is now YAML.",
		"used to, relative clause": "The gate skips the classes that used to drift.",
	}
	for name, sentence := range cases {
		t.Run(name, func(t *testing.T) {
			root := docsFixture(t)
			writeFile(t, root, "docs/guide.md", "# Guide\n\nIntro line.\n"+sentence+"\n")

			report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
			if err != nil {
				t.Fatalf("DryRun: %v", err)
			}
			row := gateRow(t, report.Gates, "change-narration")
			if row.Tier != TierHardFail || len(row.Findings) != 1 {
				t.Fatalf("change-narration row = %+v, want one hard-fail finding", row)
			}
			if !anyContains(report.WouldRefuseOn, "docs/guide.md:4", strings.TrimSuffix(sentence, ".")) {
				t.Errorf("would_refuse_on does not name the file, line and sentence:\n%s", strings.Join(report.WouldRefuseOn, "\n"))
			}
			if !anyContains(report.WouldRefuseOn, "docs/guide.md:4", "<!-- docs-lint: allow -->") {
				t.Errorf("the finding must name the escape marker:\n%s", strings.Join(report.WouldRefuseOn, "\n"))
			}
		})
	}
}

// TestNarrationGatePassesPresentTense is AC10's first half and the gate's scope:
// bare present-tense "now" and "previously" — apart, and together in one
// sentence with no change verb beside either — a construct inside code, a line
// carrying the docs-lint escape, and the release records (a changelog is
// narration by definition) do not hard-fail; nor does a file outside the doc
// bodies (commands/ is plugin surface, not a doc body). The four sentences a
// review found refused (iss-2609251827286563) state present state: a
// participle "used to", a "no longer" whose subject is not abcd, a present
// "renamed", and "previously" beside "now" with nothing changing.
func TestNarrationGatePassesPresentTense(t *testing.T) {
	root := docsFixture(t)
	writeFile(t, root, "docs/present.md", "# Present\n\n"+
		"The command now accepts a path.\n"+
		"Run the previously saved query with `--replay`.\n"+
		"A passive key is used to sign the archive.\n\n"+
		"The token used to authenticate the request is read from the environment.\n"+
		"Files that are no longer present in the tree are skipped.\n"+
		"The output is renamed to match the tag.\n"+
		"Now, as previously noted, the report lists every gate.\n"+
		"Set the token used to authenticate the request.\n\n"+
		"```\nthe verb no longer writes\n```\n\n"+
		"Use `no longer` sparingly.\n"+
		"The page lists what changed from one release to the next. <!-- docs-lint: allow -->\n")
	writeFile(t, root, "docs/CHANGELOG.md", "# Changelog\n\nThe verb no longer writes a receipt.\n")
	writeFile(t, root, "commands/x.md", "The verb no longer writes a receipt.\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	row := gateRow(t, report.Gates, "change-narration")
	if len(row.Findings) != 0 {
		t.Fatalf("present-tense prose was flagged: %+v", row.Findings)
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

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/intentdriven/abcd/internal/core/release"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/gittest"
)

// shipFixture is a repository the ship verb can actually cut from: a tagged
// release whose tree carries the surface baseline, and whose baseline is the
// surface THIS binary walks — because the guardrail refuses to compare a
// snapshot it cannot prove came from the tree being released.
//
// It is built through the front door's own SurfaceSnapshot, so the fixture and
// the verb agree on the surface by construction rather than by a hand-written
// copy that would rot the moment a command is added.
func shipFixture(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(".claude-plugin/plugin.json", `{"name":"abcd","description":"fixture"}`+"\n")
	r.Write(".claude-plugin/marketplace.json", `{"name":"abcd","plugins":[{"name":"abcd","source":"./"}]}`+"\n")
	// The empty [Unreleased] heading is the post-cutover state (outcome 7) and the
	// anchor the ingest step inserts beneath.
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.4.0] - 2026-07-01\n\n### Added\n\n- the base.\n")

	live, err := SurfaceSnapshot(r.Root())
	if err != nil {
		t.Fatalf("SurfaceSnapshot: %v", err)
	}
	data, err := surface.Encode(live)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	r.Write(SurfaceSnapshotPath, string(data))
	r.Commit("the released state")
	r.Git("tag", "v0.4.0")
	return r
}

// shipIn runs the CLI with the working directory pointed at the fixture, the way
// the verb is really invoked.
func shipIn(t *testing.T, r *gittest.Repo, args ...string) ([]byte, error) {
	t.Helper()
	t.Chdir(r.Root())
	return runCLIErr(t, args...)
}

// TestLaunchShipEmitsAReadyCut is the wired path: `abcd launch ship` with no
// payload flag runs the deterministic emit step and exits 0 on a cut that may
// proceed.
func TestLaunchShipEmitsAReadyCut(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived.\n")
	r.Commit("ship an intent")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	for _, want := range []string{"v0.4.1", "additive", "itd-73", "A Version Is A Fact"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("render does not mention %q:\n%s", want, out)
		}
	}
}

// TestLaunchShipRefusalExits1 pins the exit contract: a refusal is a REPORT, not
// a crash — the whole cut renders, the exit code is 1, and the blocking record
// is named. Exit 2 is reserved for a structural fault.
func TestLaunchShipRefusalExits1(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Write(".abcd/development/intents/planned/itd-94-gate.md",
		"---\nid: itd-94\nkind: standalone\nspec_id: spc-9\n---\n# gate\n")
	r.Write(".abcd/development/specs/closed/spc-9-gate.md",
		"---\nid: spc-9\nslug: gate\nintent: itd-94\n---\n# spc-9\n")
	r.Commit("a merged feature whose intent never left planned/")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"REFUSED", "stale-intent", "itd-94"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("refusal render does not mention %q:\n%s", want, out)
		}
	}
}

// TestLaunchShipRefusesReleaseInFlight is outcome 1 wired end to end: the newest
// CHANGELOG heading is ahead of the newest tag, so a release sits between its
// merge and its tag and the verb refuses rather than deriving against a
// mismatched base.
func TestLaunchShipRefusesReleaseInFlight(t *testing.T) {
	r := shipFixture(t)
	r.Write("CHANGELOG.md", "# Changelog\n\n## [0.5.0] - 2026-07-20\n\n### Added\n\n- the ship PR merged.\n")
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Commit("the ship PR merged; auto-release has not tagged it yet")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"release-in-flight", "v0.5.0"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("refusal render does not mention %q:\n%s", want, out)
		}
	}
}

// composedPayload writes a well-formed composer payload citing exactly ids.
func composedPayload(t *testing.T, dir, nextTag string, ids ...string) string {
	t.Helper()
	entries := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, map[string]any{
			"section": "Added",
			"records": []string{id},
			"text":    "**Something shipped.** " + id + " landed.",
		})
	}
	// The release page tells every intent the payload cites, in one headline.
	var intents []string
	for _, id := range ids {
		if strings.HasPrefix(id, "itd-") {
			intents = append(intents, id)
		}
	}
	var page any
	if len(intents) > 0 {
		page = map[string]any{
			"headlines": []map[string]any{{"records": intents, "text": "What shipped in this release."}},
			"listed":    []string{},
			"quotes":    []any{},
		}
	}
	data, err := json.Marshal(map[string]any{
		"schema_version": 2,
		"prompt_version": "1.0.0",
		"next_tag":       nextTag,
		"entries":        entries,
		"press_release":  page,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "changelog.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// shipReadyRepo is a fixture whose cut is ready with one required record.
func shipReadyRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived.\n")
	r.Commit("ship an intent")
	return r
}

// TestLaunchShipIngestWritesTheHeading is the wired write path: a payload whose
// citations match the cut lands the dated section and exits 0.
func TestLaunchShipIngestWritesTheHeading(t *testing.T) {
	r := shipReadyRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	for _, want := range []string{"wrote:", "CHANGELOG.md", "## [0.4.1] - "} {
		if !strings.Contains(string(out), want) {
			t.Errorf("render does not mention %q:\n%s", want, out)
		}
	}
	data, err := os.ReadFile(filepath.Join(r.Root(), "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "### Added\n\n- **Something shipped.** itd-73 landed. (itd-73)\n") {
		t.Errorf("the composed line did not land:\n%s", got)
	}
	if !strings.Contains(got, "## [Unreleased]\n\n## [0.4.1] - ") {
		t.Errorf("the section was not inserted beneath an empty [Unreleased]:\n%s", got)
	}
}

// TestLaunchShipIngestReadsStdin pins the `-` operand: an orchestrating command
// pipes the composer's output straight in rather than staging a temp file.
func TestLaunchShipIngestReadsStdin(t *testing.T) {
	r := shipReadyRepo(t)
	data, err := os.ReadFile(composedPayload(t, t.TempDir(), "v0.4.1", "itd-73"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(r.Root())
	out, err := runCLIStdinErr(t, string(data), "launch", "ship", "--changelog-json", "-")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(string(out), "## [0.4.1] - ") {
		t.Errorf("render does not report the written heading:\n%s", out)
	}
}

// TestLaunchShipIngestBijectionExits2 is the loud stage at the front door: a
// composed changelog that omits a shipped record is a structural fault, the
// whole document is refused, and CHANGELOG.md is byte-identical afterwards.
func TestLaunchShipIngestBijectionExits2(t *testing.T) {
	r := shipReadyRepo(t)
	r.Write(".abcd/work/issues/resolved/iss-51-crash.md", "---\nid: iss-51\nimpact: fix\n---\n# x\n")
	r.Commit("resolve an issue too")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	before := cliTreeDigest(t, r.Root())
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s", code, out)
	}
	for _, want := range []string{"MISSING", "iss-51", "nothing was written"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Error("a refused ingest changed the working tree")
	}
}

// TestLaunchShipIngestOnARefusedCutExits1 keeps the two failure modes apart: the
// CUT refusing is a report (exit 1), not a payload fault (exit 2) — and it must
// not write, whatever the payload says.
func TestLaunchShipIngestOnARefusedCutExits1(t *testing.T) {
	r := shipFixture(t)
	r.Commit("nothing shipped")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	before := cliTreeDigest(t, r.Root())
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(string(out), "empty-cut") {
		t.Errorf("the refusal report is missing:\n%s", out)
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Error("a refused cut changed the working tree")
	}
}

// TestChangelogPreviewWritesNothing is deliverable 4's contract, asserted the
// only way that means anything: hash the whole tree before and after.
func TestChangelogPreviewWritesNothing(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	// A standing release page, so the digest proves the preview neither
	// rewrites it nor archives it.
	r.Write("RELEASE.md", "# Release 0.4.0 (2026-07-01)\n\nThe base. (itd-1)\n")
	r.Commit("ship an intent")

	before := cliTreeDigest(t, r.Root())
	out, err := shipIn(t, r, "changelog")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0 (a preview always reports)\n%s", code, out)
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Errorf("the preview changed the working tree:\nbefore %s\nafter  %s", before, after)
	}
}

// TestChangelogPreviewRefusalStillExitsZero separates the preview from the gate:
// `abcd changelog` REPORTS a refused cut (like `launch --dry-run`), while
// `launch ship` exits non-zero on the same repository.
func TestChangelogPreviewRefusalStillExitsZero(t *testing.T) {
	r := shipFixture(t)
	r.Commit("nothing shipped at all")

	out, err := shipIn(t, r, "changelog")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(string(out), "empty-cut") {
		t.Errorf("preview does not name the refusal:\n%s", out)
	}

	if _, err := shipIn(t, r, "launch", "ship"); exitCodeOf(err) != 1 {
		t.Errorf("launch ship exit = %d on the same repo, want 1", exitCodeOf(err))
	}
}

// TestChangelogPreviewJSON pins the machine surface the next stages read.
func TestChangelogPreviewJSON(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# Title\n\nsummary.\n")
	r.Commit("ship an intent")

	out, err := shipIn(t, r, "changelog", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	var got struct {
		Ready   bool   `json:"ready"`
		NextTag string `json:"next_tag"`
		Added   []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"added"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("changelog --json is not JSON: %v\n%s", err, out)
	}
	if !got.Ready || got.NextTag != "v0.4.1" {
		t.Errorf("ready=%v next_tag=%q, want true v0.4.1", got.Ready, got.NextTag)
	}
	if len(got.Added) != 1 || got.Added[0].ID != "itd-73" || got.Added[0].Title != "Title" {
		t.Errorf("added = %+v, want the one record with its title", got.Added)
	}
}

// TestLaunchShipWritesNothingYet pins that the emit stage of the verb is as
// read-only as the preview. The write path arrives with the ingest step; until
// then a ship that ran and a ship that did not are indistinguishable on disk.
func TestLaunchShipWritesNothingYet(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Commit("ship an intent")

	before := cliTreeDigest(t, r.Root())
	if _, err := shipIn(t, r, "launch", "ship"); exitCodeOf(err) != 0 {
		t.Fatalf("launch ship failed unexpectedly")
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Errorf("the emit step changed the working tree:\nbefore %s\nafter  %s", before, after)
	}
}

// TestShipStructuralFaultExits2 pins the third exit code. A directory that is
// not an abcd repository cannot be read at all, which is a fault rather than a
// refusal — and the diagnostic must be path-scrubbed, because it names files
// under the caller's working directory.
func TestShipStructuralFaultExits2(t *testing.T) {
	for _, args := range [][]string{{"launch", "ship"}, {"changelog"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Chdir(t.TempDir())
			out, err := runCLIErr(t, args...)
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("exit = %d, want 2\n%s", code, out)
			}
			if strings.Contains(err.Error(), os.TempDir()) {
				t.Errorf("diagnostic leaks an absolute path: %q", err.Error())
			}
		})
	}
}

// TestLaunchStillRunsWithoutASubcommand is the regression guard on hanging
// `ship` off the launch command: a cobra parent that gains a subcommand can stop
// running its own RunE, which would silently turn `abcd launch` into a usage
// dump. It must still reach its own refusal.
func TestLaunchStillRunsWithoutASubcommand(t *testing.T) {
	r := shipFixture(t)
	out, err := shipIn(t, r, "launch")
	if err == nil {
		t.Fatalf("bare `abcd launch` must still refuse without --dry-run\n%s", out)
	}
	if !strings.Contains(err.Error(), "pass --dry-run") {
		t.Errorf("error = %q, want the launch command's own refusal", err.Error())
	}
}

// cliTreeDigest hashes every path and byte under root except .git, whose
// internals git rewrites for reasons unrelated to the code under test.
func cliTreeDigest(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		lines = append(lines, filepath.ToSlash(rel)+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// A record whose `shipped_in` could not be read must say so in the HUMAN cut
// render, not only in --json (iss-2608241612087533).
//
// Such a record is in the cut precisely because its value was unreadable, so it
// prints as an ordinary entry and the operator never learns that the exclusion
// they wrote did not take. An adversarial review found the field reachable only
// through `--json`, which made "reported, not silent" true for a machine and
// false for the person running the documented ship flow.
func TestRenderEntriesShowsAnUnreadableShippedIn(t *testing.T) {
	var w strings.Builder
	renderEntries(&w, "added", []release.Entry{
		{ID: "iss-1", Impact: "fix", Title: "a real fix", InChangelog: true},
		{ID: "iss-2", Impact: "fix", Title: "a swept record", InChangelog: true,
			ShippedInErr: `shipped_in "v9.9.9" names no tag in this repository`},
	})
	got := w.String()

	if !strings.Contains(got, "names no tag") {
		t.Errorf("the human render drops the shipped_in fault; an operator running the "+
			"documented flow would never see it.\n%s", got)
	}
	if !strings.Contains(got, "still in this cut") {
		t.Errorf("the render must say the record is still in the cut — that is the "+
			"consequence the operator has to act on.\n%s", got)
	}
	// The clean record must not sprout a fault line.
	if strings.Count(got, "still in this cut") != 1 {
		t.Errorf("exactly one entry has a fault; got %d annotations.\n%s",
			strings.Count(got, "still in this cut"), got)
	}
}

// TestChangelogPreviewListsThePageSet: the read-only preview names the intents
// the release page will be composed from, and marks them in the entry list.
func TestChangelogPreviewListsThePageSet(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n")
	r.Write(".abcd/development/intents/shipped/itd-97-y.md", "---\nid: itd-97\nimpact: internal\n---\n# Plumbing\n")
	r.Write(".abcd/work/issues/resolved/iss-51-crash.md", "---\nid: iss-51\nimpact: fix\n---\n# x\n")
	r.Commit("ship a mixed cut")

	out, err := shipIn(t, r, "changelog")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	got := string(out)
	if !strings.Contains(got, "release page: 1 intent(s)") {
		t.Errorf("the preview does not count the page set:\n%s", got)
	}
	_, block, _ := strings.Cut(got, "release page:")
	if !strings.Contains(block, "itd-73") || strings.Contains(block, "itd-97") || strings.Contains(block, "iss-51") {
		t.Errorf("the page block lists the wrong records:\n%s", block)
	}
	if !strings.Contains(got, "A Version Is A Fact  (on the release page)") {
		t.Errorf("the entry list does not mark the page's intent:\n%s", got)
	}

	jsonOut, err := shipIn(t, r, "changelog", "--json")
	if exitCodeOf(err) != 0 || !strings.Contains(string(jsonOut), `"in_press_release": true`) {
		t.Errorf("changelog --json does not carry in_press_release:\n%s", jsonOut)
	}
}

// TestChangelogPreviewSaysNoPageForAFixesOnlyCut: with no user-facing intent the
// preview says no page will be written, and why.
func TestChangelogPreviewSaysNoPageForAFixesOnlyCut(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/work/issues/resolved/iss-51-crash.md", "---\nid: iss-51\nimpact: fix\n---\n# x\n")
	r.Commit("a fix alone")

	out, err := shipIn(t, r, "changelog")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(string(out), "release page: none (no user-facing intent shipped; RELEASE.md stays as it is)") {
		t.Errorf("the preview does not say no page will be written:\n%s", out)
	}
}

// TestLaunchShipFixesOnlyReportsNoPage: a fixes-only ship writes the changelog,
// leaves RELEASE.md alone, and says why in the report.
func TestLaunchShipFixesOnlyReportsNoPage(t *testing.T) {
	r := shipFixture(t)
	r.Write("RELEASE.md", "# Release 0.4.0 (2026-07-01)\n\nThe base. (itd-1)\n")
	r.Write(".abcd/work/issues/resolved/iss-51-crash.md", "---\nid: iss-51\nimpact: fix\n---\n# x\n")
	r.Commit("a fix alone")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "iss-51")

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
	}
	if !strings.Contains(string(out), "No release page written: no user-facing intent shipped in this cut; RELEASE.md stays on 0.4.0") {
		t.Errorf("the report does not say why no page was written:\n%s", out)
	}
	data, err := os.ReadFile(filepath.Join(r.Root(), "RELEASE.md"))
	if err != nil || string(data) != "# Release 0.4.0 (2026-07-01)\n\nThe base. (itd-1)\n" {
		t.Errorf("RELEASE.md changed on a fixes-only ship: %q (%v)", data, err)
	}
}

// TestLaunchShipWritesTheReleasePage: a feature ship reports the page it wrote.
func TestLaunchShipWritesTheReleasePage(t *testing.T) {
	r := shipReadyRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")

	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
	}
	for _, want := range []string{"page:", "RELEASE.md", "# Release 0.4.1 (", "1 headline(s), 0 listed, 0 quote(s)"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the report does not mention %q:\n%s", want, out)
		}
	}
}

// TestLaunchShipPayloadRefusalJSON pins the retry loop's machine seam: a refused
// payload exits 2 WITH a payload_refusal carrying stable codes (recompose), and
// the tree is untouched.
func TestLaunchShipPayloadRefusalJSON(t *testing.T) {
	r := shipReadyRepo(t)
	path := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(data), `"What shipped in this release."`, `"# A forged heading"`, 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	before := cliTreeDigest(t, r.Root())
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", path, "--json")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s", code, out)
	}
	var got struct {
		Written        bool `json:"written"`
		PayloadRefusal *struct {
			Reasons []struct {
				Code   string `json:"code"`
				At     string `json:"at"`
				Detail string `json:"detail"`
			} `json:"reasons"`
		} `json:"payload_refusal"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("the refusal is not JSON: %v\n%s", err, out)
	}
	if got.Written || got.PayloadRefusal == nil || len(got.PayloadRefusal.Reasons) == 0 {
		t.Fatalf("written=%v refusal=%+v, want an unwritten payload_refusal", got.Written, got.PayloadRefusal)
	}
	if r0 := got.PayloadRefusal.Reasons[0]; r0.Code != "heading" || r0.At == "" || r0.Detail == "" {
		t.Errorf("reason = %+v, want a heading reason with a path and a detail", r0)
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Error("a refused payload changed the working tree")
	}

	// A STOP carries no payload_refusal: the repository, not the composer, is at
	// fault, and no rewrite can fix it.
	r.Write("RELEASE.md", "a hand-written page\n")
	r.Commit("a release page with no heading")
	stop, err := shipIn(t, r, "launch", "ship", "--changelog-json", composedPayload(t, t.TempDir(), "v0.4.1", "itd-73"), "--json")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s", code, stop)
	}
	if strings.Contains(string(stop), "payload_refusal") {
		t.Errorf("a structural stop carries payload_refusal, which tells the host to recompose:\n%s", stop)
	}
}

// TestLaunchPageNamesEveryRefusalCode pins the command page to the binary: the
// retry loop it documents names every reason code the ingest can return, has no
// attempt limit, and reports every refused attempt.
func TestLaunchPageNamesEveryRefusalCode(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "commands", "launch.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	for _, code := range release.ReasonCodes {
		if !strings.Contains(page, "`"+string(code)+"`") {
			t.Errorf("commands/launch.md does not name the refusal code `%s`", code)
		}
	}
	for _, want := range []string{"no attempt limit", "report every refused attempt", "payload_refusal"} {
		if !strings.Contains(page, want) {
			t.Errorf("commands/launch.md does not say %q", want)
		}
	}
}

package release

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// baseChangelog is the post-cutover state outcome 7 leaves behind: an EMPTY
// [Unreleased] heading above the last dated section. Every derived cut is
// inserted between the two.
const baseChangelog = "# Changelog\n\n## [Unreleased]\n\n## [0.4.0] - 2026-07-01\n\n### Added\n\n- the base.\n"

// cutAt is the injected clock. The date in a heading is deterministic input, not
// a wall-clock read buried in the writer.
var cutAt = time.Date(2026, 7, 21, 9, 30, 0, 0, time.UTC)

// shippableRepo is a ready cut with a known required set: itd-73 (additive) and
// iss-51 (fix) must be cited; iss-97 (internal) must not.
func shippableRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := releasedRepo(t)
	r.Write("CHANGELOG.md", baseChangelog)
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived.\n")
	r.Record(resolvedDir+"iss-51-crash.md", "iss-51", "fix")
	r.Record(resolvedDir+"iss-97-toctou.md", "iss-97", "internal")
	r.Commit("ship an intent, a fix, and an internal issue")
	return r
}

// goodEntries is the payload that satisfies the bijection for shippableRepo.
func goodEntries() []ChangelogEntry {
	return []ChangelogEntry{
		{Section: SectionFixed, Records: []string{"iss-51"}, Text: "**The crash is gone.** It no longer crashes."},
		{Section: SectionAdded, Records: []string{"itd-73"}, Text: "**A version is a fact.** Derived, not typed."},
	}
}

func marshalPayload(t *testing.T, nextTag string, entries []ChangelogEntry) []byte {
	t.Helper()
	data, err := json.Marshal(ChangelogPayload{
		SchemaVersion: ChangelogSchemaVersion,
		PromptVersion: "1.0.0",
		NextTag:       nextTag,
		Entries:       entries,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return data
}

func readChangelog(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("reading CHANGELOG.md: %v", err)
	}
	return string(data)
}

// TestIngestWritesTheDatedSection is the happy path: a payload whose cited ids
// are exactly the required set writes one dated section, in Keep-a-Changelog
// order, leaving [Unreleased] empty (outcome 7 — no fold).
func TestIngestWritesTheDatedSection(t *testing.T) {
	r := shippableRepo(t)

	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if !res.Written {
		t.Fatalf("nothing was written; cut refusals = %v", refusalKinds(res.Cut))
	}
	if res.Heading != "## [0.4.1] - 2026-07-21" {
		t.Errorf("Heading = %q, want ## [0.4.1] - 2026-07-21", res.Heading)
	}
	if res.Path != "CHANGELOG.md" {
		t.Errorf("Path = %q, want CHANGELOG.md", res.Path)
	}
	if strings.Join(res.Cited, ",") != "iss-51,itd-73" {
		t.Errorf("Cited = %v, want the sorted required set [iss-51 itd-73]", res.Cited)
	}

	got := readChangelog(t, r.Root())
	want := "# Changelog\n\n" +
		"## [Unreleased]\n\n" +
		"## [0.4.1] - 2026-07-21\n\n" +
		sectionNotice + "\n\n" +
		"### Added\n\n" +
		"- **A version is a fact.** Derived, not typed. (itd-73)\n\n" +
		"### Fixed\n\n" +
		"- **The crash is gone.** It no longer crashes. (iss-51)\n\n" +
		"## [0.4.0] - 2026-07-01\n\n### Added\n\n- the base.\n"
	if got != want {
		t.Errorf("CHANGELOG.md =\n%q\nwant\n%q", got, want)
	}
}

// TestIngestHeadingMatchesTheWorkflowGrep pins the ADR-37 contract against the
// workflow's OWN pattern, read out of .github/workflows/auto-release.yml at test
// time. If that grep ever changes, this breaks loudly — which is the point: the
// heading this writes is the only thing that turns a merge into a git tag.
func TestIngestHeadingMatchesTheWorkflowGrep(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := fsutil.ModuleRoot(cwd)
	if err != nil {
		t.Fatal(err)
	}
	workflow := filepath.Join(root, ".github", "workflows", "auto-release.yml")
	data, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatalf("reading the release workflow: %v", err)
	}
	// The line this mirrors is auto-release.yml's "Detect the newest CHANGELOG
	// version needing a tag" step:
	//   grep -m1 -E '^## \[v?[0-9]+\.[0-9]+\.[0-9]+\] - ' CHANGELOG.md
	grepRe := regexp.MustCompile(`grep -m1 -E '([^']+)' CHANGELOG\.md`)
	m := grepRe.FindSubmatch(data)
	if m == nil {
		t.Fatalf("auto-release.yml no longer greps CHANGELOG.md with a -m1 -E pattern; "+
			"the heading contract moved and %s must be re-pinned", workflow)
	}
	pattern, err := regexp.Compile(string(m[1]))
	if err != nil {
		t.Fatalf("the workflow pattern %q does not compile in Go: %v", m[1], err)
	}

	r := shippableRepo(t)
	if _, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	var matched []string
	for _, line := range strings.Split(readChangelog(t, r.Root()), "\n") {
		if pattern.MatchString(line) {
			matched = append(matched, line)
		}
	}
	if len(matched) == 0 {
		t.Fatalf("the workflow pattern %q matches no line of the written CHANGELOG", m[1])
	}
	// grep -m1 takes the FIRST match, so the newly written heading must sit above
	// every older one or the workflow would re-tag a past release.
	if matched[0] != "## [0.4.1] - 2026-07-21" {
		t.Errorf("the workflow would tag %q, want the derived ## [0.4.1] - 2026-07-21", matched[0])
	}
}

// TestIngestBijection is the heart of the phase. Each row is a payload that
// differs from the required set in exactly one way; every one must refuse whole,
// name what is wrong, and leave the file byte-identical.
func TestIngestBijection(t *testing.T) {
	tests := []struct {
		name         string
		entries      []ChangelogEntry
		wantMissing  []string
		wantInvented []string
		wantInternal []string
	}{
		{
			name:        "one required record is missing — a lie by omission",
			entries:     []ChangelogEntry{{Section: SectionAdded, Records: []string{"itd-73"}, Text: "derived versions."}},
			wantMissing: []string{"iss-51"},
		},
		{
			name: "one cited record did not ship — an invention",
			entries: append(goodEntries(),
				ChangelogEntry{Section: SectionAdded, Records: []string{"itd-999"}, Text: "a feature nobody wrote."}),
			wantInvented: []string{"itd-999"},
		},
		{
			name: "both at once — each named separately",
			entries: []ChangelogEntry{
				{Section: SectionAdded, Records: []string{"itd-73"}, Text: "derived versions."},
				{Section: SectionFixed, Records: []string{"iss-999"}, Text: "a fix nobody made."},
			},
			wantMissing:  []string{"iss-51"},
			wantInvented: []string{"iss-999"},
		},
		{
			name: "an internal record is cited — it is in the cut but earns no line",
			entries: append(goodEntries(),
				ChangelogEntry{Section: SectionAdded, Records: []string{"iss-97"}, Text: "plumbing."}),
			wantInternal: []string{"iss-97"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := shippableRepo(t)
			before := readChangelog(t, r.Root())

			res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", tt.entries), cutAt)
			var incomplete *IncompleteError
			if !errors.As(err, &incomplete) {
				t.Fatalf("Ingest err = %v (written=%v), want an *IncompleteError", err, res.Written)
			}
			if got := strings.Join(incomplete.Missing, ","); got != strings.Join(tt.wantMissing, ",") {
				t.Errorf("Missing = %v, want %v", incomplete.Missing, tt.wantMissing)
			}
			if got := strings.Join(incomplete.Invented, ","); got != strings.Join(tt.wantInvented, ",") {
				t.Errorf("Invented = %v, want %v", incomplete.Invented, tt.wantInvented)
			}
			if got := strings.Join(incomplete.Internal, ","); got != strings.Join(tt.wantInternal, ",") {
				t.Errorf("Internal = %v, want %v", incomplete.Internal, tt.wantInternal)
			}
			for _, id := range append(append(append([]string{}, tt.wantMissing...), tt.wantInvented...), tt.wantInternal...) {
				if !strings.Contains(err.Error(), id) {
					t.Errorf("the refusal %q does not name %s", err, id)
				}
			}
			if res.Written {
				t.Error("a refused ingest reported a write")
			}
			if after := readChangelog(t, r.Root()); after != before {
				t.Errorf("a refused ingest modified CHANGELOG.md:\n%q", after)
			}
		})
	}
}

// TestIngestRequiresTheRemovedSide pins that a record which LEFT a terminal
// folder is part of the cut and must be cited too: a supersession is a
// user-visible change, and omitting it is the same lie as omitting an addition.
func TestIngestRequiresTheRemovedSide(t *testing.T) {
	r := releasedRepo(t)
	r.Write("CHANGELOG.md", baseChangelog)
	r.Record(shippedDir+"itd-40-superseded.md", "itd-40", "additive")
	r.Commit("a shipped intent at the base")
	r.Git("tag", "-d", "v0.4.0")
	r.Git("tag", "v0.4.0")

	r.Remove(shippedDir + "itd-40-superseded.md")
	r.Record(shippedDir+"itd-73-derived.md", "itd-73", "additive")
	r.Commit("supersede one intent, ship another")

	entries := []ChangelogEntry{{Section: SectionAdded, Records: []string{"itd-73"}, Text: "derived versions."}}
	_, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", entries), cutAt)
	var incomplete *IncompleteError
	if !errors.As(err, &incomplete) {
		t.Fatalf("Ingest err = %v, want an *IncompleteError naming the removed record", err)
	}
	if strings.Join(incomplete.Missing, ",") != "itd-40" {
		t.Errorf("Missing = %v, want [itd-40] — the record that left shipped/", incomplete.Missing)
	}
}

// TestIngestRequiresAnUnlabelledRemovedRecord pins the removed side's other
// half. A record whose blob at the BASE TAG carries no valid impact still left a
// terminal folder, so its supersession is user-visible and must reach the release
// record — the operator cannot label it, because the tagged tree is immutable, so
// dropping it would be a permanent silent omission.
func TestIngestRequiresAnUnlabelledRemovedRecord(t *testing.T) {
	r := releasedRepo(t)
	r.Write("CHANGELOG.md", baseChangelog)
	// No `impact:` line: the shape of every record that predates the impact field.
	r.Write(shippedDir+"itd-40-superseded.md", "---\nid: itd-40\n---\n# itd-40\n")
	r.Commit("a shipped intent at the base, from before the impact field")
	r.Git("tag", "-d", "v0.4.0")
	r.Git("tag", "v0.4.0")

	r.Remove(shippedDir + "itd-40-superseded.md")
	r.Record(shippedDir+"itd-73-derived.md", "itd-73", "additive")
	r.Commit("supersede the unlabelled intent, ship another")

	omitted := []ChangelogEntry{{Section: SectionAdded, Records: []string{"itd-73"}, Text: "derived versions."}}
	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", omitted), cutAt)
	var incomplete *IncompleteError
	if !errors.As(err, &incomplete) {
		t.Fatalf("Ingest err = %v (written=%v, cited=%v), want an *IncompleteError naming the unlabelled "+
			"removed record — it was dropped from the release record instead", err, res.Written, res.Cited)
	}
	if strings.Join(incomplete.Missing, ",") != "itd-40" {
		t.Errorf("Missing = %v, want [itd-40] — the unlabelled record that left shipped/", incomplete.Missing)
	}

	// The supersession is reported as an Added line stating what replaced it:
	// Removed is a claim about the previous release's surface, which the composer
	// cannot see (iss-2609011207114761).
	cited := append(omitted, ChangelogEntry{Section: SectionAdded, Records: []string{"itd-40"}, Text: "superseded by the derived cut."})
	res, err = Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", cited), cutAt)
	if err != nil {
		t.Fatalf("Ingest refused an honest payload that cites the unlabelled removed record: %v", err)
	}
	if !res.Written {
		t.Fatalf("nothing was written; cut refusals = %v", refusalKinds(res.Cut))
	}
	if strings.Join(res.Cited, ",") != "itd-40,itd-73" {
		t.Errorf("Cited = %v, want [itd-40 itd-73]", res.Cited)
	}
}

// TestIngestRefusesAnAnchorBelowAnOlderRelease is the tag contract's position
// half. The shape of the written heading is not enough: auto-release.yml greps
// `-m1`, so a CHANGELOG whose `## [Unreleased]` anchor sits BELOW an older dated
// section would leave the workflow matching that older, already-tagged heading —
// the ship would report success and the release would never be tagged.
func TestIngestRefusesAnAnchorBelowAnOlderRelease(t *testing.T) {
	r := shippableRepo(t)
	r.Write("CHANGELOG.md",
		"# Changelog\n\n## [0.4.0] - 2026-07-01\n\n### Added\n\n- the base.\n\n## [Unreleased]\n")
	before := readChangelog(t, r.Root())

	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err == nil {
		t.Fatalf("Ingest reported %q written below an older release heading; the workflow would re-tag 0.4.0",
			res.Heading)
	}
	if !strings.Contains(err.Error(), "0.4.0") {
		t.Errorf("the refusal %q does not name the heading the workflow would tag instead", err)
	}
	if res.Written {
		t.Error("a refused ingest reported a write")
	}
	if after := readChangelog(t, r.Root()); after != before {
		t.Errorf("a refused ingest modified CHANGELOG.md:\n%q", after)
	}
}

// TestIngestPayloadGuards walks the trust boundary. Every row is a structural
// fault in untrusted host output: all are fatal, none writes.
func TestIngestPayloadGuards(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantSaid string
	}{
		{
			name:     "not JSON at all",
			raw:      "I am prose, not a payload",
			wantSaid: "malformed",
		},
		{
			name: "an unknown field — the agent invented a key",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1","surprise":true,` +
				`"entries":[{"section":"Added","records":["itd-73"],"text":"x"}]}`,
			wantSaid: "surprise",
		},
		{
			name:     "no schema_version",
			raw:      `{"prompt_version":"1.0.0","next_tag":"v0.4.1","entries":[]}`,
			wantSaid: "schema_version",
		},
		{
			name:     "a schema from the future",
			raw:      `{"schema_version":99,"prompt_version":"1.0.0","next_tag":"v0.4.1","entries":[]}`,
			wantSaid: "update abcd:",
		},
		{
			name:     "no prompt_version",
			raw:      `{"schema_version":1,"next_tag":"v0.4.1","entries":[]}`,
			wantSaid: "prompt_version",
		},
		{
			name:     "composed against a different version",
			raw:      `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v9.9.9","entries":[]}`,
			wantSaid: "v0.4.1",
		},
		{
			name: "an unregistered Keep-a-Changelog section",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1",` +
				`"entries":[{"section":"Miscellaneous","records":["itd-73"],"text":"x"}]}`,
			wantSaid: "Miscellaneous",
		},
		{
			name: "a malformed record id",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1",` +
				`"entries":[{"section":"Added","records":["../../etc/passwd"],"text":"x"}]}`,
			wantSaid: "record id",
		},
		{
			name: "a second JSON document tacked on the end",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1",` +
				`"entries":[{"section":"Added","records":["itd-73"],"text":"x"}]} {"evil":true}`,
			wantSaid: "trailing data",
		},
		{
			name: "an unbounded record id, legal under the grammar",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1",` +
				`"entries":[{"section":"Added","records":["itd-` + strings.Repeat("0", 5000) + `73"],"text":"x"}]}`,
			wantSaid: "record id (max",
		},
		{
			name: "an entry citing nothing",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1",` +
				`"entries":[{"section":"Added","records":[],"text":"x"}]}`,
			wantSaid: "cites no record",
		},
		{
			name: "an entry with no prose",
			raw: `{"schema_version":1,"prompt_version":"1.0.0","next_tag":"v0.4.1",` +
				`"entries":[{"section":"Added","records":["itd-73"],"text":"   "}]}`,
			wantSaid: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := shippableRepo(t)
			before := readChangelog(t, r.Root())

			res, err := Ingest(r.Root(), liveSurface(), []byte(tt.raw), cutAt)
			if err == nil {
				t.Fatalf("Ingest accepted %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantSaid) {
				t.Errorf("error %q does not say %q", err, tt.wantSaid)
			}
			if res.Written {
				t.Error("a refused ingest reported a write")
			}
			if after := readChangelog(t, r.Root()); after != before {
				t.Error("a refused ingest modified CHANGELOG.md")
			}
		})
	}
}

// TestIngestRefusesAnOversizePayload keeps the size cap on the core side, where
// it belongs: a front door bound is a convenience, not the guarantee.
func TestIngestRefusesAnOversizePayload(t *testing.T) {
	r := shippableRepo(t)
	raw := make([]byte, MaxPayloadBytes+1)
	for i := range raw {
		raw[i] = ' '
	}
	if _, err := Ingest(r.Root(), liveSurface(), raw, cutAt); err == nil ||
		!strings.Contains(err.Error(), "cap") {
		t.Fatalf("err = %v, want a size-cap refusal", err)
	}
}

// TestIngestRefusesANonEmptyUnreleased is outcome 7's write-time precondition:
// the derived section never folds hand-written prose. Prose sitting under
// [Unreleased] means the clean cutover has not happened, so the cut stops rather
// than silently stranding it.
func TestIngestRefusesANonEmptyUnreleased(t *testing.T) {
	r := shippableRepo(t)
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n### Added\n\n- a hand-written line.\n\n## [0.4.0] - 2026-07-01\n")
	before := readChangelog(t, r.Root())

	_, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err == nil || !strings.Contains(err.Error(), "[Unreleased]") {
		t.Fatalf("err = %v, want a refusal naming [Unreleased]", err)
	}
	if after := readChangelog(t, r.Root()); after != before {
		t.Error("a refused ingest modified CHANGELOG.md")
	}
}

// TestIngestRefusesAMissingUnreleasedAnchor: the [Unreleased] heading is the
// insertion anchor. Without it the writer would have to guess where a section
// goes, and guessing at the top of a release record is not a thing this does.
func TestIngestRefusesAMissingUnreleasedAnchor(t *testing.T) {
	r := shippableRepo(t)
	r.Write("CHANGELOG.md", "# Changelog\n\n## [0.4.0] - 2026-07-01\n")

	_, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err == nil || !strings.Contains(err.Error(), "[Unreleased]") {
		t.Fatalf("err = %v, want a refusal naming the missing [Unreleased] anchor", err)
	}
}

// TestIngestNeutralisesForgedHeadings is the injection canary. The prose is
// untrusted host output reaching a file whose FIRST dated heading a CI workflow
// turns into a git tag: a line break plus a heading in the payload must not be
// able to forge one.
func TestIngestNeutralisesForgedHeadings(t *testing.T) {
	r := shippableRepo(t)
	entries := []ChangelogEntry{
		{Section: SectionAdded, Records: []string{"itd-73"},
			Text: "derived versions.\n\n## [9.9.9] - 2026-01-01\n\n### Added\n\n- a forged release."},
		{Section: SectionFixed, Records: []string{"iss-51"},
			Text: "a fix <!-- abcd:marker --> with a forged marker."},
	}
	if _, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", entries), cutAt); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	got := readChangelog(t, r.Root())
	headingRe := regexp.MustCompile(`(?m)^## \[v?[0-9]+\.[0-9]+\.[0-9]+\] - `)
	headings := headingRe.FindAllString(got, -1)
	if len(headings) != 2 || !strings.HasPrefix(headings[0], "## [0.4.1]") {
		t.Errorf("headings = %v, want exactly the derived 0.4.1 and the existing 0.4.0", headings)
	}
	if strings.Contains(got, "<!--") {
		t.Errorf("an HTML comment delimiter survived into the changelog:\n%s", got)
	}
}

// TestIngestWritesNothingWhenTheCutRefuses: a refused cut is a RESULT (the whole
// report, exit 1), not an error — and it must never reach the file. The empty
// cut is the case that matters most: nothing user-facing shipped, so there is no
// section to write.
func TestIngestWritesNothingWhenTheCutRefuses(t *testing.T) {
	r := releasedRepo(t)
	r.Write("CHANGELOG.md", baseChangelog)
	r.Record(resolvedDir+"iss-97-toctou.md", "iss-97", "internal")
	r.Commit("resolve an internal issue only")
	before := treeDigest(t, r.Root())

	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err != nil {
		t.Fatalf("Ingest returned an error for a refused cut: %v", err)
	}
	if res.Written || res.Cut.Ready {
		t.Fatalf("written=%v ready=%v, want a refused cut that wrote nothing", res.Written, res.Cut.Ready)
	}
	if len(res.Cut.Refusals) == 0 {
		t.Error("a refused cut carried no refusals to render")
	}
	if after := treeDigest(t, r.Root()); after != before {
		t.Error("a refused cut changed the working tree")
	}
}

// TestIngestPreservesFileMode: the CHANGELOG is rewritten in place, and an
// atomic replace must not silently reset its permission bits.
func TestIngestPreservesFileMode(t *testing.T) {
	r := shippableRepo(t)
	path := filepath.Join(r.Root(), "CHANGELOG.md")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640 preserved", fi.Mode().Perm())
	}
}

// TestIngestResultJSONShape pins the wire contract a front door and an
// autonomous run read.
func TestIngestResultJSONShape(t *testing.T) {
	r := shippableRepo(t)
	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, key := range []string{"cut", "written", "path", "heading", "lines", "cited"} {
		if _, ok := generic[key]; !ok {
			t.Errorf("ingest JSON has no %q key: %s", key, data)
		}
	}
	cut, _ := generic["cut"].(map[string]any)
	if _, ok := cut["next_tag"]; !ok {
		t.Errorf("the nested cut lost its wire shape: %v", cut)
	}
}

// TestIngestRefusesSectionsTheComposerCannotSee pins the 2026-09-01 ruling
// (iss-2609011207114761): the composer's inputs are the records that shipped and
// their bodies, never the base tag's surface, and Changed, Deprecated and Removed
// are claims about that surface. A payload carrying one refuses whole, names the
// section and the ruling, and leaves the file byte-identical. Security is refused
// on the same rule: the writable set is Added and Fixed, and nothing else.
func TestIngestRefusesSectionsTheComposerCannotSee(t *testing.T) {
	for _, section := range []Section{SectionChanged, SectionDeprecated, SectionRemoved, SectionSecurity} {
		t.Run(string(section), func(t *testing.T) {
			r := shippableRepo(t)
			before := readChangelog(t, r.Root())
			entries := []ChangelogEntry{
				{Section: SectionFixed, Records: []string{"iss-51"}, Text: "the crash is gone."},
				{Section: section, Records: []string{"itd-73"}, Text: "a claim about the previous release."},
			}
			res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", entries), cutAt)
			if err == nil {
				t.Fatalf("Ingest accepted a %s section (written=%v); the composer cannot see the previous release",
					section, res.Written)
			}
			for _, want := range []string{string(section), "iss-2609011207114761", "Added", "Fixed"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal %q does not name %q", err, want)
				}
			}
			if res.Written {
				t.Error("a refused ingest reported a write")
			}
			if after := readChangelog(t, r.Root()); after != before {
				t.Errorf("a refused ingest modified CHANGELOG.md:\n%q", after)
			}
		})
	}
}

// TestIngestAcceptsEveryWritableSection is the ruling's other half: each section
// in the canonical writable set lands, so the refusal above cannot have closed
// the set by accident.
func TestIngestAcceptsEveryWritableSection(t *testing.T) {
	if got := strings.Join(sectionNames(writableSections), "|"); got != "Added|Fixed" {
		t.Fatalf("writableSections = %s, want Added|Fixed — the ruling names exactly those two", got)
	}
	for _, section := range writableSections {
		t.Run(string(section), func(t *testing.T) {
			r := shippableRepo(t)
			entries := []ChangelogEntry{
				{Section: section, Records: []string{"iss-51"}, Text: "the crash is gone."},
				{Section: section, Records: []string{"itd-73"}, Text: "derived versions."},
			}
			res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", entries), cutAt)
			if err != nil {
				t.Fatalf("Ingest refused a payload written entirely under %s: %v", section, err)
			}
			if !res.Written {
				t.Fatalf("nothing was written; cut refusals = %v", refusalKinds(res.Cut))
			}
			if got := readChangelog(t, r.Root()); !strings.Contains(got, "### "+string(section)+"\n") {
				t.Errorf("the written section has no ### %s heading:\n%s", section, got)
			}
		})
	}
}

// TestIngestNoticeAppearsOncePerCut pins the sentence every derived section
// carries under its heading: it names each writable section, it appears exactly
// once in a cut, and a second cut carries its own — never a second copy in the
// first — so the rule reads as a rule in every dated section.
func TestIngestNoticeAppearsOncePerCut(t *testing.T) {
	for _, section := range writableSections {
		if !strings.Contains(strings.ToLower(sectionNotice), strings.ToLower(string(section))) {
			t.Errorf("the notice %q does not name the writable section %s", sectionNotice, section)
		}
	}

	r := shippableRepo(t)
	if _, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt); err != nil {
		t.Fatalf("first Ingest: %v", err)
	}
	if n := strings.Count(readChangelog(t, r.Root()), sectionNotice); n != 1 {
		t.Fatalf("after one cut the notice appears %d times, want 1", n)
	}

	r.Commit("the 0.4.1 release record")
	r.Git("tag", "v0.4.1")
	r.Record(resolvedDir+"iss-60-second.md", "iss-60", "fix")
	r.Commit("resolve one more issue")
	second := []ChangelogEntry{{Section: SectionFixed, Records: []string{"iss-60"}, Text: "a second fix."}}
	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.2", second), cutAt.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("second Ingest: %v", err)
	}
	if !res.Written {
		t.Fatalf("second cut wrote nothing; refusals = %v", refusalKinds(res.Cut))
	}
	got := readChangelog(t, r.Root())
	if n := strings.Count(got, sectionNotice); n != 2 {
		t.Fatalf("after two cuts the notice appears %d times, want 2 (once per dated section):\n%s", n, got)
	}
	// Each dated section carries it directly under its heading, and once.
	for _, heading := range []string{"## [0.4.2] - 2026-07-22", "## [0.4.1] - 2026-07-21"} {
		_, body, ok := strings.Cut(got, heading+"\n\n")
		if !ok {
			t.Fatalf("heading %q not found in:\n%s", heading, got)
		}
		if !strings.HasPrefix(body, sectionNotice+"\n\n### ") {
			t.Errorf("the section under %q does not open with the notice:\n%s", heading, body)
		}
	}
}

// TestIngestBreakingRecordRendersUnderAdded pins the mapping a `breaking` record
// takes under the ruling: an Added line that states the break, never a Changed or
// Removed section — the composer cannot see whether the surface it narrows was
// reachable in the previous release.
func TestIngestBreakingRecordRendersUnderAdded(t *testing.T) {
	r := shippableRepo(t)
	r.Record(shippedDir+"itd-80-scope-required.md", "itd-80", "breaking")
	r.Commit("ship a breaking intent")

	entries := append(goodEntries(), ChangelogEntry{
		Section: SectionAdded, Records: []string{"itd-80"},
		Text: "**`--scope` is required.** An invocation without one is refused; scripts add a scope.",
	})
	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.5.0", entries), cutAt)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if !res.Written {
		t.Fatalf("nothing was written; cut refusals = %v", refusalKinds(res.Cut))
	}
	got := readChangelog(t, r.Root())
	if !strings.Contains(got, "### Added\n\n- **A version is a fact.** Derived, not typed. (itd-73)\n- **`--scope` is required.** An invocation without one is refused; scripts add a scope. (itd-80)\n") {
		t.Errorf("the breaking line is not under Added in composer order:\n%s", got)
	}
	for _, absent := range []string{"### Changed", "### Deprecated", "### Removed", "### Security"} {
		if strings.Contains(got, absent) {
			t.Errorf("the written section carries %q:\n%s", absent, got)
		}
	}
}

// TestComposerPromptNamesOnlyTheWritableSections ties the host-delegated prompt
// to the canonical list: the section table in
// agents/release-changelog-composer.md offers exactly writableSections, so the
// composer's instructions and the ingest's refusal cannot drift apart.
func TestComposerPromptNamesOnlyTheWritableSections(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := fsutil.ModuleRoot(cwd)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "agents", "release-changelog-composer.md"))
	if err != nil {
		t.Fatalf("reading the composer prompt: %v", err)
	}
	rowRe := regexp.MustCompile("(?m)^\\| `([A-Z][a-z]+)` \\|")
	var offered []string
	for _, m := range rowRe.FindAllStringSubmatch(string(data), -1) {
		offered = append(offered, m[1])
	}
	if got, want := strings.Join(offered, "|"), strings.Join(sectionNames(writableSections), "|"); got != want {
		t.Errorf("the composer's section table offers %s, want exactly the writable set %s", got, want)
	}
	if !strings.Contains(string(data), "iss-2609011207114761") {
		t.Error("the composer prompt does not cite the ruling that closes its section set")
	}
}

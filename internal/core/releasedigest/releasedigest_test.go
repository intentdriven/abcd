package releasedigest

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
)

// doc renders a digest document from releases, as a test fixture.
func doc(t *testing.T, releases ...Release) []byte {
	t.Helper()
	out, err := Marshal(&Digest{Releases: releases})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return out
}

func rel(tag, base string, entries ...Entry) Release {
	return Release{Tag: tag, BaseTag: base, Entries: entries}
}

func entry(line string, recs ...Record) Entry { return Entry{Line: line, Records: recs} }

func fix(id string) Record      { return Record{ID: id, Impact: ImpactFix} }
func additive(id string) Record { return Record{ID: id, Impact: ImpactAdditive} }
func breaking(id string) Record { return Record{ID: id, Impact: ImpactBreaking} }

// chain is three releases, each cut from the one below it.
func chain(t *testing.T) *Digest {
	t.Helper()
	d, err := Parse(doc(t,
		rel("v0.16.0", "v0.15.0",
			entry("A new verb.", additive("itd-1")),
			entry("Breaking: a flag is gone.", breaking("itd-2"), fix("iss-3")),
			entry("A fix.", fix("iss-4"))),
		rel("v0.15.0", "v0.14.2", entry("Another fix.", fix("iss-5"))),
		rel("v0.14.2", "v0.14.1", entry("An older fix.", fix("iss-6"))),
	))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func tags(rs []Release) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Tag)
	}
	return out
}

// TestParseRoundTrips: a well-formed digest parses, and marshals back to the
// same bytes, so appending a release never reformats the releases already
// there.
func TestParseRoundTrips(t *testing.T) {
	raw := doc(t,
		rel("v0.2.0", "v0.1.0", entry("Line with `code` & <angle> \"quotes\".", additive("itd-7"), fix("iss-8"))),
		rel("v0.1.0", "", entry("First.", fix("iss-1"))),
	)
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	back, err := Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, raw) {
		t.Fatalf("round trip changed the document:\n%s\n---\n%s", raw, back)
	}
	if d.Releases[0].Entries[0].Records[1] != fix("iss-8") {
		t.Errorf("records not kept: %+v", d.Releases[0].Entries[0].Records)
	}
}

// TestParseRefusesAMalformedDigest: the read is strict and capped. Unknown
// fields, repeated keys, trailing content, a wrong schema version, a version
// that is not exactly a release tag, an order the chain cannot be walked in,
// and every entry fault refuse the whole digest.
func TestParseRefusesAMalformedDigest(t *testing.T) {
	good := `{"tag":"v0.2.0","base_tag":"v0.1.0","entries":[{"line":"L.","records":[{"id":"iss-1","impact":"fix"}]}]}`
	wrap := func(releases string) string { return `{"schema_version":1,"releases":[` + releases + `]}` }
	withEntry := func(e string) string {
		return wrap(`{"tag":"v0.2.0","base_tag":"v0.1.0","entries":[` + e + `]}`)
	}
	manyRecords := make([]string, maxRecordsPerEntry+1)
	for i := range manyRecords {
		manyRecords[i] = `{"id":"iss-` + strings.Repeat("1", i+1) + `","impact":"fix"}`
	}
	for name, raw := range map[string]string{
		"not JSON":                     `{`,
		"an unknown top-level key":     `{"schema_version":1,"releases":[],"extra":1}`,
		"an unknown entry key":         withEntry(`{"line":"L.","records":[{"id":"iss-1","impact":"fix"}],"section":"Added"}`),
		"a repeated key":               `{"schema_version":1,"schema_version":1,"releases":[]}`,
		"trailing content":             wrap(good) + `{}`,
		"no schema version":            `{"releases":[]}`,
		"a newer schema version":       `{"schema_version":2,"releases":[]}`,
		"a tag with a leading zero":    wrap(`{"tag":"v0.02.0","base_tag":"v0.1.0","entries":[{"line":"L.","records":[{"id":"iss-1","impact":"fix"}]}]}`),
		"a pre-release tag":            wrap(`{"tag":"v0.2.0-rc.1","base_tag":"v0.1.0","entries":[{"line":"L.","records":[{"id":"iss-1","impact":"fix"}]}]}`),
		"a base tag that is not a tag": wrap(`{"tag":"v0.2.0","base_tag":"main","entries":[{"line":"L.","records":[{"id":"iss-1","impact":"fix"}]}]}`),
		"a base tag not older":         wrap(`{"tag":"v0.2.0","base_tag":"v0.2.0","entries":[{"line":"L.","records":[{"id":"iss-1","impact":"fix"}]}]}`),
		"releases oldest first":        wrap(`{"tag":"v0.1.0","base_tag":"","entries":[{"line":"L.","records":[{"id":"iss-1","impact":"fix"}]}]},` + good),
		"a repeated release":           wrap(good + `,` + good),
		"a release with no entries":    wrap(`{"tag":"v0.2.0","base_tag":"v0.1.0","entries":[]}`),
		"an entry with no records":     withEntry(`{"line":"L.","records":[]}`),
		"an empty line":                withEntry(`{"line":"  ","records":[{"id":"iss-1","impact":"fix"}]}`),
		"an oversized line":            withEntry(`{"line":"` + strings.Repeat("x", maxLineBytes+1) + `","records":[{"id":"iss-1","impact":"fix"}]}`),
		"a malformed record id":        withEntry(`{"line":"L.","records":[{"id":"adr-1","impact":"fix"}]}`),
		"an oversized record id":       withEntry(`{"line":"L.","records":[{"id":"iss-` + strings.Repeat("1", maxRecordIDBytes) + `","impact":"fix"}]}`),
		"a repeated record id":         withEntry(`{"line":"L.","records":[{"id":"iss-1","impact":"fix"},{"id":"iss-1","impact":"fix"}]}`),
		"an internal impact":           withEntry(`{"line":"L.","records":[{"id":"iss-1","impact":"internal"}]}`),
		"an unknown impact":            withEntry(`{"line":"L.","records":[{"id":"iss-1","impact":"major"}]}`),
		"too many records":             withEntry(`{"line":"L.","records":[` + strings.Join(manyRecords, ",") + `]}`),
		"an oversized document":        `{"schema_version":1,"releases":[]}` + strings.Repeat(" ", maxDigestBytes),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(raw)); err == nil {
				t.Fatalf("Parse accepted %s", name)
			}
		})
	}
	if _, err := Parse([]byte(wrap(good))); err != nil {
		t.Fatalf("the control document is refused: %v", err)
	}
}

// TestParseSanitisesEveryLine: a line is passed through termsafe again on
// read, so a control sequence or a hidden rune in the embedded file never
// reaches a terminal or an agent's context as written.
func TestParseSanitisesEveryLine(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"releases": []any{map[string]any{
			"tag": "v0.2.0", "base_tag": "v0.1.0",
			"entries": []any{map[string]any{
				"line":    "Shown \x1b]0;owned\x07 here‮ and​ there.",
				"records": []any{map[string]any{"id": "iss-1", "impact": "fix"}},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := d.Releases[0].Entries[0].Line
	for _, bad := range []string{"\x1b", "\x07", "‮", "​"} {
		if strings.Contains(got, bad) {
			t.Errorf("line %q still carries %q", got, bad)
		}
	}
}

// TestRangeWalksTheChain: every release from the running one down to the one
// the range starts after is returned, newest first, with no gap.
func TestRangeWalksTheChain(t *testing.T) {
	d := chain(t)
	s, err := d.Range("v0.14.1", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tags(s.Releases), " "); got != "v0.16.0 v0.15.0 v0.14.2" {
		t.Fatalf("releases = %s", got)
	}
	if s.Partial || s.Gap != nil {
		t.Fatalf("a complete chain reported a gap: %+v", s.Gap)
	}
	// A start that is no release of the chain stops at the first release at
	// or below it.
	s, err = d.Range("v0.14.3", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tags(s.Releases), " "); got != "v0.16.0 v0.15.0" || s.Partial {
		t.Fatalf("releases = %s partial %v", got, s.Partial)
	}
}

// TestRangePutsActionFirst: within a release, the entries that cite a
// breaking record come first, the rest keep their order.
func TestRangePutsActionFirst(t *testing.T) {
	s, err := chain(t).Range("v0.15.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, e := range s.Releases[0].Entries {
		lines = append(lines, e.Line)
	}
	want := "Breaking: a flag is gone.|A new verb.|A fix."
	if got := strings.Join(lines, "|"); got != want {
		t.Fatalf("order = %s, want %s", got, want)
	}
	if !s.Releases[0].Entries[0].NeedsAction() || s.Releases[0].Entries[1].NeedsAction() {
		t.Errorf("NeedsAction must be true exactly for an entry citing a breaking record")
	}
	if s.Releases[0].NeedsAction() != 1 {
		t.Errorf("NeedsAction count = %d, want 1", s.Releases[0].NeedsAction())
	}
}

// TestRangeNamesAGapWithWhereItsNotesAre: a release the chain does not reach
// is named as not described with its published address, and the summary is
// partial; nothing below it is guessed.
func TestRangeNamesAGapWithWhereItsNotesAre(t *testing.T) {
	d, err := Parse(doc(t,
		rel("v0.16.0", "v0.15.0", entry("New.", additive("itd-1"))),
		rel("v0.14.2", "v0.14.1", entry("Old.", fix("iss-2"))),
	))
	if err != nil {
		t.Fatal(err)
	}
	s, err := d.Range("v0.14.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tags(s.Releases), " "); got != "v0.16.0" {
		t.Fatalf("releases = %s, want only v0.16.0: the chain stops at the gap", got)
	}
	if !s.Partial || s.Gap == nil {
		t.Fatalf("no gap reported: %+v", s)
	}
	want := Gap{
		After:       "v0.14.0",
		Through:     "v0.15.0",
		NotesURL:    core.ReleaseOrigin + "/releases/tag/v0.15.0",
		ReleasesURL: core.ReleaseOrigin + "/releases",
	}
	if *s.Gap != want {
		t.Fatalf("gap = %+v, want %+v", *s.Gap, want)
	}

	// A running release the digest does not carry is itself the gap.
	s, err = d.Range("v0.14.2", "v0.17.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Releases) != 0 || s.Gap == nil || s.Gap.Through != "v0.17.0" || s.Gap.After != "v0.14.2" || !s.Partial {
		t.Fatalf("a missing running release: %+v gap %+v", s, s.Gap)
	}
}

// TestRangeWithNoStartIsTheRunningReleaseAlone: with nothing known about
// where the computer moved from, the range is the running release only.
func TestRangeWithNoStartIsTheRunningReleaseAlone(t *testing.T) {
	s, err := chain(t).Range("", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tags(s.Releases), " "); got != "v0.16.0" || s.Partial {
		t.Fatalf("releases = %s partial %v", got, s.Partial)
	}
}

// TestRangeOfNoReleases: a start at or above the running release names
// nothing and no gap.
func TestRangeOfNoReleases(t *testing.T) {
	for _, from := range []string{"v0.16.0", "v0.17.0"} {
		s, err := chain(t).Range(from, "v0.16.0")
		if err != nil {
			t.Fatal(err)
		}
		if len(s.Releases) != 0 || s.Partial {
			t.Fatalf("Range(%s, v0.16.0) = %+v", from, s)
		}
	}
}

// TestRangeRefusesAVersionThatIsNotAReleaseTag: neither end may be anything
// but exactly vMAJOR.MINOR.PATCH; a dev build takes no part.
func TestRangeRefusesAVersionThatIsNotAReleaseTag(t *testing.T) {
	d := chain(t)
	for _, c := range [][2]string{{"v0.15.0", "dev"}, {"dev", "v0.16.0"}, {"v0.15", "v0.16.0"}, {"v0.15.0", "0.16.0"}} {
		if _, err := d.Range(c[0], c[1]); !errors.Is(err, ErrNotReleaseTag) {
			t.Errorf("Range(%q, %q) err = %v, want ErrNotReleaseTag", c[0], c[1], err)
		}
	}
}

// TestAppendAddsTheNewestRelease: the cut's write. The release goes on top,
// the releases already there keep their bytes, and a release that is not
// newer than the newest, or that does not validate, is refused.
func TestAppendAddsTheNewestRelease(t *testing.T) {
	first, err := Append(nil, rel("v0.15.0", "v0.14.2", entry("Fix.", fix("iss-5"))))
	if err != nil {
		t.Fatalf("Append to nothing: %v", err)
	}
	next, err := Append(first, rel("v0.16.0", "v0.15.0", entry("New.", additive("itd-1"), breaking("itd-2"))))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	d, err := Parse(next)
	if err != nil {
		t.Fatalf("the appended digest does not parse: %v", err)
	}
	if got := strings.Join(tags(d.Releases), " "); got != "v0.16.0 v0.15.0" {
		t.Fatalf("releases = %s", got)
	}
	want, _ := Marshal(&Digest{Releases: []Release{
		rel("v0.16.0", "v0.15.0", entry("New.", additive("itd-1"), breaking("itd-2"))),
		rel("v0.15.0", "v0.14.2", entry("Fix.", fix("iss-5"))),
	}})
	if !bytes.Equal(next, want) {
		t.Fatalf("appended bytes are not canonical:\n%s", next)
	}
	for name, r := range map[string]Release{
		"the same release":        rel("v0.16.0", "v0.15.0", entry("Again.", fix("iss-9"))),
		"an older release":        rel("v0.15.1", "v0.15.0", entry("Late.", fix("iss-9"))),
		"a malformed entry":       rel("v0.17.0", "v0.16.0", entry("No records.")),
		"a dev tag":               rel("dev", "v0.16.0", entry("Dev.", fix("iss-9"))),
		"an internal record":      rel("v0.17.0", "v0.16.0", entry("Plumbing.", Record{ID: "iss-9", Impact: "internal"})),
		"a base that is no older": rel("v0.17.0", "v0.17.0", entry("Self.", fix("iss-9"))),
	} {
		if _, err := Append(next, r); err == nil {
			t.Errorf("Append accepted %s", name)
		}
	}
	if _, err := Append([]byte(`{"schema_version":1,"releases":[],"x":1}`), rel("v0.1.0", "", entry("A.", fix("iss-1")))); err == nil {
		t.Errorf("Append wrote over a digest that does not parse")
	}
}

// TestEmbeddedDigestParses: the digest the binary carries reads cleanly.
func TestEmbeddedDigestParses(t *testing.T) {
	d, err := Embedded()
	if err != nil {
		t.Fatalf("the embedded digest does not parse: %v", err)
	}
	if len(d.Releases) == 0 {
		t.Fatal("the embedded digest carries no release")
	}
}

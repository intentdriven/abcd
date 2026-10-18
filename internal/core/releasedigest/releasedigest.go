// Package releasedigest is the release digest: per release, its tag, the tag
// it was cut from, and each changelog line as `abcd launch ship` composed it,
// with the records the line cites and their impact (spc-2610170635097509).
//
// The digest travels inside the binary. `launch ship` appends the release it
// cuts to the file a repository declares under `release.digest`, and abcd's
// own declaration is this package's digest.json, which go:embed carries into
// the binary built from the release's tag. So a binary holds every release up
// to and including its own, and says what changed in an update without
// reading the repository it runs in or the network. The `.abcd/` namespace
// never reaches a released binary, which is why the file lives here.
//
// The embedded file is read as strictly as any other embedded default: one
// JSON document, no unknown field, no repeated key, nothing after it, capped
// in size, every version exactly a release tag, and every line sanitised again
// on read.
package releasedigest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SchemaVersion is the digest document's schema version. A document from a
// newer schema is refused, never half-read.
const SchemaVersion = 1

const (
	// MaxBytes caps the document, for the reader here and for the cut's read
	// of the file it appends to. The seed is a few hundred KiB and a release
	// adds tens; the cap leaves years of headroom and bounds a file that has
	// grown wrong.
	MaxBytes = 8 << 20
	// maxReleases caps the releases one digest holds.
	maxReleases = 4096
	// maxEntriesPerRelease, maxRecordsPerEntry, maxLineBytes and
	// maxRecordIDBytes are the caps the cut holds a composed line to
	// (internal/core/release), so every line a cut accepts fits.
	maxEntriesPerRelease = 500
	maxRecordsPerEntry   = 32
	maxLineBytes         = 4096
	maxRecordIDBytes     = 32
)

// Impact is a cited record's declared impact. Only the three that earn a
// changelog line appear: an internal record is never cited.
type Impact string

const (
	ImpactAdditive Impact = "additive"
	ImpactBreaking Impact = "breaking"
	ImpactFix      Impact = "fix"
)

// recordIDRe is a cited record id: the grammar the cut's bijection reads.
var recordIDRe = regexp.MustCompile(`\A(?:itd|iss)-[0-9]+\z`)

// ErrNotReleaseTag is Range's refusal of a version that is not exactly a
// release tag; a dev build takes no part.
var ErrNotReleaseTag = errors.New("not a release tag of the form vMAJOR.MINOR.PATCH")

//go:embed digest.json
var embedded []byte

// Record is one record a line cites, with its declared impact.
type Record struct {
	ID     string `json:"id"`
	Impact Impact `json:"impact"`
}

// Entry is one changelog line, stored once with every record it cites. Line
// is the line's wording without the citation suffix the cut appends.
type Entry struct {
	Line    string   `json:"line"`
	Records []Record `json:"records"`
}

// NeedsAction reports whether the entry asks the person to act: any record it
// cites is breaking (the changelog states a break as an Added line, so the
// impact, not the section, is the signal).
func (e Entry) NeedsAction() bool {
	for _, r := range e.Records {
		if r.Impact == ImpactBreaking {
			return true
		}
	}
	return false
}

// Release is one cut release: its tag, the tag it was cut from ("" for the
// first release a repository cut), and its lines in the CHANGELOG's order.
type Release struct {
	Tag     string  `json:"tag"`
	BaseTag string  `json:"base_tag"`
	Entries []Entry `json:"entries"`
}

// NeedsAction counts the release's entries that need the person to act.
func (r Release) NeedsAction() int {
	n := 0
	for _, e := range r.Entries {
		if e.NeedsAction() {
			n++
		}
	}
	return n
}

// Digest is the whole document, releases newest first.
type Digest struct {
	Releases []Release
	byTag    map[string]int
}

// document is the wire form.
type document struct {
	SchemaVersion int       `json:"schema_version"`
	Releases      []Release `json:"releases"`
}

// Gap names what a range cannot describe: the release Through, and any
// release after After below it, which the chain cannot list. NotesURL is
// where Through's release notes are published, ReleasesURL where every
// release's are. Nothing in a gap is guessed.
type Gap struct {
	After       string `json:"after,omitempty"`
	Through     string `json:"through"`
	NotesURL    string `json:"notes_url"`
	ReleasesURL string `json:"releases_url"`
}

// Summary is a range of releases: those the digest describes, newest first
// with each release's entries needing action first, and the gap, if any.
// Partial is true exactly when there is a gap.
type Summary struct {
	From     string    `json:"from,omitempty"`
	To       string    `json:"to"`
	Releases []Release `json:"releases"`
	Gap      *Gap      `json:"gap,omitempty"`
	Partial  bool      `json:"partial"`
}

// notesURL is where a release's notes are published: the release origin
// internal/core/update pins, so the address is derived and never fetched.
func notesURL(tag string) string { return core.ReleaseOrigin + "/releases/tag/" + tag }

// releasesURL is where every release's notes are listed.
func releasesURL() string { return core.ReleaseOrigin + "/releases" }

var loadEmbedded = sync.OnceValues(func() (*Digest, error) { return Parse(embedded) })

// embeddedDigest is the digest this binary carries.
func embeddedDigest() (*Digest, error) { return loadEmbedded() }

// Range is the embedded digest's Range.
func Range(from, to string) (Summary, error) {
	d, err := embeddedDigest()
	if err != nil {
		return Summary{}, err
	}
	return d.Range(from, to)
}

// Parse reads a digest document strictly and validates every release.
func Parse(data []byte) (*Digest, error) {
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("release digest exceeds the %d-byte cap", MaxBytes)
	}
	var doc document
	if err := jsonstrict.Decode(data, &doc); err != nil {
		return nil, fmt.Errorf("release digest: %w", err)
	}
	switch {
	case doc.SchemaVersion == 0:
		return nil, errors.New("release digest: schema_version is missing")
	case doc.SchemaVersion != SchemaVersion:
		return nil, fmt.Errorf("release digest: schema_version %d is not %d, the one this binary reads", doc.SchemaVersion, SchemaVersion)
	}
	return build(doc.Releases)
}

// build validates releases, sanitises their lines and indexes them.
func build(releases []Release) (*Digest, error) {
	if len(releases) > maxReleases {
		return nil, fmt.Errorf("release digest: %d releases (max %d)", len(releases), maxReleases)
	}
	d := &Digest{Releases: make([]Release, 0, len(releases)), byTag: map[string]int{}}
	var newer launch.Semver
	for i, r := range releases {
		v, err := launch.ParseReleaseTag(r.Tag)
		if err != nil {
			return nil, fmt.Errorf("release digest: release %d: tag: %w", i+1, err)
		}
		if i > 0 && !launch.CoreGreater(newer, v) {
			return nil, fmt.Errorf("release digest: %s is not older than the release above it; releases are listed newest first, once each", r.Tag)
		}
		newer = v
		if r.BaseTag != "" {
			b, err := launch.ParseReleaseTag(r.BaseTag)
			if err != nil {
				return nil, fmt.Errorf("release digest: %s: base_tag: %w", r.Tag, err)
			}
			if !launch.CoreGreater(v, b) {
				return nil, fmt.Errorf("release digest: %s: base_tag %s is not older than the release", r.Tag, r.BaseTag)
			}
		}
		entries, err := cleanEntries(r.Tag, r.Entries)
		if err != nil {
			return nil, err
		}
		d.byTag[r.Tag] = len(d.Releases)
		d.Releases = append(d.Releases, Release{Tag: r.Tag, BaseTag: r.BaseTag, Entries: entries})
	}
	return d, nil
}

// cleanEntries validates one release's entries and sanitises each line.
func cleanEntries(tag string, in []Entry) ([]Entry, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("release digest: %s carries no entry", tag)
	}
	if len(in) > maxEntriesPerRelease {
		return nil, fmt.Errorf("release digest: %s carries %d entries (max %d)", tag, len(in), maxEntriesPerRelease)
	}
	out := make([]Entry, 0, len(in))
	for i, e := range in {
		at := fmt.Sprintf("release digest: %s entry %d", tag, i+1)
		if len(e.Line) > maxLineBytes {
			return nil, fmt.Errorf("%s: line exceeds %d bytes", at, maxLineBytes)
		}
		line := termsafe.Sanitize(e.Line)
		if strings.TrimSpace(line) == "" {
			return nil, fmt.Errorf("%s: line is empty", at)
		}
		if len(e.Records) == 0 {
			return nil, fmt.Errorf("%s: cites no record", at)
		}
		if len(e.Records) > maxRecordsPerEntry {
			return nil, fmt.Errorf("%s: cites %d records (max %d)", at, len(e.Records), maxRecordsPerEntry)
		}
		seen := map[string]bool{}
		recs := make([]Record, 0, len(e.Records))
		for _, r := range e.Records {
			if len(r.ID) > maxRecordIDBytes || !recordIDRe.MatchString(r.ID) {
				return nil, fmt.Errorf("%s: %s is not a record id (want itd-N or iss-N)", at, termsafe.DescribeRefused(r.ID))
			}
			if seen[r.ID] {
				return nil, fmt.Errorf("%s: cites %s twice", at, r.ID)
			}
			seen[r.ID] = true
			switch r.Impact {
			case ImpactAdditive, ImpactBreaking, ImpactFix:
			default:
				return nil, fmt.Errorf("%s: %s has impact %s; a cited record is additive, breaking or fix", at, r.ID, termsafe.DescribeRefused(string(r.Impact)))
			}
			recs = append(recs, r)
		}
		out = append(out, Entry{Line: line, Records: recs})
	}
	return out, nil
}

// marshal writes the digest in its one canonical form: two-space indented,
// HTML characters unescaped, one trailing newline.
func marshal(d *Digest) ([]byte, error) {
	releases := d.Releases
	if releases == nil {
		releases = []Release{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(document{SchemaVersion: SchemaVersion, Releases: releases}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Append adds rel as the newest release of the digest existing holds (empty
// for a digest not yet written) and returns the whole new document. The
// release must validate and be newer than every release already there; a
// digest that does not parse is refused rather than rebuilt.
func Append(existing []byte, rel Release) ([]byte, error) {
	var releases []Release
	if len(bytes.TrimSpace(existing)) > 0 {
		d, err := Parse(existing)
		if err != nil {
			return nil, err
		}
		releases = d.Releases
	}
	// A line sanitising would change is refused rather than stored changed:
	// the digest line is the CHANGELOG line, byte for byte, and a line the cut
	// composed is already clean.
	for i, e := range rel.Entries {
		if termsafe.Sanitize(e.Line) != e.Line {
			return nil, fmt.Errorf("release digest: %s entry %d carries a character the reader would mask", termsafe.Sanitize(rel.Tag), i+1)
		}
	}
	d, err := build(append([]Release{rel}, releases...))
	if err != nil {
		return nil, err
	}
	out, err := marshal(d)
	if err != nil {
		return nil, err
	}
	if _, err := Parse(out); err != nil {
		return nil, err
	}
	return out, nil
}

// Range returns the releases after from up to and including to, walking the
// chain of base tags down from to. A release the chain reaches that the
// digest does not carry stops the walk and is named in the gap, with where
// its notes are published, and the summary is partial. from "" means the
// running release alone; a from at or above to is an empty range. Either
// version must be exactly a release tag.
func (d *Digest) Range(from, to string) (Summary, error) {
	toV, err := launch.ParseReleaseTag(to)
	if err != nil {
		return Summary{}, ErrNotReleaseTag
	}
	var fromV launch.Semver
	if from != "" {
		if fromV, err = launch.ParseReleaseTag(from); err != nil {
			return Summary{}, ErrNotReleaseTag
		}
		if !launch.CoreGreater(toV, fromV) {
			return Summary{From: from, To: to, Releases: []Release{}}, nil
		}
	}
	s := Summary{From: from, To: to, Releases: []Release{}}
	for cur := to; ; {
		i, ok := d.byTag[cur]
		if !ok {
			s.Gap = &Gap{After: from, Through: cur, NotesURL: notesURL(cur), ReleasesURL: releasesURL()}
			s.Partial = true
			break
		}
		r := d.Releases[i]
		s.Releases = append(s.Releases, Release{Tag: r.Tag, BaseTag: r.BaseTag, Entries: actionFirst(r.Entries)})
		if from == "" || r.BaseTag == "" {
			break
		}
		// Validated on parse: the base parses and is older, so the walk ends.
		b, _ := launch.ParseReleaseTag(r.BaseTag)
		if !launch.CoreGreater(b, fromV) {
			break
		}
		cur = r.BaseTag
	}
	return s, nil
}

// actionFirst orders entries that need action first, keeping the order within
// each group.
func actionFirst(in []Entry) []Entry {
	out := slices.Clone(in)
	slices.SortStableFunc(out, func(a, b Entry) int {
		switch {
		case a.NeedsAction() == b.NeedsAction():
			return 0
		case a.NeedsAction():
			return -1
		}
		return 1
	})
	return out
}

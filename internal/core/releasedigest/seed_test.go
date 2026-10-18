package releasedigest

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// repoRoot is the repository root, three levels above this package.
const repoRoot = "../../.."

// firstDescribed is the oldest release the digest describes: the first whose
// CHANGELOG section `launch ship` derived from records. Every release below it
// was rolled by hand and is named as not described.
const firstDescribed = "v0.7.0"

var (
	headingRe = regexp.MustCompile(`^## \[(v?[0-9]+\.[0-9]+\.[0-9]+)\] - `)
	citedRe   = regexp.MustCompile(`^- (.*) \(((?:itd|iss)-[0-9]+(?:, (?:itd|iss)-[0-9]+)*)\)$`)
)

// changelogSection is one dated CHANGELOG section's tag and entry lines.
type changelogSection struct {
	tag   string
	lines []string
}

// readChangelog reads CHANGELOG.md's dated sections, newest first.
func readChangelog(t *testing.T) []changelogSection {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	var out []changelogSection
	for _, line := range strings.Split(string(data), "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			out = append(out, changelogSection{tag: "v" + strings.TrimPrefix(m[1], "v")})
			continue
		}
		if len(out) > 0 && strings.HasPrefix(line, "- ") {
			out[len(out)-1].lines = append(out[len(out)-1].lines, line)
		}
	}
	return out
}

// rendered is an entry as the cut renders it into CHANGELOG.md.
func rendered(e Entry) string {
	ids := make([]string, 0, len(e.Records))
	for _, r := range e.Records {
		ids = append(ids, r.ID)
	}
	return "- " + e.Line + " (" + strings.Join(ids, ", ") + ")"
}

// TestEmbeddedDigestIsTheChangelog: every described release is the CHANGELOG
// section of the same tag, line for line and in order, cut from the release
// below it; every derived section is described and no hand-rolled one is.
func TestEmbeddedDigestIsTheChangelog(t *testing.T) {
	d, err := embeddedDigest()
	if err != nil {
		t.Fatal(err)
	}
	first, err := launch.ParseReleaseTag(firstDescribed)
	if err != nil {
		t.Fatal(err)
	}
	sections := readChangelog(t)
	byTag := map[string]int{}
	for i, s := range sections {
		byTag[s.tag] = i
	}
	described := map[string]bool{}
	for _, r := range d.Releases {
		described[r.Tag] = true
		i, ok := byTag[r.Tag]
		if !ok {
			t.Errorf("%s is in the digest but has no CHANGELOG section", r.Tag)
			continue
		}
		if i+1 >= len(sections) || sections[i+1].tag != r.BaseTag {
			t.Errorf("%s: base_tag %q is not the CHANGELOG release below it", r.Tag, r.BaseTag)
		}
		var got []string
		for _, e := range r.Entries {
			got = append(got, rendered(e))
		}
		if strings.Join(got, "\n") != strings.Join(sections[i].lines, "\n") {
			t.Errorf("%s: the digest's lines are not its CHANGELOG lines", r.Tag)
		}
	}
	for _, s := range sections {
		v, err := launch.ParseReleaseTag(s.tag)
		if err != nil {
			continue
		}
		derived := !launch.CoreGreater(first, v)
		if derived != described[s.tag] {
			t.Errorf("%s: described = %v, want %v (derived sections from %s on are described, hand-rolled ones never)",
				s.tag, described[s.tag], derived, firstDescribed)
		}
	}
}

// TestEmbeddedDigestIsCanonical: the embedded file is exactly what marshal
// writes for it, so the cut's append changes only the release it adds.
func TestEmbeddedDigestIsCanonical(t *testing.T) {
	d, err := embeddedDigest()
	if err != nil {
		t.Fatal(err)
	}
	out, err := marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, embedded) {
		t.Fatal("digest.json is not in the canonical form marshal writes")
	}
}

// impactRe reads a record's declared impact from its frontmatter.
var impactRe = regexp.MustCompile(`(?m)^impact:\s*"?([a-z]+)"?\s*$`)

// TestEmbeddedDigestImpactsAreTheRecords: each cited record's impact is the
// one the record itself declares, read from the terminal folder it shipped in.
func TestEmbeddedDigestImpactsAreTheRecords(t *testing.T) {
	d, err := embeddedDigest()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, dir := range []string{
		".abcd/development/intents/shipped",
		".abcd/work/issues/resolved",
		".abcd/work/issues/wontfix",
	} {
		entries, err := os.ReadDir(filepath.Join(repoRoot, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := e.Name()
			if id, _, ok := strings.Cut(strings.TrimSuffix(name, ".md"), "-"); ok {
				rest := strings.TrimPrefix(name, id+"-")
				num, _, _ := strings.Cut(rest, "-")
				files[id+"-"+strings.TrimSuffix(num, ".md")] = filepath.Join(repoRoot, dir, name)
			}
		}
	}
	for _, r := range d.Releases {
		for _, e := range r.Entries {
			for _, rec := range e.Records {
				path, ok := files[rec.ID]
				if !ok {
					t.Errorf("%s: %s is in no terminal folder", r.Tag, rec.ID)
					continue
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				fm, _, _ := strings.Cut(strings.TrimPrefix(string(data), "---\n"), "\n---")
				m := impactRe.FindStringSubmatch(fm)
				if m == nil || m[1] != string(rec.Impact) {
					t.Errorf("%s: %s carries impact %q in the digest, the record says %v", r.Tag, rec.ID, rec.Impact, m)
				}
			}
		}
	}
}

package reflect

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// AuditRollup is the per-criterion verdict count an ingested audit writes on
// its `Acceptance rollup:` line.
type AuditRollup struct {
	Met             int `json:"met"`
	MetWithConcerns int `json:"met_with_concerns"`
	NotMet          int `json:"not_met"`
	Inconclusive    int `json:"inconclusive"`
}

func (a AuditRollup) add(b AuditRollup) AuditRollup {
	return AuditRollup{a.Met + b.Met, a.MetWithConcerns + b.MetWithConcerns, a.NotMet + b.NotMet, a.Inconclusive + b.Inconclusive}
}

// GapCounts is the honoured / diverged / missing count of an audit's gap audit.
type GapCounts struct {
	Honoured int `json:"honoured"`
	Diverged int `json:"diverged"`
	Missing  int `json:"missing"`
}

func (g GapCounts) add(b GapCounts) GapCounts {
	return GapCounts{g.Honoured + b.Honoured, g.Diverged + b.Diverged, g.Missing + b.Missing}
}

// SeedIntent is one intent the release shipped, as the interview opens from
// it: what it is, where it lives now, the impact it declared, and what its
// audit notes say, as counts. The notes themselves stay on the intent; the
// retrospective links to them.
type SeedIntent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Path is repo-relative and names where the record lives now, so a link
	// resolves; a record the intent store no longer holds is read from, and
	// named by, the tag's tree.
	Path   string `json:"path"`
	Impact string `json:"impact"`
	// Audited is true when the record carries audit notes: an ingested review
	// (its `Acceptance rollup:` line) or a hand-written audit. A placeholder, an
	// owed review and an absent section are not audit notes.
	Audited bool        `json:"audited"`
	Receipt string      `json:"receipt,omitempty"`
	Rollup  AuditRollup `json:"rollup"`
	Gaps    GapCounts   `json:"gaps"`
}

// AuditOffer names a shipped intent without audit notes and the command that
// audits it (criterion 2). It is an offer, never a gate.
type AuditOffer struct {
	IntentID string `json:"intent_id"`
	Command  string `json:"command"`
}

// TargetedIntent is a planned intent whose target_release names the release
// and which has not shipped (criterion 7).
type TargetedIntent struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// ChangelogSection is the section the cut composed for the tag. Found is false
// when CHANGELOG.md carries no dated heading for the release; the seed goes
// ahead without it.
type ChangelogSection struct {
	Found   bool   `json:"found"`
	Heading string `json:"heading,omitempty"`
	// Anchor is the fragment a link to the heading takes on a rendered page.
	Anchor string `json:"anchor,omitempty"`
	Body   string `json:"body,omitempty"`
}

// MetricsBlock is the metrics section, computed from the seed (spec scope 2):
// the intents shipped, the audit notes' verdict distribution, and the dates of
// this tag and the previous one.
type MetricsBlock struct {
	IntentsShipped  int         `json:"intents_shipped"`
	Audited         int         `json:"audited"`
	Unaudited       int         `json:"unaudited"`
	Rollup          AuditRollup `json:"rollup"`
	Gaps            GapCounts   `json:"gaps"`
	TagDate         string      `json:"tag_date"`
	PreviousTag     string      `json:"previous_tag,omitempty"`
	PreviousTagDate string      `json:"previous_tag_date,omitempty"`
}

// Seed is what a retrospective interview opens from (spec scope 1).
type Seed struct {
	Tag       string           `json:"tag"`
	Intents   []SeedIntent     `json:"intents"`
	Unaudited []AuditOffer     `json:"unaudited"`
	Unshipped []TargetedIntent `json:"unshipped_targets"`
	Changelog ChangelogSection `json:"changelog"`
	Metrics   MetricsBlock     `json:"metrics"`
	// Output is the repo-relative path the retrospective is written to.
	Output string `json:"output"`
}

// BuildSeed reads what the release tag shipped. It refuses a malformed tag, a
// tag the repository does not hold, a release that already has a
// retrospective, and a release that shipped no intent (NothingShippedError).
// It writes nothing.
//
// Which intents a tag shipped is read the way the release cut reads it
// (changelog.ShippedSince): the intents that reached shipped/ between the
// previous release tag and this one, less any that say `shipped_in:` another
// release, plus any in shipped/ now that say `shipped_in:` this one. The
// stamp alone is not enough: it is a migration field the cut never writes, so
// a release cut the ordinary way stamps none of its intents.
func BuildSeed(root, tag string) (Seed, error) {
	if err := validTag(tag); err != nil {
		return Seed{}, err
	}
	if _, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", tag+"^{commit}"); err != nil {
		return Seed{}, fmt.Errorf("reflect: no release tag %s in this repository", tag)
	}
	out := outputRel(tag)
	if present, err := outputPresent(root, out); err != nil {
		return Seed{}, err
	} else if present {
		return Seed{}, &ExistsError{Tag: tag, Path: out}
	}

	prev, err := previousTag(root, tag)
	if err != nil {
		return Seed{}, err
	}
	corpus, err := intent.Load(root)
	if err != nil {
		return Seed{}, err
	}
	members, err := shippedBy(root, tag, prev, corpus)
	if err != nil {
		return Seed{}, err
	}
	if len(members) == 0 {
		return Seed{}, &NothingShippedError{Tag: tag}
	}

	s := Seed{Tag: tag, Output: out, Unaudited: []AuditOffer{}, Unshipped: []TargetedIntent{}}
	for _, m := range members {
		it := readSeedIntent(m)
		s.Intents = append(s.Intents, it)
		s.Metrics.IntentsShipped++
		if it.Audited {
			s.Metrics.Audited++
			s.Metrics.Rollup = s.Metrics.Rollup.add(it.Rollup)
			s.Metrics.Gaps = s.Metrics.Gaps.add(it.Gaps)
		} else {
			s.Metrics.Unaudited++
			s.Unaudited = append(s.Unaudited, AuditOffer{IntentID: it.ID, Command: "abcd intent audit " + it.ID})
		}
	}
	for _, it := range corpus.Intents {
		if it.Bucket == intent.BucketPlanned && it.TargetRelease == tag {
			s.Unshipped = append(s.Unshipped, TargetedIntent{ID: it.ID, Path: filepath.ToSlash(it.Path)})
		}
	}
	sort.SliceStable(s.Unshipped, func(i, j int) bool { return idNum(s.Unshipped[i].ID) < idNum(s.Unshipped[j].ID) })

	if s.Changelog, err = changelogSection(root, tag); err != nil {
		return Seed{}, err
	}
	s.Metrics.TagDate = tagDate(root, tag)
	if prev != "" {
		s.Metrics.PreviousTag = prev
		s.Metrics.PreviousTagDate = tagDate(root, prev)
	}
	return s, nil
}

// outputPresent reports whether the retrospective for the release exists. A
// store that a symlink or a file occupies is refused rather than read.
func outputPresent(root, rel string) (bool, error) {
	ok, err := fsutil.ProbeRealDirAll(root, path.Dir(rel))
	if err != nil {
		return false, fmt.Errorf("reflect: %w", err)
	}
	if !ok {
		return false, nil
	}
	_, err = os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("reflect: %w", err)
}

// previousTag is the newest release tag below tag, by SemVer order, or "" when
// tag is the first release.
func previousTag(root, tag string) (string, error) {
	cur, err := launch.ParseSemver(strings.TrimPrefix(tag, "v"))
	if err != nil {
		return "", err
	}
	tags, err := launch.GitExistingTags(root)
	if err != nil {
		return "", err
	}
	var best launch.Semver
	found := false
	for _, t := range tags {
		if launch.CoreGreater(cur, t) && (!found || launch.CoreGreater(t, best)) {
			best, found = t, true
		}
	}
	if !found {
		return "", nil
	}
	return best.Tag(), nil
}

// tagDate is the date a tag carries: the tagger's date for an annotated tag,
// the commit's for a lightweight one, as YYYY-MM-DD.
func tagDate(root, tag string) string {
	out, err := gitutil.Run(root, "for-each-ref", "--format=%(creatordate:short)", "refs/tags/"+tag)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// member is one intent the release shipped: its id, where it lives now (empty
// when the store no longer holds it), and the tag-tree path to fall back on.
type member struct {
	id      string
	path    string
	tagPath string
	tag     string
	root    string
}

var shippedFileRe = regexp.MustCompile(`^(itd-[0-9]+)[^/]*\.md$`)

const shippedRelDir = intent.IntentsRelDir + "/" + intent.BucketShipped

// shippedAt lists the intents in shipped/ at ref, id to path.
func shippedAt(root, ref string) (map[string]string, error) {
	out, err := gitutil.Run(root, "ls-tree", "-r", "--name-only", ref, "--", shippedRelDir)
	if err != nil {
		return nil, fmt.Errorf("reflect: listing %s at %s: %w", shippedRelDir, ref, err)
	}
	ids := map[string]string{}
	for _, p := range strings.Split(out, "\n") {
		p = strings.TrimSpace(p)
		if p == "" || path.Dir(p) != shippedRelDir {
			continue
		}
		if m := shippedFileRe.FindStringSubmatch(path.Base(p)); m != nil {
			ids[m[1]] = p
		}
	}
	return ids, nil
}

func shippedBy(root, tag, prev string, corpus intent.Corpus) ([]member, error) {
	atTag, err := shippedAt(root, tag)
	if err != nil {
		return nil, err
	}
	atPrev := map[string]string{}
	if prev != "" {
		if atPrev, err = shippedAt(root, prev); err != nil {
			return nil, err
		}
	}
	current := map[string]intent.Intent{}
	for _, it := range corpus.Intents {
		current[it.ID] = it
	}
	stamp := func(id string) string {
		it, ok := current[id]
		if !ok {
			return ""
		}
		data, err := fsutil.ReadGuarded(filepath.Join(root, it.Path), maxIntentBytes)
		if err != nil {
			return ""
		}
		v, ok := frontmatter.ScalarString(frontmatter.Fields(strings.Split(string(data), "\n"))["shipped_in"].Value)
		if !ok || frontmatter.IsNull(v) {
			return ""
		}
		return strings.TrimSpace(v)
	}

	chosen := map[string]member{}
	for id, p := range atTag {
		if _, before := atPrev[id]; before {
			continue
		}
		if s := stamp(id); s != "" && s != tag {
			continue
		}
		m := member{id: id, tagPath: p, tag: tag, root: root}
		if it, ok := current[id]; ok {
			m.path = filepath.ToSlash(it.Path)
		}
		chosen[id] = m
	}
	for _, it := range corpus.Intents {
		if it.Bucket != intent.BucketShipped {
			continue
		}
		if _, ok := chosen[it.ID]; ok {
			continue
		}
		if stamp(it.ID) == tag {
			chosen[it.ID] = member{id: it.ID, path: filepath.ToSlash(it.Path), root: root}
		}
	}
	out := make([]member, 0, len(chosen))
	for _, m := range chosen {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return idNum(out[i].id) < idNum(out[j].id) })
	return out, nil
}

// maxIntentBytes caps a record read, as the intent store caps its own.
const maxIntentBytes = 256 * 1024

// readSeedIntent reads one member's record: from the store where it lives now,
// else from the tag's tree. A record that cannot be read is still named, with
// no audit notes, so the interview can say so rather than lose it.
func readSeedIntent(m member) SeedIntent {
	it := SeedIntent{ID: m.id, Path: m.path}
	var text string
	if m.path != "" {
		if data, err := fsutil.ReadGuarded(filepath.Join(m.root, filepath.FromSlash(m.path)), maxIntentBytes); err == nil {
			text = string(data)
		}
	}
	if text == "" && m.tagPath != "" {
		if blob, err := gitutil.RunLimited(m.root, maxIntentBytes, "cat-file", "blob", m.tag+":"+m.tagPath); err == nil {
			text = blob
		}
		if it.Path == "" {
			it.Path = m.tagPath
		}
	}
	lines := strings.Split(text, "\n")
	fields := frontmatter.Fields(lines)
	if v, ok := frontmatter.ScalarString(fields["impact"].Value); ok && !frontmatter.IsNull(v) {
		it.Impact = strings.TrimSpace(v)
	}
	it.Title = titleOf(text, m.id)
	readAuditNotes(text, &it)
	return it
}

// titleOf is the record's first `# ` heading after its frontmatter, else its id.
func titleOf(text, id string) string {
	_, body := frontmatter.Split(text)
	for _, ln := range strings.Split(body, "\n") {
		if strings.HasPrefix(ln, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(ln, "# "))
		}
	}
	return id
}

func idNum(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "itd-"))
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return n
}

// changelogSection reads the dated section CHANGELOG.md carries for tag, by the
// same heading predicate the cut and the tagger use.
func changelogSection(root, tag string) (ChangelogSection, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(root, "CHANGELOG.md"), changelog.MaxChangelogBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return ChangelogSection{}, nil
		}
		return ChangelogSection{}, fmt.Errorf("reflect: reading CHANGELOG.md: %w", err)
	}
	version := strings.TrimPrefix(tag, "v")
	lines := strings.Split(string(data), "\n")
	start := -1
	for i, ln := range lines {
		ln = strings.TrimRight(ln, "\r")
		if !changelog.IsDatedHeading(ln) {
			continue
		}
		if start >= 0 {
			return section(lines, start, i), nil
		}
		if strings.HasPrefix(ln, "## ["+version+"] - ") || strings.HasPrefix(ln, "## [v"+version+"] - ") {
			start = i
		}
	}
	if start < 0 {
		return ChangelogSection{}, nil
	}
	return section(lines, start, len(lines)), nil
}

func section(lines []string, start, end int) ChangelogSection {
	heading := strings.TrimRight(lines[start], "\r")
	body := strings.TrimSpace(strings.Join(lines[start+1:end], "\n"))
	return ChangelogSection{Found: true, Heading: heading, Anchor: anchorOf(strings.TrimPrefix(heading, "## ")), Body: body}
}

// anchorOf is the fragment a rendered page gives a heading: lower-cased, every
// character but a letter, a digit, a space, a hyphen or an underscore dropped,
// and each space a hyphen.
func anchorOf(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

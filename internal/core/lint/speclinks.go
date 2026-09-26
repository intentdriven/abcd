package lint

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// IntentLink is one intent record as the link index sees it: which bucket holds
// it (the lifecycle state, because directory-as-truth) and which spec it names.
type IntentLink struct {
	// ID is the intent id (itd-N), taken from the frontmatter when well-formed
	// and from the filename otherwise.
	ID string
	// Bucket is the lifecycle directory holding it (drafts, planned, shipped,
	// disciplines, superseded), or "" for a file outside a known bucket.
	Bucket string
	// Path is the record's repo-relative path.
	Path string
	// SpecID is the raw spec_id frontmatter value (which may be a YAML null).
	SpecID string
}

// SpecLink is one spec record as the link index sees it. The unexported fields
// carry what the spec-lifecycle lint needs from the same read — the frontmatter
// with its line numbers, and whether the file is content-exempt — so the tree is
// walked once for both consumers.
type SpecLink struct {
	// ID is the spec id (spc-N) from frontmatter, "" when malformed or absent.
	ID string
	// Bucket is the lifecycle directory holding it: "open" or "closed".
	Bucket string
	// Path is the spec's repo-relative path.
	Path string
	// IntentID is the raw intent frontmatter value — the back-link.
	IntentID string
	// Intents is the raw `intents:` list a bundle's shared spec carries: every
	// member it realises, the first being IntentID (itd-34). Empty on an
	// ordinary spec.
	Intents []string

	fields map[string]fmField
	exempt bool
	// preamble is the 1-based line the leading `---` sits on when something
	// precedes it, 0 otherwise. spec.Load reads the delimiter at line 0 only, so
	// a preamble hides the whole block from the loader (iss-2608270926031827).
	preamble int
}

// SpecLinkIndex is ONE traversal of the intent buckets and the spec store,
// shared by every consumer that must reason about the intent↔spec link.
//
// It exists because two consumers ask opposite questions of the same two trees.
// spec_lifecycle walks the specs and asks whether each names an intent that
// agrees with it; the release cut walks the intents and asks whether any still
// sitting in planned/ has no OPEN spec left — a merged feature whose record never
// moved, which is invisible to the shipped/-tree diff and would silently
// under-bump the release. Two walks would be two answers about one tree, so
// there is one scan and two readings of it — and since an intent owns one or
// more specs (adr-2609151513118583), both readings derive that set through
// SpecsForIntent rather than each resolving a scalar of its own.
type SpecLinkIndex struct {
	Intents []IntentLink
	Specs   []SpecLink
}

// IntentSpecID maps each known intent id to its raw spec_id value.
func (x SpecLinkIndex) IntentSpecID() map[string]string {
	out := make(map[string]string, len(x.Intents))
	for _, i := range x.Intents {
		out[i.ID] = i.SpecID
	}
	return out
}

// KnownIntents is the set of intent ids present anywhere in the corpus.
func (x SpecLinkIndex) KnownIntents() map[string]bool {
	out := make(map[string]bool, len(x.Intents))
	for _, i := range x.Intents {
		out[i.ID] = true
	}
	return out
}

// SpecsForIntent returns every spec whose back-link names the given intent.
//
// An intent owns one or more specs (adr-2609151513118583, invariant 17), and the
// spec's own `intent:` field is the source of truth for that link: nothing on
// the intent side carries a set. Every consumer of this index that must reason
// about the whole set — the bidirectional check, and the release cut's
// stale-intent question — derives it here, so there is one answer to "which
// specs realise this intent".
//
// Matching is canonical on both sides, so a zero-padded spelling on either side
// resolves. It goes through recordid.SameID — the SAME primitive the spec store
// compares with — rather than a private canonicaliser, because this index and
// that store answer one question ("which specs realise this intent") for two
// callers, and two implementations of one question are two answers waiting to
// diverge. SameID also fails closed where the private form did not: a value that
// is not a record id at all (`intent: null`) matches nothing, so two unresolvable
// back-links no longer group into one pseudo-intent.
func (x SpecLinkIndex) SpecsForIntent(intentID string) []SpecLink {
	var out []SpecLink
	for _, s := range x.Specs {
		if s.names(intentID) {
			out = append(out, s)
		}
	}
	return out
}

// names reports whether the spec realises intentID through its `intent:`
// back-link or its bundle `intents:` list — the same question spec.Spec.Names
// answers for the store, through the same primitive.
func (s SpecLink) names(intentID string) bool {
	if recordid.SameID(s.IntentID, intentID) {
		return true
	}
	for _, m := range s.Intents {
		if recordid.SameID(m, intentID) {
			return true
		}
	}
	return false
}

// SpecBucket resolves a spec_id value to the lifecycle bucket holding that spec.
//
// Matching is on the spec NUMBER, not the literal string, because a spec_id is
// written both bare (`spc-9`) and with its slug (`spc-9-thing`) across the
// record; comparing literals would silently fail to resolve half the corpus and
// make a fail-closed gate fail open.
func (x SpecLinkIndex) SpecBucket(specID string) (string, bool) {
	n := specNum(specID)
	if n < 0 {
		return "", false
	}
	for _, s := range x.Specs {
		if specNum(s.ID) == n {
			return s.Bucket, true
		}
	}
	return "", false
}

// ScanSpecLinks reads the intent buckets and the spec store once, both relative
// to repoRoot. A missing tree contributes nothing and is not an error, mirroring
// the rest of the record lint: an unpopulated repository is a state, not a fault.
// A tree that is present but cannot be read IS a fault and is returned: both
// halves fail closed, so no consumer ever reads an index that says "nothing
// here" about records it merely failed to open.
//
// top supplies the content exemptions. They are recorded per spec rather than
// applied here, because they exempt a file from the lint's CONTENT checks — they
// do not mean the record stopped existing, which is the only thing the release
// cut asks about.
func ScanSpecLinks(repoRoot, intentsDir, specsDir string, top Config) (SpecLinkIndex, error) {
	var idx SpecLinkIndex

	intentsRoot := filepath.Join(repoRoot, filepath.FromSlash(intentsDir))
	if err := filepath.WalkDir(intentsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// A missing tree is the one soft case (and the guard that keeps the
			// nil DirEntry WalkDir hands the failed root out of the checks below);
			// every other walk error is a tree that could not be read, which is a
			// fault, not an absence — fail closed exactly as the spec half does.
			if path == intentsRoot && os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !intentFileRe.MatchString(d.Name()) {
			return nil
		}
		content, rerr := readRepoAbs(repoRoot, path, maxRepoFileBytes)
		if rerr != nil {
			return rerr
		}
		fields := frontmatterFields(strings.Split(string(content), "\n"))
		id := fields["id"].value
		if !recordid.ValidIntentID(id) {
			id = intentIDRe.FindString(d.Name())
		}
		if id == "" {
			return nil
		}
		// Canonicalise so the KnownIntents/IntentSpecID maps key on the canonical
		// spelling: a zero-padded intent (itd-047) and its canonical twin are one
		// handle everywhere else in the record (iss-392's canonRecordID keyspace).
		id = canonRecordID(id)
		bucket := filepath.Base(filepath.Dir(path))
		if !intentBuckets[bucket] {
			bucket = ""
		}
		idx.Intents = append(idx.Intents, IntentLink{
			ID:     id,
			Bucket: bucket,
			Path:   repoRel(repoRoot, path),
			SpecID: fields["spec_id"].value,
		})
		return nil
	}); err != nil {
		return SpecLinkIndex{}, err
	}

	specsRoot := filepath.Join(repoRoot, filepath.FromSlash(specsDir))
	for _, bucket := range specBucketNames {
		entries, err := os.ReadDir(filepath.Join(specsRoot, bucket))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return SpecLinkIndex{}, err
		}
		for _, e := range entries {
			if e.IsDir() || !specFileRe.MatchString(e.Name()) {
				continue
			}
			fileAbs := filepath.Join(specsRoot, bucket, e.Name())
			content, err := readRepoAbs(repoRoot, fileAbs, maxRepoFileBytes)
			if err != nil {
				return SpecLinkIndex{}, err
			}
			rel := repoRel(repoRoot, fileAbs)
			lines := strings.Split(string(content), "\n")
			fields := frontmatterFields(lines)
			var members []string
			if v := fields["intents"].value; !isNull(v) {
				members = frontmatter.StringList(v)
			}
			idx.Specs = append(idx.Specs, SpecLink{
				Intents:  members,
				ID:       fields["id"].value,
				Bucket:   bucket,
				Path:     rel,
				IntentID: fields["intent"].value,
				fields:   fields,
				exempt:   contentExempt(rel, fields, top),
				preamble: preambleLine(lines),
			})
		}
	}
	return idx, nil
}

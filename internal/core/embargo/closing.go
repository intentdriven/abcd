package embargo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// DraftDirRel is where triage keeps each closing draft inside the store,
// slash-separated: the local tier, beside the state, one file per advisory.
const DraftDirRel = ".abcd/.work.local/embargo/close"

// maxDraftBytes bounds a draft read back for closing.
const maxDraftBytes = 64 << 10

// publicTrackPrefix begins the reason the state keeps for a public-track
// verdict, ahead of the person's own words.
const publicTrackPrefix = "moved to the public track: "

// closingDraft composes the explanation a verdict that ends the private track
// posts to the reporter, for the person to read and approve. It names neither
// the advisory nor who decided: the advisory is where it is posted, and the
// decider is the store's private record. The "not abcd's" text states both
// halves of abcd-fixes-what-it-adds in the person's own words: why the problem
// would exist without abcd, and why abcd cannot help without changing how the
// repository works. The public-track text says the work moved to the public
// track and why; it names no public issue, since the store keeps none (the
// person files it with the capture command triage prints, in their own
// words).
func closingDraft(verdict, reason, existsWithout, cannotHelp string) string {
	switch verdict {
	case VerdictNotAbcds:
		return "Thank you for reporting this privately.\n\n" +
			"We looked into it and judged that it is not a vulnerability abcd adds, so we are closing this advisory " +
			"without a change to abcd.\n\n" +
			"Why the problem would exist without abcd:\n\n" + existsWithout + "\n\n" +
			"Why abcd cannot help without changing how the repository works:\n\n" + cannotHelp + "\n\n" +
			"abcd fixes what it adds: a weakness abcd brings into a repository is abcd's to fix, and a fix never " +
			"changes how the repository works on its own. A problem that would exist without abcd, and that abcd " +
			"could prevent only by changing how the repository works, is not abcd's to fix.\n"
	case VerdictNotAVulnerability:
		return "Thank you for reporting this privately.\n\n" +
			"We looked into it and judged that it is not a security vulnerability, so we are closing this advisory.\n\n" +
			"Why:\n\n" + reason + "\n"
	case VerdictPublicTrack:
		return "Thank you for reporting this privately.\n\n" +
			"We looked into it and judged that it does not need to stay private, so we have moved it to the public track: " +
			"the work is tracked in the open, described in our own words and without the details of your report, " +
			"and we are closing this advisory.\n\n" +
			"Why:\n\n" + reason + "\n"
	}
	return ""
}

// closingText is the text close shows and posts for advisory a: its draft,
// read back through readDraft. A public-track verdict recorded before triage
// drafted one for it has none, so its text is drafted here from the reason
// the state keeps, the same text triage drafts now; one with no reason kept
// refuses. Either way the caller scans the text before it is shown.
func (s Store) closingText(a Advisory) (string, error) {
	if a.Verdict != nil && *a.Verdict == VerdictPublicTrack {
		if _, err := os.Lstat(s.draftPath(a.GHSAID)); errors.Is(err, fs.ErrNotExist) {
			why := ""
			if a.Reason != nil {
				why = strings.TrimSpace(strings.TrimPrefix(*a.Reason, publicTrackPrefix))
			}
			if why == "" {
				return "", refuse("there is no closing draft for this advisory, and the store keeps no reason to draft one from",
					"record its verdict again with `abcd embargo triage` to draft one")
			}
			return closingDraft(VerdictPublicTrack, why, "", ""), nil
		}
	}
	return s.readDraft(a.GHSAID)
}

// draftPath is the closing draft of one advisory.
func (s Store) draftPath(ghsa string) string {
	return filepath.Join(s.Root, filepath.FromSlash(DraftDirRel), ghsa+".md")
}

// writeDraft saves a closing draft owner-only and atomically.
func (s Store) writeDraft(ghsa, text string) error {
	if err := fsutil.EnsureRealDirAll(s.Root, DraftDirRel, 0o700); err != nil {
		return refuse("the closing draft's folder could not be made as a real directory in the store", "move aside whatever occupies it")
	}
	if err := fsutil.WriteFileAtomic(s.draftPath(ghsa), []byte(text), 0o600); err != nil {
		return refuse("the closing draft could not be written in the store", "check the store's permissions, then run it again")
	}
	return nil
}

// removeDraft drops a draft a later verdict made stale; an absent one is fine.
func (s Store) removeDraft(ghsa string) error {
	if err := os.Remove(s.draftPath(ghsa)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return refuse("the stale closing draft could not be removed from the store", "check the store's permissions, then run it again")
	}
	return nil
}

// readDraft reads a closing draft back through the same trust checks as the
// state: a regular file this account alone can write, within its cap.
func (s Store) readDraft(ghsa string) (string, error) {
	p := s.draftPath(ghsa)
	fi, err := os.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", refuse("there is no closing draft for this advisory", "record its verdict again with `abcd embargo triage` to draft one")
	}
	if err != nil || !fi.Mode().IsRegular() || fsutil.CallersAlone(p, fi) != nil {
		return "", refuse("the closing draft is not a regular file this account alone can write", "record its verdict again with `abcd embargo triage` to redraft it")
	}
	if err := untracked(s.Root, DraftDirRel+"/"+ghsa+".md", "the closing draft"); err != nil {
		return "", err
	}
	data, err := fsutil.ReadGuarded(p, maxDraftBytes)
	if err != nil {
		return "", refuse("the closing draft could not be read within its size limit", "record its verdict again with `abcd embargo triage` to redraft it")
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", refuse("the closing draft is empty", "record its verdict again with `abcd embargo triage` to redraft it")
	}
	return string(data), nil
}

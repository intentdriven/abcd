package lint

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// reviewsSubdir is the receipts directory the semantic-gate receipts live under,
// as immediate children keyed by the content commit they gate
// (.abcd/work/reviews/<content-sha>/<gate>.json).
const reviewsSubdir = ".abcd/work/reviews"

// DeriveReleaseContentSha resolves the reviewed CONTENT commit a release's
// semantic gate must be armed against, by reading the RECEIPTS DIRECTORY of the
// released tree rather than by walking commit ancestry (HEAD^2^ / HEAD^).
//
// The old HEAD-ancestry derivation is unsafe under a batched merge queue
// (iss-355): main advances one push per batch, so `github.sha` is the batch TIP,
// and `HEAD^2^` of that tip resolves the LAST-merged PR's pre-merge commit — an
// unrelated PR whenever the release roll is not the final entry in its batch. The
// `git merge-base --is-ancestor` guard passes for that unrelated commit (it is on
// the released lineage), and the run wedges after the immutable tag exactly as
// iss-326 records, by a route the pre-merge check cannot close because batch
// composition is decided by the queue.
//
// The receipts name the commit they gate: a semantic-pass receipt lives at
// .abcd/work/reviews/<content-sha>/<gate>.json, so <content-sha> is carried by
// the directory name, present in the released tree, and independent of where the
// release roll sat in its batch. This derivation therefore enumerates the receipt
// subdirectories of the released tree, keeps those whose name is a commit that is
// an ancestor of `released`, and returns the NEAREST such ancestor — the content
// commit of THIS release, since every earlier release's receipts sit further back
// on the shared history.
//
// It fails closed: an unreadable listing, no receipt directory on the released
// lineage, or two equidistant candidates all return an error rather than a guess,
// so an armed gate never resolves to the wrong commit or to nothing.
func DeriveReleaseContentSha(root, released string) (string, error) {
	if !receiptShaRe.MatchString(released) {
		return "", fmt.Errorf("release-gate: released commit %q is not a resolved commit sha", released)
	}
	names, err := receiptDirNames(root, released)
	if err != nil {
		return "", err
	}

	// Keep only receipt directories whose name is a real commit on the released
	// lineage. A name that is not a commit at all (a rewritten or absent sha) or
	// not an ancestor (a receipt off this line) is skipped, never fatal — the
	// derivation must survive a stray directory, and only a genuine git failure
	// aborts it.
	var candidates []string
	for _, name := range names {
		if !receiptShaRe.MatchString(name) {
			continue
		}
		typ, err := gitutil.Run(root, "cat-file", "-t", name)
		if err != nil || typ != "commit" {
			continue
		}
		anc, err := gitutil.IsAncestor(root, name, released)
		if err != nil {
			return "", fmt.Errorf("release-gate: ancestry check for receipt commit %s: %w", name, err)
		}
		if anc {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("release-gate: no receipts directory under %s names a commit on the released lineage; the semantic gate has nothing to arm (fail-closed)", reviewsSubdir)
	}

	// Nearest ancestor wins: the fewest commits between a candidate and the
	// released commit is this release's content commit; every earlier release's
	// receipts sit further back on the shared first-parent history. Two candidates
	// equidistant from `released` make the content commit ambiguous — fail closed
	// rather than pick one.
	best, bestCount, tie := "", -1, false
	for _, c := range candidates {
		out, err := gitutil.Run(root, "rev-list", "--count", c+".."+released)
		if err != nil {
			return "", fmt.Errorf("release-gate: distance from receipt commit %s to released: %w", c, err)
		}
		n, err := strconv.Atoi(strings.TrimSpace(out))
		if err != nil {
			return "", fmt.Errorf("release-gate: parsing distance from receipt commit %s: %w", c, err)
		}
		switch {
		case bestCount == -1 || n < bestCount:
			best, bestCount, tie = c, n, false
		case n == bestCount:
			tie = true
		}
	}
	if tie {
		return "", fmt.Errorf("release-gate: two receipts directories are equidistant from the released commit; the content commit is ambiguous (fail-closed)")
	}

	// Nearest is not enough: the nearest receipts directory can be an EARLIER
	// release's, when this release recorded none of its own, and its PROMOTE
	// receipts would then admit a release nobody reviewed (iss-2609251755386183).
	// The content commit a release's receipts name is the commit that rolled the
	// CHANGELOG to this release's version, so it carries the released tree's own
	// newest dated version. A candidate carrying any other version is not this
	// release's content commit, and the derivation fails closed on it.
	want, err := releaseVersionAt(root, released)
	if err != nil {
		return "", err
	}
	got, err := releaseVersionAt(root, best)
	if err != nil {
		return "", err
	}
	if got != want {
		return "", fmt.Errorf("release-gate: the nearest receipts directory names %s, whose newest CHANGELOG version is %s, "+
			"not this release's %s; no receipts directory names this release's content commit (fail-closed)",
			best, versionOrNone(got), versionOrNone(want))
	}

	// Return the full 40/64-hex sha so the armed gate's receiptShaRe check and the
	// receipt's subject digest compare against a canonical form, never an
	// abbreviated directory name.
	full, err := gitutil.Run(root, "rev-parse", "--verify", best+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("release-gate: resolving receipt commit %s: %w", best, err)
	}
	return full, nil
}

// receiptDirNames lists the immediate subdirectory names of the receipts
// directory AS IT EXISTS IN THE RELEASED TREE (git ls-tree), not on the working
// filesystem — the released tree is the authority for which receipts a release
// carries, and reading git keeps the derivation independent of any working-tree
// state. An absent receipts directory yields no names and no error (the caller
// fails closed on the empty candidate set); only a git failure is returned.
func receiptDirNames(root, released string) ([]string, error) {
	out, err := gitutil.Run(root, "ls-tree", released, reviewsSubdir+"/")
	if err != nil {
		return nil, fmt.Errorf("release-gate: listing %s in the released tree: %w", reviewsSubdir, err)
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each ls-tree line is "<mode> <type> <object>\t<path>"; keep tree entries.
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		fields := strings.Fields(line[:tab])
		if len(fields) < 2 || fields[1] != "tree" {
			continue
		}
		names = append(names, filepath.Base(line[tab+1:]))
	}
	return names, nil
}

// releaseVersionAt reads the newest dated CHANGELOG version out of rev's tree
// — the version auto-release tags when rev is released. "" means rev carries no
// CHANGELOG.md or no dated heading in it; an unreadable blob or a malformed
// heading is an error, never a silent "".
func releaseVersionAt(root, rev string) (string, error) {
	if _, err := gitutil.Run(root, "cat-file", "-e", rev+":CHANGELOG.md"); err != nil {
		return "", nil
	}
	blob, err := gitutil.Run(root, "cat-file", "blob", rev+":CHANGELOG.md")
	if err != nil {
		return "", fmt.Errorf("release-gate: reading CHANGELOG.md at %s: %w", rev, err)
	}
	v, found, err := changelog.LatestVersionIn([]byte(blob))
	if err != nil {
		return "", fmt.Errorf("release-gate: CHANGELOG.md at %s: %w", rev, err)
	}
	if !found {
		return "", nil
	}
	return v.String(), nil
}

// versionOrNone renders a release version for a refusal, naming its absence.
func versionOrNone(v string) string {
	if v == "" {
		return "none"
	}
	return v
}

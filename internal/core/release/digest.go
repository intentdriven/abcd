package release

// digest.go — the cut's fourth write: the release digest (spc-2610170635097509).
//
// A repository that declares `release.digest` in its .abcd/config.json gets
// each cut's release appended to that file: the cut's next tag, the tag it was
// cut from, and every validated changelog line with the records it cites and
// their impact (internal/core/releasedigest). abcd declares the file its binary
// embeds, so the binary built from a release's tag carries every release up to
// its own. A repository that declares nothing gets no digest.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/releasedigest"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// DigestKey is the configuration key that declares the digest's path.
const DigestKey = "release.digest"

// maxDigestPathBytes bounds the declared path; a real one is a few dozen
// bytes.
const maxDigestPathBytes = 512

// declaredDigest is the repository-relative path root's .abcd/config.json
// declares under release.digest, or "" when it declares none. Only the
// repository layer is read: the digest is part of the repository's release,
// so a machine's own config never decides where a cut writes. The release
// namespace is claimed, so a misspelt key is refused naming the file.
func declaredDigest(root string) (string, error) {
	s, err := layered.Load(layered.Config, layered.Roots{Repo: root, HomeRefused: true})
	if err != nil {
		return "", err
	}
	if err := s.Claim("release", "digest"); err != nil {
		return "", err
	}
	v, err := layered.Get(s, DigestKey, "", checkDigestPath)
	if err != nil {
		return "", err
	}
	if v.Layer != layered.Repo {
		return "", nil
	}
	return v.V, nil
}

// checkDigestPath refuses a declared path that is not a clean,
// slash-separated, relative path to a .json file inside the tree and outside
// .abcd/ and .git/. .abcd/ never reaches a released binary (AGENTS.md,
// Boundaries), so a digest there could never be embedded; .git/ is not the
// tree. The first segment is compared case-insensitively, because on a
// case-insensitive filesystem .ABCD/ is .abcd/.
func checkDigestPath(p string) error {
	switch {
	case p == "":
		return errors.New("an empty path; remove the key to write no digest")
	case len(p) > maxDigestPathBytes:
		return fmt.Errorf("a path longer than %d bytes", maxDigestPathBytes)
	case strings.Contains(p, `\`):
		return errors.New("a path with a backslash; write it with forward slashes, relative to the repository root")
	case !fs.ValidPath(p):
		return errors.New("not a clean relative path inside the repository (no leading /, no . or .. segment, no empty segment)")
	}
	first, _, _ := strings.Cut(p, "/")
	switch {
	case strings.EqualFold(first, ".abcd"):
		return errors.New("a path under .abcd/, which never reaches a released binary, so the digest could not be embedded")
	case strings.EqualFold(first, ".git"):
		return errors.New("a path under .git/, which is not the working tree")
	case path.Ext(p) != ".json":
		return errors.New("a path that does not end in .json")
	}
	return nil
}

// planDigest resolves the declared digest and reads the file the cut appends
// to, before anything is written: a declaration that does not resolve, a
// directory that is not in the tree, or a present file that cannot be read is
// a stop. It records the undo; the bytes are composed by composeDigest once
// the payload is accepted.
func planDigest(root string, plan *cutPlan) error {
	rel, err := declaredDigest(root)
	if err != nil {
		return fmt.Errorf("reading %s: %w", DigestKey, err)
	}
	if rel == "" {
		return nil
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if dir := path.Dir(rel); dir != "." {
		fi, err := r.Stat(dir)
		if err != nil || !fi.IsDir() {
			return fmt.Errorf("%s declares %s, but its directory %s is not a directory inside the repository; "+
				"create it, or declare a path whose directory exists", DigestKey, rel, dir)
		}
	}
	existing, err := fsutil.ReadGuardedInRoot(r, rel, releasedigest.MaxBytes)
	switch {
	case err == nil:
		if _, err := releasedigest.Parse(existing); err != nil {
			return fmt.Errorf("the release digest %s cannot take this cut: %w; restore the file the last cut wrote", rel, err)
		}
		plan.undo.digestExisted = true
		plan.undo.digestBefore = existing
	case errors.Is(err, os.ErrNotExist):
	default:
		return fmt.Errorf("reading the release digest %s: %w", rel, err)
	}
	plan.digestRel = rel
	plan.undo.digestRel = rel
	return nil
}

// composeDigest appends the cut's release to the digest planDigest read: the
// next tag, the base tag, and each accepted line in the CHANGELOG's order with
// every record it cites and that record's impact.
func composeDigest(plan *cutPlan, cut Cut, entries []ChangelogEntry) error {
	if plan.digestRel == "" {
		return nil
	}
	impact := map[string]releasedigest.Impact{}
	for _, e := range append(append([]Entry{}, cut.Added...), cut.Removed...) {
		impact[e.ID] = releasedigest.Impact(e.Impact)
	}
	rel := releasedigest.Release{Tag: cut.NextTag, BaseTag: cut.BaseTag}
	for _, e := range orderedEntries(entries) {
		recs := make([]releasedigest.Record, 0, len(e.Records))
		for _, id := range e.Records {
			recs = append(recs, releasedigest.Record{ID: id, Impact: impact[id]})
		}
		rel.Entries = append(rel.Entries, releasedigest.Entry{Line: e.Text, Records: recs})
	}
	out, err := releasedigest.Append(plan.undo.digestBefore, rel)
	if err != nil {
		return fmt.Errorf("the release digest %s cannot take this cut: %w", plan.digestRel, err)
	}
	plan.digest = out
	return nil
}

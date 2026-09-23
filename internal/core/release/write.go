package release

// write.go — the cut's writes, their order, and their undo.
//
// A feature cut makes up to three writes, and they land together or not at all:
//
//  1. the ARCHIVE: the outgoing RELEASE.md's bytes, created under
//     .abcd/development/releases/<its version>.md, never overwriting;
//  2. the PAGE: the new RELEASE.md, replaced atomically (a copy then a replace,
//     so there is no instant with no RELEASE.md);
//  3. the CHANGELOG heading, written LAST, so the file the tagging workflow reads
//     lands only after the page did.
//
// Everything that can be checked is checked before step 1 (the payload, the
// outgoing page's heading, the archive target). A failure at a later step undoes
// the earlier ones in reverse, and the report says whether that worked.

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// fileOps is the seam the writes go through, so a test can observe their order
// and fail any one of them. Paths are repo-relative and slash-separated.
type fileOps interface {
	createExclusive(rel string, data []byte) error
	replace(rel string, data []byte) error
	remove(rel string) error
}

// osOps performs the writes inside the repository root. The archive is created
// through an os.Root, so a symlinked ancestor cannot carry it outside the tree.
type osOps struct{ root string }

func (o osOps) createExclusive(rel string, data []byte) error {
	r, err := os.OpenRoot(o.root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := r.MkdirAll(path.Dir(rel), 0o755); err != nil {
		return err
	}
	return fsutil.CreateExclusiveIn(r, rel, data, 0o644)
}

func (o osOps) replace(rel string, data []byte) error {
	return fsutil.WriteFileAtomicPreserveMode(filepath.Join(o.root, filepath.FromSlash(rel)), data)
}

func (o osOps) remove(rel string) error {
	return os.Remove(filepath.Join(o.root, filepath.FromSlash(rel)))
}

// UndoPlan is what it takes to put the tree back as it was before a cut wrote.
// The ship verb applies it when a step AFTER the ingest refuses (the payload
// render), so a refused ship leaves no release record behind.
type UndoPlan struct {
	changelogBefore []byte
	pageBefore      []byte
	pageExisted     bool
	pageWritten     bool
	archived        string
	archiveDirMade  bool
	changelogDone   bool
}

// Apply undoes every write the cut made, in reverse, and returns a description
// of each undo that failed (empty when the tree is restored).
func (u UndoPlan) Apply(root string) []string {
	return u.apply(osOps{root: root}, root)
}

func (u UndoPlan) apply(ops fileOps, root string) []string {
	var failures []string
	if u.changelogDone {
		if err := ops.replace(changelogFile, u.changelogBefore); err != nil {
			failures = append(failures, changelogFile+": "+err.Error())
		}
	}
	if u.pageWritten {
		var err error
		if u.pageExisted {
			err = ops.replace(PageFile, u.pageBefore)
		} else {
			err = ops.remove(PageFile)
		}
		if err != nil {
			failures = append(failures, PageFile+": "+err.Error())
		}
	}
	if u.archived != "" {
		if err := ops.remove(u.archived); err != nil {
			failures = append(failures, u.archived+": "+err.Error())
		} else if u.archiveDirMade {
			// Only a directory this cut created, and only when empty: os.Remove
			// refuses a non-empty directory, which is the guard.
			_ = os.Remove(filepath.Join(root, filepath.FromSlash(path.Dir(u.archived))))
		}
	}
	return failures
}

// cutPlan is the validated set of writes, built before any of them runs.
type cutPlan struct {
	changelog []byte
	page      []byte // nil when no page is written
	undo      UndoPlan
}

// planPage reads the outgoing page and decides the archive move. It is called
// only when a new page will be written; every fault here is structural (a stop,
// not a recompose).
func planPage(root string, plan *cutPlan) error {
	cur := filepath.Join(root, PageFile)
	data, err := fsutil.ReadGuarded(cur, maxPageBytes)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		// A first cut: nothing to archive, and rolling back removes the page.
		return nil
	default:
		return fmt.Errorf("reading the outgoing %s: %w", PageFile, err)
	}
	version, _, ok := parsePageHeading(string(data))
	if !ok {
		return fmt.Errorf("the outgoing %s does not open with a `# Release X.Y.Z (YYYY-MM-DD)` heading, so it cannot "+
			"be archived under the release it describes; restore the heading the cut wrote, or move the page to "+
			"%s/<its version>.md by hand, and cut again", PageFile, ArchiveDir)
	}
	archive := ArchiveDir + "/" + version + ".md"
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(archive))); err == nil {
		return fmt.Errorf("the archive page %s already exists; the outgoing %s names release %s, which is already "+
			"archived, and the cut never overwrites an archived page", archive, PageFile, version)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking the archive page %s: %w", archive, err)
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(ArchiveDir))); errors.Is(err, os.ErrNotExist) {
		plan.undo.archiveDirMade = true
	}
	plan.undo.pageBefore = data
	plan.undo.pageExisted = true
	plan.undo.archived = archive
	return nil
}

// execute performs the plan's writes in order and rolls back on failure. It
// returns the undo that reverses a completed plan.
func execute(ops fileOps, root string, plan cutPlan) (UndoPlan, error) {
	done := plan.undo
	done.archived = ""
	done.pageWritten = false
	done.changelogDone = false

	fail := func(step string, err error) (UndoPlan, error) {
		msg := fmt.Sprintf("writing %s failed: %v", step, err)
		if failures := done.apply(ops, root); len(failures) > 0 {
			return UndoPlan{}, fmt.Errorf("%s\n  THE ROLLBACK FAILED — recover by hand: %s", msg, strings.Join(failures, "; "))
		}
		return UndoPlan{}, fmt.Errorf("%s\n  the steps already taken were rolled back; the tree is as it was", msg)
	}

	if plan.page != nil && plan.undo.archived != "" {
		if err := ops.createExclusive(plan.undo.archived, plan.undo.pageBefore); err != nil {
			// The create may have made the directory before failing.
			if plan.undo.archiveDirMade {
				_ = os.Remove(filepath.Join(root, filepath.FromSlash(ArchiveDir)))
			}
			return fail(plan.undo.archived, err)
		}
		done.archived = plan.undo.archived
	}
	if plan.page != nil {
		if err := ops.replace(PageFile, plan.page); err != nil {
			return fail(PageFile, err)
		}
		done.pageWritten = true
	}
	if err := ops.replace(changelogFile, plan.changelog); err != nil {
		return fail(changelogFile, err)
	}
	done.changelogDone = true
	return done, nil
}

// noPageReason is the report of a fixes-only cut.
func noPageReason(root string) string {
	const lead = "No release page written: no user-facing intent shipped in this cut; "
	data, err := fsutil.ReadGuarded(filepath.Join(root, PageFile), maxPageBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return lead + "no " + PageFile + " exists yet"
		}
		return lead + PageFile + " stays as it is"
	}
	if version, _, ok := parsePageHeading(string(data)); ok {
		return lead + PageFile + " stays on " + version
	}
	return lead + PageFile + " stays as it is"
}

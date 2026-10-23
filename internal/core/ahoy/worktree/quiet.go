package worktree

// quiet.go is the quiet rule (spc-2609301811532881, step 5; decision 3): an
// unmerged worktree in the lane is quiet when its last activity is older than
// the quiet window and no pull request for its branch is open (where a forge
// answered; without one the window alone decides). Prune lists each quiet
// candidate with a dossier and leaves it, unless the person names it with
// --yes; the core never prompts. A named candidate takes prune's own archive,
// refusals and removal, so uncommitted work stays until the person commits or
// discards it.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// OutcomeQuiet is a quiet candidate's outcome: listed with its dossier and
// left in place.
const OutcomeQuiet = "quiet"

// The quiet window's configuration: the layered key, its bundled default,
// and the most days it takes. The ceiling keeps the window's arithmetic far
// from overflow, where a huge window would wrap round and read every
// worktree as quiet.
const (
	QuietDaysKey     = "worktrees.quiet_days"
	DefaultQuietDays = 14
	maxQuietDays     = 36500
)

// The dossier's bounds. A dossier is read by a person deciding whether to
// name a worktree, so it shows the first few of each list, and its reads are
// bounded like every other read of another checkout's history.
const (
	dossierPaths    = 5
	dossierSubjects = 10
	dossierRecords  = 20
	subjectsCap     = 1 << 20
	namesCap        = 1 << 20
	shortstatCap    = 4 << 10
)

// QuietDays reads the quiet window, worktrees.quiet_days, through the layered
// configuration resolver: the repository's .abcd/config.json, then
// ~/.abcd.noindex/config.json, then DefaultQuietDays. It claims the worktrees
// namespace, so a misspelt key, a value below 1 or one that is not a whole
// number is refused naming its file, never replaced by the default.
func QuietDays(r layered.Roots) (int, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return 0, err
	}
	if err := s.Claim("worktrees", "quiet_days"); err != nil {
		return 0, err
	}
	v, err := layered.Get(s, QuietDaysKey, DefaultQuietDays, func(n int) error {
		if n < 1 || n > maxQuietDays {
			return fmt.Errorf("want a whole number of days from 1 to %d", maxQuietDays)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return v.V, nil
}

// Dossier is what prune shows of a quiet candidate, dirty state first, so
// the person can tell what naming it would and would not remove.
type Dossier struct {
	Dirty DirtyState `json:"dirty"`
	// Commits counts the branch's commits not on the default branch, and
	// Subjects are the first of them, newest first.
	Commits  int      `json:"commits"`
	Subjects []string `json:"subjects"`
	// Diff is `git diff --shortstat` from the merge base to the tip.
	Diff string `json:"diff"`
	// Records are the record ids the subjects and changed paths cite.
	Records []string `json:"records"`
	// LastActivity is when the worktree was last active, in UTC.
	LastActivity   string `json:"last_activity"`
	Recommendation string `json:"recommendation"`
}

// DirtyState is a candidate's uncommitted changes: counts, and the first
// paths.
type DirtyState struct {
	Modified  int      `json:"modified"`
	Untracked int      `json:"untracked"`
	Paths     []string `json:"paths"`
}

// quietness is the quiet rule's answer for one worktree.
type quietness struct {
	quiet bool
	// open is a pull request open for the branch.
	open bool
	// last is the last activity; known says it could be read.
	last  time.Time
	known bool
}

// quietFor applies the quiet rule to an unmerged worktree. A last activity
// that cannot be read is not quiet, because a quiet candidate is one a
// person may name for removal.
func quietFor(e entry, opts PruneOptions) quietness {
	if opts.QuietDays <= 0 {
		return quietness{}
	}
	q := quietness{open: e.openPR}
	q.last, q.known = lastActivity(e)
	cutoff := opts.Now().AddDate(0, 0, -opts.QuietDays)
	q.quiet = q.known && !q.open && q.last.Before(cutoff)
	return q
}

// lastActivity is the later of the tip's committer date and the newest
// modification time of the files git status reports changed or untracked.
// The files are read with Lstat, never followed.
func lastActivity(e entry) (time.Time, bool) {
	out, err := gitutil.Run(e.path, "log", "-1", "--format=%ct", "HEAD", "--")
	if err != nil {
		return time.Time{}, false
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	last := time.Unix(secs, 0)
	for _, st := range e.status {
		fi, err := os.Lstat(filepath.Join(e.path, filepath.FromSlash(strings.TrimSuffix(st.Path, "/"))))
		if err != nil {
			// A deleted file has no time of its own; the tree is dirty, so
			// a named candidate holding it is kept regardless.
			continue
		}
		if m := fi.ModTime(); m.After(last) {
			last = m
		}
	}
	return last, true
}

// notQuiet is why a worktree named with --yes that is not a quiet candidate
// this run is kept.
func notQuiet(e entry, q quietness, days int) string {
	const named = "named with --yes, but "
	switch {
	case e.merged.merged:
		return named + "merged: a merged worktree needs no name, and prune alone reclaims it once it is clean"
	case days <= 0:
		return named + "this run judges no worktree quiet"
	case q.open:
		return named + "not quiet: a pull request for its branch is open"
	case !q.known:
		return named + "not quiet: its last activity could not be read"
	}
	return fmt.Sprintf("%snot quiet: active within the last %d days", named, days)
}

// dossierFor is e's dossier. Every string is another checkout's bytes and
// is masked; a read that fails leaves its part empty rather than failing
// the row, since the dossier informs and the refusals protect.
func dossierFor(repoRoot string, e entry, q quietness) *Dossier {
	d := &Dossier{Subjects: []string{}, Records: []string{}, LastActivity: q.last.UTC().Format(time.RFC3339)}
	d.Dirty.Paths = []string{}
	cited := map[string]bool{}
	for _, st := range e.status {
		if st.XY == "??" {
			d.Dirty.Untracked++
		} else {
			d.Dirty.Modified++
		}
		if len(d.Dirty.Paths) < dossierPaths {
			d.Dirty.Paths = append(d.Dirty.Paths, termsafe.Sanitize(st.Path))
		}
		cite(cited, st.Path)
	}
	if def := gitutil.DefaultRef(repoRoot); def != "" {
		branchHistory(e.path, def, d, cited)
	}
	for id := range cited {
		d.Records = append(d.Records, id)
	}
	slices.Sort(d.Records)
	if len(d.Records) > dossierRecords {
		d.Records = d.Records[:dossierRecords]
	}
	d.Recommendation = recommend(e, d)
	return d
}

// branchHistory fills d with the branch's commits not on def, the diff
// summary from the merge base, and the records they cite.
func branchHistory(dir, def string, d *Dossier, cited map[string]bool) {
	if subjects, err := gitutil.RunCappedAsStored(dir, subjectsCap, "log", "--format=%s", def+"..HEAD", "--"); err == nil && subjects != "" {
		lines := strings.Split(subjects, "\n")
		d.Commits = len(lines)
		for i, s := range lines {
			cite(cited, s)
			if i < dossierSubjects {
				d.Subjects = append(d.Subjects, termsafe.Sanitize(s))
			}
		}
	}
	base, err := gitutil.Run(dir, "merge-base", def, "HEAD")
	if err != nil || !gitutil.IsFullSHA(base) {
		return
	}
	if stat, err := asStored(dir, "", shortstatCap, "diff", "--shortstat", "--no-ext-diff", "--no-textconv", base, "HEAD", "--"); err == nil {
		d.Diff = termsafe.Sanitize(strings.TrimSpace(stat))
	}
	if names, err := asStored(dir, "", namesCap, "diff", "--name-only", "--no-ext-diff", base, "HEAD", "--"); err == nil {
		for _, p := range strings.Split(names, "\n") {
			cite(cited, p)
		}
	}
}

// cite adds each record id text carries to cited, in its lower-case
// spelling: recordid.HandleInText's reading of a handle, so a path's slug
// after the number does not hide it.
func cite(cited map[string]bool, text string) {
	for text != "" {
		h, ok := recordid.HandleInText(text)
		if !ok {
			return
		}
		cited[strings.ToLower(h)] = true
		i := strings.Index(text, h)
		if i < 0 {
			return
		}
		text = text[i+len(h):]
	}
}

// recommend is the dossier's next step: uncommitted work first, since a
// named candidate holding it is kept; then a branch with work of its own,
// which a removal leaves standing as a branch; then a worktree with nothing
// to lose.
func recommend(e entry, d *Dossier) string {
	yes := "`abcd ahoy worktree prune --yes " + termsafe.Sanitize(e.name) + "`"
	switch {
	case !e.clean:
		return "commit or discard the changes, then name it with " + yes
	case d.Commits > 0:
		return "open a pull request if the work is wanted; otherwise reclaim with " + yes + " (its branch stays)"
	}
	return "reclaim with " + yes
}

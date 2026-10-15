package loop

// conflict.go is the landing's reading of a pull request the forge cannot merge
// (iss-2610110908443341). The forge computes a pull request's mergeability
// itself and applies none of the merge drivers .gitattributes names, so a lane
// that appended to a `merge=union` file the default branch also appended to
// merges cleanly here and conflicts there; a conflicting pull request never
// enters the merge queue, and an armed one waits forever. The arm and the
// merged check ask the forge (`gh pr view <n> --json mergeable,
// mergeStateStatus`) and refuse a pull request it reports CONFLICTING, naming
// what this checkout can say about the paths: a merge of the default branch
// and the pushed head with every merge driver switched to git's built-in text
// merge, as the forge merges, and which of the paths it conflicts in carry a
// driver here. UNKNOWN is a verdict the forge has not computed yet, never a
// conflict: the merged check keeps waiting on it.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The forge's mergeable verdicts.
const (
	MergeableConflicting = "CONFLICTING"
	MergeableUnknown     = "UNKNOWN"
)

// maxConflictPathsNamed caps the paths a conflict refusal names.
const maxConflictPathsNamed = 10

// mergeability is the forge's verdict on whether a pull request merges.
type mergeability struct {
	Mergeable        string `json:"mergeable"`
	MergeStateStatus string `json:"mergeStateStatus"`
}

// conflicting reports whether the forge reports the pull request conflicting.
func (m mergeability) conflicting() bool { return m.Mergeable == MergeableConflicting }

// note is the verdict in words, for the merged check's wait.
func (m mergeability) note() string {
	if m.Mergeable == MergeableUnknown {
		return "the forge has not yet computed whether it merges"
	}
	return "the forge reports it " + strings.ToLower(m.Mergeable) + ", merge state " + m.MergeStateStatus
}

// prMergeability asks the forge whether the lane's pull request merges.
func prMergeability(c Context, lane Lane) (mergeability, error) {
	n := strconv.Itoa(lane.PR)
	out, err := forge(c, lane, "pr", "view", n, "--json", "mergeable,mergeStateStatus")
	if err != nil {
		return mergeability{}, err
	}
	var m mergeability
	if err := json.Unmarshal([]byte(out), &m); err != nil || m.Mergeable == "" {
		return mergeability{}, refuse(string(StageLand), "", lane.ID, "the forge's mergeability of pull request #"+n+" does not parse",
			"check the forge client (`gh --version`), then run `abcd implement step` again")
	}
	return m, nil
}

// conflictRefusal is the refusal of a pull request the forge reports
// conflicting with the default branch def, whose tracking ref the caller has
// just fetched. It needs a person: the loop pushes nothing to a lane after
// arming and merges nothing itself, so it is a refusal (exit 2), never the
// merge's ordinary wait (contention, exit 3), which clears on its own.
func conflictRefusal(c Context, lane Lane, def, tracking string, m mergeability) *Refusal {
	ld := lane.Landing
	n := strconv.Itoa(lane.PR)
	verdict := fmt.Sprintf("(mergeable %s, merge state %s)", m.Mergeable, m.MergeStateStatus)
	var reason, remedy string
	where := "in the lane's worktree " + fsutil.RedactHome(lane.Worktree)
	local := describeConflict(localConflict(c.RepoRoot, tracking, ld.Pushed), def, ld.Pushed)
	switch {
	case ld.Armed:
		rearm := "`gh pr merge " + n + " --auto` with the queue's method"
		if rule, err := readMergeRule(c, lane, def); err == nil && landing.MergeMethods[rule.Method] != "" {
			rearm = "`gh pr merge " + n + " --auto " + landing.MergeMethods[rule.Method] + "`"
		}
		reason = "pull request #" + n + " is armed but conflicts with " + def + " as the forge computes it " + verdict +
			", and the merge queue never takes a conflicting pull request, so it will not merge; " + local
		remedy = "a person disarms it (`gh pr merge " + n + " --disable-auto`), merges " + Remote + "/" + def + " into " + lane.Branch + " " + where +
			" resolving what conflicts, runs `make preflight` there, pushes the branch and re-arms it (" + rearm + "), then runs `abcd implement step` again, which waits for the merge; or, rather than merging, closes it and hands the lane back, so the step is built again. The loop pushes nothing to a lane after arming and merges nothing itself"
	case ld.Merge != "":
		reason = "pull request #" + n + " (" + ld.Merge + ") conflicts with " + def + " as the forge computes it " + verdict +
			", so it cannot merge as it stands; " + local
		remedy = "a person merges " + Remote + "/" + def + " into " + lane.Branch + " " + where +
			" resolving what conflicts, runs `make preflight` there and pushes the branch, then runs `abcd implement step` again; or, rather than merging, closes it and hands the lane back, so the step is built again. The loop merges nothing itself"
	default:
		reason = "pull request #" + n + " conflicts with " + def + " as the forge computes it " + verdict +
			", and the merge queue never takes a conflicting pull request, so the loop does not arm it; " + local
		remedy = "a person merges " + Remote + "/" + def + " into " + lane.Branch + " " + where +
			" resolving what conflicts, runs `make preflight` there and pushes the branch, then runs `abcd implement step` again, which arms it once the forge reports it mergeable; or, rather than merging, closes it and hands the lane back, so the step is built again. The loop merges nothing itself"
	}
	return refuse(string(StageLand), "", lane.ID, reason, remedy)
}

// conflictFinding is what this checkout can say about a conflict the forge
// reports: the paths a driver-free merge conflicts in, the merge driver each
// carries here (none for a path without one), or why it could not be computed.
type conflictFinding struct {
	paths   []string
	drivers map[string]string
	err     error
}

// localConflict merges tracking and head without writing a ref, the index or
// a worktree (`merge-tree --write-tree`), with every merge driver git's
// built-in text merge: those the repository configures
// (gitutil.MergeDriverOverrides) and the built-in union driver, which a
// configured driver of the same name outranks. The merge does not
// renormalise either (merge.renormalize=false), whatever the checkout sets:
// renormalising runs each side of a conflicted path through the content
// filters an attribute in either tree names, and the forge's merge does
// neither. That is the merge the forge computes. The probe is a read-only
// prediction of it, writing nothing and changing nothing about how the
// repository's own merges behave, so the pin is fidelity to the forge rather
// than a setting switched off; syncMerge, a real merge in the lane's worktree,
// keeps the repository's renormalisation. The merge driver each conflicted
// path carries at head is read with check-attr; where that cannot be read the
// paths are still named.
func localConflict(root, tracking, head string) conflictFinding {
	overrides, err := gitutil.MergeDriverOverrides(root)
	if err != nil {
		return conflictFinding{err: err}
	}
	args := append(overrides, "-c", "merge.union.driver="+gitutil.BuiltinMergeDriver(), "-c", "merge.renormalize=false",
		"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", tracking, head)
	out, conflicted, err := gitutil.RunCappedExitOne(root, maxGitOutput, args...)
	if err != nil {
		return conflictFinding{err: err}
	}
	if !conflicted {
		return conflictFinding{}
	}
	// The merged tree's object name first, then one conflicted path per
	// entry. git exits 1 on a refusal too, with no tree to name.
	entries := strings.Split(string(out), "\x00")
	if !gitutil.IsFullSHA(entries[0]) {
		return conflictFinding{err: fmt.Errorf("git merge-tree named no merged tree for %s and %s", tracking, shortSHA(head))}
	}
	var paths []string
	seen := map[string]bool{}
	for i, p := range entries {
		if i == 0 || p == "" || seen[p] {
			continue
		}
		seen[p] = true
		paths = append(paths, p)
	}
	return conflictFinding{paths: paths, drivers: mergeDrivers(root, head, paths)}
}

// mergeDrivers is the merge driver each path carries in head's
// .gitattributes: a value of the merge attribute that names a driver (union,
// or one the repository configures), not git's own text and binary merges.
// A path with none is absent; an unreadable answer is an empty map.
func mergeDrivers(root, head string, paths []string) map[string]string {
	out := map[string]string{}
	if len(paths) == 0 {
		return out
	}
	raw, err := gitutil.RunCappedBytes(root, maxGitOutput, append([]string{"check-attr", "-z", "--source=" + head, "merge", "--"}, paths...)...)
	if err != nil {
		return out
	}
	f := strings.Split(string(raw), "\x00")
	for i := 0; i+2 < len(f); i += 3 {
		switch v := f[i+2]; v {
		case "unspecified", "unset", "set", "text", "binary":
		default:
			out[f[i]] = v
		}
	}
	return out
}

// describeConflict says what this checkout found, never guessing silently:
// the paths that carry a merge driver the forge ignores, the paths that
// conflict here even with the drivers, or that the conflict cannot be
// reproduced here.
func describeConflict(f conflictFinding, def, head string) string {
	if f.err != nil {
		return "this checkout could not merge " + Remote + "/" + def + " and " + shortSHA(head) + " to name the paths: " + fsutil.RedactHome(f.err.Error())
	}
	if len(f.paths) == 0 {
		return "GitHub reports a conflict this checkout cannot reproduce: " + Remote + "/" + def + " and " + shortSHA(head) +
			" merge cleanly here even with every merge driver off, so the forge's verdict may be stale (iss-2608290810033503), and pushing a merge of " +
			def + " into the branch makes it compute again"
	}
	var driven, plain []string
	for _, p := range f.paths {
		if d, ok := f.drivers[p]; ok {
			driven = append(driven, p+" (merge="+d+")")
		} else {
			plain = append(plain, p)
		}
	}
	var parts []string
	if len(driven) > 0 {
		parts = append(parts, "GitHub reports a conflict; locally these files carry merge drivers the forge ignores, and conflict without them: "+namedPaths(driven))
	}
	if len(plain) > 0 {
		parts = append(parts, "a merge of "+Remote+"/"+def+" and "+shortSHA(head)+" conflicts here too, in: "+namedPaths(plain))
	}
	return strings.Join(parts, "; ")
}

// namedPaths lists paths, capped at maxConflictPathsNamed.
func namedPaths(ps []string) string {
	if len(ps) <= maxConflictPathsNamed {
		return strings.Join(ps, ", ")
	}
	return strings.Join(ps[:maxConflictPathsNamed], ", ") + fmt.Sprintf(" and %d more", len(ps)-maxConflictPathsNamed)
}

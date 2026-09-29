package loop

// pickcommit.go is the pick's record-only commit (itd-2609211116005482,
// criterion 2 and 4; spc-2609212015048113 scope 5): after the lane's branch is
// cut and before the implementer starts, the worktree step of a picked run's
// first lane appends the pick's grounds entry to the intent in the lane's own
// worktree, through the intent store's grounds writer and its lock, and
// commits that one file on the lane's branch. The reason so reaches the
// default branch with the work, and stays on the branch as history if the lane
// is discarded. The checkout the run was started from is never written.
//
// The commit is made in the lane's worktree only, on the lane's own branch,
// which the worktree step made or adopted a moment before; it is the one git
// write the loop makes outside `git worktree add`. The environment is the
// isolated one less the global-config neutralisers (gitutil.ScrubbedEnv): the
// commit is authored by the person whose identity git is configured with, as
// every commit of the repository is, and no inherited GIT_DIR, GIT_WORK_TREE
// or injected configuration can redirect it. Hooks and the fsmonitor are off:
// the message and the entry are computed, the lane's pull request runs every
// gate over the commit, and a hook dispatcher is code the loop does not run.
// Every argument is derived: the paths come from the intent store's validated
// ids, after `--`, and the message from the run and intent ids.
//
// The body is idempotent, as every step's is: a branch already carrying the
// pick commit as its first commit past the base is adopted; an entry written
// and not committed (a process killed between the two) is committed; anything
// else on the branch or in the worktree is refused and left where it is.

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxGitOutput caps what the pick commit reads from git.
const maxGitOutput = 1 << 20

// pickSubject is the pick commit's subject line.
func pickSubject(st State) string {
	return "chore(record): " + st.Intent + " picked by run " + st.RunID
}

// pickMessage is the pick commit's whole message. The trailer declares the
// commit human-only in the attribution convention's sense: its text is
// computed by abcd, and no model composed it.
func pickMessage(st State) string {
	n := 0
	if st.Pick != nil {
		n = len(st.Pick.Pick.Candidates)
	}
	return pickSubject(st) + "\n\n" +
		fmt.Sprintf("`abcd build next` picked %s from %d candidate(s); this commit appends the\n", st.Intent, n) +
		"reason to the intent's grounds, marked as the run's. It is record-only: the\n" +
		"implementer's commits follow it.\n\n" +
		"Refs: " + st.Intent + "\n" +
		"Assisted-by: None\n"
}

// pickGit runs one git command in the lane's worktree for the pick commit.
func pickGit(dir string, args ...string) (string, error) {
	full := append([]string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "core.quotePath=false", "-C", dir}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = gitutil.ScrubbedEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 2048 {
			msg = msg[:2048]
		}
		return "", fmt.Errorf("git %s: %v (%s)", args[0], err, msg)
	}
	if stdout.Len() > maxGitOutput {
		return "", fmt.Errorf("git %s wrote more than %d bytes", args[0], maxGitOutput)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// pickCommit makes, or finds, the pick's record-only commit on the lane's
// branch and returns its object name.
func pickCommit(c Context, lane *Lane, wt, branch, base string) (string, error) {
	st := c.State
	branchRef := "refs/heads/" + branch
	revs, err := gitutil.Run(c.RepoRoot, "rev-list", "--reverse", base+".."+branchRef, "--")
	if err != nil {
		return "", fmt.Errorf("listing %s past its base: %v", branch, err)
	}
	if revs != "" {
		first := strings.Fields(revs)[0]
		if ok, why := isPickCommit(c.RepoRoot, st, first, base); !ok {
			return "", refuse(string(StepWorktree), "", lane.ID,
				fmt.Sprintf("%s carries commits past its base, and the first (%s) is not the pick's record commit: %s", branch, shortSHA(first), why),
				"the pick's entry is the lane's first commit; remove the lane's worktree and branch, then run the step again")
		}
		return first, nil
	}

	corpus, err := intent.Load(wt)
	if err != nil {
		return "", fmt.Errorf("reading the intent store in the lane's worktree: %w", err)
	}
	it, ok := corpus.Lookup(st.Intent)
	if !ok || it.Bucket != intent.BucketPlanned {
		return "", refuse(string(StepWorktree), "", lane.ID,
			st.Intent+" is not planned on the default branch the lane was cut from, so the pick's entry has no record to land on",
			"land the intent's planning on the default branch, then run the step again")
	}
	dirty, err := pickGit(wt, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", fmt.Errorf("reading the lane's worktree: %w", err)
	}
	switch {
	case dirty == "":
		g, err := grounds.New(grounds.Pursued, st.Pick.Entry)
		if err != nil {
			return "", refuse(string(StepWorktree), "", lane.ID, "the pick's reason cannot be written as a grounds entry: "+err.Error(),
				"report this: the reason is computed, and a computed reason the writer refuses is a defect")
		}
		if _, err := intent.RecordGrounds(wt, st.Intent, g); err != nil {
			return "", refuse(string(StepWorktree), "", lane.ID, "the pick's entry could not be written: "+fsutil.RedactHome(err.Error()),
				"settle what the reason names, then run the step again")
		}
	case dirty == "M "+it.Path || dirty == "M  "+it.Path:
		// Written before a kill and not committed: the entry must be ours.
		if !carriesPickEntry(filepath.Join(wt, filepath.FromSlash(it.Path)), st) {
			return "", refuse(string(StepWorktree), "", lane.ID,
				it.Path+" is changed in the lane's worktree, and its last grounds entry is not this run's pick",
				"the loop never commits what it did not write; restore the file (`git restore`), then run the step again")
		}
	default:
		return "", refuse(string(StepWorktree), "", lane.ID,
			"the lane's worktree holds changes the pick did not make, so its record-only commit is not made over them",
			"clean the lane's worktree, then run the step again")
	}
	if _, err := pickGit(wt, "commit", "-q", "-m", pickMessage(st), "--", it.Path); err != nil {
		return "", refuse(string(StepWorktree), "", lane.ID,
			"git could not make the pick's record commit (is a git identity configured?): "+fsutil.RedactHome(err.Error()),
			"settle what git reports, then run the step again")
	}
	sha, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", branchRef+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(sha) {
		return "", fmt.Errorf("resolving the lane branch %s after the pick commit: %v", branch, err)
	}
	if ok, why := isPickCommit(c.RepoRoot, st, sha, base); !ok {
		return "", fmt.Errorf("the pick commit %s is not the one the loop made: %s", shortSHA(sha), why)
	}
	return sha, nil
}

// isPickCommit reports whether sha is this run's record-only commit: its
// parent is the base, its subject is the pick's, and it changes the intent's
// record alone, adding this run's entry.
func isPickCommit(repoRoot string, st State, sha, base string) (bool, string) {
	parent, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", sha+"^1^{commit}", "--")
	if err != nil || parent != base {
		return false, "its parent is not the lane's base"
	}
	subject, err := gitutil.Run(repoRoot, "log", "-1", "--format=%s", sha, "--")
	if err != nil || subject != pickSubject(st) {
		return false, "its subject is not the pick's"
	}
	files, err := gitutil.Run(repoRoot, "diff-tree", "--no-commit-id", "--name-only", "-r", "--no-renames", sha, "--")
	if err != nil || len(strings.Fields(files)) != 1 || !strings.HasPrefix(files, intent.IntentsRelDir+"/"+intent.BucketPlanned+"/") {
		return false, "it changes more than the intent's record"
	}
	body, err := gitutil.RunLimited(repoRoot, maxIntentBytes, "cat-file", "blob", sha+":"+files)
	if err != nil {
		return false, "its record cannot be read"
	}
	entries := intent.ParseGrounds(body)
	if len(entries) == 0 || !intent.IsRunPick(entries[len(entries)-1]) || !strings.HasPrefix(entries[len(entries)-1].Text, intent.RunPickMarker+st.RunID+" ") {
		return false, "its record's last grounds entry is not this run's pick"
	}
	return true, ""
}

// carriesPickEntry reports whether the intent file's last grounds entry is
// this run's pick.
func carriesPickEntry(abs string, st State) bool {
	data, err := fsutil.ReadGuarded(abs, maxIntentBytes)
	if err != nil {
		return false
	}
	entries := intent.ParseGrounds(string(data))
	if len(entries) == 0 {
		return false
	}
	last := entries[len(entries)-1]
	return intent.IsRunPick(last) && strings.HasPrefix(last.Text, intent.RunPickMarker+st.RunID+" ")
}

package loop

// pickcommit.go is the pick's record-only commit (itd-2609211116005482,
// criterion 2 and 4; spc-2609212015048113 scope 5): after the lane's branch is
// cut and before the implementer starts, the worktree stage of a picked run's
// first lane appends the pick's grounds entry to the intent in the lane's own
// worktree, through the intent store's grounds writer and its lock, and
// commits that one file on the lane's branch. The reason so reaches the
// default branch with the work, and stays on the branch as history if the lane
// is discarded. The checkout the run was started from is never written.
//
// The commit is made in the lane's worktree only, on the lane's own branch,
// which the worktree stage made or adopted a moment before; it is the one git
// write the loop makes outside `git worktree add`. The environment is the
// isolated one less the global-config neutralisers (gitutil.ScrubbedEnv): the
// commit is authored by the person whose identity git is configured with, as
// every commit of the repository is, and no inherited GIT_DIR, GIT_WORK_TREE
// or injected configuration can redirect it. Hooks, the fsmonitor and commit
// signing are off (gitutil.ExecPins): the message and the entry are computed,
// the lane's pull request runs every gate over the commit, and a hook
// dispatcher or a signing program the repository names is code the loop does
// not run (iss-2610090821520843).
// Every argument is derived: the paths come from the intent store's validated
// ids, after `--`, and the message from the run and intent ids.
//
// The body is idempotent, as every stage's is: a branch already carrying the
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

// pickMessage is the pick commit's whole message. Its text is computed by abcd
// from the run's records and no model composed it, so its trailer is the abcd
// label (composed_attribution.go, ruling PC1), never None: None declares that
// no tool touched the text, and abcd did.
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
		composedAssistedBy() + "\n"
}

// pickGit runs one git command in the lane's worktree for the pick commit and
// returns its stdout verbatim: a porcelain status line opens with a space when
// the change is unstaged, and the comparison below is made against the line as
// git wrote it.
func pickGit(dir string, args ...string) (string, error) {
	full := append(append(gitutil.ExecPins(), "-c", "core.quotePath=false", "-C", dir), args...)
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
	return stdout.String(), nil
}

// pickCommit makes, or finds, the pick's record-only commit on the lane's
// branch and returns its object name.
func pickCommit(c Context, lane *Lane, wt, branch, base string) (string, error) {
	st := c.State
	corpus, err := intent.Load(wt)
	if err != nil {
		return "", fmt.Errorf("reading the intent store in the lane's worktree: %w", err)
	}
	it, ok := corpus.Lookup(st.Intent)
	if !ok || it.Bucket != intent.BucketPlanned {
		return "", refuse(string(StageWorktree), "", lane.ID,
			st.Intent+" is not planned in the lane's worktree, cut from the default branch, so the pick's entry has no record to land on",
			"land the intent's planning on the default branch, then run `abcd implement step` again")
	}
	rel := filepath.ToSlash(it.Path)

	branchRef := "refs/heads/" + branch
	revs, err := gitutil.Run(c.RepoRoot, "rev-list", "--reverse", base+".."+branchRef, "--")
	if err != nil {
		return "", fmt.Errorf("listing %s past its base: %v", branch, err)
	}
	if revs != "" {
		first := strings.Fields(revs)[0]
		if ok, why := isPickCommit(c.RepoRoot, wt, st, rel, first, base); !ok {
			return "", refuse(string(StageWorktree), "", lane.ID,
				fmt.Sprintf("%s carries commits past its base, and the first (%s) is not the pick's record commit: %s", branch, shortSHA(first), why),
				"the pick's entry is the lane's first commit; remove the lane's worktree and branch, then run `abcd implement step` again")
		}
		return first, nil
	}

	dirty, err := pickGit(wt, "status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return "", fmt.Errorf("reading the lane's worktree: %w", err)
	}
	switch {
	case dirty == "":
		g, err := grounds.New(grounds.Pursued, st.Pick.Entry)
		if err != nil {
			return "", refuse(string(StageWorktree), "", lane.ID, "the pick's reason cannot be written as a grounds entry: "+err.Error(),
				"report this: the reason is computed, and a computed reason the writer refuses is a defect")
		}
		if _, err := intent.RecordGrounds(wt, st.Intent, g); err != nil {
			return "", refuse(string(StageWorktree), "", lane.ID, "the pick's entry could not be written: "+fsutil.RedactHome(err.Error()),
				"settle what the reason names, then run `abcd implement step` again")
		}
	case dirty == " M "+rel+"\x00" || dirty == "M  "+rel+"\x00":
		// Written before a kill and not committed: the file must be exactly
		// what the loop writes, the base's record plus this run's entry.
		if why := carriesPickEntry(c.RepoRoot, wt, st, rel, base); why != "" {
			return "", refuse(string(StageWorktree), "", lane.ID,
				rel+" is changed in the lane's worktree, and it is not the base's record with this run's pick appended: "+why,
				"the loop never commits what it did not write; restore the file (`git restore`), then run `abcd implement step` again")
		}
	default:
		return "", refuse(string(StageWorktree), "", lane.ID,
			"the lane's worktree holds changes the pick did not make, so its record-only commit is not made over them",
			"clean the lane's worktree, then run `abcd implement step` again")
	}
	if _, err := pickGit(wt, "commit", "-q", "-m", pickMessage(st), "--", rel); err != nil {
		return "", refuse(string(StageWorktree), "", lane.ID,
			"git could not make the pick's record commit (is a git identity configured?): "+fsutil.RedactHome(err.Error()),
			"settle what git reports, then run `abcd implement step` again")
	}
	sha, err := gitutil.Run(c.RepoRoot, "rev-parse", "--verify", "--quiet", branchRef+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(sha) {
		return "", fmt.Errorf("resolving the lane branch %s after the pick commit: %v", branch, err)
	}
	if ok, why := isPickCommit(c.RepoRoot, wt, st, rel, sha, base); !ok {
		return "", fmt.Errorf("the pick commit %s is not the one the loop made: %s", shortSHA(sha), why)
	}
	return sha, nil
}

// expectedPickRecord is the content the pick commit gives the intent's record
// at rel: the record at the lane's base with this run's entry appended, built
// as intent.RecordGrounds builds it in the lane's worktree (the same entry,
// prepared by the same redactor and validator, appended by the same writer).
func expectedPickRecord(repoRoot, wt string, st State, rel, base string) (string, error) {
	if st.Pick == nil {
		return "", fmt.Errorf("the run carries no pick")
	}
	g, err := grounds.New(grounds.Pursued, st.Pick.Entry)
	if err != nil {
		return "", err
	}
	prepared, _, err := intent.PrepareGrounds(wt, g)
	if err != nil {
		return "", err
	}
	was, err := gitutil.RunCappedBytes(repoRoot, maxIntentBytes, "cat-file", "blob", base+":"+rel)
	if err != nil {
		return "", fmt.Errorf("the record at the lane's base cannot be read: %v", err)
	}
	return grounds.AppendToRecord(string(was), prepared)
}

// isPickCommit reports whether sha is this run's record-only commit: its
// parent is the base, its subject is the pick's, it changes the picked
// intent's record at rel and no other path, and that record is byte for byte
// the base's with this run's one entry appended.
func isPickCommit(repoRoot, wt string, st State, rel, sha, base string) (bool, string) {
	parent, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", sha+"^1^{commit}", "--")
	if err != nil || parent != base {
		return false, "its parent is not the lane's base"
	}
	subject, err := gitutil.Run(repoRoot, "log", "-1", "--format=%s", sha, "--")
	if err != nil || subject != pickSubject(st) {
		return false, "its subject is not the pick's"
	}
	files, err := gitutil.RunCappedBytes(repoRoot, maxGitOutput, "diff-tree", "--no-commit-id", "--name-only", "-z", "-r", "--no-renames", sha, "--")
	if err != nil || string(files) != rel+"\x00" {
		return false, "it changes a path other than " + st.Intent + "'s record, or more than that record"
	}
	want, err := expectedPickRecord(repoRoot, wt, st, rel, base)
	if err != nil {
		return false, "the record it should carry cannot be built: " + fsutil.RedactHome(err.Error())
	}
	got, err := gitutil.RunCappedBytes(repoRoot, maxIntentBytes, "cat-file", "blob", sha+":"+rel)
	if err != nil {
		return false, "its record cannot be read"
	}
	if string(got) != want {
		return false, "its record is not the base's with this run's one entry appended and nothing else changed"
	}
	return true, ""
}

// carriesPickEntry names why the intent's record in the lane's worktree is not
// the base's record with this run's entry appended, or returns "" when it is.
func carriesPickEntry(repoRoot, wt string, st State, rel, base string) string {
	data, err := fsutil.ReadGuarded(filepath.Join(wt, filepath.FromSlash(rel)), maxIntentBytes)
	if err != nil {
		return "it cannot be read"
	}
	want, err := expectedPickRecord(repoRoot, wt, st, rel, base)
	if err != nil {
		return "the record it should carry cannot be built: " + fsutil.RedactHome(err.Error())
	}
	if string(data) != want {
		return "it differs from that record"
	}
	return ""
}

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
// or injected configuration can redirect it. The repository's hooks and its
// commit signing run as they do on the person's own commit (commitGit): they
// are how the repository commits, and a hook that refuses the commit refuses
// the stage (iss-2610111729399920).
// Every argument is derived: the paths come from the intent store's validated
// ids, after `--`, and the message from the run and intent ids.
//
// The body is idempotent, as every stage's is: a branch already carrying the
// pick commit as its first commit past the base is adopted; an entry written
// and not committed (a process killed between the two) is committed; anything
// else on the branch or in the worktree is refused and left where it is.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxGitOutput caps what the pick commit reads from git.
const maxGitOutput = 1 << 20

// commitTimeout bounds one commitGit command: the repository's hooks and its
// signing program run on it, and one can wait on a person (a GUI pinentry, a
// hook that prompts) who is not there, as on an unattended resume. A var so a
// test can shorten it.
var commitTimeout = 10 * time.Minute

// worktreeAddTimeout bounds the lane's `git worktree add` (worktreeAddGit):
// gitutil.CheckoutBound, the checkout's own bound, which the store's add
// shares, not commitTimeout. A var so a test can shorten it.
var worktreeAddTimeout = gitutil.CheckoutBound

// resetTimeout bounds the lane's reset --hard (resetLaneTree), as
// worktreeAddTimeout bounds the checkout and for the same reason: the reset
// rewrites the tree's files through the repository's content filters, a Git
// LFS smudge downloading what it names, so its time scales with the tree and
// the network. A var so a test can shorten it.
var resetTimeout = 60 * time.Minute

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

// pickGit runs one git command in the lane's worktree and returns its stdout
// verbatim: a porcelain status line opens with a space when the change is
// unstaged, and the comparison below is made against the line as git wrote it.
// It is the loop's one route for a git command that reads or judges a lane's
// working tree, or writes it where the person's own git runs no hook (the
// land's staging, a restart's clean, a merge's abort); a command that runs
// the repository's hooks on the person's own git goes through commitGit (a
// commit, a merge), resetLaneTree (a reset --hard) or worktreeAddGit (the
// lane's `git worktree add`). Each sees the repository as the person's own
// git does: gitutil.WorkingTreeCommand, which keeps the person's global config (a
// content filter a global `git lfs install` defines, a global ignore file)
// with the program-starting knobs and automatic maintenance pinned off.
func pickGit(dir string, args ...string) (string, error) {
	return gitutil.RunTreeCommand(gitutil.WorkingTreeCommand(dir, args...), args)
}

// commitGit is pickGit for a command that runs the repository's hooks on the
// person's own git, in a lane's worktree: the pick commit and a sync's merge,
// and, on the same command with a bound of its own, the reset --hard that a
// restart and the undo of a stopped sync make (resetLaneTree), which runs the
// repository's reference-transaction and post-index-change hooks. It runs
// gitutil.WorkingTreeCommitCommand, so the repository's hooks, its commit
// signing, its merge's signature check and its merge drivers run as they do
// on the person's own commit, merge and reset
// (iss-2610111729399920, iss-2610120640452939). A restart's clean stays on
// pickGit: `git clean` runs no hook, so the pins change nothing there.
//
// Because those programs run, the command is bounded by commitTimeout through
// gitutil.RunCommitCommandBounded: past it git and the processes below it are
// stopped together, so git removes its lock files and a stopped commit leaves
// no index.lock, and the call is refused naming the deadline
// (errCommitStalled). A hook's child is stopped with git because it would
// otherwise go on writing into the lane's worktree after the loop has given up
// on the command, and after undoStalledSync has reset that worktree; one that
// has detached (reparented to init) before the stop is out of reach. A git
// that exited 0 is a made commit even then.
func commitGit(dir string, args ...string) (string, error) {
	return gitutil.RunCommitCommandBounded(dir, commitTimeout, errCommitStalled, args...)
}

// worktreeAddGit is commitGit for the lane's `git worktree add`, which runs
// the repository's post-checkout hook as the person's own does
// (iss-2610120640452939), bounded by worktreeAddTimeout instead: a checkout's
// time scales with the tree, and a stop at its bound is refused as a checkout
// that did not finish (errCheckoutStalled), never as a signing prompt.
func worktreeAddGit(dir string, args ...string) (string, error) {
	return gitutil.RunCommitCommandBounded(dir, worktreeAddTimeout, errCheckoutStalled, args...)
}

// errCommitStalled names a commitGit command stopped at its deadline.
var errCommitStalled = errors.New("a signing prompt or hook did not finish")

// errCheckoutStalled names a worktreeAddGit command stopped at its bound.
var errCheckoutStalled = gitutil.ErrCheckoutStalled

// errResetStalled names a resetLaneTree stopped at its bound.
var errResetStalled = errors.New("the reset (its content filters, such as a Git LFS download, and the repository's hooks) did not finish")

// resetLaneTree resets the lane's worktree at dir to head, index and files,
// as the person's own `git reset --hard` does (commitGit's command): the
// repository's reference-transaction and post-index-change hooks run, and the
// files its filters smudge are written through them. It is the one reset
// --hard the loop makes: a restart's, once the uncommitted work is saved
// aside, and the undo of a sync's merge stopped at its deadline. It is bounded
// by resetTimeout rather than commitTimeout, and a stop there is refused as a
// reset that did not finish (errResetStalled), never as a signing prompt.
func resetLaneTree(dir, head string) error {
	_, err := gitutil.RunCommitCommandBounded(dir, resetTimeout, errResetStalled, "reset", "--hard", "--quiet", head)
	return err
}

// pickGitIndex is pickGit on a scratch index, for a command that must read the
// working tree as the person's git does without touching the lane's own index.
func pickGitIndex(dir, index string, args ...string) (string, error) {
	if !filepath.IsAbs(index) {
		return "", fmt.Errorf("the scratch index %q is not an absolute path", index)
	}
	cmd := gitutil.WorkingTreeCommand(dir, args...)
	cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+index)
	return gitutil.RunTreeCommand(cmd, args)
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
	if _, err := commitGit(wt, "commit", "-q", "-m", pickMessage(st), "--", rel); errors.Is(err, errCommitStalled) {
		return "", refuse(string(StageWorktree), "", lane.ID,
			"git could not make the pick's record commit: "+err.Error(),
			"answer the repository's signing prompt, or settle the hook that waits, then run `abcd implement step` again")
	} else if err != nil {
		return "", refuse(string(StageWorktree), "", lane.ID,
			"git could not make the pick's record commit (a repository hook refused it, or no git identity is configured): "+fsutil.RedactHome(err.Error()),
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

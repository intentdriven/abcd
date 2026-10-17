package claim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// traceGit, when a test sets it, is shown every git command this package runs,
// before it runs: the proof that a refusal came before any git at all, and
// that no value a writer chose ever reached git's argument list.
var traceGit func(args []string)

func trace(args ...string) {
	if traceGit != nil {
		traceGit(append([]string(nil), args...))
	}
}

// Bounds on a git command run under the developer's own configuration.
const (
	personalTimeout = 2 * time.Minute
	personalOutCap  = 64 << 10
	stderrCap       = 2048
)

// personalGit runs one local git command in the checkout at root under the
// developer's own configuration less any injected GIT_DIR or GIT_CONFIG_*
// (gitutil.ScrubbedEnv), with gitutil.ExecPins prepended so the repository's
// config starts no hook, fsmonitor or signing program, and no terminal prompt.
// It is for the commands that need the person's identity: the config reads
// that name them and the commit-tree that writes as them. stdin is the
// command's standard input; env is appended to the scrubbed environment. It
// returns stdout, git's exit code (0 on success, -1 when git did not run) and
// an error carrying git's bounded stderr.
func personalGit(root, stdin string, env []string, args ...string) (string, int, error) {
	trace(args...)
	ctx, cancel := context.WithTimeout(context.Background(), personalTimeout)
	defer cancel()
	full := append(append(gitutil.ExecPins(), "-c", "core.quotePath=false", "-C", root), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(append(gitutil.ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0"), env...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil {
		code := -1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		}
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > stderrCap {
			msg = msg[:stderrCap]
		}
		return "", code, fmt.Errorf("git %s: %v (%s)", args[0], err, msg)
	}
	if stdout.Len() > personalOutCap {
		return "", 0, fmt.Errorf("git %s wrote more than %d bytes", args[0], personalOutCap)
	}
	return stdout.String(), 0, nil
}

// localGit runs one read-only git command in the checkout at root under the
// isolated configuration (gitutil.RunCapped): a read of refs and objects
// needs nothing of the person's.
func localGit(root string, maxBytes int, args ...string) (string, error) {
	trace(args...)
	return gitutil.RunCapped(root, maxBytes, args...)
}

// storedGit is localGit reading history as stored
// (gitutil.RunCappedAsStored), as the history walk and the commit reads need:
// a refs/replace/* ref, which a fetch can bring in (remote.<name>.fetch
// naming refs/replace/*), would stand one commit in for another, its parents
// included, and a .git/info/grafts file would give a commit other parents.
// The trace is the subcommand's own argv, as it is for every command here:
// the global options and environment git runs under are not traced.
func storedGit(root string, maxBytes int, args ...string) (string, error) {
	trace(args...)
	return gitutil.RunCappedAsStored(root, maxBytes, args...)
}

// emptyTree is gitutil.EmptyTree, traced.
func emptyTree(root string) (string, error) {
	trace("hash-object", "-w", "-t", "tree", "--stdin")
	return gitutil.EmptyTree(root)
}

// storedGitCtx is storedGit bound to ctx (gitutil.RunCappedBytesAsStoredContext):
// git is killed when ctx ends, so a reader on a deadline, the prompt hook's
// status read among them, is never held past it by a slow history (a commit
// whose message is huge costs git a parse each). A ctx that cannot end runs
// storedGit itself, unchanged.
func storedGitCtx(ctx context.Context, root string, maxBytes int, args ...string) (string, error) {
	if ctx.Done() == nil {
		return storedGit(root, maxBytes, args...)
	}
	out, err := storedGitInput(ctx, root, "", maxBytes, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), " \t\r\n"), nil
}

// storedRefs reads every ref under prefix as stored, under ctx, line by line
// (gitutil.EachAsStoredContext), so however many this checkout holds the
// listing is read: each ref and its object, in git's ref order, to each. A
// line past maxListingLine, whose ref name is cut, is handed to each with
// cut set, its object still named.
func storedRefs(ctx context.Context, root, prefix string, each func(ref, sha string, cut bool)) error {
	args := []string{"for-each-ref", "--format=%(objectname) %(refname)", prefix}
	trace(args...)
	return gitutil.EachAsStoredContext(ctx, root, maxListingLine, func(line string) {
		sha, ref, ok := strings.Cut(line, " ")
		if !ok || !gitutil.IsFullSHA(sha) || !strings.HasPrefix(ref, prefix) {
			return
		}
		each(ref, sha, len(line) >= maxListingLine)
	}, args...)
}

// storedGitInput is storedGitCtx with stdin as git's standard input and
// stdout returned verbatim: a batch of objects named one per line
// (`cat-file --batch`), read in one command rather than one per object.
func storedGitInput(ctx context.Context, root, stdin string, maxBytes int, args ...string) ([]byte, error) {
	trace(args...)
	return gitutil.RunCappedBytesAsStoredContext(ctx, root, stdin, maxBytes, args...)
}

package gitutil

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/procgroup"
)

// asStored makes an isolated command read history as stored: no
// refs/replace/* ref stands one object in for another
// (GIT_NO_REPLACE_OBJECTS), no .git/info/grafts file gives a commit
// parents it does not store (noGraftFile), which git honours even with
// replace refs off, and no commit-graph file stands in for a commit's tree
// and parents (AsStoredEnv).
func asStored(cmd *exec.Cmd) *exec.Cmd {
	cmd.Env = AsStoredEnv(cmd.Env)
	return cmd
}

// commitGraphOff is the GIT_CONFIG_PARAMETERS entry that turns the
// commit-graph off, quoted whole as git before 2.31 writes it: every git
// parses that form, also after an entry in the 'key'='value' form git 2.31
// and later write, which a git before 2.31 cannot parse.
const commitGraphOff = "'core.commitgraph=false'"

// AsStoredEnv is env made to read objects as stored, for a git command built
// outside this package as well as asStored's own: GIT_NO_REPLACE_OBJECTS=1,
// GIT_GRAFT_FILE naming no file, core.commitGraph false as the last entry of
// GIT_CONFIG_PARAMETERS, and GIT_TEST_COMMIT_GRAPH=0. A commit-graph is a cache
// git reads a commit's tree and parents from in place of the commit, and one
// written by hand can claim another tree. The entry follows whatever entries
// env's GIT_CONFIG_PARAMETERS carries, which stay, and the last entry wins;
// GIT_TEST_COMMIT_GRAPH, which loads a graph whatever the setting says, is
// off. This is what .githooks/pre-push sets. The variables come last and any
// earlier value of them is dropped. env itself is not changed.
func AsStoredEnv(env []string) []string {
	out := make([]string, 0, len(env)+4)
	params := ""
	for _, kv := range env {
		switch key, value, _ := strings.Cut(kv, "="); key {
		case "GIT_CONFIG_PARAMETERS":
			params = value
		case "GIT_NO_REPLACE_OBJECTS", "GIT_GRAFT_FILE", "GIT_TEST_COMMIT_GRAPH":
		default:
			out = append(out, kv)
		}
	}
	if params != "" {
		params += " "
	}
	return append(out,
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_GRAFT_FILE="+noGraftFile,
		"GIT_CONFIG_PARAMETERS="+params+commitGraphOff,
		"GIT_TEST_COMMIT_GRAPH=0",
	)
}

// RunCappedAsStored is RunCapped reading history as stored (asStored): for a
// read whose answer is a commit's parents or what a branch holds, which a
// replace ref or a graft could otherwise change.
func RunCappedAsStored(root string, maxBytes int, args ...string) (string, error) {
	out, overflowed, err := runBoundedCmd(context.Background(), asStored(isolatedGit(root, args...)), maxBytes)
	if err != nil {
		return "", err
	}
	if overflowed {
		return "", fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated history is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return strings.TrimRight(string(out), " \t\r\n"), nil
}

// IsAncestorAsStored is IsAncestor reading history as stored (asStored): for
// an ancestry verdict a gate acts on, which a graft, a refs/replace entry or
// a commit-graph could otherwise give parents the commits do not store.
func IsAncestorAsStored(root, ancestor, descendant string) (bool, error) {
	return isAncestor(root, true, ancestor, descendant)
}

// RunCappedExitOneAsStored is RunCappedExitOne reading history as stored
// (asStored).
func RunCappedExitOneAsStored(root string, maxBytes int, args ...string) ([]byte, bool, error) {
	return runCappedExitOneCmd(asStored(isolatedGit(root, args...)), maxBytes, args)
}

// RunCappedExitOneAsStoredContext is RunCappedExitOneContext reading history
// as stored (asStored): for a deadline-bound ancestry verdict
// (`merge-base --is-ancestor`, which answers no with exit 1) that a removal
// or a landing acts on.
func RunCappedExitOneAsStoredContext(ctx context.Context, root string, maxBytes int, args ...string) ([]byte, bool, error) {
	return runCappedExitOneContextCmd(ctx, func() *exec.Cmd { return asStored(isolatedGitContext(ctx, root, args...)) }, maxBytes, args)
}

// RunCappedBytesAsStoredContext is RunCappedAsStored bound to ctx, with stdin
// as the command's standard input ("" for none), returning stdout verbatim:
// for a deadline-bound read of history as stored, a batch of objects named on
// standard input among them. When ctx ends first, git is killed
// (isolatedGitContext) and the error wraps ctx's own; output past maxBytes is
// an error saying the cap was exceeded, never a truncated answer. git's
// output is read to EOF before Wait, as RunLimitedContext reads it, so a git
// that answered is read whole however late its output closes, until ctx ends.
func RunCappedBytesAsStoredContext(ctx context.Context, root, stdin string, maxBytes int, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	cmd := asStored(isolatedGitContext(ctx, root, args...))
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, overflowed, err := runBoundedCmd(ctx, cmd, maxBytes)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), ctxErr)
	}
	if err != nil {
		return nil, err
	}
	if overflowed {
		return nil, fmt.Errorf("git %s: output exceeded the %d-byte cap; the answer would be truncated, and a truncated history is a wrong one",
			strings.Join(args, " "), maxBytes)
	}
	return out, nil
}

// EachAsStoredContext runs one git command as stored (asStored) under ctx,
// for a command whose output may run to any length (a listing of every ref
// under a prefix a fetch of a whole namespace can fill): each line of stdout
// is handed to each as it is read, without its newline, and none is kept, so
// no bound on the whole output applies. A line longer than maxLine bytes is
// handed cut to its first maxLine. git is killed when ctx ends, and ctx's
// error is then answered; a failure carries git's bounded stderr.
func EachAsStoredContext(ctx context.Context, root string, maxLine int, each func(line string), args ...string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	cmd := asStored(isolatedGitContext(ctx, root, args...))
	w := &lineSink{max: maxLine, each: each}
	e := &capWriter{remaining: 4096}
	err := procgroup.RunToEOF(ctx, cmd, w, e)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), ctxErr)
	}
	if err != nil {
		return fmt.Errorf("%w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	w.flush()
	return nil
}

// StreamAsStored runs one git command as stored (asStored) with its stdout
// written to w as git writes it, for an answer the caller keeps whole rather
// than reads, such as a bundle written to a file the caller opened itself: no
// cap applies, and git never opens a path of its own for it. A failure
// carries git's bounded stderr.
func StreamAsStored(root string, w io.Writer, args ...string) error {
	cmd := asStored(isolatedGit(root, args...))
	e := &capWriter{remaining: 4096}
	if err := procgroup.RunToEOF(context.Background(), cmd, w, e); err != nil {
		return fmt.Errorf("git %s: %w (stderr: %q)", strings.Join(args, " "), err, strings.TrimSpace(string(e.buf)))
	}
	return nil
}

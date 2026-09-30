package runner

// proc.go is the one place this package starts a process. The trust boundary:
// a runner starts a harness with a prompt and a repository path, so
//
//   - the argv is a vector handed to exec, never a shell line, and the prompt
//     travels as one argument behind the end-of-options marker;
//   - the binary is resolved on PATH by its fixed name and refused when it is
//     not absolute, when other can write it or its directory, when a group
//     other than the system administrator group can write either, or when
//     it resolves inside the repository the role runs in, or the
//     checkout a lane's worktree belongs to, lexically or through a symlink (a
//     PATH entry into either is repository content, never run);
//   - the environment is the parent's with every git repository-selection and
//     config-injection variable scrubbed (gitutil.ScrubbedEnv), so an inherited
//     GIT_DIR cannot aim the role's git at another repository, while the
//     person's own git identity and the harness's own credential variables
//     pass through untouched: abcd never logs a harness in;
//   - stdin is the null device, the working directory is the repository;
//   - stdout and stderr are each bounded; a harness writing past the bound is
//     cut off and the run fails naming the bound;
//   - the child leads its own process group, and a run past its time, or one
//     whose context ends, is killed with that group through the pid this
//     handle holds: never a pattern, never another process;
//   - an error is written by abcd and never carries the harness's output.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Output bounds. A transcript is the whole event stream of one role's run;
// stderr is diagnostics only.
const (
	defaultMaxStdout = 32 << 20
	defaultMaxStderr = 64 << 10
)

// pipeGrace is how long a run keeps reading output once the harness has exited
// or been killed: a descendant that left the group can hold the pipe open.
const pipeGrace = 5 * time.Second

// launcher starts one harness binary. Its fields are the seams the tests
// narrow; production uses defaultLauncher.
type launcher struct {
	lookPath  func(string) (string, error)
	maxStdout int
	maxStderr int
}

func defaultLauncher() launcher {
	return launcher{lookPath: exec.LookPath, maxStdout: defaultMaxStdout, maxStderr: defaultMaxStderr}
}

// procResult is what one run produced.
type procResult struct {
	stdout, stderr []byte
	// overflow names the stream that passed its bound, "" when neither did.
	overflow string
}

// admit resolves name on PATH and refuses a result that is not absolute or
// that lies inside any of repos (the directory the role runs in, and the
// checkout it belongs to), lexically or after symlink resolution.
func (l launcher) admit(runner, name string, repos ...string) (string, error) {
	p, err := l.lookPath(name)
	if err != nil {
		return "", fail(runner, ReasonAbsent, "%s is not on PATH", name)
	}
	if !filepath.IsAbs(p) {
		return "", fail(runner, ReasonAbsent, "%s resolves to a relative path, which is never run", name)
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", fail(runner, ReasonAbsent, "%s does not resolve to a file", name)
	}
	// Whoever can write the binary, the directory PATH reaches it through, or
	// the directory it resolves into chooses what runs.
	for _, c := range []string{resolved, filepath.Dir(resolved), filepath.Dir(filepath.Clean(p))} {
		fi, err := os.Stat(c)
		if err != nil {
			return "", fail(runner, ReasonAbsent, "%s could not be examined before it is run", name)
		}
		if fsutil.WritableByOthers(fi) && !adminGroupWritableOnly(fi) {
			return "", fail(runner, ReasonAbsent, "%s, or a directory it is reached through, is one group or other can write, "+
				"so it is never run; chmod go-w it", name)
		}
	}
	var guards []string
	for _, repo := range repos {
		if repo == "" {
			continue
		}
		guards = append(guards, filepath.Clean(repo))
		if g, err := filepath.EvalSymlinks(repo); err == nil {
			guards = append(guards, g)
		}
	}
	fold := fsutil.CaseFoldingFS()
	for _, g := range guards {
		for _, c := range []string{filepath.Clean(p), resolved} {
			if fsutil.PathWithin(c, g, fold) {
				return "", fail(runner, ReasonAbsent, "%s resolves inside the repository the role runs in or the checkout it belongs to; "+
					"a program there is repository content and is never run", name)
			}
		}
	}
	return resolved, nil
}

// adminGroupWritableOnly reports whether fi's only write bit beyond its
// owner's is the group's, and that group is the system administrator group:
// gid 0 (root, or wheel on darwin) anywhere, and gid 80 (admin) on darwin.
// Homebrew installs /opt/homebrew/bin as root- or user-owned, group admin,
// mode 0775, so a harness installed through it is reached through a
// group-writable directory. Members of the administrator group can already
// act as root, so that write grants them nothing they do not hold, and the
// directory is admitted. Other-writable is never admitted, whatever the
// group, and neither is any other group, nor a group that cannot be read.
func adminGroupWritableOnly(fi os.FileInfo) bool {
	if fi.Mode().Perm()&0o002 != 0 {
		return false
	}
	gid, ok := fileGroup(fi)
	if !ok {
		return false
	}
	return gid == 0 || (runtime.GOOS == "darwin" && gid == 80)
}

// fileGroup reads the owning group of a stat result; ok is false when the
// platform does not report one. A variable so a test can name a group it
// cannot chown to.
var fileGroup = func(fi os.FileInfo) (uint32, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return st.Gid, true
}

// run starts bin with args in dir and waits for it, at most timeout.
func (l launcher) run(ctx context.Context, runner, bin string, args []string, dir string, timeout time.Duration) (procResult, error) {
	// #nosec G204 -- bin is a fixed harness name resolved by admit (absolute,
	// outside the repository); args are a vector built by the adapter, never a
	// shell line.
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = gitutil.ScrubbedEnv()
	cmd.Stdin = nil
	var out, errb bytes.Buffer
	ow := &bounded{w: &out, remaining: l.maxStdout}
	ew := &bounded{w: &errb, remaining: l.maxStderr}
	cmd.Stdout = ow
	cmd.Stderr = ew
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = pipeGrace
	if err := cmd.Start(); err != nil {
		return procResult{}, fail(runner, ReasonFailed, "%s could not be started", filepath.Base(bin))
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var werr error
	killed := ""
	select {
	case werr = <-done:
	case <-timer.C:
		killed = fmt.Sprintf("did not finish within the time allowed (%s)", timeout)
	case <-ctx.Done():
		killed = "was stopped: the run it belongs to ended"
	}
	if killed != "" {
		// The group this child leads (Setpgid), through the pid this handle
		// holds; the leader is not yet reaped, so the group id is still its.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
	}
	res := procResult{stdout: out.Bytes(), stderr: errb.Bytes()}
	switch {
	case ow.overflowed:
		res.overflow = "stdout"
	case ew.overflowed:
		res.overflow = "stderr"
	}
	name := filepath.Base(bin)
	switch {
	case killed != "":
		return res, fail(runner, ReasonFailed, "%s %s; its process group was killed", name, killed)
	case res.overflow != "":
		return res, fail(runner, ReasonFailed, "%s wrote past the %s bound (%d bytes); the rest was discarded",
			name, res.overflow, l.bound(res.overflow))
	case errors.Is(werr, exec.ErrWaitDelay):
		return res, fail(runner, ReasonFailed, "%s exited, but a process it started left its group and held its output", name)
	case werr != nil:
		var ee *exec.ExitError
		if errors.As(werr, &ee) {
			return res, fail(runner, ReasonFailed, "%s exited with status %d", name, ee.ExitCode())
		}
		return res, fail(runner, ReasonFailed, "%s did not complete", name)
	}
	return res, nil
}

func (l launcher) bound(stream string) int {
	if stream == "stderr" {
		return l.maxStderr
	}
	return l.maxStdout
}

// bounded keeps at most remaining bytes and drops the rest without failing the
// writer, recording that it dropped some.
type bounded struct {
	w          *bytes.Buffer
	remaining  int
	overflowed bool
}

func (b *bounded) Write(p []byte) (int, error) {
	n := len(p)
	k := n
	if k > b.remaining {
		k = b.remaining
		b.overflowed = true
	}
	if k > 0 {
		b.w.Write(p[:k])
		b.remaining -= k
	}
	return n, nil
}

// transcript is what a run's record keeps: the event stream, then the
// harness's stderr under a marker line when it wrote any. The store redacts it
// on write.
func (r procResult) transcript() []byte {
	if len(r.stderr) == 0 {
		return r.stdout
	}
	var b bytes.Buffer
	b.Write(r.stdout)
	if len(r.stdout) > 0 && r.stdout[len(r.stdout)-1] != '\n' {
		b.WriteByte('\n')
	}
	// The marker opens with "==", never a "---" run, so no frontmatter reader
	// takes it for a block delimiter.
	b.WriteString("== abcd runner: the harness's stderr ==\n")
	b.Write(r.stderr)
	return b.Bytes()
}

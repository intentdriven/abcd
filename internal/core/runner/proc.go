package runner

// proc.go is the one place this package starts a process, and Exec is the one
// place another package starts a vendor binary through it. The trust boundary:
// a runner starts a harness with a prompt and a repository path, so
//
//   - the argv is a vector handed to exec, never a shell line, and the prompt
//     travels as one argument behind the end-of-options marker;
//   - the binary is resolved on PATH by its fixed name and refused when it is
//     not absolute, when other can write it or its directory, when a group
//     other than darwin's admin group (gid 80, on darwin only) can write
//     either, or when it resolves inside the repository the role runs in, or
//     the checkout a lane's worktree belongs to, lexically or through a
//     symlink (a PATH entry into either is repository content, never run);
//   - the environment is the parent's with every git repository-selection and
//     config-injection variable scrubbed (gitutil.ScrubbedEnv), so an inherited
//     GIT_DIR cannot aim the role's git at another repository, while the
//     person's own git identity and the harness's own credential variables
//     pass through untouched: abcd never logs a harness in; an adapter's own
//     sealing switches (seal) are set over whatever the parent holds for
//     them, so the parent cannot reopen what the adapter closes;
//   - stdin is the null device, the working directory is the repository (for
//     Exec, the directory its caller names);
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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/procgroup"
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
	lookPath func(string) (string, error)
	// reachedThrough is where a pinned harness was found on PATH, judged against the
	// repository as what lookPath answers is; "" for none.
	reachedThrough string
	maxStdout      int
	maxStderr      int
	// gateWaiting reports whether the process tree a run's harness leads
	// holds a waiter queued at abcd's load gate (queuedAtGate); nil asks
	// nothing, and the run's clock never stops. gatePoll is how often it is
	// asked; zero is gatePollInterval.
	gateWaiting func(pid int) bool
	gatePoll    time.Duration
	// carry, when set, is all a launch carries of the parent's environment.
	carry func(env []string) []string
}

// defaultLauncher is a harness's launcher: its clock stops while the role
// waits at the load gate (gatewait.go).
func defaultLauncher() launcher {
	return launcher{lookPath: exec.LookPath, maxStdout: defaultMaxStdout, maxStderr: defaultMaxStderr,
		gateWaiting: queuedAtGate}
}

// procResult is what one run produced.
type procResult struct {
	stdout, stderr []byte
	// overflow names the stream that passed its bound, "" when neither did.
	overflow string
	// exit is the process's status, -1 when it was killed or did not exit.
	exit int
	// silent is how long it had written nothing when it ended.
	silent time.Duration
}

// admit resolves name on PATH and refuses a result that is not absolute or
// that lies inside any of repos (the directory the role runs in, and the
// checkout it belongs to), lexically or after symlink resolution.
func (l launcher) admit(runner, name string, repos ...string) (string, error) {
	p, err := l.lookPath(name)
	if err != nil {
		return "", failAt(ErrNotOnPath, runner, ReasonAbsent, "%s is not on PATH", name)
	}
	if !filepath.IsAbs(p) {
		return "", failAt(ErrRefused, runner, ReasonAbsent, "%s resolves to a relative path, which is never run", name)
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", failAt(ErrNotOnPath, runner, ReasonAbsent, "%s does not resolve to a file", name)
	}
	// Whoever can write the binary, the directory PATH reaches it through, or
	// the directory it resolves into chooses what runs.
	for _, c := range []string{resolved, filepath.Dir(resolved), filepath.Dir(filepath.Clean(p))} {
		fi, err := os.Stat(c)
		if err != nil {
			return "", failAt(ErrRefused, runner, ReasonAbsent, "%s could not be examined before it is run", name)
		}
		if fsutil.WritableByOthers(fi) && !adminGroupWritableOnly(fi) {
			return "", failAt(ErrRefused, runner, ReasonAbsent, "%s, or a directory it is reached through, is one group or other can write, "+
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
		for _, c := range []string{filepath.Clean(p), resolved, l.reachedThrough} {
			if c != "" && fsutil.PathWithin(c, g, fold) {
				return "", failAt(ErrRefused, runner, ReasonAbsent, "%s resolves inside the repository the role runs in or the checkout it belongs to; "+
					"a program there is repository content and is never run", name)
			}
		}
	}
	return resolved, nil
}

// adminGroupWritableOnly reports whether fi's only write bit beyond its
// owner's is the group's, and that group is darwin's admin group (gid 80),
// on darwin only. Homebrew installs /opt/homebrew/bin as root- or
// user-owned, group admin, mode 0775, so a harness installed through it is
// reached through a group-writable directory. Members of darwin's admin
// group can sudo by default, so that write grants them nothing they do not
// hold, and the directory is admitted. gid 0 is never admitted: being in
// Linux's root group, or darwin's wheel, does not by itself let a member act
// as root. Other-writable is never admitted, whatever the group, and neither
// is any other group, gid 80 off darwin, nor a group that cannot be read.
func adminGroupWritableOnly(fi os.FileInfo) bool {
	if fi.Mode().Perm()&0o002 != 0 {
		return false
	}
	gid, ok := fileGroup(fi)
	if !ok {
		return false
	}
	return runtime.GOOS == "darwin" && gid == 80
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

// run starts bin with args in dir and waits for it, at most timeout. seal is
// the adapter's KEY=value switches, set over the parent's.
func (l launcher) run(ctx context.Context, runner, bin string, args, seal []string, dir string, timeout time.Duration) (procResult, error) {
	// #nosec G204 -- bin is a fixed harness name resolved by admit (absolute,
	// outside the repository); args are a vector built by the adapter, never a
	// shell line.
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	env := gitutil.ScrubbedEnv()
	if l.carry != nil {
		env = l.carry(env)
	}
	cmd.Env = sealedEnv(env, seal)
	cmd.Stdin = nil
	var out, errb bytes.Buffer
	ow := &bounded{w: &out, remaining: l.maxStdout}
	ew := &bounded{w: &errb, remaining: l.maxStderr}
	cmd.Stdout = ow
	cmd.Stderr = ew
	procgroup.Lead(cmd)
	cmd.WaitDelay = pipeGrace
	// Read before the start returns: a process's start is when it forked, and
	// Start returns only once exec has, which can take seconds.
	forked := time.Now()
	if err := cmd.Start(); err != nil {
		return procResult{}, failAt(ErrNotStarted, runner, ReasonFailed, "%s could not be started", filepath.Base(bin))
	}
	launchStarted(ctx, cmd.Process.Pid, forked)
	// The clock stands still while the harness's tree waits at the load gate
	// (gatewait.go): each look that finds a waiter queued moves the deadline
	// by the time since the last look, up to gateCredit.
	deadline := time.Now().Add(timeout)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var looks <-chan time.Time
	if l.gateWaiting != nil {
		every := l.gatePoll
		if every <= 0 {
			every = gatePollInterval
		}
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		looks = ticker.C
	}
	lastLook := time.Now()
	var waited time.Duration
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var werr error
	killed := ""
wait:
	for {
		select {
		case werr = <-done:
			break wait
		case at := <-looks:
			d := at.Sub(lastLook)
			lastLook = at
			if left := gateCredit(timeout) - waited; left > 0 && l.gateWaiting(cmd.Process.Pid) {
				d = min(d, left)
				waited += d
				deadline = deadline.Add(d)
				timer.Reset(time.Until(deadline))
			}
		case <-timer.C:
			killed = fmt.Sprintf("did not finish within the time allowed (%s)", timeout)
			if waited > 0 {
				killed = fmt.Sprintf("did not finish within the time allowed (%s, and %s it waited at the load gate)",
					timeout, waited.Round(time.Second))
			}
			break wait
		case <-ctx.Done():
			killed = "was stopped: the run it belongs to ended"
			break wait
		}
	}
	if killed != "" {
		// The group this child leads (Setpgid), through the pid this handle
		// holds; the leader is not yet reaped, so the group id is still its.
		_ = procgroup.Kill(cmd)
		<-done
	} else {
		// Nothing the harness started outlives it, however it ended.
		procgroup.KillLeft(cmd)
	}
	res := procResult{stdout: out.Bytes(), stderr: errb.Bytes(), exit: -1,
		silent: time.Since(latest(forked, ow.last, ew.last))}
	var ee *exec.ExitError
	switch {
	case killed != "":
	case werr == nil:
		res.exit = 0
	case errors.As(werr, &ee):
		res.exit = ee.ExitCode()
	}
	switch {
	case ow.overflowed:
		res.overflow = "stdout"
	case ew.overflowed:
		res.overflow = "stderr"
	}
	name := filepath.Base(bin)
	switch {
	case killed != "":
		return res, failAt(ErrKilled, runner, ReasonFailed, "%s %s; its process group was killed", name, killed)
	case res.overflow != "":
		return res, fail(runner, ReasonFailed, "%s wrote past the %s bound (%d bytes); the rest was discarded",
			name, res.overflow, l.bound(res.overflow))
	case errors.Is(werr, exec.ErrWaitDelay):
		return res, fail(runner, ReasonFailed, "%s exited, but a process it started left its group and held its output", name)
	case werr != nil:
		if errors.As(werr, &ee) {
			return res, fail(runner, ReasonFailed, "%s exited with status %d", name, ee.ExitCode())
		}
		return res, fail(runner, ReasonFailed, "%s did not complete", name)
	}
	return res, nil
}

// The launch stages a Failure can name through errors.Is: the program was not
// found, it was found and refused, or it was admitted and could not start.
// ErrKilled names a run that started and was killed, past its time or when
// the run it belongs to ended (iss-2611021751406429): its work may stand in
// its working tree, and what it spent shows only in its transcript. Every
// other failure (a non-zero exit, an overrun bound) is none of them.
var (
	ErrNotOnPath  = errors.New("not on PATH")
	ErrRefused    = errors.New("refused before it is run")
	ErrNotStarted = errors.New("could not be started")
	ErrKilled     = errors.New("killed")
)

// Command is one program found on PATH that another package runs under this
// file's discipline: the one launch primitive abcd has for a vendor binary,
// so a second copy of the admission, the group kill and the bounds is never
// written. Zero output bounds take a harness's own.
type Command struct {
	// Name is the program's fixed name, looked up on PATH; a name carrying a
	// path separator is refused, so no caller can point Exec at a file.
	Name string
	// Args is the argument vector, never a shell line.
	Args []string
	// Dir is the working directory the program runs in; empty runs it at the
	// file-system root, never in the caller's directory, which for abcd is
	// usually a project whose settings a vendor binary must not read.
	Dir string
	// Guards are the folders a program resolving inside is refused from,
	// lexically or through a link: repository content is never run. A caller
	// running in or for a project passes it here; Exec cannot know it.
	Guards []string
	// Timeout bounds the run; past it the program's group is killed. A
	// command without one is never run.
	Timeout time.Duration
	// MaxStdout and MaxStderr bound each stream; past either the run fails.
	MaxStdout, MaxStderr int
}

// Output is what an Exec run wrote, each stream up to its bound.
type Output struct {
	Stdout, Stderr []byte
}

// Exec admits c.Name exactly as a harness is admitted (admit) and runs it
// exactly as a harness is run (run): the scrubbed environment, the null
// device on stdin, the bounds, and the process group killed through this
// handle at the timeout or when ctx ends. A failure is a *Failure, whose
// errors.Is names the stage for ErrNotOnPath, ErrRefused and ErrNotStarted;
// the output is returned whenever the program produced some. Its clock never
// stops: the load gate's credit is a role's (gatewait.go), and c.Timeout is
// the caller's own bound.
func Exec(ctx context.Context, c Command) (Output, error) {
	l := defaultLauncher()
	l.gateWaiting = nil
	if c.MaxStdout > 0 {
		l.maxStdout = c.MaxStdout
	}
	if c.MaxStderr > 0 {
		l.maxStderr = c.MaxStderr
	}
	if c.Timeout <= 0 {
		return Output{}, fail(c.Name, ReasonFailed, "%s was given no time bound, so it is never run", c.Name)
	}
	if c.Name == "" || filepath.Base(c.Name) != c.Name {
		return Output{}, failAt(ErrRefused, c.Name, ReasonAbsent, "%s is not a fixed program name looked up on PATH, so it is never run", c.Name)
	}
	dir := c.Dir
	if dir == "" {
		dir = string(filepath.Separator)
	}
	bin, err := l.admit(c.Name, c.Name, c.Guards...)
	if err != nil {
		return Output{}, err
	}
	res, err := l.run(ctx, c.Name, bin, c.Args, nil, dir, c.Timeout)
	return Output{Stdout: res.stdout, Stderr: res.stderr}, err
}

// stageEnvExtra are further variables a walled launch carries: only a test's
// stand-in harness needs any.
var stageEnvExtra []string

// stageEnvNames are what a walled launch carries of the parent's environment:
// the harness's own needs, a shell's, and a build's. Nothing else passes.
var stageEnvNames = []string{
	"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR", "TERM", "TZ", "LANG",
	"LC_ALL", "LC_COLLATE", "LC_CTYPE", "LC_MESSAGES", "LC_MONETARY", "LC_NUMERIC", "LC_TIME",
	"NODE_EXTRA_CA_CERTS", "SSL_CERT_FILE", "SSL_CERT_DIR",
	"GOROOT", "GOFLAGS", "GOPRIVATE", "GONOSUMDB", "GONOPROXY", "GOTOOLCHAIN", "GOMAXPROCS",
}

// stageProxyNames are carried with any credentials in their URLs taken out.
var stageProxyNames = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"}

// stageEnv is what a walled launch carries of env.
func stageEnv(env []string) []string {
	var out []string
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case slices.Contains(stageProxyNames, k):
			if clean, ok := proxyWithoutCredentials(v); ok {
				out = append(out, k+"="+clean)
			}
		case k == "GOPROXY":
			if clean, ok := proxyListWithoutCredentials(v); ok {
				out = append(out, k+"="+clean)
			}
		case slices.Contains(stageEnvNames, k) || slices.Contains(stageEnvExtra, k):
			out = append(out, kv)
		}
	}
	return out
}

// proxyWithoutCredentials is a proxy value with its URL's credentials removed; ok
// is false for a value that holds some in a shape that cannot be taken apart.
func proxyWithoutCredentials(v string) (string, bool) {
	if !strings.Contains(v, "@") {
		return v, true
	}
	u, err := url.Parse(v)
	if err != nil || u.User == nil {
		return "", false
	}
	u.User = nil
	return u.String(), true
}

// proxyListWithoutCredentials is a Go proxy list, whose entries a comma or a bar
// separates, with each entry's credentials taken out.
func proxyListWithoutCredentials(v string) (string, bool) {
	var b strings.Builder
	start := 0
	for i := 0; i <= len(v); i++ {
		if i < len(v) && v[i] != ',' && v[i] != '|' {
			continue
		}
		clean, ok := proxyWithoutCredentials(v[start:i])
		if !ok {
			return "", false
		}
		b.WriteString(clean)
		if i < len(v) {
			b.WriteByte(v[i])
		}
		start = i + 1
	}
	return b.String(), true
}

// stageShellNames are what the harness and the shell itself add to a stage's
// shell, beside what it carries.
var stageShellNames = []string{"PWD", "OLDPWD", "SHLVL", "_", "CLAUDECODE", "AI_AGENT", "GIT_EDITOR", "COREPACK_ENABLE_AUTO_PIN",
	"NoDefaultCurrentDirectoryInExePath", "__CF_USER_TEXT_ENCODING", "GOCACHE", "GOMODCACHE", "GOPATH", "GOBIN", "GOPROXY", abcdhome.PersonHomeEnv,
	stageUpdatesOff}

// stageShellUnset are what a stage's shell file takes back out of what the harness
// starts the shell with: the walled directory, under each name it goes by, the file the
// run proved and the harness's messaging variables. The hooks and the approver read their
// own environment and flags, not the shell's. These stay outside what a stage's shell may
// hold.
var stageShellUnset = []string{"CLAUDE_CONFIG_DIR", StageProtectEnv, StageHarnessEnv, "CLAUDE_CODE_MESSAGING_TOKEN", "CLAUDE_CODE_MESSAGING_SOCKET"}

// harnessSecretWords are what a word of a name under the harness's prefix may not end in,
// so a joined spelling reads as the plain one. The prefix stays open because each harness
// release adds names of its own, and a stage holding a new setting costs nothing; a new
// name of one of these shapes stops the drive and says the version.
var harnessSecretWords = []string{"TOKEN", "OAUTH", "KEY", "SECRET", "PASSWORD", "PASSWD", "PASSPHRASE", "CREDENTIAL", "CREDENTIALS", "CREDS",
	"AUTH", "AUTHENTICATE", "AUTHENTICATION", "AUTHORIZATION", "COOKIE", "GRANT", "SOCKET", "SOCK", "JWT", "BEARER"}

// stageShellSecret reports whether name reads as a credential or a socket: one abcd
// knows, or one under the harness's prefix with a word of that shape.
func stageShellSecret(name string) bool {
	if slices.Contains(credentialVars, name) {
		return true
	}
	rest, ok := strings.CutPrefix(name, "CLAUDE_")
	if !ok {
		return false
	}
	return slices.ContainsFunc(strings.Split(rest, "_"), func(word string) bool {
		return slices.ContainsFunc(harnessSecretWords, func(secret string) bool { return strings.HasSuffix(word, secret) })
	})
}

// stageShellAllowed reports whether a stage's shell may hold the variable name.
func stageShellAllowed(name string) bool {
	if stageShellSecret(name) || slices.Contains(stageShellUnset, name) {
		return false
	}
	return slices.Contains(stageEnvNames, name) || slices.Contains(stageProxyNames, name) || slices.Contains(stageShellNames, name) ||
		strings.HasPrefix(name, "CLAUDE_") || strings.HasPrefix(name, "ABCD_STAGE_")
}

// sealedEnv is env with every key seal names removed, then each of seal's
// KEY=value entries appended: one entry per switch, the adapter's value. A
// bare name in seal takes the variable out and sets nothing.
func sealedEnv(env, seal []string) []string {
	if len(seal) == 0 {
		return env
	}
	set := map[string]bool{}
	for _, kv := range seal {
		k, _, _ := strings.Cut(kv, "=")
		set[k] = true
	}
	out := make([]string, 0, len(env)+len(seal))
	for _, kv := range env {
		if k, _, _ := strings.Cut(kv, "="); !set[k] {
			out = append(out, kv)
		}
	}
	for _, kv := range seal {
		if strings.Contains(kv, "=") {
			out = append(out, kv)
		}
	}
	return out
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
	// last is when anything was last written, the zero time for nothing yet.
	last time.Time
}

// latest is the latest of times.
func latest(times ...time.Time) time.Time {
	var at time.Time
	for _, t := range times {
		if t.After(at) {
			at = t
		}
	}
	return at
}

func (b *bounded) Write(p []byte) (int, error) {
	b.last = time.Now()
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

// stderrMarker is the line a transcript's stderr follows.
const stderrMarker = "== abcd runner: the harness's stderr =="

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
	b.WriteString(stderrMarker + "\n")
	b.Write(r.stderr)
	return b.Bytes()
}

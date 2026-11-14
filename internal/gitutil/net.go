package gitutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/procgroup"
)

// Bounds on a git command that reaches the remote.
const (
	// NetTimeout is the longest one networked git command may run.
	NetTimeout = 10 * time.Minute
	// netOutputCap bounds what such a command may write to stdout.
	netOutputCap = 1 << 20
	// netStderrCap bounds how much of its stderr an error carries.
	netStderrCap = 2048
	// The bounds a stalled connection meets.
	netLowSpeedLimit = "1000"
	netLowSpeedTime  = "60"
	// BatchMode keeps ssh from asking anything on the terminal (a first
	// host-key question, a passphrase): GIT_TERMINAL_PROMPT reaches git's
	// own prompts, and ssh reads the terminal itself.
	netSSHOptions = "-o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=15 -o ServerAliveCountMax=4"
)

// Net runs one git command that reaches the remote, from the checkout at dir:
// a fetch or a push. It is the one networked git runner, shared by the
// implement loop's landing, `abcd land` and the reservation on the shared
// remote. A git command that reaches no remote runs through LocalGit instead, so
// a hook it runs sees none of the network settings below.
//
// The repository's hooks run (the pre-push gate is the point), under the
// developer's own configuration less any injected GIT_DIR or GIT_CONFIG_*
// (ScrubbedEnv), with GIT_TERMINAL_PROMPT=0 so git's own credential prompt
// cannot hang a session. ssh reads the terminal itself, so the ssh command Net
// composes runs in batch mode: a host-key question or a passphrase prompt is a
// failure, not a wait. A developer's own ssh command is left as it is, and
// whether it can prompt is theirs. A connection that stalls is cut off in
// about a minute rather than at NetTimeout, so a lost network is met as a
// failure a caller can read (iss-2610080620372731): an HTTP transfer slower
// than netLowSpeedLimit bytes a second for netLowSpeedTime is aborted, and
// ssh, unless the developer configured their own ssh command, gives up a
// connect after 30 seconds and a silent connection after four missed
// keepalives. Stdout is returned within
// netOutputCap, on a failure too (a porcelain push reports its rejections
// there); a failure's error carries git's bounded stderr. On a failure both are
// scrubbed of credential URLs (ScrubURLCredentials) before they are cut.
// Under a test binary, networked git is restricted to local transports with
// no credential helper (netEnv); outside one nothing about it changes.
func Net(dir string, args ...string) (string, error) {
	return runHooked(dir, true, "", args, nil)
}

// NetEach is Net for a command whose output may run to any length (a
// listing of every ref under a prefix a pusher can fill): each line of
// stdout is handed to each as it is read, without its newline, and none is
// kept, so no bound on the whole output applies. A line longer than maxLine
// bytes is handed cut to its first maxLine. A failure's error carries git's
// bounded stderr, scrubbed, as Net's does.
func NetEach(dir string, maxLine int, each func(line string), args ...string) error {
	w := &lineSink{max: maxLine, each: each}
	_, err := runHooked(dir, true, "", args, w)
	if err == nil {
		w.flush()
	}
	return err
}

// lineSink splits what it is written into lines for each, keeping at most
// max bytes of the line it is in.
type lineSink struct {
	max  int
	each func(string)
	buf  []byte
	open bool
}

func (w *lineSink) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			w.each(string(w.buf))
			w.buf, w.open = w.buf[:0], false
			continue
		}
		w.open = true
		if len(w.buf) < w.max {
			w.buf = append(w.buf, b)
		}
	}
	return len(p), nil
}

// flush hands each the last line when the output did not end in a newline.
func (w *lineSink) flush() {
	if w.open {
		w.each(string(w.buf))
		w.buf, w.open = w.buf[:0], false
	}
}

// NetInput is Net with stdin fed to the command: a `git fetch --stdin` reads
// its refspecs there, one per line, so however many a fetch names, none
// lengthens its command line past the system's argument limit.
func NetInput(dir, stdin string, args ...string) (string, error) {
	return runHooked(dir, true, stdin, args, nil)
}

// LocalGit runs one git command that reaches no remote (a commit, a read of
// origin's URLs) from the checkout at dir, with the repository's hooks running
// under the developer's own configuration exactly as their own git runs them:
// ScrubbedEnv, GIT_TERMINAL_PROMPT=0 and core.quotePath=false, and nothing
// else, so none of Net's network settings (its low-speed bounds, which reach a
// hook through GIT_CONFIG_PARAMETERS, and its batch-mode ssh command) changes
// how a hook of the repository's runs. Its bounds and its output are Net's.
func LocalGit(dir string, args ...string) (string, error) {
	return runHooked(dir, false, "", args, nil)
}

// NetContext is Net bounded by ctx as well as by NetTimeout: a caller with a
// deadline of its own (the shell guard reading the merge mode inside a host's
// hook) has git killed when ctx ends, and the error says so. It runs with the
// person's own configuration and credentials, as Net does, never prompting.
//
// Unlike Net it runs none of the checkout's hooks (core.hooksPath=/dev/null):
// its callers read, and have no hook of the repository's to honour, so a
// fetch it makes runs no program the checkout chose. Once ctx ends, Wait
// waits netContextWaitDelay for a helper git started (a remote-https) to let
// go of the output, then returns, so the caller's deadline holds.
func NetContext(ctx context.Context, dir string, args ...string) (string, error) {
	return runHookedContext(ctx, dir, true, "", append([]string{"-c", "core.hooksPath=/dev/null"}, args...), nil)
}

// netContextWaitDelay is NetContext's grace, once its context has ended, for
// git and the helpers it started to close the output. It is far longer than
// contextWaitDelay because exec's own copy of the output runs under it (this
// runner does not read to EOF first), so a short grace on a loaded machine
// could cut an answer git already gave; a cut answer is an error, which the
// callers read as the cautious one.
const netContextWaitDelay = 2 * time.Second

// runHooked is Net (network true) and LocalGit (false), with stdin fed to
// the command when it is not "". With sink set, stdout is written to it as
// it is read, none is kept, and the output returned is "".
func runHooked(dir string, network bool, stdin string, args []string, sink io.Writer) (string, error) {
	return runHookedContext(context.Background(), dir, network, stdin, args, sink)
}

// beforeHooked is the check SetBeforeHookedGit sets, nil for none.
var beforeHooked atomic.Pointer[func(args []string) error]

// NotRunError is a git command abcd did not run. It has two makers: the
// check SetBeforeHookedGit set refused the command (askBeforeHooked, for a
// hooked or networked runner, and askIfContentRuns, for an isolated one), or,
// for a networked command, the read of core.sshCommand that decides the ssh
// command it runs gave no answer of git's (hookedCommand). Verb is its
// subcommand; Err is the check's refusal or the failed read. Whatever exit
// status Err wraps is not the command's: ExitCode reads none.
type NotRunError struct {
	Verb string
	Err  error
}

func (e *NotRunError) Error() string { return "git " + e.Verb + " was not run: " + e.Err.Error() }

func (e *NotRunError) Unwrap() error { return e.Err }

// ExitCode is the exit status of the git err reports, -1 when there is none:
// a git that was not run (a NotRunError) has none, whatever its cause wraps,
// so a caller that reads a status as an answer (none, no, too old) reads it
// as a failed read instead.
func ExitCode(err error) int {
	var nr *NotRunError
	if errors.As(err, &nr) {
		return -1
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// ErrBeforeHookedGitSet is SetBeforeHookedGit's refusal of a second check
// while one is set: one drive runs in a process.
var ErrBeforeHookedGitSet = errors.New("a check before every hooked or networked git is already set in this process, and one drive runs in a process")

// SetBeforeHookedGit sets a check every git command abcd starts with the
// repository's hooks, merge drivers or content filters on, or that reaches a
// remote, asks first, before the command is built: Net, NetEach, NetInput,
// NetContext and LocalGit (runHookedContext), RunCommitCommandBounded and
// RunCommitCommandBoundedAsStored (runCommitBounded), WorkingTreeCommand and
// WorkingTreeCommandContext, and RunLimitedNetContext. An isolated command
// (Run and its kin) has its hooks and the other programs ExecPins names
// pinned off, but not content filters, textconv and external diff programs
// or merge drivers, so one that can start one of those
// (LeavesContentProgramsOn: a hash-object, a status with the filters left
// on, a diff of the working tree that does not blank them, a subcommand it
// does not know) asks too, and one that blanks them or reads no content
// through them does not. An error from
// the check is the command's, and git does not run. A drive sets it to its
// comparison of what it holds while it runs (implement/loop's
// heldBeforeGit); it is given the command's arguments. One drive runs in a
// process, so a check already set is not replaced: the call is refused with
// ErrBeforeHookedGitSet. The returned clear removes this check and only this
// one. The comparison's own reads never ask it: they run through
// NetAskContext, WorkingTreeCommitCommand built directly, and isolated
// `config`, `var` and `rev-parse` reads, which start no program. With no
// check set nothing changes.
func SetBeforeHookedGit(f func(args []string) error) (clear func(), err error) {
	if f == nil {
		return nil, errors.New("no check was given")
	}
	p := &f
	if !beforeHooked.CompareAndSwap(nil, p) {
		return nil, ErrBeforeHookedGitSet
	}
	return func() { beforeHooked.CompareAndSwap(p, nil) }, nil
}

// askBeforeHooked asks the check SetBeforeHookedGit set, when one is set,
// for the git command args would run: its error is a NotRunError naming the
// subcommand.
func askBeforeHooked(args []string) error {
	if f := beforeHooked.Load(); f != nil {
		if err := (*f)(args); err != nil {
			return &NotRunError{Verb: gitVerb(args), Err: err}
		}
	}
	return nil
}

// runHookedContext is runHooked under parent, after the check
// SetBeforeHookedGit set, when one is set.
func runHookedContext(parent context.Context, dir string, network bool, stdin string, args []string, sink io.Writer) (string, error) {
	if err := askBeforeHooked(args); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(parent, NetTimeout)
	defer cancel()
	cmd := hookedCommand(ctx, dir, network, args)
	if cmd.Err != nil {
		if perr := parent.Err(); perr != nil {
			return "", fmt.Errorf("git %s: %w", gitVerb(args), perr)
		}
		return "", notStarted(args, cmd.Err)
	}
	if parent.Done() != nil {
		cmd.WaitDelay = netContextWaitDelay
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if sink != nil {
		cmd.Stdout = sink
	}
	what := "git " + gitVerb(args)
	if err := cmd.Run(); err != nil {
		if perr := parent.Err(); perr != nil {
			return "", fmt.Errorf("%s: %w", what, perr)
		}
		// Scrubbed BEFORE the cut: a cut across a credential URL leaves a
		// fragment with no host, which the scrubber no longer recognises, and
		// the token's start would go out with it.
		msg := ScrubURLCredentials(strings.TrimSpace(stderr.String()))
		if len(msg) > netStderrCap {
			msg = msg[:netStderrCap]
		}
		out := ScrubURLCredentials(stdout.String())
		if len(out) > netOutputCap {
			out = out[:netOutputCap]
		}
		return out, fmt.Errorf("%s: %v (%s)", what, err, msg)
	}
	if stdout.Len() > netOutputCap {
		return "", fmt.Errorf("%s wrote more than %d bytes", what, netOutputCap)
	}
	return stdout.String(), nil
}

// notStarted is the error of a hooked or networked command that never
// started (err, its cmd.Err), led by the git command it is, as a failure of
// cmd.Run is: a NotRunError already is, and is returned as it is.
func notStarted(args []string, err error) error {
	var nr *NotRunError
	if errors.As(err, &nr) {
		return err
	}
	return fmt.Errorf("git %s: %w", gitVerb(args), err)
}

// hookedCommand is the git command runHookedContext runs: Net's (network
// true) or LocalGit's, bound to ctx, with no stdin or output set. A networked
// command whose read of core.sshCommand gave no answer of git's carries that
// as its Err (a NotRunError), so it never starts.
func hookedCommand(ctx context.Context, dir string, network bool, args []string) *exec.Cmd {
	full := []string{"-c", "core.quotePath=false"}
	if network {
		full = append(full, "-c", "http.lowSpeedLimit="+netLowSpeedLimit, "-c", "http.lowSpeedTime="+netLowSpeedTime)
	}
	full = append(append(full, "-C", dir), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0")
	if !network {
		cmd.Env = narrowHooked(cmd.Env)
	}
	if network {
		cmd.Env = netEnv(cmd.Env)
		ssh, err := netSSHCommand(ctx, dir)
		if err != nil {
			cmd.Err = &NotRunError{Verb: gitVerb(args), Err: err}
		} else if ssh != "" {
			cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+ssh)
		}
	}
	return cmd
}

// NetAskContext runs one git command that reads and writes nothing (a
// `config` listing, `var`, `rev-parse --git-path`) in exactly the
// environment Net, NetEach and NetInput give git: the person's own home,
// configuration and credentials, the repository's hooks not pinned off. It
// is for a caller that must know what that environment reads (the settings,
// the hooks directory a push runs, the attributes files), never for a
// command that runs a hook or reaches a remote. It is bound to ctx, git is
// killed when ctx ends, and its output past limit bytes is an error. The
// error of a git that ran and failed wraps its *exec.ExitError, so the
// caller can read the exit status; its stderr is bounded and scrubbed of
// credentials.
func NetAskContext(ctx context.Context, dir string, limit int, args ...string) (string, error) {
	cmd := hookedCommand(ctx, dir, true, args)
	if cmd.Err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("git %s: %w", gitVerb(args), ctxErr)
		}
		return "", notStarted(args, cmd.Err)
	}
	cmd.WaitDelay = netContextWaitDelay
	out := &capWriter{remaining: limit}
	errOut := &capWriter{remaining: netStderrCap}
	cmd.Stdout, cmd.Stderr = out, errOut
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("git %s: %w", gitVerb(args), ctxErr)
		}
		return string(out.buf), fmt.Errorf("git %s: %w (%s)", gitVerb(args), err, ScrubURLCredentials(strings.TrimSpace(string(errOut.buf))))
	}
	if out.overflowed {
		return "", fmt.Errorf("git %s: output exceeded the %d-byte cap", gitVerb(args), limit)
	}
	return string(out.buf), nil
}

// NetEnviron is the environment NetAskContext, Net, NetEach and NetInput
// give git, less the ssh command Net composes: for a caller that must read
// what that environment tells git (its home, its XDG_CONFIG_HOME,
// GIT_ATTR_NOSYSTEM).
func NetEnviron() []string { return netEnv(append(ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0")) }

// testTransportGuard is the configuration netEnv adds to a networked git
// command run by a test binary, so a test can neither reach a real forge nor
// ask anyone for a credential (iss-2610170718356567). An empty
// credential.helper clears every helper a lower scope named, the system
// config's osxkeychain among them, so git calls none. protocol.allow=never
// refuses every transport that names no policy of its own, and http, https,
// ssh and git are named never outright, so a config that allows one cannot
// let it back in; git refuses them before it connects. A local path, a
// file:// URL and a URL a repository's url.<base>.insteadOf rewrites to one
// keep git's own default for file. A repository's own policy for another
// protocol (a test's ext:: remote standing in for a slow one) still applies.
var testTransportGuard = [][2]string{
	{"credential.helper", ""},
	{"protocol.allow", "never"},
	{"protocol.http.allow", "never"},
	{"protocol.https.allow", "never"},
	{"protocol.ssh.allow", "never"},
	{"protocol.git.allow", "never"},
	{"protocol.file.allow", "user"},
}

// netGuardOff switches the guard off for a test of this package that drives
// a transport through a fake (the ssh command Net composes). It is atomic,
// since a networked read can run on a goroutine of its caller's (a drive's
// comparison) while a test sets it back.
var netGuardOff atomic.Bool

// netEnv is the environment of every git command abcd runs that may reach a
// remote, built on env, the environment it would otherwise run with: Net,
// NetEach, NetInput (and through them RemoteHeadBranch, the landing and the
// reservation code) and RunLimitedNetContext. Outside a test binary it is env
// unchanged, so the person's own configuration and credential helpers apply
// exactly as their own git applies them. Under a test binary
// (testing.Testing) networked git is restricted to local transports with no
// credential helper: testTransportGuard rides in the GIT_CONFIG_COUNT form,
// after any entries env already carries, so it outranks every config file.
func netEnv(env []string) []string { return netEnvFor(env, testing.Testing() && !netGuardOff.Load()) }

// netEnvFor is netEnv with the test-binary question answered by underTest.
func netEnvFor(env []string, underTest bool) []string {
	if !underTest {
		return env
	}
	n := 0
	out := make([]string, 0, len(env)+2*len(testTransportGuard)+1)
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "GIT_CONFIG_COUNT="); ok {
			n, _ = strconv.Atoi(v)
		}
		// GIT_ALLOW_PROTOCOL outranks every protocol.*.allow entry, so one
		// inherited from the developer's shell would let the guarded
		// transports back in; under a test it is dropped.
		if strings.HasPrefix(kv, "GIT_ALLOW_PROTOCOL=") {
			continue
		}
		out = append(out, kv)
	}
	for i, kv := range testTransportGuard {
		out = append(out, "GIT_CONFIG_KEY_"+strconv.Itoa(n+i)+"="+kv[0], "GIT_CONFIG_VALUE_"+strconv.Itoa(n+i)+"="+kv[1])
	}
	return append(out, "GIT_CONFIG_COUNT="+strconv.Itoa(n+len(testTransportGuard)))
}

// netSSHCommand is the ssh command Net runs git's ssh transport through: ssh
// in batch mode with the connect and keepalive bounds, or "" when the developer chose their
// own (GIT_SSH_COMMAND, GIT_SSH, or core.sshCommand), which is left as it is.
//
// Its read of core.sshCommand runs under ctx, the command's own, so it cannot
// hold a caller past its deadline (NetAskContext's, inside a drive's
// comparison), and reads git's answer to its end before Wait
// (procgroup.RunToEOF), so a program git left holding the output delays the
// answer and never cuts it. The default is composed only on git's own answer
// that none is set: exit status 1 with nothing written, or exit 0 with nothing
// but white space (what a setting given an empty value answers). Any other
// outcome (the context ended, another exit, an answer past its cap) is an
// error naming the read, and the networked command is not started: exporting
// the default then would run git through an ssh command other than the one the
// repository or the person set, since GIT_SSH_COMMAND outranks
// core.sshCommand.
func netSSHCommand(ctx context.Context, dir string) (string, error) {
	if os.Getenv("GIT_SSH_COMMAND") != "" || os.Getenv("GIT_SSH") != "" {
		return "", nil
	}
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "config", "--get", "core.sshCommand")
	cmd.Env = ScrubbedEnv()
	if ctx.Done() != nil {
		cmd.WaitDelay = contextWaitDelay
	}
	out := &capWriter{remaining: netSSHReadCap}
	errOut := &capWriter{remaining: netStderrCap}
	err := procgroup.RunToEOF(ctx, cmd, out, errOut)
	var ee *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return "", fmt.Errorf("the read of core.sshCommand, which decides the ssh command git runs, did not finish: %w", ctx.Err())
	case err == nil && out.overflowed:
		return "", fmt.Errorf("the read of core.sshCommand, which decides the ssh command git runs, answered more than %d bytes; remedy: %s", netSSHReadCap, netSSHReadRemedy)
	case err == nil && strings.TrimSpace(string(out.buf)) != "":
		return "", nil
	case err == nil, errors.As(err, &ee) && ee.ExitCode() == 1 && len(out.buf) == 0:
		return "ssh " + netSSHOptions, nil
	}
	return "", fmt.Errorf("the read of core.sshCommand, which decides the ssh command git runs, failed: %w (%s); remedy: %s",
		err, ScrubURLCredentials(strings.TrimSpace(string(errOut.buf))), netSSHReadRemedy)
}

// netSSHReadRemedy is what to do about a read of core.sshCommand that failed
// or answered past its cap.
const netSSHReadRemedy = "check that git can read this repository's settings and your own " +
	"(git config --show-origin --get core.sshCommand names where it is set), then run the command again"

// netSSHReadCap bounds the answer of the read of core.sshCommand: one command
// line.
const netSSHReadCap = 64 << 10

// gitVerb is the subcommand among args, past git's global options
// (Subcommand), for an error to name.
func gitVerb(args []string) string {
	if v := Subcommand(args); v != "" {
		return v
	}
	return "git"
}

// hookedEnv narrows what a repository's hooks are given (SetHookedEnv).
var hookedEnv atomic.Pointer[func([]string) []string]

// SetHookedEnv narrows the environment of the commands that run a repository's
// hooks on this process's behalf, the push's excepted, since a push is what needs
// the person's credentials; nil leaves it as it is.
func SetHookedEnv(f func([]string) []string) {
	if f == nil {
		hookedEnv.Store(nil)
		return
	}
	hookedEnv.Store(&f)
}

// narrowHooked is env as a hooked command that opens no connection gets it.
func narrowHooked(env []string) []string {
	if f := hookedEnv.Load(); f != nil {
		return (*f)(env)
	}
	return env
}

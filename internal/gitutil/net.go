package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
var beforeHooked atomic.Pointer[func() error]

// SetBeforeHookedGit sets a check every command runHookedContext runs asks
// first (Net, NetEach, NetInput, NetContext and LocalGit: each git command
// that runs a repository's hooks or reaches a remote): an error from it is
// the command's, and git does not run. A drive sets it to its comparison of
// what it holds while it runs (implement/loop's heldBeforeGit); nil clears it.
// Commands that read only (NetAskContext, the isolated readers) never ask it.
func SetBeforeHookedGit(f func() error) {
	if f == nil {
		beforeHooked.Store(nil)
		return
	}
	beforeHooked.Store(&f)
}

// runHookedContext is runHooked under parent, after the check
// SetBeforeHookedGit set, when one is set.
func runHookedContext(parent context.Context, dir string, network bool, stdin string, args []string, sink io.Writer) (string, error) {
	if f := beforeHooked.Load(); f != nil {
		if err := (*f)(); err != nil {
			return "", fmt.Errorf("git %s was not run: %w", gitVerb(args), err)
		}
	}
	ctx, cancel := context.WithTimeout(parent, NetTimeout)
	defer cancel()
	cmd := hookedCommand(ctx, dir, network, args)
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

// hookedCommand is the git command runHookedContext runs: Net's (network
// true) or LocalGit's, bound to ctx, with no stdin or output set.
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
		if ssh := netSSHCommand(dir); ssh != "" {
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
// a transport through a fake (the ssh command Net composes).
var netGuardOff bool

// netEnv is the environment of every git command abcd runs that may reach a
// remote, built on env, the environment it would otherwise run with: Net,
// NetEach, NetInput (and through them RemoteHeadBranch, the landing and the
// reservation code) and RunLimitedNetContext. Outside a test binary it is env
// unchanged, so the person's own configuration and credential helpers apply
// exactly as their own git applies them. Under a test binary
// (testing.Testing) networked git is restricted to local transports with no
// credential helper: testTransportGuard rides in the GIT_CONFIG_COUNT form,
// after any entries env already carries, so it outranks every config file.
func netEnv(env []string) []string { return netEnvFor(env, testing.Testing() && !netGuardOff) }

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
func netSSHCommand(dir string) string {
	if os.Getenv("GIT_SSH_COMMAND") != "" || os.Getenv("GIT_SSH") != "" {
		return ""
	}
	cmd := exec.Command("git", "-C", dir, "config", "--get", "core.sshCommand")
	cmd.Env = ScrubbedEnv()
	if out, err := cmd.Output(); err == nil && strings.TrimSpace(string(out)) != "" {
		return ""
	}
	return "ssh " + netSSHOptions
}

// gitVerb is the subcommand among args, past any leading -c options, for an
// error to name.
func gitVerb(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			i++
			continue
		}
		return args[i]
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

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

// runHooked is Net (network true) and LocalGit (false), with stdin fed to
// the command when it is not "". With sink set, stdout is written to it as
// it is read, none is kept, and the output returned is "".
func runHooked(dir string, network bool, stdin string, args []string, sink io.Writer) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), NetTimeout)
	defer cancel()
	full := []string{"-c", "core.quotePath=false"}
	if network {
		full = append(full, "-c", "http.lowSpeedLimit="+netLowSpeedLimit, "-c", "http.lowSpeedTime="+netLowSpeedTime)
	}
	full = append(append(full, "-C", dir), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(ScrubbedEnv(), "GIT_TERMINAL_PROMPT=0")
	if network {
		cmd.Env = netEnv(cmd.Env)
		if ssh := netSSHCommand(dir); ssh != "" {
			cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+ssh)
		}
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if sink != nil {
		cmd.Stdout = sink
	}
	what := "git " + args[0]
	if err := cmd.Run(); err != nil {
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
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "GIT_CONFIG_COUNT="); ok {
			n, _ = strconv.Atoi(v)
		}
	}
	out := append(make([]string, 0, len(env)+2*len(testTransportGuard)+1), env...)
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

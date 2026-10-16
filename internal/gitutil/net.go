package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func Net(dir string, args ...string) (string, error) {
	return runHooked(dir, true, args)
}

// LocalGit runs one git command that reaches no remote (a commit, a read of
// origin's URLs) from the checkout at dir, with the repository's hooks running
// under the developer's own configuration exactly as their own git runs them:
// ScrubbedEnv, GIT_TERMINAL_PROMPT=0 and core.quotePath=false, and nothing
// else, so none of Net's network settings (its low-speed bounds, which reach a
// hook through GIT_CONFIG_PARAMETERS, and its batch-mode ssh command) changes
// how a hook of the repository's runs. Its bounds and its output are Net's.
func LocalGit(dir string, args ...string) (string, error) {
	return runHooked(dir, false, args)
}

// runHooked is Net (network true) and LocalGit (false).
func runHooked(dir string, network bool, args []string) (string, error) {
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
		if ssh := netSSHCommand(dir); ssh != "" {
			cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND="+ssh)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
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

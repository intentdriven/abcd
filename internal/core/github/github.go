// Package github is abcd's one door to GitHub's API: the `gh` runner every
// forge read and write goes through, and the parser that turns an origin URL
// into the only repository a request may name (spc-2610120557432880, step 1;
// one-canonical-primitive). It was lifted out of internal/core/ahoy, whose
// remote verbs, and the site verb through them, were its only callers, so the
// embargo verbs share it rather than growing a second runner.
//
// The call goes through `gh` rather than a raw HTTP client on purpose: it
// inherits the caller's own authenticated identity, so abcd never handles a
// token and a write is made by the person who invoked it, which is what
// adr-44's "no uninvited remote mutation" rests on.
package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// Host is the API host every request names EXPLICITLY. Without it `gh api`
// picks its host from GH_HOST, or from whichever single host the caller's gh is
// authenticated to, so the github.com-only rule ParseRemote enforces would
// constrain the PATH while the environment chose the ENDPOINT. An ambient
// GH_HOST (a repo direnv file, a CI job, an agent harness) would then send an
// authenticated write, with whatever credential gh holds for that host, to a
// machine the origin URL never named.
const Host = "github.com"

// Timeout bounds each `gh` call. A subprocess reading from the network with no
// bound is a verb that hangs a session indefinitely on a stalled connection.
var Timeout = 30 * time.Second

// ErrTimeout is what Run's error wraps when gh did not answer within Timeout.
var ErrTimeout = errors.New("gh did not answer")

// MaxOutputBytes caps what is read back from `gh`, stdout and stderr each
// (trust boundary): the response is remote-controlled, and an unbounded read
// into memory is the shape a hostile or broken endpoint exploits.
const MaxOutputBytes = 1 << 20

// plainNameRe is the charset a GitHub owner or repository name may use. It is
// applied to BOTH segments before either reaches an API path, because the path
// is assembled by concatenation: a segment carrying a slash or a `..` would not
// merely name a different repository, it would address a different ENDPOINT.
// Anchored, and deliberately narrower than GitHub's own acceptance: refusing a
// name abcd cannot vouch for costs one error message, and the alternative costs
// a request nobody intended.
var plainNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// PlainName reports whether s is an owner or repository name abcd will put
// into an API path.
func PlainName(s string) bool { return plainNameRe.MatchString(s) }

// ErrNotGitHub is ParseRemote's refusal of a URL that is not on github.com at
// all, told apart from a github.com URL that names no one repository.
var ErrNotGitHub = errors.New("the origin remote is not a github.com URL, and this verb speaks only the GitHub API")

// Remote is a remote URL split into what names the repository it reaches:
// Form is "scp" for git's scp-like `[user@]host:path`, else the URL's
// scheme; User and Port are as written (a password is never kept); Host is
// lower-cased; Path is the repository's path with its slashes and a `.git`
// suffix trimmed.
type Remote struct {
	Form, User, Host, Port, Path string
}

// SplitRemote splits a remote URL that names a network host, the forms git
// takes (spc-2610161044461448, step 10): the scp-like `[user@]host:path`
// and a scheme URL with a host, its userinfo and port included. A local
// path, a `file://` URL, and anything with no host are not one (ok false).
func SplitRemote(raw string) (r Remote, ok bool) {
	if !strings.Contains(raw, "://") {
		colon := strings.Index(raw, ":")
		if colon <= 0 || strings.Contains(raw[:colon], "/") {
			return Remote{}, false
		}
		host := raw[:colon]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			r.User, host = host[:at], host[at+1:]
		}
		if host == "" {
			return Remote{}, false
		}
		r.Form, r.Host, r.Path = "scp", strings.ToLower(host), trimRepoPath(raw[colon+1:])
		return r, true
	}
	u, err := neturl.Parse(raw)
	if err != nil || u.Scheme == "file" || u.Hostname() == "" {
		return Remote{}, false
	}
	r = Remote{Form: strings.ToLower(u.Scheme), Host: strings.ToLower(u.Hostname()), Port: u.Port(), Path: trimRepoPath(u.Path)}
	if u.User != nil {
		r.User = u.User.Username()
	}
	return r, true
}

func trimRepoPath(p string) string {
	return strings.TrimSuffix(strings.TrimSuffix(strings.Trim(p, "/"), ".git"), "/")
}

// OnGitHub reports whether r reaches github.com by name: the host github.com
// in any form, or ssh.github.com (GitHub's ssh over port 443) over ssh.
func (r Remote) OnGitHub() bool {
	switch r.Host {
	case Host:
		return true
	case "ssh." + Host:
		return r.Form == "scp" || r.Form == "ssh" || r.Form == "git+ssh" || r.Form == "ssh+git"
	}
	return false
}

// NamesHostedRepository reports whether url names a network host and exactly
// two path parts, an owner and a name, on any host: the shape a github.com
// URL takes, so a caller can tell a repository on a host it does not
// recognise (an ssh host alias, another forge) from a local path.
func NamesHostedRepository(url string) bool {
	r, ok := SplitRemote(url)
	return ok && len(strings.Split(r.Path, "/")) == 2
}

// ParseRemote splits a github.com remote URL into owner and repository. It
// accepts the forms git takes for github.com (SplitRemote): the three it
// writes (`https://github.com/owner/repo[.git]`, `git@github.com:owner/repo[.git]`,
// `ssh://git@github.com/owner/repo`), and with them userinfo, a port,
// `ssh.github.com` over ssh, ssh or scp with no user, and a host in any case.
// It refuses everything else, without echoing the URL's segments: the URL is
// repo-controlled input, and a refusal is output like any other.
func ParseRemote(url string) (owner, name string, err error) {
	r, ok := SplitRemote(url)
	if !ok || !r.OnGitHub() {
		return "", "", ErrNotGitHub
	}
	parts := strings.Split(r.Path, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("the origin remote does not name exactly one owner and one repository")
	}
	owner, name = parts[0], parts[1]
	if !PlainName(owner) || !PlainName(name) {
		return "", "", fmt.Errorf("the origin remote's owner or repository name is not a plain GitHub name, " +
			"so it will not be used to build an API path")
	}
	return owner, name, nil
}

// API runs `gh api --hostname github.com <args...>` under cwd: Run with the
// host named by construction, so no caller can forget it.
func API(cwd string, stdin []byte, args ...string) ([]byte, error) {
	return Run(cwd, stdin, append([]string{"api", "--hostname", Host}, args...)...)
}

// Run executes one `gh` subcommand under cwd and returns its bounded stdout.
// The environment is passed through unchanged; scrubbing it would blind gh to
// the credentials that make the caller the actor. A missing gh is refused with
// the tool registry's explanation (itd-63): what gh is, that these verbs
// require it, and the exact install step.
func Run(cwd string, stdin []byte, args ...string) ([]byte, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, tools.Missing(errors.New("the GitHub CLI (gh) is not on PATH; abcd speaks to GitHub through it so that a "+
			"remote write is made by your own authenticated identity, never by a token abcd holds"), "gh", tools.GitHubSettings)
	}
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &out, remaining: MaxOutputBytes}
	cmd.Stderr = &limitedWriter{w: &errBuf, remaining: MaxOutputBytes}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w within %s", ErrTimeout, Timeout)
		}
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%w (%s)", err, msg)
	}
	return out.Bytes(), nil
}

// limitedWriter caps what a subprocess can put into memory. It DISCARDS the
// overflow rather than erroring the command: the cap is a memory bound, and
// turning a large-but-valid response into a failure would refuse a repository
// for being verbose.
type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		return len(p), nil
	}
	if len(p) > l.remaining {
		if _, err := l.w.Write(p[:l.remaining]); err != nil {
			return 0, err
		}
		l.remaining = 0
		return len(p), nil
	}
	l.remaining -= len(p)
	return l.w.Write(p)
}

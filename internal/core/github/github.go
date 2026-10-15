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
const Timeout = 30 * time.Second

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

// ParseRemote splits a github.com remote URL into owner and repository. It
// accepts the forms git itself writes (`https://github.com/owner/repo[.git]`,
// `git@github.com:owner/repo[.git]`, `ssh://git@github.com/owner/repo`) and
// refuses everything else, without echoing the URL's segments: the URL is
// repo-controlled input, and a refusal is output like any other.
func ParseRemote(url string) (owner, name string, err error) {
	rest := ""
	switch {
	case strings.HasPrefix(url, "git@github.com:"):
		rest = strings.TrimPrefix(url, "git@github.com:")
	case strings.HasPrefix(url, "ssh://git@github.com/"):
		rest = strings.TrimPrefix(url, "ssh://git@github.com/")
	case strings.HasPrefix(url, "https://github.com/"):
		rest = strings.TrimPrefix(url, "https://github.com/")
	case strings.HasPrefix(url, "http://github.com/"):
		rest = strings.TrimPrefix(url, "http://github.com/")
	default:
		return "", "", fmt.Errorf("the origin remote is not a github.com URL, and this verb speaks only the GitHub API")
	}
	rest = strings.TrimSuffix(strings.TrimSuffix(strings.Trim(rest, "/"), ".git"), "/")
	parts := strings.Split(rest, "/")
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
			return nil, fmt.Errorf("gh did not answer within %s", Timeout)
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

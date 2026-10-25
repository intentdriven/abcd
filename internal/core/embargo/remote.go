package embargo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// remoteAdvisory is the part of a GitHub repository security advisory this
// package reads. GitHub's answer carries many more fields; they are ignored
// rather than refused, because the answer is GitHub's shape, not abcd's.
type remoteAdvisory struct {
	GHSAID      string  `json:"ghsa_id"`
	CVEID       *string `json:"cve_id"`
	Summary     string  `json:"summary"`
	Description *string `json:"description"`
	Severity    *string `json:"severity"`
	State       string  `json:"state"`
	PublishedAt *string `json:"published_at"`
	ClosedAt    *string `json:"closed_at"`

	Vulnerabilities []remoteVulnerability `json:"vulnerabilities"`
}

// endedAt is the timestamp GitHub gives for the end its state reports:
// published_at for published, closed_at for closed, nil otherwise.
func (a remoteAdvisory) endedAt() *string {
	switch a.State {
	case "published":
		return a.PublishedAt
	case "closed":
		return a.ClosedAt
	}
	return nil
}

// remoteVulnerability is one affected package of an advisory, as GitHub
// carries it; a field GitHub leaves unset is null.
type remoteVulnerability struct {
	Package                *Package `json:"package"`
	VulnerableVersionRange *string  `json:"vulnerable_version_range"`
	PatchedVersions        *string  `json:"patched_versions"`
	VulnerableFunctions    []string `json:"vulnerable_functions"`
}

func (a remoteAdvisory) description() string {
	if a.Description == nil {
		return ""
	}
	return *a.Description
}

// advisoriesPath is the advisory collection of repo ("owner/name", already
// held to github.PlainName by ParseRemote or the state's validation).
func advisoriesPath(repo string) string { return "repos/" + repo + "/security-advisories" }

// The scope lines of the spec's request table, named on a refusal GitHub
// answers with a missing scope, a 403 or a 404.
const (
	scopeRead = "reading repository advisories needs a classic token with `repo` or `repository_advisories:read`, " +
		"or a fine-grained token with Repository security advisories: read, on an account that is an admin or " +
		"security manager of the repository, or a collaborator on the advisory"
	scopeWrite = "changing a repository advisory needs a classic token with `repo` or `repository_advisories:write`, " +
		"or a fine-grained token with Repository security advisories: write, on an account that is an admin or " +
		"security manager of the repository, or a collaborator on the advisory"
	scopeRemedy = "grant it with `gh auth refresh -s repository_advisories:write` (abcd never runs gh auth itself), then run it again"

	signedOutReason = "gh is signed in to no GitHub account, or GitHub rejects its token, so no advisory can be read"
	signedOutRemedy = "sign in with your own identity with `gh auth login` (abcd never runs gh auth itself), then run it again"
)

// ghAuthRequired is the exit status gh documents for a command that needs
// authentication and has none (`gh help exit-codes`).
const ghAuthRequired = 4

// ghSignedOut reports whether a failed gh call failed for want of an
// accepted sign-in: gh's own authentication-required exit status, which it
// gives when no account is signed in and no token is set, or GitHub's
// HTTP 401, which gh reports when the token it holds is rejected. The
// status is matched in err's text, which carries gh's own words, but only
// the "HTTP 401" pattern is looked for, and nothing of that text is echoed
// in the refusal: gh's words can carry an identity.
func ghSignedOut(err error) bool {
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == ghAuthRequired {
		return true
	}
	return strings.Contains(err.Error(), "HTTP 401")
}

// remoteError explains a failed gh call. A missing gh passes through with the
// tool registry's explanation; a gh signed in to no account, or holding a
// token GitHub rejects (ghSignedOut), and a refusal GitHub gives in words
// that name a scope or access, each name the person's own remedy and refuse
// a run whole (accessRefusal). A bare 403 or 404 names the scope too, but is
// told apart (notShownRefusal): on one advisory's read it is that
// advisory's answer, not the run's.
func remoteError(err error, write bool) error {
	var missing *tools.MissingError
	if errors.As(err, &missing) {
		return err
	}
	if ghSignedOut(err) {
		return &accessRefusal{&RefusalError{Reason: signedOutReason, Remedy: signedOutRemedy}}
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	scope := scopeRead
	if write {
		scope = scopeWrite
	}
	refused := &RefusalError{Reason: "GitHub refused the request: " + scope, Remedy: scopeRemedy}
	if strings.Contains(low, "scope") || strings.Contains(low, "not accessible") {
		return &accessRefusal{refused}
	}
	if strings.Contains(msg, "403") || strings.Contains(msg, "404") {
		return &notShownRefusal{refused}
	}
	return refuse("gh could not complete the request to GitHub ("+msg+")", "settle what gh reports, then run it again")
}

// accessRefusal is remoteError's refusal for want of a sign-in, a scope or
// access: a RefusalError (errors.As finds it), told apart so a caller that
// reads many advisories one at a time refuses whole on it, as on a missing
// gh, instead of taking it for one advisory GitHub could not show.
type accessRefusal struct{ *RefusalError }

func (e *accessRefusal) Unwrap() error { return e.RefusalError }

// notShownRefusal is remoteError's refusal for a bare 403 or 404: GitHub
// shows the thing asked for to no one this token speaks for. A token
// without the scope meets it on every request; a deleted advisory, or one
// this account cannot see, meets it on that advisory's read alone. A caller
// making one request refuses on it as on accessRefusal (it is the same
// RefusalError); readHeld leaves that one advisory unread instead, and
// when every read it made met it, asks the listing whether the scope is
// held (scopeAnswers).
type notShownRefusal struct{ *RefusalError }

func (e *notShownRefusal) Unwrap() error { return e.RefusalError }

// refusesWhole reports whether err, from a gh call, is one that every other
// call would meet too: gh missing, gh signed in to no account or holding a
// token GitHub rejects, or GitHub refusing in words that name a scope or
// access. It is passed up whole, with its remedy, never read as one
// advisory's answer. A bare 403 or 404 (notShownRefusal) is not one.
func refusesWhole(err error) bool {
	var missing *tools.MissingError
	var access *accessRefusal
	return errors.As(err, &missing) || errors.As(err, &access)
}

// listAdvisories lists the repository's advisories in one state, every page:
// `gh api --paginate` writes each page's JSON array back to back, so the
// answer is read as a stream of arrays. An answer that is not that (including
// one cut at the runner's output cap) refuses rather than reading part of it.
func listAdvisories(cwd, repo, state string) ([]remoteAdvisory, error) {
	out, err := github.API(cwd, nil, "--paginate", "-X", "GET", advisoriesPath(repo),
		"-f", "state="+state, "-f", "per_page=100")
	if err != nil {
		return nil, remoteError(err, false)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	all := []remoteAdvisory{}
	for {
		var items []remoteAdvisory
		err := dec.Decode(&items)
		if errors.Is(err, io.EOF) {
			return all, nil
		}
		if err != nil {
			return nil, refuse("GitHub's answer to the advisory listing is not the JSON this abcd reads, or was cut at its size limit; nothing was recorded",
				"run it again; if it persists, the repository holds more advisory text than one listing can carry")
		}
		all = append(all, items...)
	}
}

// getAdvisory reads one advisory. The answer is GitHub's, read leniently for
// its shape and held to the id asked for.
func getAdvisory(cwd, repo, ghsa string) (remoteAdvisory, error) {
	out, err := github.API(cwd, nil, advisoriesPath(repo)+"/"+ghsa)
	if err != nil {
		return remoteAdvisory{}, remoteError(err, false)
	}
	var a remoteAdvisory
	if err := json.Unmarshal(out, &a); err != nil {
		return remoteAdvisory{}, refuse("GitHub's answer for the advisory is not the JSON this abcd reads, or was cut at its size limit",
			"run it again; if it persists, report it")
	}
	if !strings.EqualFold(a.GHSAID, ghsa) {
		return remoteAdvisory{}, refuse("GitHub answered for another advisory than the one asked for", "run it again; if it persists, report it")
	}
	return a, nil
}

// patchAdvisory sends one update of an advisory, its body on stdin as JSON
// (never as -f fields). It is the only write this package makes: close and
// publish each send their updates through it.
func patchAdvisory(cwd, repo, ghsa string, body []byte) error {
	if _, err := github.API(cwd, body, "-X", "PATCH", advisoriesPath(repo)+"/"+ghsa, "--input", "-"); err != nil {
		return remoteError(err, true)
	}
	return nil
}

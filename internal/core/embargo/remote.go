package embargo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
)

// remoteError explains a failed gh call. A missing gh passes through with the
// tool registry's explanation; a refusal GitHub gives for want of a scope or
// access names the scope and the person's own remedy.
func remoteError(err error, write bool) error {
	var missing *tools.MissingError
	if errors.As(err, &missing) {
		return err
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(msg, "403") || strings.Contains(msg, "404") || strings.Contains(low, "scope") ||
		strings.Contains(low, "not accessible") {
		scope := scopeRead
		if write {
			scope = scopeWrite
		}
		return refuse("GitHub refused the request: "+scope, scopeRemedy)
	}
	return refuse("gh could not complete the request to GitHub ("+msg+")", "settle what gh reports, then run it again")
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

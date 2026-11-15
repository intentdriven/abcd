package sweep

// forge.go holds a sweep to origin's one repository: what git says origin is
// fetched from and pushed to, and the forge calls pinned to that repository.
//
// Origin must be one repository: exactly one fetch address, and every push
// address naming the same repository as it (forgerepo.Key, the comparison
// `abcd land` makes, so an https fetch and an ssh push of one repository
// agree). A remote delete is held otherwise (remote.go), since the sweep
// could not check the repository its push would change.
//
// Every forge call the package makes names that repository: its host, owner
// and name are read from origin's address, the forge is asked for the
// repository by that name (`gh repo view <host/owner/name>`), every one of
// origin's addresses must name the repository the forge answers for
// (forgerepo.Stranger, the check `abcd land` makes), and every later call
// carries `--repo=<host/owner/name>` as one argument. When any step fails, no
// forge call is made: the worktrees are judged merged from git alone, every
// branch on origin is kept, and the reservation workflow is not dispatched.

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/forgerepo"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// originAddresses reads origin's fetch and push addresses; it is
// readOriginAddresses, which a test replaces to give origin a forge's
// address while git itself reaches a local repository.
var originAddresses = readOriginAddresses

// readOriginAddresses reads origin's every fetch address and every push
// address, each as git resolves it under the person's own configuration (the
// one the delete's push runs under), its insteadOf and pushInsteadOf rewrites
// applied. It reads the configuration and reaches no remote.
func readOriginAddresses(ctx context.Context, root string) (fetch, push []string, err error) {
	ctx, cancel := context.WithTimeout(ctx, remoteNetTimeout)
	defer cancel()
	out, err := gitutil.NetContext(ctx, root, "remote", "get-url", "--all", claim.Remote)
	if err != nil {
		return nil, nil, fmt.Errorf("origin's fetch address could not be read: %w", err)
	}
	fetch = strings.Fields(out)
	out, err = gitutil.NetContext(ctx, root, "remote", "get-url", "--push", "--all", claim.Remote)
	if err != nil {
		return nil, nil, fmt.Errorf("origin's push address could not be read: %w", err)
	}
	return fetch, strings.Fields(out), nil
}

// oneRepository is the key of the one repository origin is fetched from and
// pushed to, or why origin is not one repository. No address is shown in the
// reason: one may carry credentials, and the reason needs none.
func oneRepository(fetch, push []string) (key, why string) {
	switch {
	case len(fetch) == 0:
		return "", "origin has no fetch address"
	case len(fetch) > 1:
		return "", fmt.Sprintf("origin has %d fetch addresses, so the sweep cannot check what it would delete", len(fetch))
	case len(push) == 0:
		return "", "origin has no push address"
	}
	key = forgerepo.Key(fetch[0])
	if key == "" {
		return "", "origin's fetch address does not read as a repository's"
	}
	for _, p := range push {
		if forgerepo.Key(p) != key {
			return "", "origin is read from one repository and pushed to another, so the sweep cannot check what it would delete"
		}
	}
	return key, ""
}

// repoNameRe is the host, or one segment of a forge repository's
// owner/name: letters, digits, '.', '_' and '-', never opening with '-', so
// no part of the name the forge client is given reads as a flag.
var repoNameRe = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._-]*$`)

// forgeRepoName is the repository a key names as the forge client takes it,
// host/owner/name, and whether the key names one: a host with no port and a
// path of exactly an owner and a name, none of the three opening with '-'.
func forgeRepoName(key string) (string, bool) {
	host, path := forgerepo.SplitKey(key)
	if host == "" || strings.Contains(host, ":") || !repoNameRe.MatchString(host) {
		return "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) != 2 || !repoNameRe.MatchString(parts[0]) || !repoNameRe.MatchString(parts[1]) ||
		parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return "", false
	}
	return host + "/" + parts[0] + "/" + parts[1], true
}

// forgeView is the forge's word on a repository, in gh's field names.
type forgeView struct {
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
}

// unmatched opens every reason no forge call is made.
const unmatched = "the forge's repository could not be matched to origin"

// pinForge is gh with every call pinned to origin's repository, and that
// repository's key (forgerepo.Key); or nil and why no forge call can be
// pinned there.
func pinForge(ctx context.Context, root string, gh store.GH) (store.GH, string, string) {
	fail := func(why string) (store.GH, string, string) { return nil, "", unmatched + " (" + why + ")" }
	if gh == nil {
		return fail("no forge client was given")
	}
	fetch, push, err := originAddresses(ctx, root)
	if err != nil {
		return fail(answerText(err))
	}
	key, why := oneRepository(fetch, push)
	if why != "" {
		return fail(why)
	}
	repo, ok := forgeRepoName(key)
	if !ok {
		return fail("origin's address names no owner and repository on a forge's host")
	}
	// The repository rides as gh repo view's one positional argument, which
	// takes no --repo; forgeRepoName admits no name that opens with '-', the
	// character a flag opens with.
	out, err := forgeCall(ctx, gh, root, "repo", "view", repo, "--json", "nameWithOwner,url,sshUrl")
	if err != nil {
		return fail("the forge could not be asked for " + repo + ": " + answerText(err))
	}
	var v forgeView
	if json.Unmarshal([]byte(out), &v) != nil {
		return fail("the forge's view of " + repo + " does not parse")
	}
	if _, _, found := forgerepo.Stranger(append(slices.Clone(fetch), push...), forgerepo.Keys(v.URL, v.SSHURL)); found {
		return fail("the forge answers for another repository than the one origin's addresses name")
	}
	pin := "--repo=" + repo
	return func(dir string, args ...string) (string, error) {
		return gh(dir, append(slices.Clone(args), pin)...)
	}, key, ""
}

// pinnedForge is o with its Forge pinned to origin's repository, once: a run
// that pins it hands the pinned options to every half, which then pin
// nothing again. When no call can be pinned, Forge is nil and forgeUnmatched
// says why.
func (o Options) pinnedForge(ctx context.Context, root string) Options {
	if o.forgeChecked {
		return o
	}
	o.forgeChecked = true
	if o.Forge == nil {
		return o
	}
	o.Forge, o.forgeKey, o.forgeUnmatched = pinForge(ctx, root, o.Forge)
	return o
}

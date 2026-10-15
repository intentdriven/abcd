package claim

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// noreplyRe matches GitHub's noreply commit addresses, both forms:
// `77722411+user@users.noreply.github.com` and the older
// `user@users.noreply.github.com`. The pattern itself names the forge, so the
// derivation needs no forge configuration and degrades by absence everywhere
// else (itd-140). It is the one spelling of the forge's two forms: the site's
// contributor profiles and the reservation's holder both read it.
var noreplyRe = regexp.MustCompile(`(?i)^(?:\d+\+)?([a-z\d](?:[a-z\d-]*[a-z\d])?)@users\.noreply\.github\.com$`)

// NoReplyUsername is the GitHub username a noreply commit address names, or ""
// when the address is not one.
func NoReplyUsername(email string) string {
	m := noreplyRe.FindStringSubmatch(email)
	if m == nil {
		return ""
	}
	return m[1]
}

// usernameRe is a GitHub username: alphanumerics and single interior hyphens,
// one to 39 characters.
var usernameRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

func validUsername(s string) bool { return usernameRe.MatchString(s) }

// sameHolder compares two usernames as the forge does, without regard to case.
func sameHolder(a, b string) bool { return strings.EqualFold(a, b) }

// The machine-layer key that names the holder when user.email does not.
const (
	holderNamespace = "reservation"
	holderKey       = holderNamespace + ".github_username"
)

// identity is the person's git identity, read under their own configuration:
// the author and committer of every reservation commit.
type identity struct {
	Name  string
	Email string
}

// gitIdentity reads user.name and user.email under the developer's own git
// configuration (gitutil.ScrubbedEnv), where the identity lives. A checkout
// whose git has neither is refused before anything is written: git would
// otherwise compose one from the machine's name, which a public commit must
// never carry.
func gitIdentity(root string) (identity, error) {
	var id identity
	for _, f := range []struct {
		key  string
		into *string
	}{{"user.name", &id.Name}, {"user.email", &id.Email}} {
		v, err := personalConfig(root, f.key)
		if err != nil {
			return identity{}, refuse("could not read git's %s: %v; %s", f.key, err, nothingPushed)
		}
		*f.into = v
	}
	if id.Name == "" || id.Email == "" {
		return identity{}, refuse("git has no user.name or user.email to author the reservation; set them, then retry; %s", nothingPushed)
	}
	return id, nil
}

// resolveHolder is the person's GitHub username, from these places in order
// and from nowhere else:
//
//  1. user.email, when it is a GitHub noreply address;
//  2. reservation.github_username in the machine layer of the layered
//     configuration (~/.abcd.noindex/config.json), taken only from that layer.
//
// The repository's own .abcd/config.json is never read for it: a value there
// would let a commit decide whose name a reservation is pushed under. With
// neither user-level source the write is refused naming both.
func resolveHolder(root, home, email string) (string, error) {
	if u := NoReplyUsername(email); u != "" {
		if !validUsername(u) {
			return "", refuse("user.email names %s, which is not a GitHub username; %s", echo(u), nothingPushed)
		}
		return u, nil
	}
	stack, err := layered.Load(layered.Config, layered.Roots{Repo: root, Home: home})
	if err != nil {
		return "", refuse("could not read the configuration that names your GitHub username: %v; %s", err, nothingPushed)
	}
	if err := stack.Claim(holderNamespace, "github_username"); err != nil {
		return "", refuse("%v; %s", err, nothingPushed)
	}
	found, err := stack.Lookup(holderKey)
	if err != nil {
		return "", refuse("%v; %s", err, nothingPushed)
	}
	for _, f := range found {
		if f.Layer != layered.Machine {
			continue
		}
		u, err := layered.Decode[string](f.Raw)
		if err == nil && !validUsername(u) {
			err = errors.New("it is not a GitHub username")
		}
		if err != nil {
			return "", refuse("%s (%s layer): %s: %v; %s", f.Origin, f.Layer, holderKey, err, nothingPushed)
		}
		return u, nil
	}
	return "", refuse("cannot tell which GitHub user you are: user.email is not a GitHub no-reply address, and %s sets no %s; "+
		"set one of them (the repository's own configuration is never read for this); %s",
		layered.Config.MachineOrigin(), holderKey, nothingPushed)
}

// personalConfig reads one git config value under the developer's own
// configuration; "" when it is unset.
func personalConfig(root, key string) (string, error) {
	out, code, err := personalGit(root, "", nil, "config", "--get", key)
	if code == 1 {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("git config %s: %w", key, err)
	}
	return strings.TrimSpace(out), nil
}

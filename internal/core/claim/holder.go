package claim

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	toolid "github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/tools"
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

// identity is a git identity: the one read under the person's own
// configuration, and the author and committer of every reservation commit.
type identity struct {
	Name  string
	Email string
}

// person reports whether the identity is a person's: both halves set, and not
// a machine's as the attribution gate judges an author (toolid.IsToolIdentity).
// An identity that is not a person's is never shown back, in a refusal or
// anywhere else.
func (id identity) person() bool {
	return id.Name != "" && id.Email != "" && !toolid.IsToolIdentity(toolid.RoleAuthor, id.Name, id.Email)
}

// gitIdentity reads user.name and user.email under the developer's own git
// configuration (gitutil.ScrubbedEnv), where the identity lives. Either may be
// unset: the holder resolver then asks the session's forge token whose account
// it is, and that account authors the reservation, so git never composes an
// author from the machine's name, which a public commit must never carry.
func gitIdentity(root string) (identity, error) {
	var id identity
	for _, f := range []struct {
		key  string
		into *string
	}{{"user.name", &id.Name}, {"user.email", &id.Email}} {
		v, err := personalConfig(root, f.key)
		if err != nil {
			return identity{}, refuse("could not read git's %s: %v; %s", f.key, err, nothingSent)
		}
		*f.into = v
	}
	return id, nil
}

// nothingSent ends every refusal the holder resolver makes: it may have asked
// GitHub's user endpoint, so it says that nothing was sent (no reservation, no
// commit) as well as that nothing was pushed.
const nothingSent = "nothing was sent or pushed"

// Holder is the GitHub account a reservation act would be made as: its
// login, its numeric id and no-reply address when they are known, and which
// source named it.
type Holder struct {
	Login   string `json:"login"`
	ID      int64  `json:"id,omitempty"`
	NoReply string `json:"noreply,omitempty"`
	Source  string `json:"source"`
}

// The sources a Holder is named from.
const (
	HolderFromGit    = "git"    // user.email is a GitHub no-reply address
	HolderFromConfig = "config" // the machine layer's reservation.github_username
	HolderFromToken  = "token"  // the forge token gh resolves from the session's environment
)

// ResolveHolder is the account a reservation act in the checkout at root
// would be made as, with the forge's user endpoint allowed: `abcd reserve
// --holder` asks it. It reserves nothing and writes nothing.
func ResolveHolder(root string, o Options) (Holder, error) {
	who, err := gitIdentity(root)
	if err != nil {
		return Holder{}, err
	}
	return resolveHolder(root, who, o, true)
}

// resolveHolder is the person's GitHub account, from these places in order
// and from nowhere else. When git's identity is a person's:
//
//  1. user.email, when it is a GitHub noreply address whose local part is a
//     GitHub username (one that is not falls through to the next source);
//  2. reservation.github_username in the machine layer of the layered
//     configuration (~/.abcd.noindex/config.json), the only layer read for it;
//  3. when net allows it, the forge token gh resolves from the session's
//     environment, asked whose account it is through the user endpoint
//     (o.ForgeUser).
//
// When it is not a person's (unset, or a machine's), the token alone: the
// author then comes from the token's account, and the machine key is skipped
// because the author's address must come from the same account as the login.
// The repository's own .abcd/config.json is never read for it: a value there
// would let a commit decide whose name a reservation is pushed under, and a
// malformed one would let a commit block the person. Only an
// explicit act passes net; a render never does, so it makes no network call.
// With no source the act is refused, and the refusal never echoes the git
// identity it rejected.
func resolveHolder(root string, who identity, o Options, net bool) (Holder, error) {
	if who.person() {
		if u := NoReplyUsername(who.Email); u != "" && validUsername(u) {
			return Holder{Login: u, ID: noreplyID(who.Email), NoReply: who.Email, Source: HolderFromGit}, nil
		}
		u, err := machineHolder(o.home())
		if err != nil {
			return Holder{}, err
		}
		if u != "" {
			return Holder{Login: u, Source: HolderFromConfig}, nil
		}
		if !net {
			return Holder{}, refuse("cannot tell which GitHub user you are without asking GitHub: user.email is not a GitHub no-reply address, "+
				"%s names no %s, and only a reservation act asks GitHub; %s",
				layered.Config.MachineOrigin(), holderKey, nothingSent)
		}
	} else if !net {
		return Holder{}, refuse("cannot tell which GitHub user you are without asking GitHub: git's identity is not a person's, and only a reservation act asks GitHub; %s", nothingSent)
	}
	return tokenHolder(root, o)
}

// machineHolder is reservation.github_username from the machine layer, ""
// when that layer does not set it or no home resolves to hold one. Only the
// machine layer is loaded (Roots.Repo is ""), so nothing the repository
// commits, malformed or not, can refuse the person here; the person's own
// file, malformed for this key, still refuses, naming the file.
func machineHolder(home string) (string, error) {
	if home == "" {
		return "", nil
	}
	stack, err := layered.Load(layered.Config, layered.Roots{Home: home})
	if err != nil {
		return "", refuse("could not read the configuration that names your GitHub username: %v; %s", err, nothingSent)
	}
	if err := stack.Claim(holderNamespace, reservationKeys...); err != nil {
		return "", refuse("%v; %s", err, nothingSent)
	}
	found, err := stack.Lookup(holderKey)
	if err != nil {
		return "", refuse("%v; %s", err, nothingSent)
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
			return "", refuse("%s (%s layer): %s: %v; %s", f.Origin, f.Layer, holderKey, err, nothingSent)
		}
		return u, nil
	}
	return "", nil
}

// noreplyIDRe is the numeric account id a noreply address's newer form leads
// with.
var noreplyIDRe = regexp.MustCompile(`^(\d+)\+`)

// noreplyID is the account id a noreply address carries, 0 when it carries
// none.
func noreplyID(email string) int64 {
	m := noreplyIDRe.FindStringSubmatch(email)
	if m == nil {
		return 0
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// The classes a token lookup fails in. A refusal names the class, never gh's
// own words: its stderr can carry an identity.
var (
	errForgeMissing = errors.New("the GitHub CLI (gh) is not on PATH")
	errForgeNoToken = errors.New("gh could not read the account (it is not logged in to github.com, holds no usable token, or GitHub was unreachable)")
	errForgeTimeout = errors.New("gh did not answer in time")
	errForgeAnswer  = errors.New("GitHub's answer did not name an account")
	errForgeMachine = errors.New("the token belongs to a machine account, not a person's")
)

// tokenHolder asks the session's forge token whose account it is: one call of
// the user endpoint, through o.ForgeUser.
func tokenHolder(root string, o Options) (Holder, error) {
	ask := o.ForgeUser
	if ask == nil {
		ask = forgeUser
	}
	login, id, err := ask(root)
	if err == nil && (!validUsername(login) || id <= 0) {
		err = errForgeAnswer
	}
	noreply := fmt.Sprintf("%d+%s@users.noreply.github.com", id, login)
	if err == nil && toolid.IsToolIdentity(toolid.RoleAuthor, login, noreply) {
		// A release bot's or an automation account's token names no person.
		err = errForgeMachine
	}
	if err != nil {
		cause := errForgeNoToken
		for _, c := range []error{errForgeMissing, errForgeTimeout, errForgeAnswer, errForgeMachine} {
			if errors.Is(err, c) {
				cause = c
			}
		}
		return Holder{}, refuse("could not work out whose GitHub account this session uses: %v; %s", cause, nothingSent)
	}
	return Holder{Login: login, ID: id, NoReply: noreply, Source: HolderFromToken}, nil
}

// forgeUser is the real user-endpoint call: `gh api user` through the one
// door to GitHub, its failure reduced to a class.
func forgeUser(root string) (string, int64, error) {
	out, err := github.API(root, nil, "user")
	if err != nil {
		return "", 0, forgeClass(err)
	}
	var u struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	}
	if err := json.Unmarshal(out, &u); err != nil {
		return "", 0, errForgeAnswer
	}
	return u.Login, u.ID, nil
}

// forgeClass is the class a failed user-endpoint call falls in: gh missing,
// gh out of time, or else no account read. gh's own words are dropped.
func forgeClass(err error) error {
	var missing *tools.MissingError
	switch {
	case errors.As(err, &missing):
		return errForgeMissing
	case errors.Is(err, github.ErrTimeout):
		return errForgeTimeout
	}
	return errForgeNoToken
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

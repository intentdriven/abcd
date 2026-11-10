package loop

// remoteid.go is origin's identity across a drive's lanes
// (iss-2611071820538335, ruling recorded in its decision): the settings that
// say which remote `git ls-remote origin` reaches, read by the drive as it
// starts and held in its own memory, never in a file anything in a lane could
// rewrite. Each run of a drain records its watched settings afresh, so a lane's
// agent that changed origin's URL would otherwise choose the remote that vouches
// for the next lane's pin.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteIdentity is what a drive read of origin's identity: for each key it
// covers that git reads for the checkout, a hash of its values in order, and
// the salt its key names are hidden under when a refusal names them
// (hiddenKey). No value is kept.
type remoteIdentity struct {
	keys map[string]string
	salt string
}

// remoteIdentityKey says whether a canonical config key is one of origin's
// identity: its URL and push URL, and every URL rewrite (url.<base>.insteadOf
// and pushInsteadOf). Every rewrite is covered, not only one whose base is a
// prefix of origin's URL: telling which applies is git's own longest-match
// rule, and a change to any of them is rare enough to stop a lane for.
func remoteIdentityKey(key string) bool {
	switch key {
	case "remote." + Remote + ".url", "remote." + Remote + ".pushurl":
		return true
	}
	if rest, ok := strings.CutPrefix(key, "url."); ok {
		return strings.HasSuffix(rest, ".insteadof") || strings.HasSuffix(rest, ".pushinsteadof")
	}
	return false
}

// readRemoteIdentity reads origin's identity at repoRoot as the worktree
// stage's ask of origin reads it: `git config --null --list` through the same
// runner (gitutil.NetContext), so the person's global and system settings, a
// conditional include and the environment's own settings are read as the ask
// reads them. It is bounded by originAskTimeout.
func readRemoteIdentity(repoRoot string) (*remoteIdentity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	out, err := gitutil.NetContext(ctx, repoRoot, "config", "--null", "--list")
	if err != nil {
		return nil, fmt.Errorf("reading %s's settings: %s", Remote, gitutil.ScrubURLCredentials(err.Error()))
	}
	values := map[string][]string{}
	for _, rec := range strings.Split(out, "\x00") {
		key, value, _ := strings.Cut(rec, "\n")
		if remoteIdentityKey(key) {
			values[key] = append(values[key], value)
		}
	}
	salt, err := newSalt()
	if err != nil {
		return nil, err
	}
	id := &remoteIdentity{keys: map[string]string{}, salt: salt}
	for k, vs := range values {
		h := sha256.New()
		for _, v := range vs {
			writePart(h, v)
		}
		id.keys[k] = hex.EncodeToString(h.Sum(nil))
	}
	return id, nil
}

// changed names, sorted, each key of was's that now lacks or holds otherwise,
// and each key now holds that was lacks, with a URL in its name hidden
// (hiddenKey). Never a value.
func (was *remoteIdentity) changed(now *remoteIdentity) []string {
	var out []string
	for k, h := range was.keys {
		if now.keys[k] != h {
			out = append(out, hiddenKey(was.salt, k))
		}
	}
	for k := range now.keys {
		if _, ok := was.keys[k]; !ok {
			out = append(out, hiddenKey(was.salt, k))
		}
	}
	slices.Sort(out)
	return out
}

// remoteUnchanged refuses the lane's worktree stage when origin's identity
// differs from what the drive read as it started (Context.remote), before
// anything asks origin; with nothing carried it checks nothing.
func remoteUnchanged(c Context, laneID string) error {
	if c.remote == nil {
		return nil
	}
	now, err := readRemoteIdentity(c.RepoRoot)
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"could not read "+Remote+"'s settings again, so the remote the lane would be checked against cannot be shown to be the one the drive started with: "+err.Error(),
			"settle what git reports (`git config --list`), then run the step again; nothing was made")
	}
	keys := c.remote.changed(now)
	if len(keys) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		fmt.Sprintf("%s's settings changed since the drive started (%s), so the remote that would vouch for the lane's hooks is not the one the drive started with", Remote, strings.Join(keys, ", ")),
		"look at what changed them (`git config --show-origin --get-regexp '^(remote\\."+Remote+"\\.(url|pushurl)|url\\..*insteadof)$'`); the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

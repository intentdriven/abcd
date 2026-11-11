package loop

// remoteid.go is what a drive holds across its lanes (iss-2611071820538335,
// rulings recorded in its decisions): the settings that say which remote `git
// ls-remote origin` reaches and what runs for it (the program it asks for a
// credential and the protocols it may use among them), the hooks directory the
// next lane's add is told, and the checkout's hooks directory, whose
// post-checkout hook each lane's add runs. The drive reads them as it starts
// and holds hashes of them in its own memory, never in a file anything in a
// lane could rewrite. Each run of a drain records its watched settings and the
// checkout's hooks afresh at its first check, so a lane's agent that changed
// origin's URL would otherwise choose the remote that vouches for the next
// lane's pin, and one that changed the checkout's post-checkout hook would
// have it run in the next lane's add.
//
// The hold on the checkout's hooks has three limits. The directory is read
// just before the add; a change made after that read and before git runs the
// hook is not seen, and nothing holds the directory through the add. The
// directory read is the one the repository's own settings name (checkoutHooks);
// with a relative core.hooksPath set only in global settings, the copy the add
// runs is not the one read. What is compared is each top-level entry, a file by
// its bytes and mode, a subdirectory by its kind only and a symlink by its
// target text, up to maxHookEntries (past it, the count and the first
// entries); nothing a hook sources or runs from a subdirectory or from
// elsewhere is held.
//
// Not held: any other http.* key (a per-URL http.<url>.* among them); an
// origin defined only by a legacy remotes or branches file in the git
// directory, which has no key to compare; the run fingerprint's other watched
// settings (content filters, diff and merge drivers, signing programs, includes
// and the rest), which each run records afresh at its first check; and
// anything changed while no drive runs.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteIdentityHeld names, for a refusal and the docs, the key families a
// drive holds across its lanes (remoteIdentityKey).
const remoteIdentityHeld = "origin's url, pushurl, uploadpack, proxy and vcs; every url.*.insteadOf and pushInsteadOf; core.sshCommand, core.gitProxy, core.hooksPath and core.askPass; every credential.*; protocol.allow and every protocol.*.allow; http.proxy and http.sslVerify"

// remoteIdentity is what a drive read of origin's identity: for each key it
// covers that git reads for the checkout, a hash of its values in order, and
// the salt its key names are hidden under when a refusal names them
// (hiddenKey). No value is kept.
type remoteIdentity struct {
	keys  map[string]string
	hooks map[string]string
	salt  string
}

// remoteIdentityKey says whether a canonical config key is one of origin's
// identity (remoteIdentityHeld): origin's url, pushurl, uploadpack, proxy and
// vcs; every URL rewrite (url.<base>.insteadOf and pushInsteadOf);
// core.sshCommand, core.gitProxy, core.hooksPath and core.askPass; every
// credential.* key (a helper is a program the ask runs); protocol.allow and
// every protocol.<name>.allow (which transports, ext:: among them, the ask
// may use); and http.proxy and http.sslVerify. Every rewrite is covered, not
// only one whose base is a prefix of origin's URL: telling which applies is
// git's own longest-match rule, and a change to any of them is rare enough to
// stop a lane for.
func remoteIdentityKey(key string) bool {
	switch key {
	case "remote." + Remote + ".url", "remote." + Remote + ".pushurl", "remote." + Remote + ".uploadpack",
		"remote." + Remote + ".proxy", "remote." + Remote + ".vcs",
		"core.sshcommand", "core.gitproxy", "core.hookspath", "core.askpass",
		"protocol.allow", "http.proxy", "http.sslverify":
		return true
	}
	if rest, ok := strings.CutPrefix(key, "url."); ok {
		return strings.HasSuffix(rest, ".insteadof") || strings.HasSuffix(rest, ".pushinsteadof")
	}
	if rest, ok := strings.CutPrefix(key, "protocol."); ok {
		return strings.HasSuffix(rest, ".allow")
	}
	return strings.HasPrefix(key, "credential.")
}

// readRemoteIdentity is what a drive holds as it starts: origin's identity
// (readRemoteSettings) and the checkout's hooks directory, hashed under the
// identity's salt through the run's fingerprint's own reader (checkoutHooks).
// It fails when the settings cannot be listed, no salt can be drawn, the
// repository's own settings cannot be read or the common git directory cannot
// be resolved; an unreadable hooks directory is recorded as such (hashDir),
// not a failure, and compared as an entry is.
func readRemoteIdentity(repoRoot string) (*remoteIdentity, error) {
	id, err := readRemoteSettings(repoRoot)
	if err != nil {
		return nil, err
	}
	if id.hooks, err = readCheckoutHooks(repoRoot, id.salt); err != nil {
		return nil, err
	}
	return id, nil
}

// readCheckoutHooks hashes, under salt, each top-level entry of the hooks
// directory the repository's own settings name (else the common git
// directory's hooks), through the fingerprint's own reader (checkoutHooks).
func readCheckoutHooks(repoRoot, salt string) (map[string]string, error) {
	hooks := map[string]string{}
	if _, _, err := checkoutHooks(repoRoot, salt, map[string]string{}, hooks); err != nil {
		return nil, fmt.Errorf("reading the checkout's hooks directory: %s", fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	return hooks, nil
}

// readRemoteSettings reads origin's identity at repoRoot as the worktree
// stage's ask of origin reads it: `git config --null --list` through the same
// runner (gitutil.NetContext), so the person's repository, global and system
// settings and a conditional include are read as the ask reads them
// (TestEachKeyFamilyTheDriveHoldsIsReadAgain has a system-scope and an
// included case). The environment is the one that runner gives the ask; no
// test sets a setting through the environment. It is bounded by
// originAskTimeout.
func readRemoteSettings(repoRoot string) (*remoteIdentity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	out, err := gitutil.NetContext(ctx, repoRoot, "config", "--null", "--list")
	if err != nil {
		return nil, fmt.Errorf("reading %s's settings: %s", Remote, fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
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
// differs from what the drive read as it started (Context.remote). The stage
// calls it before it asks origin anything and again once origin has answered,
// so a change made while the ask ran and left in place is refused too; one
// made and put back between the two reads is not seen. With nothing carried
// it checks nothing.
func remoteUnchanged(c Context, laneID string) error {
	if c.remote == nil {
		return nil
	}
	now, err := readRemoteSettings(c.RepoRoot)
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
		fmt.Sprintf("settings the drive holds across its lanes (%s) changed since it started (%s), so the remote that would vouch for the lane's hooks, or what runs to reach it, is not the drive's", remoteIdentityHeld, strings.Join(keys, ", ")),
		"look at what changed them (`git config --show-origin --get-regexp '^(remote\\."+Remote+"\\.(url|pushurl|uploadpack|proxy|vcs)|url\\..*insteadof|core\\.(sshcommand|gitproxy|hookspath|askpass)|credential\\..*|protocol\\.(.*\\.)?allow|http\\.(proxy|sslverify))$'`); the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

// checkoutHooksUnchanged refuses the lane's worktree stage when the
// checkout's hooks directory, as read now, differs from what the drive read
// as it started (Context.remote): an entry added, removed or changed, named.
// git's add runs in the checkout, so it runs the post-checkout hook of the
// hooks directory git takes there (for a relative core.hooksPath, the
// checkout's own working-tree copy) with the drive's rights; each lane of a
// drain is a run of its own, whose fingerprint takes that copy as it stands
// at its first check, so without this a copy an earlier lane changed would be
// the new run's starting point. The stage calls it just before the add, so a
// difference present at this read is refused before that add. It holds the
// three limits the file's header states: a change made after this read and
// before git runs the hook is not seen, and nothing holds the directory
// through the add; the directory read is the one the repository's own
// settings name; and only top-level entries are compared, a subdirectory by
// its kind and a symlink by its target text. With nothing carried (a step the
// host runs, a single build) it checks nothing.
func checkoutHooksUnchanged(c Context, laneID string) error {
	if c.remote == nil {
		return nil
	}
	now, err := readCheckoutHooks(c.RepoRoot, c.remote.salt)
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"could not read the checkout's hooks directory again, so the post-checkout hook the lane's add would run cannot be shown to be the one the drive started with: "+err.Error(),
			"settle what git reports (`git config --local --list --show-origin`), then run the step again; nothing was made")
	}
	changed := configFingerprint{Files: c.remote.hooks}.changes(c.RepoRoot, configFingerprint{Files: now})
	if len(changed) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		"the checkout's hooks changed since the drive started ("+strings.Join(nameChanges(c.RepoRoot, changed), ", ")+"), and git's add of the lane's worktree runs the checkout's post-checkout hook with the drive's rights, so the lane is not made",
		"look at what changed the checkout's hooks directory; the drive reads it once, as it starts, so stop it and start it again once the hooks are as they should be; nothing was made")
}

package loop

// remoteid.go is what a drive holds across its lanes (iss-2611071820538335,
// iss-2611131308266006, rulings recorded in their decisions): every setting
// the run's fingerprint watches (watchedKey: what names a program git starts,
// or a file that adds more settings, origin's identity among them), read at
// every scope git reads, with http.proxy and http.sslVerify beside them; the
// hooks directory git takes at the checkout from those same scopes
// (checkoutHooksPath), at every depth; and the common git directory's
// info/attributes. The drive reads them as it starts and holds hashes of them
// in its own memory, never in a file anything in a lane could rewrite. Each
// run of a drain records its watched settings and the checkout's hooks afresh
// at its first check, so a lane's agent that changed origin's URL would
// otherwise choose the remote that vouches for the next lane's pin, one that
// changed the checkout's post-checkout hook would have it run in the next
// lane's add, and one that set a content filter would have it run in the next
// lane's commit.
//
// As it starts, where the hooks directory is the checkout's own in-tree copy
// and the repository has an origin, the drive also proves that copy against
// the tip origin reports, and refuses to start on a difference
// (proveCheckoutHooks).
//
// What one account cannot hold, the page states: the directory is read just
// before each add, and a change made after that read and before git runs the
// hook is not seen, since nothing holds the directory through the add; a
// change put back between the two reads of the settings around the ask of
// origin is not seen; what a hook runs from elsewhere (a script, make, a
// program on PATH) is not hashed; and a hooks directory outside the tree, or a
// repository with no origin, has nothing a drive's start can prove it
// against. Not held either: any other http.* key (a per-URL http.<url>.*
// among them), an origin defined only by a legacy remotes or branches file in
// the git directory, which has no key to compare, and the global and system
// attributes files.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteIdentityHeld names, for a refusal and the docs, what settings a drive
// holds across its lanes (remoteIdentityKey).
const remoteIdentityHeld = "every setting the run's fingerprint watches, at every scope (origin's and every remote's url, pushurl, uploadpack, receivepack, proxy and vcs; every url rewrite; core.hooksPath, core.sshCommand, core.gitProxy, core.askPass, core.fsmonitor, core.pager, core.editor and core.worktree; every credential.*; the protocol allow rules; content filters, diff and merge drivers, gpg programs, include and includeIf paths), and http.proxy and http.sslVerify"

// heldPattern is the extended regular expression a refusal's remedy names
// for remoteIdentityHeld: watchedConfig with http.proxy and http.sslVerify.
var heldPattern = strings.TrimSuffix(watchedConfig, ")$") + `|http\.(proxy|sslverify))$`

// remoteIdentity is what a drive read as it started: for each setting it
// holds that git reads for the checkout, a hash of its scopes and values in
// order; each entry of the checkout's hooks directory and info/attributes,
// hashed as the run's fingerprint hashes them; the core.hooksPath git takes
// at the checkout ("" for none), which names that directory; and the
// salt its key names are hidden under when a refusal names them (hiddenKey).
// No value is kept.
type remoteIdentity struct {
	keys      map[string]string
	hooks     map[string]string
	hooksPath string
	salt      string
}

// remoteIdentityKey says whether a canonical config key is one a drive holds
// across its lanes: every key the run's fingerprint watches (watchedKey, the
// one predicate for both), and http.proxy and http.sslVerify, which choose how
// origin is reached without naming a program.
func remoteIdentityKey(key string) bool {
	return watchedKey(key) || key == "http.proxy" || key == "http.sslverify"
}

// readRemoteIdentity is what a drive holds as it starts: its settings
// (readRemoteSettings) and the checkout's hooks directory and info/attributes
// (readCheckoutHooks), hashed under the settings' salt. It fails when the
// settings cannot be listed, no salt can be drawn, the hooks directory git
// takes at the checkout cannot be told, the common git directory cannot be
// resolved, or the hooks directory holds more than maxHookEntries entries; an
// unreadable hooks directory is recorded as such (hashDir), not a failure,
// and compared as an entry is.
func readRemoteIdentity(repoRoot string) (*remoteIdentity, error) {
	id, err := readRemoteSettings(repoRoot)
	if err != nil {
		return nil, err
	}
	if id.hooksPath, err = checkoutHooksPath(repoRoot); err != nil {
		return nil, fmt.Errorf("reading the hooks directory git takes at the checkout: %s", fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	if id.hooks, err = readCheckoutHooks(repoRoot, id.salt, id.hooksPath); err != nil {
		return nil, err
	}
	return id, nil
}

// errHooksPastCap is readCheckoutHooks's refusal of a hooks directory past
// maxHookEntries entries.
var errHooksPastCap = fmt.Errorf("the checkout's hooks directory holds more than %d entries, so the drive cannot hold it whole", maxHookEntries)

// readCheckoutHooks hashes, under salt, each entry at every depth of the
// hooks directory git takes at the checkout (hooksPath, from every scope:
// checkoutHooksPath; else the common git directory's hooks), through the
// fingerprint's own walk (hashDir), and the common git directory's
// info/attributes. It fails when the common git directory cannot be resolved
// or the tree is past maxHookEntries entries.
func readCheckoutHooks(repoRoot, salt, hooksPath string) (map[string]string, error) {
	common, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("reading the checkout's hooks directory: %s", fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	dir := filepath.Join(common, "hooks")
	if hooksPath != "" {
		dir = expandHooksPath(repoRoot, hooksPath)
	}
	hooks := map[string]string{}
	if _, over := hashDir(hooks, salt, dir); over {
		return nil, errHooksPastCap
	}
	hashFile(hooks, salt, filepath.Join(common, "info", "attributes"))
	return hooks, nil
}

// readRemoteSettings reads the settings a drive holds at repoRoot as the
// worktree stage's ask of origin reads them: `git config --null --list
// --show-scope` through the same runner (gitutil.NetContext), so the
// person's repository, global and system settings and a conditional include
// are read as the ask reads them (TestEachKeyFamilyTheDriveHoldsIsReadAgain
// has a system-scope and an included case). The environment is the one that
// runner gives the ask; no test sets a setting through the environment. It is
// bounded by originAskTimeout.
func readRemoteSettings(repoRoot string) (*remoteIdentity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	out, err := gitutil.NetContext(ctx, repoRoot, "config", "--null", "--list", "--show-scope")
	if err != nil {
		return nil, fmt.Errorf("reading %s's settings: %s", Remote, fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	values := map[string][]string{}
	fields := strings.Split(out, "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		scope, rec := fields[i], fields[i+1]
		key, value, _ := strings.Cut(rec, "\n")
		if remoteIdentityKey(key) {
			values[key] = append(values[key], scope, value)
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

// proveCheckoutHooks is a drive's proof, as it starts, of the checkout's
// hooks directory: where it is the checkout's own in-tree copy (ownHooksDir)
// and the repository has an origin, what stands there must be, at every
// depth, the hooks directory of the tip origin reports for its default
// branch, as stored (hooksTreeAt, hookTreeEntries, provenHooks: each name,
// each file's bytes and exec bit, each symlink's target, no symlink leaving
// the directory). A difference refuses the start, naming each file; so does
// an origin that cannot be asked or a tip this repository does not hold. A
// hooks directory outside the tree, or no origin, has nothing to be proved
// against, and passes. It reads and refuses only: nothing is written, and a
// person's own git runs the hooks as before.
func proveCheckoutHooks(repoRoot string, id *remoteIdentity) error {
	rel := ownHooksDir(repoRoot, id.hooksPath)
	if rel == "" {
		return nil
	}
	has, err := claim.HasOrigin(repoRoot)
	if err != nil || !has {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	_, tip, err := remoteHeadTip(ctx, nil, repoRoot, Remote)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("%s did not answer within %s", Remote, originAskTimeout)
		}
		return fmt.Errorf("could not ask %s for its default branch's tip, so the checkout's hooks in %s cannot be proved against it: %s", Remote, rel, gitutil.ScrubURLCredentials(err.Error()))
	}
	if _, err := gitutil.RunCappedAsStored(repoRoot, 4096, "cat-file", "-e", tip+"^{commit}"); err != nil {
		return fmt.Errorf("%s reports its tip at %s, which this repository does not hold yet, so the checkout's hooks in %s cannot be proved against it; fetch %s", Remote, tip[:12], rel, Remote)
	}
	tree, err := hooksTreeAt(repoRoot, tip, rel)
	if err != nil {
		return err
	}
	want, err := hookTreeEntries(repoRoot, tree)
	if err == nil {
		_, err = provenHooks(id.salt, filepath.Join(repoRoot, filepath.FromSlash(rel)), want)
	}
	if err != nil {
		return &hooksDiffer{rel: rel, tip: tip, err: err}
	}
	return nil
}

// hooksDiffer is proveCheckoutHooks's refusal of a checkout whose in-tree
// hooks are not origin's tip's.
type hooksDiffer struct {
	rel, tip string
	err      error
}

func (h *hooksDiffer) Error() string {
	return fmt.Sprintf("the checkout's hooks in %s are not those of %s's tip %s: %v", h.rel, Remote, h.tip[:12], h.err)
}

// startRefusal is the drive's refusal to start on an error from
// readRemoteIdentity or proveCheckoutHooks.
func startRefusal(repoRoot string, err error) error {
	var differ *hooksDiffer
	switch {
	case errors.As(err, &differ):
		return refuse(StageDrain, "", "", "the drive cannot start: "+nameHooksDiffer(repoRoot, differ),
			"look at what changed the checkout's hooks (`git status -- "+differ.rel+"`, `git diff "+differ.tip[:12]+" -- "+differ.rel+"`); commit and land the change, or put the hooks back as "+Remote+"'s tip holds them, then start the drive again; nothing was made")
	case errors.Is(err, errHooksPastCap):
		return refuse(StageDrain, "", "", "the drive cannot start: "+err.Error(),
			"keep the hooks directory under the bound, then start the drive again; nothing was made")
	}
	return refuse(StageDrain, "", "", "the drive cannot start: "+fsutil.RedactHome(err.Error()),
		"settle what git reports for the repository's own settings (`git config --local --list --show-origin`) and for the settings the drive holds (`git config --list --show-origin`), then start the drive again")
}

// nameHooksDiffer renders a hooksDiffer through the refusal redactor.
func nameHooksDiffer(repoRoot string, h *hooksDiffer) string {
	return nameChanges(repoRoot, []string{fsutil.RedactHome(h.Error())})[0]
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

// remoteUnchanged refuses the lane's worktree stage when a setting the drive
// holds differs from what it read as it started (Context.remote). The stage
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
		fmt.Sprintf("settings the drive holds across its lanes (%s) changed since it started (%s), so the remote that would vouch for the lane's hooks, or a program the lane's git would start, is not the drive's", remoteIdentityHeld, strings.Join(keys, ", ")),
		"look at what changed them (`git config --show-origin --show-scope --get-regexp '"+heldPattern+"'`); the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

// checkoutHooksUnchanged refuses the lane's worktree stage when the
// checkout's hooks directory or info/attributes, as read now, differ from
// what the drive read as it started (Context.remote): an entry at any depth
// added, removed or changed, named. git's add runs in the checkout, so it runs
// the post-checkout hook of the hooks directory git takes there (hooksPath,
// read by the stage from every scope: checkoutHooksPath; for a relative one,
// the checkout's own working-tree copy) with the drive's rights; each lane of
// a drain is a run of its own, whose fingerprint takes that copy as it stands
// at its first check, so without this a copy an earlier lane changed would be
// the new run's starting point. A hooksPath other than the drive's is a
// change of core.hooksPath, which remoteUnchanged names; here it is read as
// the directory now named. The stage calls it just before the add, so a
// difference present at this read is refused before that add; a change made
// after this read and before git runs the hook is not seen, and nothing holds
// the directory through the add (the file's header). With nothing carried (a
// step the host runs, a single build) it checks nothing.
func checkoutHooksUnchanged(c Context, laneID, hooksPath string) error {
	if c.remote == nil {
		return nil
	}
	now, err := readCheckoutHooks(c.RepoRoot, c.remote.salt, hooksPath)
	if errors.Is(err, errHooksPastCap) {
		return refuse(string(StageWorktree), "", laneID,
			err.Error()+", and git's add of the lane's worktree runs the checkout's post-checkout hook with the drive's rights, so the lane is not made",
			"look at what grew the checkout's hooks directory; stop the drive and start it again once the hooks are as they should be; nothing was made")
	}
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"could not read the checkout's hooks directory again, so the post-checkout hook the lane's add would run cannot be shown to be the one the drive started with: "+err.Error(),
			"settle what git reports (`git config --list --show-origin`), then run the step again; nothing was made")
	}
	changed := configFingerprint{Files: c.remote.hooks}.changes(c.RepoRoot, configFingerprint{Files: now})
	if len(changed) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		"the checkout's hooks or info/attributes changed since the drive started ("+strings.Join(nameChanges(c.RepoRoot, changed), ", ")+"), and git's add of the lane's worktree runs the checkout's post-checkout hook with the drive's rights, so the lane is not made",
		"look at what changed the checkout's hooks directory or info/attributes; the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

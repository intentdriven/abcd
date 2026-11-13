package loop

// remoteid.go is what a drive holds across its lanes (iss-2611071820538335,
// iss-2611131308266006, rulings recorded in their decisions). The drive's git
// runs in two environments: the ask of origin runs through the network runner
// (gitutil.NetContext) with the person's home and settings, and the lane's
// add, commit, merge and reset run through the working-tree command
// (gitutil.WorkingTreeCommitCommand), which in a drive has the stage's home
// and the stage's global settings file. So the drive reads, in each of the
// two, the settings the run's fingerprint watches (watchedKey: what names a
// program git starts, or a file that adds more settings, origin's identity
// among them), at every scope that environment reads, with http.proxy and
// http.sslVerify beside them; the bytes of each file an include.path or
// includeIf.*.path setting names, whether or not its condition holds at the
// checkout, its path resolved by git in that environment; and the global and
// system attributes files git names there. It also reads the hooks directory
// git names at the checkout in the working-tree environment (hooksDirAt), at
// every depth, and the common git directory's info/attributes. It holds
// hashes of them in its own memory, never in a file anything in a lane could
// rewrite, and compares each environment's reading with that environment's
// own earlier one. Each run of a drain records its watched settings and the
// checkout's hooks afresh at its first check, so a lane's agent that changed
// origin's URL would otherwise choose the remote that vouches for the next
// lane's pin, one that changed the checkout's post-checkout hook would have it
// run in the next lane's add, and one that set a content filter would have it
// run in the next lane's commit.
//
// As it starts, where the hooks directory lies inside the checkout and the
// repository has an origin, the drive also compares the checkout's copy with
// the tip origin reports at that moment, under the settings as they stand at
// that moment (proveCheckoutHooks). Those settings are the drive's starting
// point and are not themselves proved, so the comparison narrows what a
// restart takes on trust and does not remove it.
//
// What one account cannot hold, the page states: the directory is read just
// before each add, and a change made after that read and before git runs the
// hook is not seen, since nothing holds the directory through the add; a
// change put back between the two reads of the settings around the ask of
// origin is not seen; what a hook runs from elsewhere (a script, make, a
// program on PATH) is not hashed; and a hooks directory outside the tree, or a
// repository with no origin, has nothing a drive's start can compare it with.
// Not held either: any other http.* key (a per-URL http.<url>.* among them),
// and an origin defined only by a legacy remotes or branches file in the git
// directory, which has no key to compare.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteIdentityHeld names, for a refusal and the docs, what settings a drive
// holds across its lanes (remoteIdentityKey), as each of its two environments
// reads them.
const remoteIdentityHeld = "the settings the run's fingerprint watches, at every scope, as the ask of origin and the lane's add and commit each read them (origin's and every remote's url, pushurl, uploadpack, receivepack, proxy and vcs; every url rewrite; core.hooksPath, core.sshCommand, core.gitProxy, core.askPass, core.fsmonitor, core.pager, core.editor and core.worktree; extensions.worktreeConfig; every credential.*; the protocol allow rules; content filters, diff and merge drivers, gpg programs and gpg.ssh.defaultKeyCommand, include and includeIf paths and the files they name), http.proxy and http.sslVerify, and the global and system attributes files"

// heldPattern is the extended regular expression a refusal's remedy names
// for remoteIdentityHeld's keys: watchedConfig with http.proxy and
// http.sslVerify.
var heldPattern = strings.TrimSuffix(watchedConfig, ")$") + `|http\.(proxy|sslverify))$`

// heldSettings is what one environment's git reads that a drive holds: for
// each held key, a hash of its scopes and values in order (no value is
// kept), and, by path, each file an include setting names and the global and
// system attributes files, hashed under the drive's salt.
type heldSettings struct {
	keys  map[string]string
	files map[string]string
}

// remoteIdentity is what a drive read as it started: the settings as the ask
// of origin reads them (ask) and as the lane's add and commit read them
// (tree); each entry of the hooks directory git names at the checkout and of
// info/attributes, hashed as the run's fingerprint hashes them (hooks); that
// directory (hooksDir); and the salt the files are hashed under and key names
// are hidden under when a refusal names them (hiddenKey).
type remoteIdentity struct {
	ask, tree heldSettings
	hooks     map[string]string
	hooksDir  string
	salt      string
}

// settingsEnv is one environment a drive's git runs in: its name, for a
// refusal, and how a read-only git command runs in it.
type settingsEnv struct {
	name string
	run  func(ctx context.Context, dir string, args ...string) (string, error)
}

// maxSettingsBytes caps one listing of the settings in the working-tree
// environment.
const maxSettingsBytes = 1 << 20

// askEnv is the environment of the ask of origin (gitutil.NetContext);
// treeEnv that of the lane's add, commit, merge and reset
// (gitutil.WorkingTreeCommitCommand).
var (
	askEnv  = settingsEnv{name: "the ask of origin", run: gitutil.NetContext}
	treeEnv = settingsEnv{name: "the lane's add and commit", run: func(ctx context.Context, dir string, args ...string) (string, error) {
		return workingTreeAsk(ctx, dir, maxSettingsBytes, args...)
	}}
)

// remoteIdentityKey says whether a canonical config key is one a drive holds
// across its lanes: every key the run's fingerprint watches (watchedKey, the
// one predicate for both), and http.proxy and http.sslVerify, which choose how
// origin is reached without naming a program.
func remoteIdentityKey(key string) bool {
	return watchedKey(key) || key == "http.proxy" || key == "http.sslverify"
}

// readRemoteIdentity is what a drive holds as it starts: its settings in both
// environments (readHeldSettings) and the checkout's hooks directory and
// info/attributes (readCheckoutHooks), hashed under a fresh salt. It fails
// when either environment's settings cannot be listed or its files cannot be
// named, no salt can be drawn, git cannot name the hooks directory it runs at
// the checkout, the common git directory cannot be resolved, or the hooks
// directory holds more than maxHookEntries entries. A hooks directory that
// cannot be read is recorded as such (hashDir) and compared as an entry is;
// where it lies inside the checkout and the repository has an origin, the
// start-time comparison refuses it (proveCheckoutHooks).
func readRemoteIdentity(repoRoot string) (*remoteIdentity, error) {
	salt, err := newSalt()
	if err != nil {
		return nil, err
	}
	id := &remoteIdentity{salt: salt}
	if id.ask, id.tree, err = readSettings(repoRoot, salt); err != nil {
		return nil, err
	}
	if id.hooksDir, err = hooksDirAt(repoRoot); err != nil {
		return nil, fmt.Errorf("git cannot name the hooks directory it runs at the checkout: %s", fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	if id.hooks, err = readCheckoutHooks(repoRoot, salt, id.hooksDir); err != nil {
		return nil, err
	}
	return id, nil
}

// readSettings reads the held settings in both environments under salt.
func readSettings(repoRoot, salt string) (heldSettings, heldSettings, error) {
	ask, err := readHeldSettings(askEnv, repoRoot, salt)
	if err != nil {
		return heldSettings{}, heldSettings{}, err
	}
	tree, err := readHeldSettings(treeEnv, repoRoot, salt)
	if err != nil {
		return heldSettings{}, heldSettings{}, err
	}
	return ask, tree, nil
}

// errHooksPastCap is readCheckoutHooks's refusal of a hooks directory past
// maxHookEntries entries.
var errHooksPastCap = fmt.Errorf("the checkout's hooks directory holds more than %d entries, so the drive cannot hold it whole", maxHookEntries)

// readCheckoutHooks hashes, under salt, each entry at every depth of hooksDir
// (the directory git names at the checkout: hooksDirAt), through the
// fingerprint's own walk (hashDir), and the common git directory's
// info/attributes. It fails when the common git directory cannot be resolved
// or the tree is past maxHookEntries entries.
func readCheckoutHooks(repoRoot, salt, hooksDir string) (map[string]string, error) {
	common, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("reading the checkout's hooks directory: %s", fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	hooks := map[string]string{}
	if _, over := hashDir(hooks, salt, hooksDir); over {
		return nil, errHooksPastCap
	}
	hashFile(hooks, salt, filepath.Join(common, "info", "attributes"))
	return hooks, nil
}

// gitScopes are the scope names `git config --show-scope` gives a setting
// read without --file or --blob.
var gitScopes = map[string]bool{"system": true, "global": true, "local": true, "worktree": true, "command": true}

// scopedSetting is one record of a `git config --null --list --show-scope
// --show-origin` listing.
type scopedSetting struct {
	scope, origin, key, value string
	noValue                   bool
}

// parseScopedList reads a `git config --null --list --show-scope
// --show-origin` listing: scope, origin and record, each NUL-terminated, the
// record a key and, after a newline, its value (no newline: a key written
// with no value). A listing that is empty, is not whole records, names a
// scope git does not give, or holds an empty key is refused: it is never read
// as holding nothing.
func parseScopedList(out string) ([]scopedSetting, error) {
	body, ok := strings.CutSuffix(out, "\x00")
	if !ok || body == "" {
		return nil, errors.New("git listed no settings, or did not end its listing as documented")
	}
	fields := strings.Split(body, "\x00")
	if len(fields)%3 != 0 {
		return nil, errors.New("git listed the settings in a form it does not document")
	}
	recs := make([]scopedSetting, 0, len(fields)/3)
	for i := 0; i < len(fields); i += 3 {
		if !gitScopes[fields[i]] {
			return nil, errors.New("git listed a setting under a scope name that is not one of git's")
		}
		key, value, found := strings.Cut(fields[i+2], "\n")
		if key == "" {
			return nil, errors.New("git listed a setting with no key")
		}
		recs = append(recs, scopedSetting{scope: fields[i], origin: fields[i+1], key: key, value: value, noValue: !found})
	}
	return recs, nil
}

// readHeldSettings reads, in env at repoRoot, what a drive holds of that
// environment's settings: `git config --null --list --show-scope
// --show-origin`, each held key's scopes and values hashed in order (a key
// with no value apart from one with an empty value); each file an
// include.path or includeIf.*.path names, whatever its condition, resolved
// by git in env (`git config --type=path`, so a home form is that
// environment's home) and, when relative, under the directory of the file
// that names it, hashed by its bytes under salt (holdInclude); and the
// global and system attributes files git names in env (`git var
// GIT_ATTR_GLOBAL`, `GIT_ATTR_SYSTEM`), hashed likewise. The listing needs git
// 2.26 or later and the attributes files 2.42; a git that cannot answer, or a
// listing of another shape, is an error, which stops the drive or the lane.
// It is bounded by originAskTimeout.
func readHeldSettings(env settingsEnv, repoRoot, salt string) (heldSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	fail := func(what string, err error) (heldSettings, error) {
		return heldSettings{}, fmt.Errorf("%s in the environment of %s: %s", what, env.name, fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())))
	}
	out, err := env.run(ctx, repoRoot, "config", "--null", "--list", "--show-scope", "--show-origin")
	if err != nil {
		return fail("reading "+Remote+"'s settings (git config --show-scope, which needs git 2.26 or later)", err)
	}
	recs, err := parseScopedList(out)
	if err != nil {
		return fail("reading "+Remote+"'s settings", err)
	}
	held := heldSettings{keys: map[string]string{}, files: map[string]string{}}
	values := map[string][]string{}
	for _, r := range recs {
		if !remoteIdentityKey(r.key) {
			continue
		}
		v := "none"
		if !r.noValue {
			v = "=" + r.value
		}
		values[r.key] = append(values[r.key], r.scope, v)
		if r.key == "include.path" || (strings.HasPrefix(r.key, "includeif.") && strings.HasSuffix(r.key, ".path")) {
			holdInclude(ctx, env, repoRoot, salt, held.files, r)
		}
	}
	for _, name := range []string{"GIT_ATTR_GLOBAL", "GIT_ATTR_SYSTEM"} {
		p, err := env.run(ctx, repoRoot, "var", name)
		if err != nil {
			return fail("asking git for its attributes files (git var "+name+", which needs git 2.42 or later)", err)
		}
		if p = strings.TrimSuffix(p, "\n"); p != "" {
			if !filepath.IsAbs(p) {
				p = filepath.Join(repoRoot, p)
			}
			hashFile(held.files, salt, p)
		}
	}
	for k, vs := range values {
		h := sha256.New()
		for _, v := range vs {
			writePart(h, v)
		}
		held.keys[k] = hex.EncodeToString(h.Sum(nil))
	}
	return held, nil
}

// holdInclude records into files the file an include setting r names, as git
// resolves its path in env: by its bytes (hashFile; a file not there records
// nothing, so its appearing is a change). A path git cannot resolve, or a
// relative one whose including file is not a file, is recorded under a name
// that carries a salted id in place of the value, so its becoming resolvable
// is a change too.
func holdInclude(ctx context.Context, env settingsEnv, repoRoot, salt string, files map[string]string, r scopedSetting) {
	unresolved := func() {
		files["an include path that cannot be resolved <hidden-"+saltedHash(salt, "include", r.value)[:16]+">"] = saltedHash(salt, "include", "unresolved")
	}
	if r.noValue {
		unresolved()
		return
	}
	p, err := env.run(ctx, repoRoot, "config", "--file", os.DevNull, "--type=path", "--default", r.value, "--get", "abcd.include")
	p = strings.TrimSuffix(p, "\n")
	if err != nil || p == "" || strings.ContainsAny(p, "\n\x00") {
		unresolved()
		return
	}
	if !filepath.IsAbs(p) {
		from, ok := strings.CutPrefix(r.origin, "file:")
		if !ok || from == "" {
			unresolved()
			return
		}
		if !filepath.IsAbs(from) {
			from = filepath.Join(repoRoot, from)
		}
		p = filepath.Join(filepath.Dir(from), p)
	}
	hashFile(files, salt, filepath.Clean(p))
}

// proveCheckoutHooks is a drive's comparison, as it starts, of the checkout's
// hooks directory with the tip origin reports at that moment, under the
// settings as they stand at that moment: where the directory git names
// (id.hooksDir) lies inside the checkout (ownHooksDir) and the repository has
// an origin, what stands there must be, at every depth, the hooks directory
// of the tip origin reports for its default branch, as stored (hooksTreeAt,
// hookTreeEntries, provenHooks: each name, each file's bytes and exec bit,
// each symlink's target, and what a link to elsewhere in the checkout points
// at). A difference refuses the start (startHooksRefusal says which kind).
// An origin that cannot be reached for a network reason is a contention
// refusal, which the drive waits on as a lane's ask of origin waits; one that
// cannot be asked otherwise, or a tip this repository does not hold yet,
// refuses the start. A hooks directory outside the checkout, or no origin,
// has nothing to be compared with, and passes. It reads and refuses only:
// nothing is written, and a person's own git runs the hooks as before.
func proveCheckoutHooks(repoRoot string, id *remoteIdentity) error {
	rel := ownHooksDir(repoRoot, id.hooksDir)
	if rel == "" {
		return nil
	}
	has, err := claim.HasOrigin(repoRoot)
	if err != nil || !has {
		return err
	}
	shown := nameChanges(repoRoot, []string{rel})[0]
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	what := "git ls-remote " + Remote
	_, tip, err := remoteHeadTip(ctx, nil, repoRoot, Remote)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("%s did not answer within %s: operation timed out", Remote, originAskTimeout)
		}
		why := gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error()))
		if netFailure(StageDrain, "", what, err) != nil {
			return contend(StageDrain, "", "",
				"the drive cannot start yet: it lost the network asking "+Remote+" for its default branch's tip, against which the checkout's hooks in "+shown+" are compared: "+why,
				"keep to offline work; the drive waits and asks again, as a lane's ask of "+Remote+" waits; nothing was made")
		}
		return refuse(StageDrain, "", "",
			"the drive cannot start: it could not ask "+Remote+" for its default branch's tip, so the checkout's hooks in "+shown+" cannot be compared with it: "+why,
			"settle what git reports when it asks "+Remote+" for its default branch (git ls-remote --symref "+Remote+" HEAD), then start the drive again; nothing was made")
	}
	if _, err := gitutil.RunCappedAsStored(repoRoot, 4096, "cat-file", "-e", tip+"^{commit}"); err != nil {
		return refuse(StageDrain, "", "",
			fmt.Sprintf("the drive cannot start: %s reports its tip at %s, which this repository does not hold yet, so the checkout's hooks in %s cannot be compared with it", Remote, tip[:12], shown),
			"fetch "+Remote+" (`git fetch "+Remote+"`), then start the drive again; nothing was made")
	}
	tree, err := hooksTreeAt(repoRoot, tip, rel)
	if err != nil {
		return refuse(StageDrain, "", "",
			fmt.Sprintf("the drive cannot start: the hooks directory %s of %s's tip %s cannot be listed: %s", shown, Remote, tip[:12], nameChanges(repoRoot, []string{fsutil.RedactHome(err.Error())})[0]),
			"settle what git reports when it lists that directory in the tip (git ls-tree), then start the drive again; nothing was made")
	}
	dir := filepath.Join(repoRoot, filepath.FromSlash(rel))
	want, err := hookTreeEntries(repoRoot, tree)
	if err == nil {
		_, err = provenHooks(id.salt, dir, want, &proofTree{repoRoot: repoRoot, commit: tip, dir: repoRoot})
	}
	if err != nil {
		return startHooksRefusal(repoRoot, rel, tip, dir, id.salt, err)
	}
	return nil
}

// startHooksRefusal is the drive's refusal to start on a checkout whose
// in-tree hooks are not those of origin's tip, saying which of four it met:
// committed hooks that link outside the checkout, which no start can prove;
// files in the hooks directory git does not track; a checkout behind the tip,
// whose hooks are its own commit's; or files that differ from the tip's. Each
// file is named through the refusal redactor; no value read from the
// settings is placed in a command to paste.
func startHooksRefusal(repoRoot, rel, tip, dir, salt string, err error) error {
	shown := nameChanges(repoRoot, []string{rel})[0]
	named := nameChanges(repoRoot, []string{fsutil.RedactHome(err.Error())})[0]
	var pe *proofError
	isProof := errors.As(err, &pe)
	switch {
	case isProof && len(pe.linkOut) > 0:
		return refuse(StageDrain, "", "",
			"the drive cannot start: the repository's committed hooks in "+shown+" link outside it, so they cannot be compared with "+Remote+"'s tip "+tip[:12]+", and a drive runs no lane there: "+named,
			"the links are the repository's own and nothing in this checkout changes them; a drive is not run in this repository while its committed hooks link outside it; nothing was made")
	case isProof && len(pe.extra) == len(pe.problems) && untracked(repoRoot, pe.extra):
		return refuse(StageDrain, "", "",
			"the drive cannot start: files git does not track stand in the checkout's hooks directory "+shown+", which "+Remote+"'s tip "+tip[:12]+" does not hold: "+named,
			"move the files git does not track out of the hooks directory, then start the drive again; nothing was made")
	case behindTip(repoRoot, rel, tip, dir, salt):
		return refuse(StageDrain, "", "",
			"the drive cannot start: the checkout is behind "+Remote+"'s tip "+tip[:12]+": its hooks in "+shown+" are those of its own commit, and the tip holds others: "+named,
			"bring the checkout up to "+Remote+"'s tip (`git pull --ff-only`), then start the drive again; nothing was made")
	}
	return refuse(StageDrain, "", "",
		"the drive cannot start: the checkout's hooks in "+shown+" differ from those of "+Remote+"'s tip "+tip[:12]+": "+named,
		"look at what changed the checkout's hooks directory (git status, and git diff against "+tip[:12]+", name it); land the change through the default branch, or put the files back as the tip holds them, then start the drive again; the drive takes the settings of every scope as they stand when it starts (`git config --list --show-scope --show-origin`); nothing was made")
}

// untracked says whether git tracks none of paths, each under repoRoot.
func untracked(repoRoot string, paths []string) bool {
	args := []string{"--literal-pathspecs", "ls-files", "-z", "--"}
	for _, p := range paths {
		rel, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return false
		}
		args = append(args, filepath.ToSlash(rel))
	}
	out, err := gitutil.RunCappedAsStored(repoRoot, 1<<16, args...)
	return err == nil && out == ""
}

// behindTip says whether the checkout's commit is an ancestor of tip that
// holds other hooks at rel than tip does, and the hooks standing in dir are
// exactly that commit's.
func behindTip(repoRoot, rel, tip, dir, salt string) bool {
	head, err := gitutil.RunCappedAsStored(repoRoot, 4096, "rev-parse", "--verify", "--quiet", "HEAD^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(head) || head == tip {
		return false
	}
	mine, err := hooksTreeAt(repoRoot, head, rel)
	if err != nil {
		return false
	}
	if theirs, err := hooksTreeAt(repoRoot, tip, rel); err != nil || theirs == mine {
		return false
	}
	if _, err := gitutil.RunCappedAsStored(repoRoot, 4096, "merge-base", "--is-ancestor", head, tip); err != nil {
		return false
	}
	want, err := hookTreeEntries(repoRoot, mine)
	if err != nil {
		return false
	}
	_, err = provenHooks(salt, dir, want, &proofTree{repoRoot: repoRoot, commit: head, dir: repoRoot})
	return err == nil
}

// startRefusal is the drive's refusal to start on an error from
// readRemoteIdentity or proveCheckoutHooks: a refusal already made is passed
// on.
func startRefusal(repoRoot string, err error) error {
	if r, ok := AsRefusal(err); ok {
		return r
	}
	if errors.Is(err, errHooksPastCap) {
		return refuse(StageDrain, "", "", "the drive cannot start: "+err.Error(),
			"keep the hooks directory under the bound, then start the drive again; nothing was made")
	}
	return refuse(StageDrain, "", "", "the drive cannot start: "+nameChanges(repoRoot, []string{fsutil.RedactHome(err.Error())})[0],
		"settle what git reports for the settings of every scope (`git config --list --show-scope --show-origin`) and the hooks directory it names (`git rev-parse --git-path hooks`), then start the drive again; nothing was made")
}

// changed names, sorted, each held key either environment now reads otherwise
// than it did (its URL subsection hidden, hiddenKey), and each held file that
// was added, removed or changed in either, through the refusal redactor.
// Each environment is compared with its own earlier reading, never with the
// other. Never a value.
func (was *remoteIdentity) changed(repoRoot string, now *remoteIdentity) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, pair := range [][2]heldSettings{{was.ask, now.ask}, {was.tree, now.tree}} {
		a, b := pair[0], pair[1]
		for k, h := range a.keys {
			if b.keys[k] != h {
				add(hiddenKey(was.salt, k))
			}
		}
		for k := range b.keys {
			if _, ok := a.keys[k]; !ok {
				add(hiddenKey(was.salt, k))
			}
		}
		for _, name := range nameChanges(repoRoot, configFingerprint{Files: a.files}.changes(repoRoot, configFingerprint{Files: b.files})) {
			add(name)
		}
	}
	slices.Sort(out)
	return out
}

// remoteUnchanged refuses the lane's worktree stage when a setting or file
// the drive holds differs, in either environment, from what that environment
// read as the drive started (Context.remote). The stage calls it before it
// asks origin anything and again once origin has answered, so a change made
// while the ask ran and left in place is refused too; one made and put back
// between the two reads is not seen. A trial's lane calls it once, since it
// asks origin nothing. With nothing carried it checks nothing.
func remoteUnchanged(c Context, laneID string) error {
	if c.remote == nil {
		return nil
	}
	ask, tree, err := readSettings(c.RepoRoot, c.remote.salt)
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"could not read the settings the drive holds again, so the remote the lane would be checked against, and the programs the lane's git would start, cannot be shown to be the ones the drive started with: "+nameChanges(c.RepoRoot, []string{err.Error()})[0],
			"settle what git reports for the settings of every scope (`git config --list --show-scope --show-origin`), then run the step again; nothing was made")
	}
	keys := c.remote.changed(c.RepoRoot, &remoteIdentity{ask: ask, tree: tree})
	if len(keys) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		fmt.Sprintf("settings the drive holds across its lanes (%s) changed since it started (%s), so the remote that would vouch for the lane's hooks, or a program the lane's git would start, is not the drive's", remoteIdentityHeld, strings.Join(keys, ", ")),
		"look at what changed them (`git config --show-origin --show-scope --get-regexp '"+heldPattern+"'`, and the files named); the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

// checkoutHooksUnchanged refuses the lane's worktree stage when the
// checkout's hooks directory or info/attributes, as read now, differ from
// what the drive read as it started (Context.remote): an entry at any depth
// added, removed or changed, named, or a tree grown past maxHookEntries
// entries. git's add runs in the checkout, so it runs the post-checkout hook
// of the hooks directory git names there (hooksDir, asked of git by the stage
// in the add's own environment: hooksDirAt) with the drive's rights; each lane
// of a drain is a run of its own, whose fingerprint takes that copy as it
// stands at its first check, so without this a copy an earlier lane changed
// would be the new run's starting point. A directory other than the drive's
// reads as its entries removed and others added. The stage calls it just
// before the add, so a difference present at this read is refused before that
// add; a change made after this read and before git runs the hook is not
// seen, and nothing holds the directory through the add (the file's header).
// With nothing carried (a step the host runs, a single build) it checks
// nothing.
func checkoutHooksUnchanged(c Context, laneID, hooksDir string) error {
	if c.remote == nil {
		return nil
	}
	now, err := readCheckoutHooks(c.RepoRoot, c.remote.salt, hooksDir)
	if errors.Is(err, errHooksPastCap) {
		return refuse(string(StageWorktree), "", laneID,
			err.Error()+", and git's add of the lane's worktree runs the checkout's post-checkout hook with the drive's rights, so the lane is not made",
			"look at what grew the checkout's hooks directory; stop the drive and start it again once the hooks are as they should be; nothing was made")
	}
	if err != nil {
		return refuse(string(StageWorktree), "", laneID,
			"could not read the checkout's hooks directory again, so the post-checkout hook the lane's add would run cannot be shown to be the one the drive started with: "+err.Error(),
			"settle what git reports for the settings of every scope (`git config --list --show-scope --show-origin`), then run the step again; nothing was made")
	}
	changed := configFingerprint{Files: c.remote.hooks}.changes(c.RepoRoot, configFingerprint{Files: now})
	if len(changed) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		"the checkout's hooks or info/attributes changed since the drive started ("+strings.Join(nameChanges(c.RepoRoot, changed), ", ")+"), and git's add of the lane's worktree runs the checkout's post-checkout hook with the drive's rights, so the lane is not made",
		"look at what changed the checkout's hooks directory or info/attributes; the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

package loop

// remoteid.go is what a drive holds across its lanes (iss-2611071820538335,
// iss-2611131308266006, rulings recorded in their decisions). The drive's git
// runs in each environment driveGitCommands names (gitenvs.go): the person's
// own, where the landing push and the reservation's pushes and fetches run
// with the repository's hooks on and the ask of origin with them off; the
// lane's add, commit, merge and reset, which in a drive have the stage's home
// and the stage's global settings file; and abcd's isolated reads. So the
// drive reads, in each of them, the settings the run's fingerprint watches
// (watchedKey: what names a program git starts, or a file that adds more
// settings, origin's identity among them), at every scope that environment
// reads, with http.proxy and http.sslVerify beside them; the bytes of each
// file an include.path or includeIf.*.path setting names, whether or not its
// condition holds at the checkout, its path resolved by git in that
// environment; and the global and system attributes files git reads there.
// In each environment whose commands run hooks it also reads the hooks
// directory git names there (hooksDirIn), at every depth, each distinct
// directory once, and the common git directory's info/attributes. A read
// that gives no answer the hold can take (readFailure) stops the drive or
// the step; it is never read as nothing there. It holds hashes of them in
// its own memory, never in a file anything in a lane could rewrite, and
// compares each environment's reading with that environment's own earlier
// one. Each run of a drain records its watched settings and the checkout's
// hooks afresh at its first check, so a lane's agent that changed origin's
// URL would otherwise choose the remote that vouches for the next lane's pin,
// one that changed the checkout's post-checkout hook would have it run in the
// next lane's add, and one that set a content filter would have it run in the
// next lane's commit.
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
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteIdentityHeld names, for a refusal and the docs, what settings a drive
// holds across its lanes (remoteIdentityKey), as each environment its git
// runs in reads them.
const remoteIdentityHeld = "the settings the run's fingerprint watches, at every scope, as each environment the drive's git runs in reads them (origin's and every remote's url, pushurl, uploadpack, receivepack, proxy and vcs; every url rewrite; core.hooksPath, core.sshCommand, core.gitProxy, core.askPass, core.fsmonitor, core.pager, core.editor and core.worktree; extensions.worktreeConfig; every credential.*; the protocol allow rules; content filters, diff and merge drivers, gpg programs and gpg.ssh.defaultKeyCommand, include and includeIf paths and the files they name), http.proxy and http.sslVerify, and the global and system attributes files"

// heldPattern is the extended regular expression a refusal's remedy names
// for remoteIdentityHeld's keys: watchedConfig with http.proxy and
// http.sslVerify.
var heldPattern = strings.TrimSuffix(watchedConfig, ")$") + `|http\.(proxy|sslverify))$`

// remoteIdentity is what a drive read as it started: in each environment
// its git runs in (driveEnvs), each held key's hash (envs); each file an
// include setting names in any of them and the global and system attributes
// files each reads, once by path (files); each entry of every distinct hooks
// directory git names in an environment whose commands run hooks, and of
// info/attributes, hashed as the run's fingerprint hashes them (hooks); the
// directory the lane's add and commit run (hooksDir) and every directory
// held (hooksDirs); and the salt the files are hashed under and key names
// are hidden under when a refusal names them (hiddenKey).
type remoteIdentity struct {
	envs      map[*gitEnv]map[string]string
	files     map[string]string
	hooks     map[string]string
	hooksDir  string
	hooksDirs []string
	salt      string
}

// maxSettingsBytes caps one listing of the settings.
const maxSettingsBytes = 1 << 20

// remoteIdentityKey says whether a canonical config key is one a drive holds
// across its lanes: every key the run's fingerprint watches (watchedKey, the
// one predicate for both), and http.proxy and http.sslVerify, which choose how
// origin is reached without naming a program.
func remoteIdentityKey(key string) bool {
	return watchedKey(key) || key == "http.proxy" || key == "http.sslverify"
}

// readRemoteIdentity is what a drive holds as it starts: its settings in
// every environment its git runs in (readSettings) and every hooks directory
// git runs and info/attributes (readHooks), hashed under a fresh salt. It
// fails when a read the hold depends on gives no answer it can take (a
// readFailure, naming the read), no salt can be drawn, the common git
// directory cannot be resolved, or a hooks directory holds more than
// maxHookEntries entries. A hooks directory that cannot be read is recorded
// as such (hashDir) and compared as an entry is; where it lies inside the
// checkout and the repository has an origin, the start-time comparison
// refuses it (proveCheckoutHooks).
func readRemoteIdentity(repoRoot string) (*remoteIdentity, error) {
	salt, err := newSalt()
	if err != nil {
		return nil, err
	}
	id := &remoteIdentity{salt: salt}
	if id.envs, id.files, err = readSettings(repoRoot, salt); err != nil {
		return nil, err
	}
	if id.hooksDir, err = hooksDirAt(repoRoot); err != nil {
		return nil, err
	}
	if id.hooks, id.hooksDirs, err = readHooks(repoRoot, salt, id.hooksDir); err != nil {
		return nil, err
	}
	return id, nil
}

// readSettings reads the held settings in every environment a drive's git
// runs in (driveEnvs) under salt: each environment's keys, and the files
// every environment names, once by path.
func readSettings(repoRoot, salt string) (map[*gitEnv]map[string]string, map[string]string, error) {
	envs, files := map[*gitEnv]map[string]string{}, map[string]string{}
	for _, e := range driveEnvs() {
		keys, err := readHeldSettings(e, repoRoot, salt, files)
		if err != nil {
			return nil, nil, err
		}
		envs[e] = keys
	}
	return envs, files, nil
}

// errHooksPastCap is readCheckoutHooks's refusal of a hooks directory past
// maxHookEntries entries.
var errHooksPastCap = fmt.Errorf("the checkout's hooks directory holds more than %d entries, so the drive cannot hold it whole", maxHookEntries)

// readHooks is every hooks directory a drive's git runs at the checkout:
// stageDir, the one the lane's add and commit run (hooksDirAt), and the one
// git names in each other environment whose commands run hooks (the landing
// push's and the reservation's, in the person's own), each distinct
// directory once, hashed with info/attributes (readCheckoutHooks). It
// returns the record and the directories.
func readHooks(repoRoot, salt, stageDir string) (map[string]string, []string, error) {
	dirs := []string{stageDir}
	for _, e := range driveEnvs() {
		if e == stageEnv || !e.hooks() {
			continue
		}
		d, err := hooksDirIn(e, repoRoot)
		if err != nil {
			return nil, nil, err
		}
		if !slices.ContainsFunc(dirs, func(h string) bool { return samePath(repoRoot, h, d) }) {
			dirs = append(dirs, d)
		}
	}
	hooks, err := readCheckoutHooks(repoRoot, salt, dirs...)
	return hooks, dirs, err
}

// readCheckoutHooks hashes, under salt, each entry at every depth of each
// of dirs (directories git names at the checkout: readHooks), through the
// fingerprint's own walk (hashDir), and the common git directory's
// info/attributes. It fails when the common git directory cannot be resolved
// or a tree is past maxHookEntries entries.
func readCheckoutHooks(repoRoot, salt string, dirs ...string) (map[string]string, error) {
	common, err := gitutil.RevParseAbsPath(repoRoot, "--git-common-dir")
	if err != nil {
		return nil, &readFailure{what: "resolving the common git directory (git rev-parse --git-common-dir)", why: fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))}
	}
	hooks := map[string]string{}
	for _, dir := range dirs {
		if _, over := hashDir(hooks, salt, dir); over {
			return nil, errHooksPastCap
		}
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
// with no value apart from one with an empty value), returned; and into
// files, under salt, by its bytes, each file an include.path or
// includeIf.*.path names, whatever its condition (holdInclude), and the
// global and system attributes files git reads in env (attributesFiles).
// The listing needs git 2.26 or later, as the run's fingerprint does. A read
// that gives no answer the hold can take is a readFailure naming it, which
// stops the drive or the step.
func readHeldSettings(env *gitEnv, repoRoot, salt string, files map[string]string) (map[string]string, error) {
	what := "listing " + Remote + "'s and the other settings a drive holds (git config --list --show-scope --show-origin)"
	a, err := env.read(repoRoot, what, maxSettingsBytes, "config", "--null", "--list", "--show-scope", "--show-origin")
	if err != nil {
		return nil, err
	}
	if a.code != 0 {
		return nil, env.exited(what, a)
	}
	recs, err := parseScopedList(a.out)
	if err != nil {
		return nil, env.failed(what, err.Error())
	}
	keys := map[string]string{}
	values := map[string][]string{}
	system := ""
	for _, r := range recs {
		if r.scope == "system" && system == "" {
			if f, ok := strings.CutPrefix(r.origin, "file:"); ok && filepath.IsAbs(f) {
				system = f
			}
		}
		if !remoteIdentityKey(r.key) {
			continue
		}
		v := "none"
		if !r.noValue {
			v = "=" + r.value
		}
		values[r.key] = append(values[r.key], r.scope, v)
		if r.key == "include.path" || (strings.HasPrefix(r.key, "includeif.") && strings.HasSuffix(r.key, ".path")) {
			if err := holdInclude(env, repoRoot, salt, files, r); err != nil {
				return nil, err
			}
		}
	}
	if _, set := lastEnv(env.environ(), "GIT_CONFIG_SYSTEM"); set {
		system = ""
	}
	attrs, err := attributesFiles(env, repoRoot, system)
	if err != nil {
		return nil, err
	}
	for _, p := range attrs {
		if !filepath.IsAbs(p) {
			p = filepath.Join(repoRoot, p)
		}
		hashFile(files, salt, p)
	}
	for k, vs := range values {
		h := sha256.New()
		for _, v := range vs {
			writePart(h, v)
		}
		keys[k] = hex.EncodeToString(h.Sum(nil))
	}
	return keys, nil
}

// holdInclude records into files the file an include setting r names, as git
// resolves its path in env, by its bytes (hashFile; a file not there records
// nothing, so its appearing is a change). A setting with no value names no
// file and holds none. A path git cannot resolve, a resolution that does not
// finish, or a relative path whose setting does not come from a file is a
// readFailure that names the setting by its key and scope.
func holdInclude(env *gitEnv, repoRoot, salt string, files map[string]string, r scopedSetting) error {
	if r.noValue {
		return nil
	}
	what := "resolving the path of the include setting " + hiddenKey(salt, r.key) + " (" + r.scope + " scope) as git does (git config --type=path)"
	// What git says of a path it cannot resolve repeats the path, so only
	// its status is named, and a read that failed otherwise names the path
	// as the path.
	a, err := env.read(repoRoot, what, 4096, "config", "--file", os.DevNull, "--type=path", "--default", r.value, "--get", "abcd.include")
	if err != nil {
		var rf *readFailure
		if errors.As(err, &rf) && r.value != "" {
			for _, v := range []string{r.value, fsutil.RedactHome(r.value)} {
				rf.why = strings.ReplaceAll(rf.why, v, "<the path>")
			}
		}
		return err
	}
	if a.code != 0 {
		return env.failed(what, "git could not resolve it (exit "+strconv.Itoa(a.code)+")")
	}
	p, ok := onePath(a.out)
	if !ok {
		return env.failed(what, "git did not answer with one path, so the answer is not taken")
	}
	if !filepath.IsAbs(p) {
		from, ok := strings.CutPrefix(r.origin, "file:")
		if !ok || from == "" {
			return env.failed(what, "the path is relative and the setting does not come from a file, so it has no directory to stand under")
		}
		if !filepath.IsAbs(from) {
			from = filepath.Join(repoRoot, from)
		}
		p = filepath.Join(filepath.Dir(from), p)
	}
	hashFile(files, salt, filepath.Clean(p))
	return nil
}

// proveCheckoutHooks is a drive's comparison, as it starts, of the checkout's
// hooks with the tip origin reports at that moment, under the settings as
// they stand at that moment: where a hooks directory the drive holds
// (id.hooksDirs) lies inside the checkout (ownHooksDir) and the repository
// has an origin, what stands there must be, at every depth, that directory
// of the tip origin reports for its default branch, as stored (hooksTreeAt,
// hookTreeEntries, provenHooks: each name, each file's bytes and exec bit,
// each symlink's target, and what a link to elsewhere in the checkout points
// at). A difference refuses the start (startHooksRefusal says which kind).
// The ask of origin is a lane's: while the run's shared outage has the
// network down it is not made and the start waits on the shared network
// probe (g), and one that loses the network is reported to that outage
// (netFailure, g.take) and is a contention refusal, which the drive waits
// on as it waits on a lane's. One that cannot be asked otherwise, or a tip
// this repository does not hold yet, refuses the start. A hooks directory
// outside the checkout, or no origin, has nothing to be compared with, and
// passes. It reads and refuses only: nothing is written, and a person's own
// git runs the hooks as before.
func proveCheckoutHooks(repoRoot string, id *remoteIdentity, g *outageGate) error {
	var rels []string
	for _, d := range id.hooksDirs {
		rel, err := ownHooksDir(repoRoot, d)
		if err != nil {
			return err
		}
		if rel != "" && !slices.Contains(rels, rel) {
			rels = append(rels, rel)
		}
	}
	if len(rels) == 0 {
		return nil
	}
	has, err := claim.HasOrigin(repoRoot)
	if err != nil || !has {
		return err
	}
	shown := strings.Join(nameChanges(repoRoot, rels), ", ")
	what := "git ls-remote " + Remote
	if g != nil && g.down(implement.ServiceNetwork) {
		r := contend(string(StageDrain), "", "",
			"the drive cannot start yet: asking "+Remote+" for its default branch's tip, against which the checkout's hooks in "+shown+" are compared, reaches the network, which is down; it "+g.waitText(implement.ServiceNetwork),
			g.waitRemedy(implement.ServiceNetwork))
		r.Outage = g.info()
		return r
	}
	ctx, cancel := context.WithTimeout(context.Background(), originAskTimeout)
	defer cancel()
	_, tip, err := remoteHeadTip(ctx, nil, repoRoot, Remote)
	if err != nil {
		if ctx.Err() != nil {
			err = fmt.Errorf("%s did not answer within %s: operation timed out", Remote, originAskTimeout)
		}
		why := gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error()))
		if r := netFailure(StageDrain, "", what, err); r != nil {
			r.Reason = "the drive cannot start yet: it lost the network asking " + Remote + " for its default branch's tip, against which the checkout's hooks in " + shown + " are compared: " + why
			r.Remedy = "keep to offline work; the drive waits as it waits on a lane's ask of " + Remote + ", within its own bounds, and asks again; nothing was made"
			r.outage.lane = startOutageLane
			if g != nil {
				g.take(r)
			}
			return r
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
	for _, rel := range rels {
		shown := nameChanges(repoRoot, []string{rel})[0]
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
	}
	return nil
}

// startOutageLane is the name a drive's start reports a lost network to the
// run's shared outage under, since it has no lane yet.
const startOutageLane = "drive-start"

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
		"look at what changed the checkout's hooks directory (git status, and git diff against "+tip[:12]+", name it); land the change through the default branch, or put the files back as the tip holds them, then start the drive again; the drive takes the settings of every scope as they stand when it starts, in each environment its git runs in (`git config --list --show-scope --show-origin`, run as yourself for the push's, and with HOME set to the stage's home for the lane's add and commit); nothing was made")
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
// on; a read the hold depends on that gave no answer it can take
// (readFailure) is refused at the config stage, naming the read.
func startRefusal(repoRoot string, err error) error {
	if r, ok := AsRefusal(err); ok {
		return r
	}
	if errors.Is(err, errHooksPastCap) {
		return refuse(StageDrain, "", "", "the drive cannot start: "+err.Error(),
			"keep the hooks directory under the bound, then start the drive again; nothing was made")
	}
	var rf *readFailure
	if errors.As(err, &rf) {
		return refuse(StageConfig, "", "", "the drive cannot start: a read its hold depends on gave no answer it can take: "+nameChanges(repoRoot, []string{fsutil.RedactHome(err.Error())})[0],
			readRemedy("start the drive again"))
	}
	return refuse(StageDrain, "", "", "the drive cannot start: "+nameChanges(repoRoot, []string{fsutil.RedactHome(err.Error())})[0],
		readRemedy("start the drive again"))
}

// readRemedy is what the person does about a read the hold could not take,
// then.
func readRemedy(then string) string {
	return "run the read the reason names in the environment it names (as yourself for the person's own; with HOME set to the stage's home, as a drive sets it, for the lane's add and commit) and settle what git answers there, then " + then + "; nothing was made"
}

// changed names, sorted, each held key any environment now reads otherwise
// than it did (its URL subsection hidden, hiddenKey), and each held file that
// was added, removed or changed, through the refusal redactor. Each
// environment's keys are compared with its own earlier reading, never with
// another's. Never a value.
func (was *remoteIdentity) changed(repoRoot string, now *remoteIdentity) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for e, a := range was.envs {
		b := now.envs[e]
		for k, h := range a {
			if b[k] != h {
				add(hiddenKey(was.salt, k))
			}
		}
		for k := range b {
			if _, ok := a[k]; !ok {
				add(hiddenKey(was.salt, k))
			}
		}
	}
	for _, name := range nameChanges(repoRoot, configFingerprint{Files: was.files}.changes(repoRoot, configFingerprint{Files: now.files})) {
		add(name)
	}
	slices.Sort(out)
	return out
}

// heldReadRefusal is the refusal of the lane's step when a read the hold
// depends on gave no answer it can take: it stops at the config stage,
// naming the read and why.
func heldReadRefusal(c Context, laneID string, err error) error {
	return refuse(StageConfig, "", laneID,
		"a read the drive's hold depends on gave no answer it can take, so what the lane's git would run cannot be shown to be what the drive started with: "+nameChanges(c.RepoRoot, []string{fsutil.RedactHome(err.Error())})[0],
		readRemedy("run the step again"))
}

// remoteUnchanged refuses the lane's worktree stage when a setting or file
// the drive holds differs, in any environment its git runs in, from what
// that environment read as the drive started (Context.remote). The stage
// calls it before it asks origin anything and again once origin has
// answered, so a change made while the ask ran and left in place is refused
// too; one made and put back between the two reads is not seen. A trial's
// lane calls it once, since it asks origin nothing. A read that gives no
// answer the hold can take stops the step at the config stage. With nothing
// carried it checks nothing.
func remoteUnchanged(c Context, laneID string) error {
	if c.remote == nil {
		return nil
	}
	envs, files, err := readSettings(c.RepoRoot, c.remote.salt)
	if err != nil {
		return heldReadRefusal(c, laneID, err)
	}
	keys := c.remote.changed(c.RepoRoot, &remoteIdentity{envs: envs, files: files})
	if len(keys) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		fmt.Sprintf("settings the drive holds across its lanes (%s) changed since it started (%s), so the remote that would vouch for the lane's hooks, or a program the lane's git would start, is not the drive's", remoteIdentityHeld, strings.Join(keys, ", ")),
		"look at what changed them (`git config --show-origin --show-scope --get-regexp '"+heldPattern+"'`, and the files named), in each environment the drive's git runs in: as yourself for the push's and the ask of origin's, and with HOME set to the stage's home, as the drive sets it, for the lane's add and commit; the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

// checkoutHooksUnchanged refuses the lane's worktree stage when a hooks
// directory the drive holds, or info/attributes, as read now, differs from
// what the drive read as it started (Context.remote): an entry at any depth
// added, removed or changed, named, or a tree grown past maxHookEntries
// entries. git's add runs in the checkout, so it runs the post-checkout hook
// of the hooks directory git names there (hooksDir, asked of git by the stage
// in the add's own environment: hooksDirAt) with the drive's rights, and the
// landing push runs the pre-push hook of the one git names in the person's
// own environment (readHooks); each lane of a drain is a run of its own,
// whose fingerprint takes the checkout's copy as it stands at its first
// check, so without this a copy an earlier lane changed would be the new
// run's starting point. A directory other than the drive's reads as its
// entries removed and others added. The stage calls it just before the add,
// so a difference present at this read is refused before that add; a change
// made after this read and before git runs the hook is not seen, and nothing
// holds the directory through the add (the file's header). A read that gives
// no answer the hold can take stops the step at the config stage. With
// nothing carried (a step the host runs, a single build) it checks nothing.
func checkoutHooksUnchanged(c Context, laneID, hooksDir string) error {
	if c.remote == nil {
		return nil
	}
	now, _, err := readHooks(c.RepoRoot, c.remote.salt, hooksDir)
	if errors.Is(err, errHooksPastCap) {
		return refuse(string(StageWorktree), "", laneID,
			err.Error()+", and git's add of the lane's worktree runs the checkout's post-checkout hook with the drive's rights, so the lane is not made",
			"look at what grew the checkout's hooks directory; stop the drive and start it again once the hooks are as they should be; nothing was made")
	}
	if err != nil {
		return heldReadRefusal(c, laneID, err)
	}
	changed := configFingerprint{Files: c.remote.hooks}.changes(c.RepoRoot, configFingerprint{Files: now})
	if len(changed) == 0 {
		return nil
	}
	return refuse(string(StageWorktree), "", laneID,
		"the checkout's hooks or info/attributes changed since the drive started ("+strings.Join(nameChanges(c.RepoRoot, changed), ", ")+"), and git's add of the lane's worktree and the landing push run the hooks with the drive's rights, so the lane is not made",
		"look at what changed the hooks directory or info/attributes; the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

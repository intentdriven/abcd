package loop

// remoteid.go is what a drive holds across its lanes (iss-2611071820538335,
// iss-2611131308266006, rulings recorded in their decisions). The drive's git
// runs in each environment driveGitCommands names (gitenvs.go): the person's
// own, where the landing push and the reservation's pushes and fetches run
// with the repository's hooks on and the ask of origin with them off; the
// lane's add, commit, merge and reset, which in a drive have the stage's home
// and the stage's global settings file; and abcd's isolated reads. So the
// drive reads, in each of them, the settings it holds (remoteIdentityKey: those
// the run's fingerprint watches, what names a program git starts or a file
// that adds more settings, origin's identity among them, less core.pager, and
// beside them driveHeldConfig), at every scope that environment reads, and,
// where git reads a worktree's own settings (extensions.worktreeConfig, as
// git decides it: worktreeScopeRead), the keys of driveHeldConfig at each
// lane worktree's own scope (laneOwnScope), a lane the drive makes read as
// the drive makes it (laneMade); the bytes of each
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
// Not held either: a setting no git of the drive reads (driveUnreadSettings
// lists them, with why); a submodule's own settings, which no fetch abcd
// makes reaches (its submodule recursion is pinned off), nor a git command of
// the drive that compares a working tree with the index: a status
// (gitutil.StatusArgs) or a diff of a working tree (gitutil.DiffWorktreeArgs)
// passes --ignore-submodules=dirty, the staging (addAllGit, addAllGitIndex)
// leaves out by pathspec each submodule whose checked-out commit has not
// moved, and a sync's merge names submodule.<name>.ignore=dirty for each
// name (submoduleIgnorePins); a commit of the drive always has something to
// commit or is made with --allow-empty, and git refuses to remove a worktree
// holding a checked-out submodule before it reads its status. What remains:
// a submodule moved, or a .gitmodules changed, between those reads and git's
// own command is not seen; and the commands a person's held submodule.recurse
// or push.recurseSubmodules makes recurse (the drive's push and reset) read
// each submodule's own settings. And an origin defined only by a legacy
// remotes or branches file in the git directory is not held, having no key
// to compare.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteIdentityHeld names, for a refusal and the docs, what settings a drive
// holds across its lanes (remoteIdentityKey), as each environment its git
// runs in reads them.
const remoteIdentityHeld = "at every scope, as each environment the drive's git runs in reads them: where a networked git goes and how it reaches the remote (origin's and every remote's url, pushurl, uploadpack, receivepack, proxy, proxy authentication method, server options and vcs; every url rewrite; core.sshCommand, core.gitProxy and ssh.variant; the protocol allow rules; fetch.bundleURI), whom it trusts and what it sends (every credential.*; core.askPass; the http.* settings that decide where an https fetch or push connects, whom it takes as the remote and what it sends there: the proxy and its authentication, certificate and key, the resolve and redirect settings, the TLS version, ciphers, backend, certificate authorities and pinned key, the client certificate and key, cookies, extra headers, credential delegation and the authentication switches, per URL too); " +
	"the programs a command of the drive starts (core.hooksPath, core.fsmonitor, core.editor, core.alternateRefsCommand, uploadpack.packObjectsHook and gc.recentObjectsHook; content filters, diff and merge drivers and diff.external; every gpg.* and the signing and verifying switches; format.pretty and pretty.*, whose format can check a signature; the Git LFS transfer and extension programs; hook.*; init.templateDir); " +
	"the settings that change which of them run (merge.renormalize, submodule.recurse and push.recurseSubmodules, status.submoduleSummary, core.attributesFile, core.worktree and extensions.worktreeConfig); include and includeIf paths and the files they name; and the global and system attributes files"

// driveHeldConfig matches, as watchedConfig does, the keys a drive holds
// across its lanes (remoteIdentityKey) beyond those the run's fingerprint
// watches: a setting that names or starts a program, or changes where a
// networked git goes or whom it trusts, that a git command the drive runs, or
// an isolated abcd command inside a drive, can read. Each, with the command
// that reads it:
//   - the http.* settings that decide whom an https fetch or push of the
//     drive (gitutil.Net) trusts, where it connects, or what it sends there,
//     plain or per URL (git applies a per-URL value over the plain one):
//     trust (sslVerify, sslVersion, sslCipherList, sslCAInfo, sslCAPath,
//     pinnedPubkey, sslBackend, schannelCheckRevoke, schannelUseSSLCAInfo,
//     sslTry, proxySSLCAInfo), destination (proxy, curloptResolve,
//     followRedirects), and what is sent (sslCert, sslCertType, sslKey,
//     sslKeyType, sslCertPasswordProtected, proxySSLCert, proxySSLKey,
//     proxySSLCertPasswordProtected, proxyAuthMethod, cookieFile,
//     saveCookies, extraHeader, delegation, emptyAuth, proactiveAuth). The
//     other http.* keys git 2.52 documents are neither a program nor trust:
//     postBuffer and maxRequests (buffers), minSessions, keepAliveCount,
//     keepAliveIdle, keepAliveInterval, lowSpeedLimit and lowSpeedTime
//     (speed), noEPSV and version (how a connection is made, not to whom),
//     and userAgent (a name git gives itself); none is unread.
//     TestEveryHTTPSettingGitDocumentsIsInOneRow holds that list.
//   - remote.<name>.proxyAuthMethod, the per-remote form of
//     http.proxyAuthMethod (what is sent to the proxy), and
//     remote.<name>.serverOption, what a fetch from that remote sends the
//     server; remote.<name>.proxy and the remote's other program and
//     destination keys are watched by the run's fingerprint.
//   - fetch.bundleURI: where a fetch first downloads from.
//   - core.alternateRefsCommand (the connectivity check of every fetch),
//     core.attributesFile (which filters and drivers apply), diff.external,
//     merge.renormalize and merge.verifySignatures (the lane's merge),
//     format.pretty and pretty.<name> (the format of a log or show, which can
//     check a signature), every gpg.* and the signing and verifying switches
//     (the lane's commit and merge, a log or show that checks a signature),
//     ssh.variant (the ssh transport), uploadpack.packObjectsHook (the
//     upload-pack a fetch from a local path runs), gc.recentObjectsHook (the
//     automatic gc a lane's commit can start), init.templateDir, and
//     status.submoduleSummary (a commit with nothing to commit prints a long
//     status).
//   - submodule.recurse and push.recurseSubmodules: the drive's push and a
//     lane's reset read them; every fetch abcd makes through its networked
//     runners, in a drive or not, pins both fetch.recurseSubmodules and
//     submodule.recurse off on its command line (gitutil.Net), and the git
//     commands of the drive that compare a working tree with the index leave
//     submodules' working trees out (the file's header says how), so the
//     per-submodule settings are not held (driveUnreadSettings says which git
//     reads them).
//   - Git LFS's custom transfer and extension programs and its standalone
//     transfer agent, plain or per URL: git-lfs, which the lfs content filter
//     the lane's worktree add and checkout run starts, reads them.
//   - hook.*: no setting of git 2.52, which the installed git is; a later git
//     reads its configured hooks there.
//
// The fingerprint of a run outside a drive keeps to watchedConfig, so nothing
// outside a drive refuses more. What is not held, and why, is
// driveUnreadSettings beside this.
const driveHeldConfig = `^(` +
	`http\.(.+\.)?(proxy|proxyauthmethod|proxysslcainfo|proxysslcert|proxysslkey|proxysslcertpasswordprotected|sslverify|sslversion|sslcipherlist|sslcainfo|sslcapath|pinnedpubkey|sslbackend|schannelcheckrevoke|schannelusesslcainfo|ssltry|sslcert|sslcerttype|sslkey|sslkeytype|sslcertpasswordprotected|curloptresolve|followredirects|cookiefile|savecookies|extraheader|delegation|emptyauth|proactiveauth)` +
	`|remote\..+\.(proxyauthmethod|serveroption)` +
	`|fetch\.bundleuri` +
	`|core\.(alternaterefscommand|attributesfile)` +
	`|diff\.external` +
	`|merge\.(renormalize|verifysignatures)` +
	`|format\.pretty` +
	`|pretty\..+` +
	`|gpg\..+` +
	`|(commit|tag|push)\.gpgsign` +
	`|tag\.forcesignannotated` +
	`|log\.showsignature` +
	`|ssh\.variant` +
	`|uploadpack\.packobjectshook` +
	`|submodule\.recurse` +
	`|push\.recursesubmodules` +
	`|status\.submodulesummary` +
	`|lfs\.(customtransfer\..+|extension\..+|(.+\.)?standalonetransferagent)` +
	`|hook\..+` +
	`|gc\.recentobjectshook` +
	`|init\.templatedir` +
	`)$`

// driveUnreadSettings are the settings that name or start a program, or
// change where a networked git goes, that a drive does not hold, each with the
// reason no git command the drive runs, or isolated abcd command inside a
// drive, reads it; and settings whose names end as a program's do
// (color.pager, advice.*) that name none. TestEachProgramSettingGitDocumentsIsHeldOrUnread
// reads git's own list (`git help --config`) against this and the held
// pattern. Left out as naming no program and changing neither where a
// networked git goes nor whom it trusts: core.excludesFile, fetch.fsckObjects
// and the other fsck and transfer checks, and the http.* keys of speed,
// buffers, connection mode and user agent (driveHeldConfig lists them).
var driveUnreadSettings = []struct{ pattern, why string }{
	{`alias\..+`, "git-config(1), alias.*: \"To avoid confusion and troubles with script usage, aliases that hide existing Git commands are ignored except for deprecated commands.\" abcd runs no deprecated command (TestNoAliasStandsForASubcommandAbcdJudges in internal/gitutil)"},
	{`core\.pager|pager\..+|color\.pager`, "git starts a pager only when its output is a terminal, and a drive gives its git none; color.pager is a switch; the run's own fingerprint still watches core.pager, so a change to it during a lane's run stops that run's next git step"},
	{`sequence\.editor|rebase\.instructionformat|interactive\.difffilter`, "read by an interactive rebase and add -p, which a drive does not run"},
	{`(diff|merge)\.(tool|guitool)|(difftool|mergetool|guitool)\..+`, "read by difftool, mergetool and git gui, which a drive does not run"},
	{`web\.browser|browser\..+|help\.browser|man\.viewer|man\..+|instaweb\..+`, "read by help, web--browse and instaweb, which a drive does not run"},
	{`sendemail\..+|imap\..+`, "read by send-email and imap-send, which a drive does not run"},
	{`trailer\..+`, "read by interpret-trailers and by a commit or tag given --trailer, none of which a drive runs"},
	{`maintenance\..+|gc\.(auto|autodetach)`, "automatic maintenance starts no program beyond the hooks directory's pre-auto-gc and gc.recentObjectsHook, which are held"},
	{`fetch\.recursesubmodules|submodule\.(.+\..+|active|fetchjobs|alternatelocation|alternateerrorstrategy|propagatebranches)`, "every fetch abcd makes through its networked runners, in a drive or not, pins fetch.recurseSubmodules and submodule.recurse off on its command line; the git commands of the drive that compare a working tree with the index start nothing in a submodule: a status or a diff of a working tree passes --ignore-submodules=dirty, which outranks submodule.<name>.ignore (and diff.ignoreSubmodules), the staging leaves out by pathspec each submodule whose checked-out commit has not moved, and a sync's merge names submodule.<name>.ignore=dirty on its command line for each name the lane's .gitmodules declares; a drive runs no git submodule; only where submodule.recurse or push.recurseSubmodules, which are held, is on does a git of the drive read the others: its push and its lane's reset go into each submodule git counts active (submodule.<name>.active), under that submodule's own settings"},
	{`transfer\.bundleuri`, "read by clone alone, which a drive does not run"},
	{`advice\..+`, "a switch for a hint, naming no program"},
}

// driveHeldKeyRE is driveHeldConfig as a Go expression.
var driveHeldKeyRE = regexp.MustCompile(driveHeldConfig)

// heldWatched is watchedConfig less core.pager, which no git of a drive
// reads (driveUnreadSettings); the run's fingerprint still watches it.
var heldWatched = strings.Replace(watchedConfig, "|pager|", "|", 1)

// heldPattern is the extended regular expression a refusal's remedy names
// for remoteIdentityHeld's keys, and the one the hold applies
// (remoteIdentityKey): heldWatched and driveHeldConfig as one.
var heldPattern = strings.TrimSuffix(heldWatched, ")$") + "|" + strings.TrimPrefix(driveHeldConfig, "^(")

// heldKeyRE is heldPattern as a Go expression.
var heldKeyRE = regexp.MustCompile(heldPattern)

// remoteIdentity is what a drive read as it started: in each environment
// its git runs in (driveEnvs), each held key's hash (envs); each file an
// include setting names in any of them and the global and system attributes
// files each reads, once by path (files); each entry of every distinct hooks
// directory git names in an environment whose commands run hooks, and of
// info/attributes, hashed as the run's fingerprint hashes them (hooks); in
// each lane worktree, by its path, the keys of driveHeldConfig it sets at its
// own scope (lanes: laneScopeSettings), a lane the drive makes added as it
// makes it (laneMade), and the same keys at the checkout's own worktree
// scope (checkoutOwn), which `git worktree add` copies into the worktree it
// makes; the directory the lane's add and commit run (hooksDir) and every
// directory held (hooksDirs); and the salt the files are hashed under and key
// names are masked under when a refusal names them (hiddenKey). mu guards
// lanes, which laneMade adds to while a drive's comparisons read it.
//
// A reading made to compare (heldReading's) sets lanesRead to the lanes it
// read: every lane git lists, or the one a git about to run runs in.
type remoteIdentity struct {
	envs        map[*gitEnv]map[string]string
	mu          sync.Mutex
	lanes       map[string]map[string]string
	checkoutOwn map[string]string
	lanesRead   lanesRead
	files       map[string]string
	hooks       map[string]string
	hooksDir    string
	hooksDirs   []string
	salt        string
}

// lanesRead is which lane worktrees a reading reads at their own scope: every
// one git lists for the repository (all), or only lane, the one a git about
// to run runs in ("" for none: a git that runs in no lane worktree).
type lanesRead struct {
	all  bool
	lane string
}

// everyLane is the reading of every lane worktree's own scope: one listing
// of the worktrees, then one listing of each lane's settings.
var everyLane = lanesRead{all: true}

// maxSettingsBytes caps one listing of the settings.
const maxSettingsBytes = 1 << 20

// remoteIdentityKey says whether a canonical config key is one a drive holds
// across its lanes: every key the run's fingerprint watches but core.pager,
// and every key of driveHeldConfig (heldPattern, which a refusal's remedy hands
// to git).
func remoteIdentityKey(key string) bool { return heldKeyRE.MatchString(key) }

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
	r, err := readHeld(repoRoot, salt, "", boundsNow(driveEnvs()...), everyLane)
	if err != nil {
		return nil, err
	}
	id.envs, id.lanes, id.checkoutOwn, id.files, id.hooks, id.hooksDirs = r.envs, r.lanes, r.checkoutOwn, r.files, r.hooks, r.dirs
	id.hooksDir = r.dirs[0]
	return id, nil
}

// heldReading is one reading of everything a drive holds (readHeld).
type heldReading struct {
	settingsReading
	hooks map[string]string
	dirs  []string
}

// settingsReading is one reading of the settings a drive holds
// (readSettings): each environment's keys (envs), each lane worktree's own
// scope as which says (lanes), the checkout's own worktree scope where which
// reads every lane (checkoutOwn), and the files they name (files).
type settingsReading struct {
	envs        map[*gitEnv]map[string]string
	lanes       map[string]map[string]string
	checkoutOwn map[string]string
	files       map[string]string
}

// readHeld reads everything a drive holds under salt, its settings
// (readSettings, the lane worktrees which names) and its hooks directories
// (readHooks, stageDir as there) side by side, under the reading's bounds b,
// and returns once both have ended. A failed read of the settings is named
// before one of the hooks.
func readHeld(repoRoot, salt, stageDir string, b readBounds, which lanesRead) (heldReading, error) {
	var r heldReading
	var serr, herr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		r.settingsReading, serr = readSettings(repoRoot, salt, b, which)
	}()
	go func() {
		defer wg.Done()
		r.hooks, r.dirs, herr = readHooks(repoRoot, salt, stageDir, b)
	}()
	wg.Wait()
	if serr != nil {
		return heldReading{}, serr
	}
	if herr != nil {
		return heldReading{}, herr
	}
	return r, nil
}

// readSettings reads the held settings in every environment a drive's git
// runs in (driveEnvs) under salt, the environments side by side, under the
// reading's bounds b: each environment's keys, the files every environment
// names, once by path, and, where git reads a worktree's own settings
// (worktreeScopeRead, asked beside them), the own scope of the lane worktrees
// which names (laneScopeSettings, or laneOwnScope for one) and, with every
// lane, the checkout's own (checkoutOwn). A failed read is named in the
// table's order of the environments, then the ask of the extension. It
// returns once every read it started has ended; one whose comparison's
// deadline passed meanwhile (a hashing of a file that outlasted compareGrace
// among them) is errCompareUnfinished, its answers its own, never written
// anywhere else.
func readSettings(repoRoot, salt string, b readBounds, which lanesRead) (settingsReading, error) {
	envs := driveEnvs()
	keys := make([]map[string]string, len(envs))
	files := make([]map[string]string, len(envs))
	errs := make([]error, len(envs))
	var ownScope bool
	var oerr error
	var wg sync.WaitGroup
	for i, e := range envs {
		files[i] = map[string]string{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			keys[i], errs[i] = readHeldSettingsBy(e, repoRoot, salt, files[i], b)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ownScope, oerr = worktreeScopeRead(repoRoot, b)
	}()
	wg.Wait()
	if b.cut() {
		return settingsReading{}, errCompareUnfinished
	}
	r := settingsReading{envs: map[*gitEnv]map[string]string{}, lanes: map[string]map[string]string{}, files: map[string]string{}}
	for i, e := range envs {
		if errs[i] != nil {
			return settingsReading{}, errs[i]
		}
		r.envs[e] = keys[i]
		maps.Copy(r.files, files[i])
	}
	if oerr != nil {
		return settingsReading{}, oerr
	}
	if !ownScope {
		return r, nil
	}
	var err error
	switch {
	case which.all:
		r.lanes, r.checkoutOwn, err = laneScopeSettings(repoRoot, b)
	case which.lane != "":
		var own map[string]string
		var gone bool
		if own, gone, err = laneOwnScope(which.lane, b); err == nil && !gone {
			r.lanes[which.lane] = own
		}
	}
	if err != nil {
		return settingsReading{}, err
	}
	return r, nil
}

// worktreeScopeRead says whether git reads each worktree's own settings
// (config.worktree): extensions.worktreeConfig as git's own check of the
// repository's format reads it, from the repository's own settings file
// alone, its last value, as a boolean. That check follows no include and
// reads no other scope, so a value set in an included file, in a
// config.worktree, in global settings or on the command line changes nothing,
// and nothing of the environment a git runs in does either: one ask answers
// for every environment a drive's git runs in
// (TestTheHoldAsksGitWhetherItReadsAWorktreesOwnSettings). It is asked of git
// (`git config --local --no-includes --type=bool --get`, isolated, under the
// reading's bounds b, which mark it the comparison's own: gitutil.WithinCheck);
// unset is false. A value git refuses is a readFailure, as git then refuses
// every command in the repository, and so is any other answer it does not
// take.
func worktreeScopeRead(repoRoot string, b readBounds) (bool, error) {
	what := "asking git whether it reads a worktree's own settings (git config --local --no-includes --type=bool --get extensions.worktreeConfig)"
	a, err := isolatedEnv.readBy(b, repoRoot, what, 4096, "config", "--local", "--no-includes", "--type=bool", "--get", "extensions.worktreeConfig")
	if err != nil {
		return false, err
	}
	switch {
	case a.code == 1:
		return false, nil
	case a.code != 0:
		return false, isolatedEnv.exited(what, a)
	}
	switch strings.TrimSuffix(a.out, "\n") {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, isolatedEnv.failed(what, "git did not answer true or false, so the answer is not taken")
}

// maxWorktreeListingBytes caps the hold's listing of the repository's
// worktrees.
const maxWorktreeListingBytes = 1 << 20

// laneScopeSettings reads, under the reading's bounds b, the keys of
// driveHeldConfig each lane worktree of the repository sets at its own scope
// (config.worktree, which git reads where extensions.worktreeConfig is on),
// each worktree by its path (laneOwnScope), and the same keys at the
// checkout's own worktree scope (checkoutOwn), which `git worktree add`
// copies into the worktree it makes. A lane worktree is a linked worktree
// named as the loop names a lane's (isLaneWorktree), other than the checkout:
// the lane's add, commit, merge and reset, and abcd's reads there, run git in
// it, and read that scope, which the checkout's own reading does not. The
// run's fingerprint compares the keys it watches there (takeFingerprint). A
// worktree whose directory is gone, or that git marks prunable, holds
// nothing; any other read that gives no answer the hold can take is a
// readFailure naming it. It lists the worktrees once and each lane's settings
// once, side by side.
func laneScopeSettings(repoRoot string, b readBounds) (lanes map[string]map[string]string, checkoutOwn map[string]string, err error) {
	what := "listing the repository's worktrees (git worktree list --porcelain -z)"
	a, err := isolatedEnv.readBy(b, repoRoot, what, maxWorktreeListingBytes, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, nil, err
	}
	if a.code != 0 {
		return nil, nil, isolatedEnv.exited(what, a)
	}
	paths := []string{repoRoot}
	for _, wt := range gitutil.ParseWorktreeList(a.out, "\x00") {
		if wt.Bare || wt.Prunable || !isLaneWorktree(wt.Path) || samePath(repoRoot, wt.Path, repoRoot) || !fsutil.IsRealDir(wt.Path) {
			continue
		}
		paths = append(paths, lanePath(wt.Path))
	}
	keys := make([]map[string]string, len(paths))
	gone := make([]bool, len(paths))
	errs := make([]error, len(paths))
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()
			keys[i], gone[i], errs[i] = laneOwnScope(p, b)
		}()
	}
	wg.Wait()
	if b.cut() {
		return nil, nil, errCompareUnfinished
	}
	lanes = map[string]map[string]string{}
	for i, p := range paths {
		if errs[i] != nil {
			return nil, nil, errs[i]
		}
		switch {
		case i == 0:
			checkoutOwn = keys[i]
		case !gone[i]:
			lanes[p] = keys[i]
		}
	}
	return lanes, checkoutOwn, nil
}

// laneOwnScope is one worktree's own scope, read under the reading's bounds
// b: each key of driveHeldConfig git lists there at the worktree scope, by
// its name, and the hash of its values in order. gone says the worktree's
// directory went away, and then it holds nothing. The listing is made once,
// in abcd's isolated environment: the worktree scope is the worktree's
// config.worktree, the same file in every environment. Where that scope
// holds an include (include.path, includeIf.*.path), whose path and condition
// can resolve otherwise in another environment (a path from the home, a
// condition on another scope's settings), it is listed in every environment
// a drive's git runs in instead, and a key every environment gives the same
// values hashes as one listing's.
func laneOwnScope(lane string, b readBounds) (keys map[string]string, gone bool, err error) {
	vals, gone, err := ownScopeValues(isolatedEnv, lane, b)
	if err != nil || gone {
		return nil, gone, err
	}
	if _, has := vals[includeMark]; !has {
		return hashOwnScope(vals, nil), false, nil
	}
	envs := driveEnvs()
	per := make([]map[string][]string, len(envs))
	gones := make([]bool, len(envs))
	errs := make([]error, len(envs))
	var wg sync.WaitGroup
	for i, e := range envs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			per[i], gones[i], errs[i] = ownScopeValues(e, lane, b)
		}()
	}
	wg.Wait()
	if b.cut() {
		return nil, false, errCompareUnfinished
	}
	for i := range envs {
		if errs[i] != nil {
			return nil, false, errs[i]
		}
		if gones[i] {
			return nil, true, nil
		}
	}
	return hashOwnScope(nil, per), false, nil
}

// includeMark is the entry ownScopeValues gives for an include setting at the
// worktree scope; it is never a key.
const includeMark = "\x00include"

// ownScopeValues is one listing of the worktree scope at lane in env, under
// the reading's bounds b: each key of driveHeldConfig, its values in order
// (a key with no value apart from one with an empty value), and includeMark
// where an include setting stands at that scope. A worktree whose directory
// went away after the listing holds nothing (gone).
func ownScopeValues(e *gitEnv, lane string, b readBounds) (map[string][]string, bool, error) {
	what := "listing a lane worktree's own settings (git config --list --show-scope --show-origin)"
	a, err := e.readBy(b, lane, what, maxSettingsBytes, "config", "--null", "--list", "--show-scope", "--show-origin")
	if err != nil {
		return nil, false, err
	}
	if a.code != 0 {
		if _, serr := os.Lstat(lane); errors.Is(serr, os.ErrNotExist) {
			return nil, true, nil
		}
		return nil, false, e.exited(what, a)
	}
	recs, err := parseScopedList(a.out)
	if err != nil {
		return nil, false, e.failed(what, err.Error())
	}
	values := map[string][]string{}
	for _, r := range recs {
		if r.scope != "worktree" {
			continue
		}
		if r.key == "include.path" || (strings.HasPrefix(r.key, "includeif.") && strings.HasSuffix(r.key, ".path")) {
			values[includeMark] = nil
		}
		if !driveHeldKeyRE.MatchString(r.key) {
			continue
		}
		v := "none"
		if !r.noValue {
			v = "=" + r.value
		}
		values[r.key] = append(values[r.key], v)
	}
	return values, false, nil
}

// hashOwnScope hashes a worktree scope's values by key: one listing's (one),
// or, where per holds every environment's listing in driveEnvs order, a key
// every environment gives the same values hashed as one listing's would be,
// and any other by each environment's values in turn.
func hashOwnScope(one map[string][]string, per []map[string][]string) map[string]string {
	hash := func(parts ...string) string {
		h := sha256.New()
		for _, p := range parts {
			writePart(h, p)
		}
		return hex.EncodeToString(h.Sum(nil))
	}
	keys := map[string]string{}
	if per == nil {
		for k, vs := range one {
			if k != includeMark {
				keys[k] = hash(vs...)
			}
		}
		return keys
	}
	all := map[string]bool{}
	for _, m := range per {
		for k := range m {
			if k != includeMark {
				all[k] = true
			}
		}
	}
	for k := range all {
		same := true
		for _, m := range per[1:] {
			same = same && slices.Equal(m[k], per[0][k])
		}
		if same {
			keys[k] = hash(per[0][k]...)
			continue
		}
		var parts []string
		for i, m := range per {
			parts = append(parts, strconv.Itoa(i), strconv.Itoa(len(m[k])))
			parts = append(parts, m[k]...)
		}
		keys[k] = hash(parts...)
	}
	return keys
}

// errHooksPastCap is readCheckoutHooks's refusal of a hooks directory past
// maxHookEntries entries.
var errHooksPastCap = fmt.Errorf("the checkout's hooks directory holds more than %d entries, so the drive cannot hold it whole", maxHookEntries)

// readHooks is every hooks directory a drive's git runs at the checkout:
// stageDir, the one the lane's add and commit run (hooksDirAt; read here
// when ""), and the one git names in each other environment whose commands
// run hooks (the landing push's and the reservation's, in the person's own),
// each distinct directory once, hashed with info/attributes
// (readCheckoutHooks). The reads of the directories and of the common git
// directory run side by side, under the reading's bounds b, and are waited
// for. It returns the record and the directories, stageDir first. Past the
// comparison's deadline it walks nothing, and a walk that outlasted it
// (compareGrace) is errCompareUnfinished, its record its own.
func readHooks(repoRoot, salt, stageDir string, b readBounds) (map[string]string, []string, error) {
	var envs []*gitEnv
	if stageDir == "" {
		envs = append(envs, stageEnv)
	}
	for _, e := range driveEnvs() {
		if e != stageEnv && e.hooks() {
			envs = append(envs, e)
		}
	}
	found := make([]string, len(envs))
	errs := make([]error, len(envs))
	var common string
	var cerr error
	var wg sync.WaitGroup
	for i, e := range envs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			found[i], errs[i] = hooksDirBy(e, repoRoot, b)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		common, cerr = commonDirBy(repoRoot, b)
	}()
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, nil, err
		}
	}
	if cerr != nil {
		return nil, nil, cerr
	}
	var dirs []string
	if stageDir != "" {
		dirs = append(dirs, stageDir)
	}
	for _, d := range found {
		if !slices.ContainsFunc(dirs, func(h string) bool { return samePath(repoRoot, h, d) }) {
			dirs = append(dirs, d)
		}
	}
	if b.cut() {
		return nil, nil, errCompareUnfinished
	}
	hooks, err := hashCheckoutHooks(salt, common, dirs...)
	if b.cut() {
		return nil, nil, errCompareUnfinished
	}
	return hooks, dirs, err
}

// commonDirBy is the common git directory at repoRoot, asked of git
// (`git rev-parse --path-format=absolute --git-common-dir`, isolated) under
// the reading's bounds b, bounded as every read the hold makes is
// (gitEnv.readBy).
func commonDirBy(repoRoot string, b readBounds) (string, error) {
	what := "resolving the common git directory (git rev-parse --git-common-dir)"
	a, err := isolatedEnv.readBy(b, repoRoot, what, 1<<16, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if a.code != 0 {
		return "", isolatedEnv.exited(what, a)
	}
	p, ok := onePath(a.out)
	if !ok || !filepath.IsAbs(p) {
		return "", isolatedEnv.failed(what, "git did not answer with one absolute path (--path-format needs git 2.31 or later), so the answer is not taken")
	}
	return p, nil
}

// readCheckoutHooks hashes, under salt, each entry at every depth of each
// of dirs (directories git names at the checkout: readHooks), through the
// fingerprint's own walk (hashDir), and the common git directory's
// info/attributes. It fails when the common git directory cannot be resolved
// or a tree is past maxHookEntries entries.
func readCheckoutHooks(repoRoot, salt string, dirs ...string) (map[string]string, error) {
	common, err := commonDirBy(repoRoot, boundsNow(isolatedEnv))
	if err != nil {
		return nil, err
	}
	return hashCheckoutHooks(salt, common, dirs...)
}

// hashCheckoutHooks is readCheckoutHooks with the common git directory
// already read.
func hashCheckoutHooks(salt, common string, dirs ...string) (map[string]string, error) {
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
// includeIf.*.path names, whatever its condition (includePath), and the
// global and system attributes files git reads in env (attributesFiles).
// The listing needs git 2.26 or later, as the run's fingerprint does. A read
// that gives no answer the hold can take is a readFailure naming it, which
// stops the drive or the step.
func readHeldSettings(env *gitEnv, repoRoot, salt string, files map[string]string) (map[string]string, error) {
	return readHeldSettingsBy(env, repoRoot, salt, files, boundsNow(env))
}

// readHeldSettingsBy is readHeldSettings under the reading's bounds b: the
// listing and the two asks for the attributes files run side by side, then
// the include paths, side by side, each waited for.
func readHeldSettingsBy(env *gitEnv, repoRoot, salt string, files map[string]string, b readBounds) (map[string]string, error) {
	asked := make(chan [2]attributeAnswer, 1)
	go func() { asked <- askAttributes(env, repoRoot, b) }()
	what := "listing " + Remote + "'s and the other settings a drive holds (git config --list --show-scope --show-origin)"
	a, err := env.readBy(b, repoRoot, what, maxSettingsBytes, "config", "--null", "--list", "--show-scope", "--show-origin")
	attrs := <-asked
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
	var includes []scopedSetting
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
			includes = append(includes, r)
		}
	}
	paths := make([]string, len(includes))
	errs := make([]error, len(includes))
	var wg sync.WaitGroup
	for i, r := range includes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			paths[i], errs[i] = includePath(env, repoRoot, salt, r, b)
		}()
	}
	wg.Wait()
	for i := range includes {
		if errs[i] != nil {
			return nil, errs[i]
		}
		if paths[i] != "" {
			hashFile(files, salt, paths[i])
		}
	}
	if _, set := lastEnv(b.environ(env), "GIT_CONFIG_SYSTEM"); set {
		system = ""
	}
	attrFiles, err := attributesFrom(env, repoRoot, system, attrs, b)
	if err != nil {
		return nil, err
	}
	for _, p := range attrFiles {
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

// includePath is the file an include setting r names, as git resolves its
// path in env under the reading's bounds b, for the
// caller to hold by its bytes (hashFile; a file not there records nothing,
// so its appearing is a change). A setting with no value names no file: "".
// A path git cannot resolve, a resolution that does not finish, or a
// relative path whose setting does not come from a file is a readFailure
// that names the setting by its key and scope.
func includePath(env *gitEnv, repoRoot, salt string, r scopedSetting, b readBounds) (string, error) {
	if r.noValue {
		return "", nil
	}
	what := "resolving the path of the include setting " + hiddenKey(salt, r.key) + " (" + r.scope + " scope) as git does (git config --type=path)"
	// What git says of a path it cannot resolve repeats the path, so only
	// its status is named, and a read that failed otherwise names the path
	// as the path.
	// The value is the option's own word (--default=<value>), so a value that
	// starts with a dash is never read as an option, by git or by abcd.
	a, err := env.readBy(b, repoRoot, what, 4096, "config", "--file", os.DevNull, "--type=path", "--default="+r.value, "--get", "abcd.include")
	if err != nil {
		var rf *readFailure
		if errors.As(err, &rf) && r.value != "" {
			for _, v := range []string{r.value, fsutil.RedactHome(r.value)} {
				rf.why = strings.ReplaceAll(rf.why, v, "<the path>")
			}
		}
		return "", err
	}
	if a.code != 0 {
		return "", env.failed(what, "git could not resolve it (exit "+strconv.Itoa(a.code)+")")
	}
	p, ok := onePath(a.out)
	if !ok {
		return "", env.failed(what, "git did not answer with one path, so the answer is not taken")
	}
	if !filepath.IsAbs(p) {
		from, ok := strings.CutPrefix(r.origin, "file:")
		if !ok || from == "" {
			return "", env.failed(what, "the path is relative and the setting does not come from a file, so it has no directory to stand under")
		}
		if !filepath.IsAbs(from) {
			from = filepath.Join(repoRoot, from)
		}
		p = filepath.Join(filepath.Dir(from), p)
	}
	return filepath.Clean(p), nil
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
		if r, ok := heldStopOf(err); ok {
			return r
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
// than it did (its URL subsection hidden, hiddenKey), each key of
// driveHeldConfig a lane worktree now (now.lanesRead) sets at its own scope
// otherwise than it did, newly, or no longer (laneChanged), and each held
// file that was added, removed or changed, through the refusal redactor.
// Each environment's keys are compared with its own earlier reading, never
// with another's. Never a value.
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
	for _, name := range was.laneChanged(repoRoot, now) {
		add(name)
	}
	for _, name := range nameChanges(repoRoot, configFingerprint{Files: was.files}.changes(repoRoot, configFingerprint{Files: now.files})) {
		add(name)
	}
	slices.Sort(out)
	return out
}

// laneChanged names, sorted by nothing (changed sorts), each key of
// driveHeldConfig whose values at a lane worktree's own scope differ between
// was and now, among the lanes now read (now.lanesRead), as "<key> in the
// worktree <path>" (the key's URL subsection masked: hiddenKey). A lane was
// holds is compared both ways: a key set, changed, emptied or gone is named;
// one whose worktree git no longer lists is compared as setting nothing, and
// only one whose directory is gone is skipped, since no git runs there any
// more. A lane was does not hold (one made since the drive started by
// another than the drive, or by the drive before laneMade read it) is
// compared with the checkout's own worktree scope as held at the start
// (checkoutOwn), which `git worktree add` copies into the worktree it makes.
func (was *remoteIdentity) laneChanged(repoRoot string, now *remoteIdentity) []string {
	was.mu.Lock()
	held := maps.Clone(was.lanes)
	was.mu.Unlock()
	var out []string
	diff := func(path string, a, b map[string]string) {
		in := " in the worktree " + fsutil.RedactHome(path)
		for k, h := range a {
			if b[k] != h {
				out = append(out, hiddenKey(was.salt, k)+in)
			}
		}
		for k := range b {
			if _, ok := a[k]; !ok {
				out = append(out, hiddenKey(was.salt, k)+in)
			}
		}
	}
	read := func(path string) bool {
		return now.lanesRead.all || now.lanesRead.lane != "" && path == now.lanesRead.lane
	}
	for path, a := range held {
		if !read(path) {
			continue
		}
		b, listed := now.lanes[path]
		if !listed && !fsutil.IsRealDir(path) {
			continue
		}
		diff(path, a, b)
	}
	for path, b := range now.lanes {
		if _, ok := held[path]; !ok {
			diff(path, was.checkoutOwn, b)
		}
	}
	return out
}

// lanePath is a lane worktree's path as the hold keys it: cleaned and, where
// it exists, its real path, so the spelling the loop derives and the one git
// lists name one lane.
func lanePath(p string) string { return fsutil.RealExistingPath(filepath.Clean(p)) }

// laneOf is the lane worktree a directory dir is in, for a comparison to read
// that lane's own scope alone (lanesFor, heldBeforeGitIn): a lane
// the drive holds whose tree holds dir, else the nearest directory at or
// above dir named as the loop names a lane's worktree (isLaneWorktree) other
// than the checkout, else "" (a git that runs in no lane worktree).
func (id *remoteIdentity) laneOf(repoRoot, dir string) string {
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	real := lanePath(abs)
	id.mu.Lock()
	held := slices.Collect(maps.Keys(id.lanes))
	id.mu.Unlock()
	slices.Sort(held)
	for _, p := range held {
		if real == p || strings.HasPrefix(real, p+string(filepath.Separator)) {
			return p
		}
	}
	for p := real; ; p = filepath.Dir(p) {
		if isLaneWorktree(p) && !samePath(repoRoot, p, repoRoot) {
			return p
		}
		if filepath.Dir(p) == p {
			return ""
		}
	}
}

// laneMade is the drive's reading of a lane worktree it has just made, or
// has adopted, at path, before an agent runs there: git's add copies the
// checkout's own worktree scope into the worktree it makes (config.worktree,
// where extensions.worktreeConfig is on), so the keys of driveHeldConfig
// there must be those the checkout held at that scope as the drive started
// (checkoutOwn), or, for an adopted lane the drive already holds, those it
// holds for it (laneChanged), and a difference is a held change that stops
// the lane's worktree stage, naming each key and the worktree. What it reads
// becomes the lane's held state (lanes), which every
// later comparison compares that lane with. With nothing carried (a step the
// host runs, a single build), or where git reads no worktree's own settings,
// it reads and holds nothing. Its reads run inside one bound
// (boundedCompare); a read that gives no answer the hold can take, or a
// comparison that does not finish, stops the step at the config stage.
func laneMade(c Context, laneID, path string) error {
	if c.remote == nil {
		return nil
	}
	type made struct {
		on, gone bool
		keys     map[string]string
	}
	m, err := boundedCompare(func(b readBounds) (made, error) {
		on, err := worktreeScopeRead(c.RepoRoot, b)
		if err != nil || !on {
			return made{}, err
		}
		keys, gone, err := laneOwnScope(path, b)
		return made{on: true, gone: gone, keys: keys}, err
	})
	if err != nil {
		return heldReadRefusal(c, laneID, err)
	}
	if !m.on || m.gone {
		return nil
	}
	path = lanePath(path)
	names := c.remote.laneChanged(c.RepoRoot, &remoteIdentity{lanes: map[string]map[string]string{path: m.keys}, lanesRead: lanesRead{lane: path}})
	if len(names) > 0 {
		slices.Sort(names)
		return refuse(string(StageWorktree), "", laneID,
			fmt.Sprintf("the lane's worktree, as the worktree stage takes it, sets settings the drive holds at its own scope otherwise than the drive holds them there (%s; a worktree made since the drive started is held to the checkout's own scope as it then stood, which git's add copies into the worktree it makes), so a program the lane's git would start is not the drive's", strings.Join(names, ", ")),
			"look at what wrote them (`git config --worktree --show-origin --list` in the lane's worktree and in the checkout), and at the checkout's post-checkout hook, which git's add runs there; the drive reads them once, as it starts, so stop it and start it again once they are as they should be")
	}
	c.remote.mu.Lock()
	if c.remote.lanes == nil {
		c.remote.lanes = map[string]map[string]string{}
	}
	c.remote.lanes[path] = m.keys
	c.remote.mu.Unlock()
	return nil
}

// heldBeforeGit is a drive's comparison of what it holds (held, read as it
// started) with what every environment its git runs in reads now: the held
// settings of each (readSettings), every lane worktree's own scope among
// them (everyLane), and every hooks directory git names in an environment
// whose commands run hooks, with info/attributes (readHooks), their reads
// side by side (readHeld), the comparison bounded as a whole by
// heldCompareTimeout (boundedCompare). It is made before each step of a
// drive (advanceWith); at each pass of the drive's loop, which ends on a
// difference or a read that gives no answer (terminal.go); and before the
// drive releases what it holds as it ends (letGo), outside the drain's lock.
// While a drive runs, the same comparison, reading the own scope of the lane
// worktrees the git runs in or names alone (heldBeforeGitIn), is made before every
// git command abcd starts with the repository's hooks, merge drivers or
// content filters on, or that reaches the network, in a step or not (the
// lane's add, commit, merge and reset, the landing push, a move's
// reservation, a renewal made while an agent runs, the network probe:
// gitutil.SetBeforeHookedGitAt, which the drive sets), which is not run on a
// difference. Each comparison answers for the git it precedes: none is
// reused. A difference, a read that gives no answer the comparison can take,
// or a comparison that does not finish stops it at the config stage, naming
// which; the drive reads what it holds once, as it starts, so the remedy for
// a difference is to stop it and start it again. With nothing held (a step
// the host runs, a single build) it compares nothing. What changes between
// this read and git's own read is not seen (the file's header).
func heldBeforeGit(repoRoot string, held *remoteIdentity) error {
	return heldBeforeGitReading(repoRoot, held, everyLane)
}

// heldBeforeGitIn is heldBeforeGit for a git about to run in dir with args:
// of the lane worktrees it reads only those that git runs in or names
// (lanesFor), none for a git that runs in and names none.
func heldBeforeGitIn(repoRoot string, held *remoteIdentity, dir string, args []string) error {
	if held == nil {
		return nil
	}
	return heldBeforeGitReading(repoRoot, held, held.lanesFor(repoRoot, dir, args))
}

// lanesFor is which lane worktrees a git about to run in dir with args reads
// at their own scope: the lane each -C before its subcommand moves it into
// (laneOf of that directory, else of dir), and, for a `git worktree`
// subcommand, the lane each path it names is in (git's `worktree remove`
// starts a status in the worktree it removes, which reads that worktree's own
// settings). One lane is read alone; two or more, or a git whose lane cannot
// be told from its arguments (a --git-dir or --work-tree anywhere in them, or
// a -C after its subcommand, before any "--"), read every lane; none reads
// none. A word given as an option's value is taken as a path too, which can
// only add a lane to read.
func (id *remoteIdentity) lanesFor(repoRoot, dir string, args []string) lanesRead {
	at := dir
	join := func(p string) string {
		if filepath.IsAbs(p) || at == "" {
			return p
		}
		return filepath.Join(at, p)
	}
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == "--git-dir" || a == "--work-tree" || strings.HasPrefix(a, "--git-dir=") || strings.HasPrefix(a, "--work-tree=") {
			return everyLane
		}
	}
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		if a == "-c" || a == "--namespace" || a == "--config-env" {
			i++
			continue
		}
		if a == "-C" {
			if i+1 < len(args) && args[i+1] != "" {
				at = join(args[i+1])
			}
			i++
			continue
		}
		if !strings.HasPrefix(a, "-") {
			break
		}
	}
	lanes := map[string]bool{}
	add := func(p string) {
		if l := id.laneOf(repoRoot, p); l != "" {
			lanes[l] = true
		}
	}
	add(at)
	rest := args[min(i+1, len(args)):]
	for _, a := range rest {
		if a == "--" {
			break
		}
		if a == "-C" {
			return everyLane
		}
	}
	if i < len(args) && args[i] == "worktree" {
		named, past := 0, false
		for j, a := range rest {
			switch {
			case a == "--" && !past:
				past = true
			case j == 0, a == "" || !past && strings.HasPrefix(a, "-"):
				// The worktree subcommand, or an option.
			default:
				add(join(a))
				named++
			}
		}
		if named == 0 && len(rest) > 0 && rest[0] == "repair" {
			return everyLane
		}
	}
	switch len(lanes) {
	case 0:
		return lanesRead{}
	case 1:
		for l := range lanes {
			return lanesRead{lane: l}
		}
	}
	return everyLane
}

// heldBeforeGitReading is heldBeforeGit reading the lane worktrees which
// names.
func heldBeforeGitReading(repoRoot string, held *remoteIdentity, which lanesRead) error {
	if held == nil {
		return nil
	}
	names, err := boundedCompare(func(b readBounds) ([]string, error) {
		now, err := readHeld(repoRoot, held.salt, "", b, which)
		if err != nil {
			return nil, err
		}
		names := held.changed(repoRoot, &remoteIdentity{envs: now.envs, lanes: now.lanes, lanesRead: which, files: now.files})
		return append(names, nameChanges(repoRoot, configFingerprint{Files: held.hooks}.changes(repoRoot, configFingerprint{Files: now.hooks}))...), nil
	})
	if errors.Is(err, errHooksPastCap) {
		return refuse(StageConfig, "", "", err.Error()+", so what the drive's next git would run cannot be shown to be what it started with, and no git step is taken",
			"look at what grew the hooks directory; stop the drive and start it again once the hooks are as they should be; nothing was made")
	}
	if err != nil {
		return heldBeforeGitRefusal(repoRoot, err)
	}
	if len(names) == 0 {
		return nil
	}
	slices.Sort(names)
	return refuse(StageConfig, "", "",
		"what the drive holds changed since it started ("+strings.Join(slices.Compact(names), ", ")+"): the settings it holds ("+remoteIdentityHeld+"), or a hooks directory its git runs or info/attributes; its next git command would run hooks or reach the network under them, so no git step is taken",
		"look at what changed them (`git config --show-origin --show-scope --get-regexp '"+heldPattern+"'`, the hooks directories and the files named), in each environment the drive's git runs in: as yourself for the push's and the ask of origin's, and with HOME set to the stage's home, as the drive sets it, for the lane's add and commit; the drive reads them once, as it starts, so stop it and start it again once they are as they should be; nothing was made")
}

// heldBeforeGitRefusal is heldBeforeGit's refusal of a read it depends on
// that gave no answer it can take.
func heldBeforeGitRefusal(repoRoot string, err error) error {
	if errors.Is(err, errCompareUnfinished) {
		return refuse(StageConfig, "", "", unfinishedReason(err)+", and no git step is taken", unfinishedRemedy)
	}
	return refuse(StageConfig, "", "",
		"a read the drive's hold depends on gave no answer it can take, so what its next git would run cannot be shown to be what it started with: "+nameChanges(repoRoot, []string{fsutil.RedactHome(err.Error())})[0],
		readRemedy("run the step again"))
}

// heldReadRefusal is the refusal of the lane's step when a read the hold
// depends on gave no answer it can take: it stops at the config stage,
// naming the read and why.
func heldReadRefusal(c Context, laneID string, err error) error {
	if errors.Is(err, errCompareUnfinished) {
		return refuse(StageConfig, "", laneID, unfinishedReason(err)+", and the lane's git is not run", unfinishedRemedy)
	}
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
// lane calls it once, since it asks origin nothing. Its reads run side by
// side inside one bound (boundedCompare). A read that gives no answer the
// hold can take, or a comparison that does not finish, stops the step at the
// config stage. With nothing carried it checks nothing.
func remoteUnchanged(c Context, laneID string) error {
	if c.remote == nil {
		return nil
	}
	keys, err := boundedCompare(func(b readBounds) ([]string, error) {
		now, err := readSettings(c.RepoRoot, c.remote.salt, b, everyLane)
		if err != nil {
			return nil, err
		}
		return c.remote.changed(c.RepoRoot, &remoteIdentity{envs: now.envs, lanes: now.lanes, lanesRead: everyLane, files: now.files}), nil
	})
	if err != nil {
		return heldReadRefusal(c, laneID, err)
	}
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
// holds the directory through the add (the file's header). Its reads run
// side by side inside one bound (boundedCompare). A read that gives no answer
// the hold can take, or a comparison that does not finish, stops the step at
// the config stage. With nothing carried (a step the host runs, a single
// build) it checks nothing.
func checkoutHooksUnchanged(c Context, laneID, hooksDir string) error {
	if c.remote == nil {
		return nil
	}
	now, err := boundedCompare(func(b readBounds) (map[string]string, error) {
		now, _, err := readHooks(c.RepoRoot, c.remote.salt, hooksDir, b)
		return now, err
	})
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

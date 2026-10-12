// Package dogfood decides which abcd binary a hook runs in a checkout the
// person has opted in to dogfooding, and builds it (iss-2610070547057816).
//
// In abcd's own source checkout the installed release guards the session,
// so a hazard rule that landed on the default branch after the last release
// is not enforced there until the next one ships. The person opts a checkout
// in by listing it in ~/.abcd.noindex/dogfood-roots, one checkout per line:
// its absolute path, a tab, the URL of the remote its reviewed branch lives
// on, and optionally a tab and that branch, main when left out; "#" starts a
// comment. Inside a listed checkout, and inside every worktree that checkout
// records as its own, the hooks run a build of that branch's tip as the
// declared remote reports it, and everywhere else the installed release,
// unchanged.
//
// Four properties are load-bearing.
//
// The opt-in is the PERSON'S, read from their home and never from a
// repository. Recognising abcd's source by its module path was rejected: a
// repository posing as abcd would get its own code run by the hooks. The list
// is read through fsutil.HomeDeclarationEntry, the fields-bearing form of the
// one reader behind every "declare this checkout" opt-in (trusted-roots,
// local-transcript-roots, filter-roots), so it is honoured only as a regular
// file this account owns that no one else can write, within its size cap,
// reached through no symlinked folder; a list failing that is IGNORED out
// loud and the installed release runs everywhere. A line that names a
// checkout but no remote builds nothing, and says so.
//
// The build's SOURCE is the tip the declared remote reports for the declared
// branch, and the store's record of that answer is the only thing that
// chooses a build. The background builder asks the remote (`git ls-remote`
// under the isolated git environment, outside every repository, never
// prompting for credentials, bounded in time and in output), records the
// answer in the store under the build lock, and builds that commit once the
// checkout holds it. Resolve, the hook's half, never touches the network and
// names a ready build only for the recorded tip. The checkout's own refs and
// config choose nothing: refs/remotes/origin/*, origin/HEAD and remote.origin.*
// are all an agent in the checkout can rewrite with ordinary git commands, so
// refs/remotes/origin/<branch> serves only to prompt a fresh check when it is
// not what it was at the last one, as the record's age does once it passes
// recheckAfter. Moving a local ref therefore costs at most one ls-remote, which
// returns the reviewed tip, and no agent commit changes the guard that judges
// the agent's next command.
//
// That holds only for a remote no agent can move, so the declared remote is
// read over https or ssh alone (remoteRefusal). A filesystem path or a file://
// URL names a repository this account can write, which an agent moves to its
// own commit with one `git fetch` into it, and git:// and http:// let anyone on
// the network path answer; a line declaring any of them is refused out loud
// and builds nothing. Which https or ssh remote is declared stays the person's
// call: it must be one the agent cannot push to, such as a protected branch
// on a forge.
//
// The build is the reviewed commit or nothing. The checkout's object store,
// its refs/replace/* and its repository-local attributes are writable by
// anyone who can commit there, so the commit and every tree under it are read
// with replacement objects off and re-hashed against their names
// (verifiedTree), and the export is checked blob by blob against that verified
// tree (verifyExport) before anything is built.
//
// A failure never weakens the guard below the installed release: no declared
// remote, a remote that cannot be read, a tip the checkout has not fetched,
// and a build that is missing, running, failed, or not runnable each leave
// Resolve naming no binary, the hook runs the installed release, and a note
// says why on stderr. The PreToolUse shim adds the run-time half: a dogfood
// guard that answers exit 1, which is a warn or "NOT CHECKED", or that dies,
// hands the same command to the installed release, whose deny stands.
//
// What this does not defend is the account itself. The declaration and the
// store are protected by being this uid's alone, so anything running as this
// account can still edit the declaration or replace a file in the store, and
// the build runs under the account's own toolchain environment (the go
// toolchain and module cache, GOENV, PATH, ~/.ssh/config), which stays the
// account's; what no longer chooses the build is anything in the checkout.
package dogfood

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

var (
	// RootsRelPath is the person's opt-in list, relative to their home.
	RootsRelPath = abcdhome.Rel("dogfood-roots")

	// RootsDisplay names the list in tilde form, so a message carries no home
	// path and a line a person pastes into a shell works.
	RootsDisplay = abcdhome.Display("dogfood-roots")

	// StoreDisplay names the build store in tilde form.
	StoreDisplay = abcdhome.Display("dogfood")

	// StoreRelPath is the machine-scoped build store, relative to the home:
	// dogfood/<root-sha>/<commit>/abcd. It is keyed on the repository's root
	// commit, as the worktree and transcript stores are, and on the commit the
	// build was made from, so a build is never shared between repositories and
	// a moved tip is a different directory.
	StoreRelPath = abcdhome.Rel("dogfood")
)

const (
	// BinaryName is the build's file name inside its commit directory.
	BinaryName = "abcd"

	// DefaultBranch is the reviewed branch of a line that names none.
	DefaultBranch = "main"

	// TrackingRemote is the remote whose remote-tracking ref for the declared
	// branch prompts a fresh check when it moves. It never names the source.
	TrackingRemote = "origin"

	// TipName is the store's record of the last check of the declared remote,
	// beside the builds it chooses between.
	TipName = "verified-tip.json"

	// maxRootsBytes caps the list read: a hand-kept list of checkouts is a
	// few hundred bytes.
	maxRootsBytes = 64 << 10

	// maxTipBytes caps the read of the store's record.
	maxTipBytes = 16 << 10

	// worktreeListCap bounds `git worktree list`.
	worktreeListCap = 1 << 20

	// keepBuilds is how many commit builds a repository keeps; older ones are
	// removed after each successful build.
	keepBuilds = 3

	// recheckAfter is how old the record of the last check may grow before a
	// hook asks for another, whatever the local refs say. The build it
	// recorded keeps running meanwhile.
	recheckAfter = 30 * time.Minute

	// retryAfter is how long a failed build of one commit is left alone before
	// a hook starts it again. A remote-tracking ref that moved starts it at
	// once.
	retryAfter = 30 * time.Minute

	// buildTimeout bounds one export and build.
	buildTimeout = 15 * time.Minute

	// lsRemoteTimeout bounds one check of the declared remote.
	lsRemoteTimeout = 60 * time.Second

	// maxLsRemoteBytes caps what the check reads back: one ref is one line.
	maxLsRemoteBytes = 64 << 10

	// allowedProtocols are the transports a check may use (GIT_ALLOW_PROTOCOL),
	// the two remoteRefusal admits. A declared URL cannot name a transport that
	// runs a program (ext::), a repository this account can write (file, which
	// git's local-path transport is too), or a plaintext one (git, http), even
	// if it got past parseRemote; there is deliberately no way to widen it.
	allowedProtocols = "https:ssh"

	lockName     = ".build.lock"
	failedSuffix = ".failed"
	logSuffix    = ".log"
	tmpPrefix    = ".build-"
)

// ErrBuildRunning is Build's answer when another build holds the store's lock.
var ErrBuildRunning = errors.New("another dogfood build is running")

// Checkout is a listed checkout a session runs in, the remote the person
// declared for it, and the tip the store last verified there.
type Checkout struct {
	// Toplevel is the session's working tree.
	Toplevel string
	// Repo is the listed checkout that vouches for Toplevel: Toplevel itself,
	// or the main working tree that records Toplevel as one of its worktrees.
	// Every git read and the export run there.
	Repo    string
	RootSHA string
	// URL and Branch are the reviewed remote and branch the person's line
	// declares for Repo.
	URL, Branch string
	// Tracking is the commit refs/remotes/origin/<Branch> names in Repo, "" when
	// none. It only prompts a check; it is never built.
	Tracking string
	// Commit is the tip the store last verified on URL, "" before a check has
	// succeeded.
	Commit string
}

// Short is the commit's abbreviation as a message shows it.
func (c Checkout) Short() string { return short(c.Commit) }

// LogDisplay is the build log's path in tilde form.
func (c Checkout) LogDisplay() string {
	return abcdhome.Display("dogfood", c.RootSHA, c.Commit+logSuffix)
}

// Source names the declared branch and remote as a message shows them, with
// any credentials in the URL left out.
func (c Checkout) Source() string { return c.Branch + " on " + displayURL(c.URL) }

// Plan is what a hook runs.
type Plan struct {
	// Binary is the build to run, "" for the installed release.
	Binary string
	// Checkout is set when the session's checkout is listed with a remote.
	Checkout *Checkout
	// StartBuild asks the caller to start a check and build of Checkout in
	// the background; Binary, or the installed release, runs meanwhile.
	StartBuild bool
	// Notes are whole lines for stderr, each starting "abcd dogfood:".
	Notes []string
}

// caseFolding is the case-folding predicate the list match uses, a variable so
// a test can force either branch.
var caseFolding = fsutil.CaseFoldingFS

// now is the clock the retry and re-check windows read, a variable for tests.
var now = time.Now

// lsRemote asks url for the commit refs/heads/<branch> names. It is the one
// network call in the package, made by Build alone, and a variable so a test
// can prove Resolve never makes it.
var lsRemote = lsRemoteGit

// Resolve decides what a hook standing in cwd runs. home is the person's home
// directory. It writes nothing and reads nothing over the network; a check or
// build it asks for is the caller's to start.
func Resolve(home, cwd string) Plan {
	if home == "" {
		return Plan{}
	}
	// No list is the ordinary case and costs one Lstat. The Lstat follows the
	// folders, so a list behind a symlinked ~/.abcd.noindex is found here and
	// refused by the reader below.
	if _, err := os.Lstat(filepath.Join(home, filepath.FromSlash(RootsRelPath))); err != nil {
		return Plan{}
	}
	c, note := listedCheckout(home, cwd)
	if c == nil {
		if note != "" {
			return Plan{Notes: []string{note}}
		}
		return Plan{}
	}
	root, err := gitutil.RootCommit(c.Repo)
	if err != nil || !gitutil.IsFullSHA(root) {
		return Plan{Notes: []string{"abcd dogfood: " + fsutil.DisplayPath(c.Toplevel) + " is listed in " + RootsDisplay +
			", but its root commit could not be read; the installed release runs"}}
	}
	c.RootSHA = root
	c.Tracking = trackingTip(c.Repo, c.Branch)
	storeAbs := filepath.Join(home, filepath.FromSlash(path.Join(StoreRelPath, root)))
	rec, recNote := readTip(home, root)
	current := rec != nil && rec.URL == c.URL && rec.Branch == c.Branch
	due := !current || rec.Tracking != c.Tracking || now().Sub(rec.Checked) >= recheckAfter || rec.Checked.After(now())
	running := building(storeAbs)
	var notes []string
	if recNote != "" {
		notes = append(notes, recNote)
	}
	if current && rec.Commit != "" {
		c.Commit = rec.Commit
		if bin, ok := readyBinary(home, root, rec.Commit); ok {
			return Plan{Binary: bin, Checkout: c, StartBuild: due && !running, Notes: notes}
		}
	}
	if running {
		return Plan{Checkout: c, Notes: append(notes, "abcd dogfood: the check and build of "+c.Source()+
			" is still running (logs in "+abcdhome.Display("dogfood", root)+"); the installed release runs until it is ready")}
	}
	if current && !due {
		if rec.Note != "" {
			return Plan{Checkout: c, Notes: append(notes, rec.Note+"; the installed release runs, and "+
				c.Source()+" is checked again after "+recheckAfter.String()+" or when "+trackingShort(c.Branch)+" moves")}
		}
		if fi, err := os.Lstat(filepath.Join(storeAbs, rec.Commit+failedSuffix)); err == nil && now().Sub(fi.ModTime()) < retryAfter {
			return Plan{Checkout: c, Notes: append(notes, "abcd dogfood: the build of "+c.Source()+" at "+c.Short()+
				" FAILED (log: "+c.LogDisplay()+"); the installed release runs instead, and the build is tried again after "+
				retryAfter.String()+" or when "+trackingShort(c.Branch)+" moves")}
		}
	}
	return Plan{Checkout: c, StartBuild: true, Notes: append(notes, "abcd dogfood: no verified build yet of "+c.Source()+
		"; checking the remote and building it in the background (logs in "+abcdhome.Display("dogfood", root)+
		"), and the installed release runs until it is ready")}
}

// listedCheckout resolves the session's working tree and reports it when the
// person's list covers it: the tree itself is listed, or the main working tree
// of its repository is listed AND that main working tree records the session's
// tree among its own worktrees. The second half is read from the listed
// checkout, never from the session's tree, so a directory whose .git merely
// points into a listed repository is not covered by it. The remote is the one
// the listed checkout's line declares. note is a refusal to say out loud,
// empty when the tree is simply not listed.
func listedCheckout(home, cwd string) (*Checkout, string) {
	top, err := gitutil.Toplevel(cwd)
	if err != nil {
		return nil, ""
	}
	declared := func(p string) ([]string, bool, string) {
		return fsutil.HomeDeclarationEntry(home, RootsRelPath, maxRootsBytes, p, caseFolding())
	}
	fields, on, why := declared(top)
	if why != "" {
		return nil, "abcd dogfood: IGNORED " + RootsDisplay + " — " + why + "; the installed release runs in every checkout"
	}
	repo := top
	if !on {
		wts, err := gitutil.ListWorktrees(top, worktreeListCap)
		if err != nil || len(wts) == 0 || wts[0].Bare || wts[0].Path == "" {
			return nil, ""
		}
		main := wts[0].Path
		if same(main, top) {
			return nil, ""
		}
		if fields, on, _ = declared(main); !on {
			return nil, ""
		}
		repo = main
	}
	if !recordsWorktree(repo, top) {
		return nil, "abcd dogfood: REFUSED — " + fsutil.DisplayPath(top) + " points at the listed checkout " +
			fsutil.DisplayPath(repo) + " as its repository, but that checkout does not record it as one of its worktrees; the installed release runs"
	}
	remote, branch, problem := parseRemote(fields)
	if problem != "" {
		return nil, "abcd dogfood: " + fsutil.DisplayPath(repo) + " is listed in " + RootsDisplay + ", but " + problem +
			", so nothing is built and the installed release runs. A line is the checkout, a tab, the https:// or ssh URL of the remote its reviewed branch lives on, and optionally a tab and the branch (" +
			DefaultBranch + " when left out): printf '%s\\t%s\\t%s\\n' '<checkout>' '<https-or-ssh-url>' '" + DefaultBranch + "' >> " + RootsDisplay
	}
	return &Checkout{Toplevel: top, Repo: repo, URL: remote, Branch: branch}, ""
}

// parseRemote reads the fields after a listed checkout's path: the remote's
// URL, then optionally the branch. problem says what is wrong with them, ""
// when they are usable.
func parseRemote(fields []string) (remote, branch, problem string) {
	if len(fields) == 0 || fields[0] == "" {
		return "", "", "its line names no remote to read the reviewed tip from"
	}
	if len(fields) > 2 {
		return "", "", "its line has more fields than a checkout, a remote URL and a branch"
	}
	remote, branch = fields[0], DefaultBranch
	if len(fields) == 2 && fields[1] != "" {
		branch = fields[1]
	}
	if strings.HasPrefix(remote, "-") || strings.ContainsFunc(remote, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return "", "", "its remote URL is not one git can be handed safely"
	}
	if why := remoteRefusal(remote); why != "" {
		return "", "", "its remote " + shownRemote(remote) + " is REFUSED, because " + why +
			" (declare an https:// or ssh remote whose branch no agent can move, such as a protected branch on a forge)"
	}
	if !validBranch(branch) {
		return "", "", "its branch is not a branch name"
	}
	return remote, branch, ""
}

// urlSchemeRe is the scheme of a URL git recognises as one ("<scheme>://").
var urlSchemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)

// scpHostRe is the host of an scp-like remote ([user@]host:path): a DNS name
// or an address, nothing git or ssh could read as an option.
var scpHostRe = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$`)

// remoteRefusal says why a declared remote cannot vouch for the reviewed tip,
// "" when it is one the builder reads over https or ssh. The tip the builder
// records is only as trustworthy as the remote it is read from, so the
// remote must be one no agent in the checkout can move:
//
//   - a filesystem path (absolute, relative or ~) and a file:// URL name a
//     repository this account can write, and an agent running as this
//     account moves its branch with one `git fetch` into it;
//   - git:// and http:// are plaintext, so anyone on the network path can
//     answer with the tip they choose;
//   - any other transport, a remote helper ("<transport>::<address>")
//     included, is not one the builder uses.
//
// What is left is https:// and ssh, as ssh://… or scp-like [user@]host:path,
// and git's own rule decides which is which: a remote with no ":", or with a
// "/" before its first ":", is a local path.
func remoteRefusal(remote string) string {
	if scheme, rest, ok := strings.Cut(remote, "://"); ok && urlSchemeRe.MatchString(scheme) {
		switch scheme {
		case "https", "ssh":
		case "file":
			return "a file:// URL names a repository anything running as this account can write"
		case "git", "http":
			return scheme + ":// is a plaintext transport, on which anyone on the network path can answer with the tip they choose"
		default:
			return "the builder reads a remote over https:// or ssh only, not the " + scheme + ":// transport"
		}
		u, err := url.Parse(remote)
		if err != nil || u.Opaque != "" || rest == "" {
			return "it is not a URL git can read"
		}
		if h := u.Hostname(); h == "" || strings.HasPrefix(h, "-") || (u.User != nil && strings.HasPrefix(u.User.Username(), "-")) {
			return "it names no host to read from"
		}
		return ""
	}
	if strings.Contains(remote, "::") {
		return "it names a remote-helper transport (\"<transport>::<address>\"), which the builder does not use"
	}
	colon := strings.IndexByte(remote, ':')
	if slash := strings.IndexByte(remote, '/'); colon < 0 || (slash >= 0 && slash < colon) {
		return "a filesystem path names a repository anything running as this account can write"
	}
	host := remote[:colon]
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		host = host[at+1:]
	}
	if !scpHostRe.MatchString(host) {
		return "it names no host to read from"
	}
	if remote[colon+1:] == "" {
		return "it names no repository on its host"
	}
	return ""
}

// shownRemote is a declared remote as a note shows it: a path with the home
// folded to ~, a URL with any credentials left out.
func shownRemote(remote string) string {
	if filepath.IsAbs(remote) {
		return fsutil.RedactHome(remote)
	}
	return displayURL(remote)
}

// branchRe is the characters a declared branch may hold; validBranch adds
// git's structural rules on top.
var branchRe = regexp.MustCompile(`^[A-Za-z0-9._/+-]+$`)

// validBranch holds a declared branch to a conservative subset of
// check-ref-format: no empty component, no component starting with "." or
// ending ".lock", no "..", and no leading "-".
func validBranch(b string) bool {
	if !branchRe.MatchString(b) || strings.HasPrefix(b, "-") || strings.Contains(b, "..") || strings.HasSuffix(b, ".") {
		return false
	}
	for _, part := range strings.Split(b, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}

// displayURL is url with any user information left out, so a token in an
// https URL never reaches stderr.
func displayURL(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.User != nil {
		u.User = nil
		return u.String()
	}
	return raw
}

// trackingShort is the remote-tracking ref that prompts a check, as a person
// names it.
func trackingShort(branch string) string { return TrackingRemote + "/" + branch }

// trackingTip is the commit refs/remotes/origin/<branch> names in repo, "" when
// it names none. It is a prompt to check the remote and nothing more.
func trackingTip(repo, branch string) string {
	commit, err := gitutil.Run(repo, "rev-parse", "--verify", "--quiet", "refs/remotes/"+TrackingRemote+"/"+branch+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(commit) {
		return ""
	}
	return commit
}

// recordsWorktree reports whether the checkout at repo lists tree among its
// working trees, compared by file identity.
func recordsWorktree(repo, tree string) bool {
	if same(repo, tree) {
		return true
	}
	wts, err := gitutil.ListWorktrees(repo, worktreeListCap)
	if err != nil {
		return false
	}
	for _, w := range wts {
		if !w.Bare && w.Path != "" && same(w.Path, tree) {
			return true
		}
	}
	return false
}

func same(a, b string) bool {
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	return err == nil && os.SameFile(fa, fb)
}

// tipRecord is the store's record of the last check of a repository's
// declared remote, written by Build under the store's lock.
type tipRecord struct {
	// Commit is the tip the remote reported, "" when the check failed.
	Commit string `json:"commit,omitempty"`
	// URL and Branch are what was checked; a record for another declaration
	// chooses nothing.
	URL    string `json:"url"`
	Branch string `json:"branch"`
	// Tracking is the remote-tracking ref's commit when the check was asked for.
	Tracking string `json:"tracking,omitempty"`
	// Checked is when the check was made.
	Checked time.Time `json:"checked"`
	// Note says why the check left nothing to build, "" when it did not.
	Note string `json:"note,omitempty"`
}

// readTip reads the store's record for the repository rooted at rootSHA. A
// record that is absent is nil with no note; one that is present but not this
// account's alone, or unreadable, is nil with a note saying so, and a check
// is asked for that replaces it.
func readTip(home, rootSHA string) (*tipRecord, string) {
	if !storeAlone(home, rootSHA) {
		return nil, ""
	}
	r, err := fsutil.OpenHomeScope(home, path.Join(StoreRelPath, rootSHA))
	if err != nil {
		return nil, ""
	}
	defer r.Close()
	fi, err := r.Lstat(TipName)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ""
	}
	shown := abcdhome.Display("dogfood", rootSHA, TipName)
	refused := "abcd dogfood: IGNORED " + shown + " — "
	if err != nil {
		return nil, refused + "it could not be read"
	}
	if !fi.Mode().IsRegular() {
		return nil, refused + "it is not a regular file"
	}
	if fsutil.CallersAlone(filepath.Join(home, filepath.FromSlash(path.Join(StoreRelPath, rootSHA, TipName))), fi) != nil {
		return nil, refused + "it is not this account's alone"
	}
	f, err := r.OpenFile(TipName, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, refused + "it could not be read"
	}
	defer f.Close()
	if st, err := f.Stat(); err != nil || !os.SameFile(st, fi) {
		return nil, refused + "it was replaced while it was read"
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxTipBytes+1))
	if err != nil || len(raw) > maxTipBytes {
		return nil, refused + "it could not be read"
	}
	var rec tipRecord
	if err := json.Unmarshal(raw, &rec); err != nil || (rec.Commit != "" && !gitutil.IsFullSHA(rec.Commit)) {
		return nil, refused + "it is not a record this build wrote"
	}
	return &rec, ""
}

// writeTip replaces the store's record through fsutil.WriteFileAtomic, so a
// reader sees the old record or the new one, never part of either.
func writeTip(storeAbs string, rec tipRecord) error {
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(storeAbs, TipName), append(body, '\n'), abcdhome.FileMode)
}

// storeAlone reports whether every level of the repository's store is a real
// directory owned by this uid that no one else can write.
func storeAlone(home, rootSHA string) bool {
	for _, level := range []string{StoreRelPath, path.Join(StoreRelPath, rootSHA)} {
		p := filepath.Join(home, filepath.FromSlash(level))
		fi, err := os.Lstat(p)
		if err != nil || !fi.IsDir() || fsutil.CallersAlone(p, fi) != nil {
			return false
		}
	}
	return true
}

// readyBinary is the finished build of commit for the repository rooted at
// rootSHA, when there is one this account alone can change: every store level
// a real directory owned by this uid that no one else can write, and the
// binary a regular, executable file of the same kind.
func readyBinary(home, rootSHA, commit string) (string, bool) {
	if !gitutil.IsFullSHA(rootSHA) || !gitutil.IsFullSHA(commit) {
		return "", false
	}
	dir := path.Join(StoreRelPath, rootSHA, commit)
	r, err := fsutil.OpenHomeScope(home, dir)
	if err != nil {
		return "", false
	}
	defer r.Close()
	if !storeAlone(home, rootSHA) {
		return "", false
	}
	p := filepath.Join(home, filepath.FromSlash(dir))
	if fi, err := os.Lstat(p); err != nil || !fi.IsDir() || fsutil.CallersAlone(p, fi) != nil {
		return "", false
	}
	fi, err := r.Lstat(BinaryName)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o100 == 0 {
		return "", false
	}
	bin := filepath.Join(p, BinaryName)
	if fsutil.CallersAlone(bin, fi) != nil {
		return "", false
	}
	return bin, true
}

// building reports whether a build holds the store's lock.
func building(storeAbs string) bool {
	if _, err := os.Lstat(storeAbs); err != nil {
		return false
	}
	err := fsutil.WithFileLock(filepath.Join(storeAbs, lockName), 0, func() error { return nil })
	return errors.Is(err, fsutil.ErrLockContention)
}

// Build checks the declared remote for the reviewed tip, records the answer
// in the store, and makes the build of that tip unless one is already there,
// returning its path. It is the only caller of the network. It holds the
// store's lock for the whole check, export and build, so two hooks firing at
// once cannot race it: the second gets ErrBuildRunning at once.
//
// c.Commit is ignored: the commit built is the one the remote reports now,
// never one a caller names. A remote that cannot be read records no tip, and
// a tip the checkout does not hold yet is recorded with a note to fetch it and
// is not built; either way Resolve then names no build and says why. The
// source is a clean export of the tip (`git archive`, checked against the
// commit's verified tree) into a fresh directory under the store, never the
// working tree, and the finished binary is moved into place by one rename, so
// a reader sees no build or a whole one. A failed build leaves a marker
// beside its log that Resolve reads.
func Build(home string, c Checkout) (string, error) {
	if !gitutil.IsFullSHA(c.RootSHA) {
		return "", errors.New("the checkout's root commit is not a full object name")
	}
	if _, _, problem := parseRemote([]string{c.URL, c.Branch}); problem != "" {
		return "", errors.New("the declared remote is not usable: " + problem)
	}
	storeRel := path.Join(StoreRelPath, c.RootSHA)
	r, err := fsutil.EnsureHomeScope(home, storeRel, abcdhome.DirMode)
	if err != nil {
		return "", fmt.Errorf("preparing the build store: %w", err)
	}
	r.Close()
	if !storeAlone(home, c.RootSHA) {
		return "", errors.New("the build store is not this account's alone")
	}
	storeAbs := filepath.Join(home, filepath.FromSlash(storeRel))
	var bin string
	err = fsutil.WithFileLock(filepath.Join(storeAbs, lockName), 0, func() error {
		rec := tipRecord{URL: c.URL, Branch: c.Branch, Tracking: c.Tracking, Checked: now()}
		ctx, cancel := context.WithTimeout(context.Background(), lsRemoteTimeout)
		tip, err := lsRemote(ctx, storeAbs, c.URL, c.Branch)
		cancel()
		if err != nil {
			rec.Note = "abcd dogfood: the reviewed remote could not be read (" + c.Source() + ": " + err.Error() + ")"
			if werr := writeTip(storeAbs, rec); werr != nil {
				return fmt.Errorf("recording the check: %w", werr)
			}
			return fmt.Errorf("reading %s: %w", c.Source(), err)
		}
		rec.Commit = tip
		if !hasCommit(c.Repo, tip) {
			rec.Note = "abcd dogfood: " + c.Source() + " is at " + short(tip) + ", which " + fsutil.DisplayPath(c.Repo) +
				" does not hold yet, so nothing is built; run `git fetch` there"
			if werr := writeTip(storeAbs, rec); werr != nil {
				return fmt.Errorf("recording the check: %w", werr)
			}
			return fmt.Errorf("%s is at %s, which the checkout does not hold yet; run `git fetch` in it", c.Source(), short(tip))
		}
		if err := writeTip(storeAbs, rec); err != nil {
			return fmt.Errorf("recording the check: %w", err)
		}
		c.Commit = tip
		if p, ok := readyBinary(home, c.RootSHA, tip); ok {
			bin = p
			return nil
		}
		p, err := build(storeAbs, c)
		if err != nil {
			_ = os.WriteFile(filepath.Join(storeAbs, tip+failedSuffix), []byte(err.Error()+"\n"), abcdhome.FileMode)
			return fmt.Errorf("%w (log: %s)", err, c.LogDisplay())
		}
		_ = os.Remove(filepath.Join(storeAbs, tip+failedSuffix))
		prune(storeAbs, tip)
		bin = p
		return nil
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return "", ErrBuildRunning
	}
	return bin, err
}

// builderEnv is the environment of every git command the builder runs: the
// isolated environment with replacement objects off, so refs/replace/* in the
// checkout (which `git replace` writes) cannot stand another object in for
// the reviewed one.
func builderEnv(extra ...string) []string {
	return append(append(gitutil.IsolatedEnv(), "GIT_NO_REPLACE_OBJECTS=1"), extra...)
}

// boundedBuffer keeps at most limit bytes and records whether more came.
type boundedBuffer struct {
	buf      bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.buf.Len(); room < len(p) {
		b.overflow = true
		if room > 0 {
			b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return b.buf.Write(p)
}

// lsRemoteGit is lsRemote: `git ls-remote -- <url> refs/heads/<branch>` run in
// dir (a directory of the store, with discovery stopped at its parent, so no
// repository's config applies), under the isolated environment, with no
// credential prompt of any kind, the transports held to allowedProtocols, and
// output capped. git matches a pattern against the tail of each ref, so the
// answer is the line naming exactly refs/heads/<branch>, and it must be the
// only one.
func lsRemoteGit(ctx context.Context, dir, remote, branch string) (string, error) {
	ref := "refs/heads/" + branch
	args := append(gitutil.ExecPins(), "ls-remote", "--", remote, ref)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = builderEnv(
		"GIT_CEILING_DIRECTORIES="+filepath.Dir(dir),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
		"GIT_ALLOW_PROTOCOL="+allowedProtocols,
	)
	cmd.Stdin = nil
	cmd.WaitDelay = 5 * time.Second
	out := &boundedBuffer{limit: maxLsRemoteBytes}
	errOut := &boundedBuffer{limit: 2048}
	cmd.Stdout, cmd.Stderr = out, errOut
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("no answer within %s", lsRemoteTimeout)
		}
		msg := strings.TrimSpace(errOut.buf.String())
		if i := strings.LastIndexByte(msg, '\n'); i >= 0 {
			msg = strings.TrimSpace(msg[i+1:])
		}
		if msg == "" {
			msg = err.Error()
		}
		// git names the URL in its own messages; credentials in it stay out.
		return "", errors.New(strings.ReplaceAll(msg, remote, displayURL(remote)))
	}
	if out.overflow {
		return "", fmt.Errorf("its answer exceeded %d bytes", maxLsRemoteBytes)
	}
	return tipFromLsRemote(out.buf.String(), branch)
}

// tipFromLsRemote reads the commit refs/heads/<branch> names from the output
// of `git ls-remote`: the one line naming exactly that ref.
func tipFromLsRemote(out, branch string) (string, error) {
	ref := "refs/heads/" + branch
	var tip string
	for _, line := range strings.Split(out, "\n") {
		oid, name, ok := strings.Cut(strings.TrimRight(line, "\r"), "\t")
		if !ok || name != ref {
			continue
		}
		if tip != "" || !gitutil.IsFullSHA(oid) {
			return "", fmt.Errorf("its answer for %s is not one commit", ref)
		}
		tip = oid
	}
	if tip == "" {
		return "", fmt.Errorf("it has no branch %s", branch)
	}
	return tip, nil
}

// hasCommit reports whether repo holds commit as a commit object, read with
// replacement objects off and nothing fetched.
func hasCommit(repo, commit string) bool {
	cmd := exec.Command("git", append(gitutil.ExecPins(), "-C", repo, "cat-file", "-t", commit)...)
	cmd.Env = builderEnv()
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "commit"
}

// build exports c.Commit and builds ./cmd/abcd from it, logging both steps.
func build(storeAbs string, c Checkout) (string, error) {
	logf, err := os.OpenFile(filepath.Join(storeAbs, c.Commit+logSuffix), os.O_CREATE|os.O_TRUNC|os.O_WRONLY|syscall.O_NOFOLLOW, abcdhome.FileMode)
	if err != nil {
		return "", fmt.Errorf("opening the build log: %w", err)
	}
	defer logf.Close()
	fmt.Fprintf(logf, "abcd dogfood: building %s at %s from %s\n", c.Source(), c.Commit, c.Repo)
	tmp, err := os.MkdirTemp(storeAbs, tmpPrefix)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	src, out := filepath.Join(tmp, "src"), filepath.Join(tmp, "out")
	for _, d := range []string{src, out} {
		if err := os.Mkdir(d, abcdhome.DirMode); err != nil {
			return "", err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
	want, err := verifiedTree(ctx, c.Repo, c.Commit)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", short(c.Commit), err)
	}
	if err := export(ctx, c.Repo, c.Commit, src, logf); err != nil {
		return "", fmt.Errorf("exporting %s: %w", short(c.Commit), err)
	}
	if err := verifyExport(want, src); err != nil {
		return "", fmt.Errorf("exporting %s: %w", short(c.Commit), err)
	}
	if err := goBuild(ctx, src, filepath.Join(out, BinaryName), logf); err != nil {
		return "", fmt.Errorf("building %s: %w", short(c.Commit), err)
	}
	// go build leaves the mode to the umask, and readyBinary runs only a
	// binary no one else can write: the build is the store's alone.
	if err := os.Chmod(filepath.Join(out, BinaryName), 0o700); err != nil {
		return "", fmt.Errorf("building %s: %w", short(c.Commit), err)
	}
	final := filepath.Join(storeAbs, c.Commit)
	// Build runs under the store's lock and only when no ready binary stands
	// for this commit, so a directory here is one readyBinary refused (a
	// binary a umask left group-writable, say): it is replaced, or every later
	// build would fail to move into place.
	if err := os.RemoveAll(final); err != nil {
		return "", fmt.Errorf("replacing a refused build: %w", err)
	}
	if err := os.Rename(out, final); err != nil {
		return "", fmt.Errorf("moving the build into place: %w", err)
	}
	fmt.Fprintf(logf, "abcd dogfood: built %s\n", filepath.Join(final, BinaryName))
	return filepath.Join(final, BinaryName), nil
}

// treeEntry is one blob of a verified tree: its mode and object name.
type treeEntry struct{ mode, oid string }

// maxTreeObjects and maxTreeBytes bound verifiedTree's walk: how many commit
// and tree objects it reads, and how many bytes of them in all.
const (
	maxTreeObjects = 1 << 20
	maxTreeBytes   = 64 << 20
)

// verifiedTree reads commit and every tree under it from repo through one
// `git cat-file --batch`, with replacement objects off, re-hashes each object
// against its name, and returns the commit's blobs by path. git checks the
// hash of a commit and of a root tree when it parses them, but `git archive`
// and `git ls-tree` read a subtree without checking it, so a loose object
// written over a subtree of the reviewed commit, or an object store borrowed
// through alternates, would otherwise put other files in the build.
func verifiedTree(ctx context.Context, repo, commit string) (map[string]treeEntry, error) {
	newHash, ok := objectHash(commit)
	if !ok {
		return nil, errors.New("the commit is not a full object name")
	}
	rawLen := newHash().Size()
	cmd := exec.CommandContext(ctx, "git", append(gitutil.ExecPins(), "-C", repo, "cat-file", "--batch")...)
	cmd.Env = builderEnv()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// The walk stops at its first refusal with git perhaps still writing an
	// answer nobody reads, so git is ended rather than waited for.
	defer func() {
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	rd := bufio.NewReader(stdout)
	objects, total := 0, 0
	read := func(oid, wantType string) ([]byte, error) {
		objects++
		if objects > maxTreeObjects {
			return nil, fmt.Errorf("the commit holds more than %d trees", maxTreeObjects)
		}
		if _, err := io.WriteString(stdin, oid+"\n"); err != nil {
			return nil, err
		}
		header, err := rd.ReadString('\n')
		if err != nil {
			return nil, err
		}
		f := strings.Fields(header)
		if len(f) != 3 || f[0] != oid || f[1] != wantType {
			return nil, fmt.Errorf("object %s is not the %s the commit records", short(oid), wantType)
		}
		size, err := strconv.Atoi(f[2])
		if err != nil || size < 0 {
			return nil, fmt.Errorf("object %s has an unreadable size", short(oid))
		}
		if total += size; total > maxTreeBytes {
			return nil, fmt.Errorf("the commit's trees exceed %d bytes", maxTreeBytes)
		}
		body := make([]byte, size+1)
		if _, err := io.ReadFull(rd, body); err != nil {
			return nil, err
		}
		body = body[:size]
		h := newHash()
		fmt.Fprintf(h, "%s %d\x00", wantType, size)
		h.Write(body)
		if hex.EncodeToString(h.Sum(nil)) != oid {
			return nil, fmt.Errorf("object %s does not hash to its name (the checkout's object store was altered)", short(oid))
		}
		return body, nil
	}
	body, err := read(commit, "commit")
	if err != nil {
		return nil, err
	}
	first, _, _ := strings.Cut(string(body), "\n")
	rootTree, ok := strings.CutPrefix(first, "tree ")
	if !ok || len(rootTree) != len(commit) {
		return nil, errors.New("the commit names no tree")
	}
	want := map[string]treeEntry{}
	type pending struct{ dir, oid string }
	queue := []pending{{"", rootTree}}
	for len(queue) > 0 {
		t := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		body, err := read(t.oid, "tree")
		if err != nil {
			return nil, err
		}
		for len(body) > 0 {
			sp := bytes.IndexByte(body, ' ')
			nul := bytes.IndexByte(body, 0)
			if sp <= 0 || nul <= sp+1 || len(body) < nul+1+rawLen {
				return nil, fmt.Errorf("tree %s is malformed", short(t.oid))
			}
			mode, name := string(body[:sp]), string(body[sp+1:nul])
			oid := hex.EncodeToString(body[nul+1 : nul+1+rawLen])
			body = body[nul+1+rawLen:]
			if name == "." || name == ".." || strings.Contains(name, "/") {
				return nil, fmt.Errorf("tree %s holds an entry named %q", short(t.oid), name)
			}
			p := name
			if t.dir != "" {
				p = t.dir + "/" + name
			}
			switch mode {
			case "40000":
				queue = append(queue, pending{p, oid})
			case "160000":
				// a submodule's commit: archive writes no content for it
			case "100644", "100755", "120000":
				want[p] = treeEntry{mode: mode, oid: oid}
			default:
				return nil, fmt.Errorf("tree %s holds %s at an unknown mode %s", short(t.oid), p, mode)
			}
		}
	}
	return want, nil
}

// export writes the tree of commit, read from the repository at repo, into
// dst: `git archive` piped into `tar`, under the builder's git environment
// (isolated, replacement objects off), with every content filter the
// repository configures blanked (gitutil.FilterOverrides), so no program the
// repository names runs.
func export(ctx context.Context, repo, commit, dst string, log io.Writer) error {
	filters, err := gitutil.FilterOverrides(repo)
	if err != nil {
		return fmt.Errorf("reading the repository's content filters: %w", err)
	}
	args := append(append(gitutil.ExecPins(), filters...), "-C", repo, "archive", "--format=tar", commit)
	git := exec.CommandContext(ctx, "git", args...)
	git.Env = builderEnv()
	git.Stderr = log
	tar := exec.CommandContext(ctx, "tar", "-x", "-f", "-", "-C", dst)
	tar.Stdout, tar.Stderr = log, log
	pipe, err := git.StdoutPipe()
	if err != nil {
		return err
	}
	tar.Stdin = pipe
	if err := git.Start(); err != nil {
		return err
	}
	terr := tar.Run()
	gerr := git.Wait()
	if gerr != nil {
		return fmt.Errorf("git archive: %w", gerr)
	}
	if terr != nil {
		return fmt.Errorf("tar: %w", terr)
	}
	return nil
}

// verifyExport proves the export in dst is the verified tree want and
// nothing else: every blob is present with the content the tree records (its
// object id recomputed from the exported bytes, a symlink's from its target),
// and nothing is present that the tree does not hold. `git archive` honours
// attributes, and repository-local ones (.git/info/attributes,
// core.attributesFile) are writable by anyone who can commit in the checkout:
// export-ignore drops a file, export-subst and eol rewrite one. The reviewed
// commit is what is built, or nothing is.
func verifyExport(want map[string]treeEntry, dst string) error {
	seen := 0
	err := filepath.WalkDir(dst, func(abs string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dst, abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		e, ok := want[rel]
		if !ok {
			return fmt.Errorf("the export holds %s, which the commit does not", rel)
		}
		var body []byte
		if e.mode == "120000" {
			if d.Type()&os.ModeSymlink == 0 {
				return fmt.Errorf("the export's %s is not the symlink the commit records", rel)
			}
			target, err := os.Readlink(abs)
			if err != nil {
				return err
			}
			body = []byte(target)
		} else {
			if !d.Type().IsRegular() {
				return fmt.Errorf("the export's %s is not the regular file the commit records", rel)
			}
			if body, err = os.ReadFile(abs); err != nil {
				return err
			}
		}
		newHash, ok := objectHash(e.oid)
		if !ok {
			return fmt.Errorf("the tree records %s under an unreadable object name", rel)
		}
		h := newHash()
		fmt.Fprintf(h, "blob %d\x00", len(body))
		h.Write(body)
		if hex.EncodeToString(h.Sum(nil)) != e.oid {
			return fmt.Errorf("the export's %s differs from the commit (a repository attribute or filter changed it)", rel)
		}
		seen++
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(want) {
		var missing []string
		for p := range want {
			if _, err := os.Lstat(filepath.Join(dst, filepath.FromSlash(p))); err != nil {
				missing = append(missing, p)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("the export is missing %s, which the commit holds (a repository attribute dropped it)", strings.Join(missing, ", "))
	}
	return nil
}

// objectHash is the hash git names objects with in a repository whose
// object names are as long as name: SHA-1 for 40 hex digits, SHA-256 for 64.
func objectHash(name string) (func() hash.Hash, bool) {
	switch len(name) {
	case 40:
		return sha1.New, true
	case 64:
		return sha256.New, true
	}
	return nil, false
}

// goModVersionRe reads the `go <version>` line of go.mod, as the Makefile's
// GO_TOOLCHAIN_VERSION and scripts/pinned-toolchain.sh's caller do.
var goModVersionRe = regexp.MustCompile(`(?m)^go ([0-9][0-9.]*)\s*$`)

// goBuild builds ./cmd/abcd in src into dst with the toolchain src's go.mod
// declares (GOTOOLCHAIN=go<version>, the pin `make preflight` builds with),
// path-clean, with no VCS stamp (an export carries no .git, and a home that
// is itself a repository must not be stamped), and with the ambient
// workspace and flags neutralised so the build is the commit's alone.
func goBuild(ctx context.Context, src, dst string, log io.Writer) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return errors.New("no go toolchain is on PATH")
	}
	env := make([]string, 0, len(os.Environ())+4)
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch k {
		case "GOFLAGS", "GOWORK", "GOTOOLCHAIN", "GOOS", "GOARCH":
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "GOWORK=off", "GOFLAGS=-mod=readonly")
	if mod, err := os.ReadFile(filepath.Join(src, "go.mod")); err == nil {
		if m := goModVersionRe.FindSubmatch(mod); m != nil {
			env = append(env, "GOTOOLCHAIN=go"+string(m[1]))
		}
	}
	cmd := exec.CommandContext(ctx, goBin, "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w", "-o", dst, "./cmd/abcd")
	cmd.Dir = src
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}

// prune keeps the keepBuilds most recent commit builds in the store and
// removes the rest with their logs and markers, the log and marker of any
// commit that never built, and any export left behind by a build that was
// killed. keep is never removed.
func prune(storeAbs, keep string) {
	entries, err := os.ReadDir(storeAbs)
	if err != nil {
		return
	}
	type build struct {
		name string
		mod  time.Time
	}
	var builds []build
	for _, e := range entries {
		name := e.Name()
		fi, err := e.Info()
		if err != nil {
			continue
		}
		switch {
		case e.IsDir() && gitutil.IsFullSHA(name):
			builds = append(builds, build{name, fi.ModTime()})
		case e.IsDir() && strings.HasPrefix(name, tmpPrefix) && now().Sub(fi.ModTime()) > buildTimeout:
			_ = os.RemoveAll(filepath.Join(storeAbs, name))
		}
	}
	sort.Slice(builds, func(i, j int) bool { return builds[i].mod.After(builds[j].mod) })
	kept, held := 0, map[string]bool{keep: true}
	for _, b := range builds {
		if b.name == keep || kept < keepBuilds-1 {
			if b.name != keep {
				kept++
			}
			held[b.name] = true
			continue
		}
		_ = os.RemoveAll(filepath.Join(storeAbs, b.name))
		_ = os.Remove(filepath.Join(storeAbs, b.name+logSuffix))
		_ = os.Remove(filepath.Join(storeAbs, b.name+failedSuffix))
	}
	// A commit whose build failed leaves its log and marker and no directory,
	// so they are pruned by name: every pair but a kept build's goes.
	for _, e := range entries {
		name := e.Name()
		for _, suffix := range []string{logSuffix, failedSuffix} {
			if sha, ok := strings.CutSuffix(name, suffix); ok && gitutil.IsFullSHA(sha) && !held[sha] && e.Type().IsRegular() {
				_ = os.Remove(filepath.Join(storeAbs, name))
			}
		}
	}
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

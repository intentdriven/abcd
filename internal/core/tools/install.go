package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/cienv"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Answer is a caller's reply to the install question: yes or no, and in words
// how the reply was reached (typed at a terminal, named with a flag, no
// terminal to ask at), which the result repeats so a no is never silent.
type Answer struct {
	Yes bool
	Why string
}

// Confirm asks the person whether to run the install the explanation shows.
// The front door supplies it; a nil Confirm is a no.
type Confirm func(Explanation) Answer

// Result reports one Install: whether a compatible copy is ready for abcd to
// run, what was downloaded and how far it was checked, or why nothing was
// written and what the capability continues on.
type Result struct {
	Tool string `json:"tool"`
	// Ready reports that a copy abcd can run is in place: one already
	// installed and compatible, abcd's own recorded copy, or the one this
	// install wrote and verified.
	Ready bool `json:"ready"`
	// Ran reports that a download was started.
	Ran bool `json:"ran"`
	// Installed reports that the checked program was written into the bin
	// folder.
	Installed bool `json:"installed"`
	// Verified reports that the verify command ran the written copy and it
	// reported the pinned version.
	Verified bool `json:"verified"`
	// Declined reports that nobody said yes (a no, no confirmation, CI).
	Declined bool `json:"declined"`
	// Path is the copy abcd runs, in the tilde form, when Ready.
	Path string `json:"path,omitempty"`
	// Version is the version of that copy.
	Version string `json:"version,omitempty"`
	// Checked is how far the download was checked, when one was written.
	Checked string `json:"checked,omitempty"`
	// ChecksumOnly reports that the download was checked by its pinned sha256
	// alone, because its makers publish no signed build record.
	ChecksumOnly bool `json:"checksum_only,omitempty"`
	// Runs says which copy abcd runs, and which one the person's shell runs.
	Runs string `json:"runs,omitempty"`
	// Why is the reason nothing was written, or what failed.
	Why string `json:"why,omitempty"`
	// Output is the bounded tail of the verify command's output on a failure,
	// or its first line on success.
	Output string `json:"output,omitempty"`
	// OnDecline is the capability's standing after a no or a failure.
	OnDecline string `json:"on_decline,omitempty"`
	pinned    string
	verify    string
}

// Summary is the one line a verb reports for the result. Every path that
// leaves no copy ready ends with the capability's standing ("continuing on the
// native secret scanner"), so a no is loud rather than silent.
func (r Result) Summary() string {
	tail := ""
	if r.OnDecline != "" {
		tail = "; " + r.OnDecline
	}
	switch {
	case r.Ready && !r.Ran:
		return r.Tool + ": nothing installed; " + r.Runs
	case !r.Ran:
		return r.Tool + " not installed (" + r.Why + ")" + tail
	case !r.Installed:
		return r.Tool + ": the download of " + r.pinned + " was not installed (" + r.Why + "); nothing was written" + tail
	case !r.Verified:
		s := r.Tool + ": installed " + r.pinned + ", but the verify " + r.verify + " failed (" + r.Why + ")"
		if r.Output != "" {
			s += ": " + r.Output
		}
		return s + tail
	}
	s := r.Tool + ": installed " + r.pinned + " into " + r.Path + ", " + r.Checked + ", and verified with " + r.verify
	if r.Output != "" {
		s += " (" + r.Output + ")"
	}
	if r.Runs != "" {
		s += "; " + r.Runs
	}
	return s
}

// Installer installs the registry's tools. Every process-, network- and
// home-touching seam is a field so the whole decision path is exercised in
// tests with no network and no real home; Default is the production wiring.
type Installer struct {
	// LookPath resolves a bare program name on the operator's PATH.
	LookPath func(string) (string, error)
	// Run executes an argv whose argv[0] is already resolved, without a shell.
	Run func(context.Context, []string) ([]byte, error)
	// Getenv reads the environment (CI detection, PATH, the XDG folders).
	Getenv func(string) string
	// GOOS and GOARCH select the platform's download.
	GOOS, GOARCH string
	// Guard is the directory a program abcd runs, and the bin folder it
	// writes into, must lie outside: the repository the verb was run from.
	Guard string
	// Home is the person's home directory: the record lives below it, and
	// ~/.local/bin is the default bin folder.
	Home string
	// Get downloads one URL whole, bounded to limit bytes.
	Get func(ctx context.Context, url string, limit int64) ([]byte, error)
	// Attest checks a download's signed build record.
	Attest Attester
	// Release is the pinned release for a registered tool; the registry's by
	// default, replaced in tests to point a download at a local server.
	Release func(name string) (Release, bool)
	// PathOnly makes Plan start no program: a copy the PATH finds counts as
	// usable without its version being read. The detection pass the hooks
	// run sets it; an install never does.
	PathOnly bool
}

// Default is the production installer for a verb run from guard (its
// repository root, or its working directory outside one). Inside a test
// binary its network is the one SwapNetworkForTest installed, and by default
// none: a test that reaches a yes is refused the download rather than making
// one, so no test of another package can fetch a real release.
func Default(guard string) *Installer {
	home, _ := os.UserHomeDir()
	in := &Installer{
		LookPath: exec.LookPath, Run: runArgv, Getenv: os.Getenv,
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Guard: guard, Home: home,
		Release: registryRelease,
	}
	if testNetwork != nil || testing.Testing() {
		in.Get, in.Attest = offlineGet, offlineAttest
		if testNetwork != nil {
			in.Get, in.Attest = testNetwork.get, testNetwork.attest
		}
		return in
	}
	dl := productionFetcher(releaseHosts)
	in.Get = func(ctx context.Context, url string, limit int64) ([]byte, error) {
		return dl.get(ctx, url, limit, nil)
	}
	in.Attest = defaultAttester()
	return in
}

// errOffline is what Default's download answers inside a test binary.
var errOffline = errors.New("a test binary never reaches the network")

func offlineGet(context.Context, string, int64) ([]byte, error)     { return nil, errOffline }
func offlineAttest(context.Context, Signer, []byte) (string, error) { return "", errOffline }

type network struct {
	get    func(ctx context.Context, url string, limit int64) ([]byte, error)
	attest Attester
}

var testNetwork *network

// SwapNetworkForTest makes Default download through get and check signed
// build records through attest, for a test of another package that drives an
// install to a yes. It returns the restore.
func SwapNetworkForTest(get func(ctx context.Context, url string, limit int64) ([]byte, error), attest Attester) (restore func()) {
	prev := testNetwork
	testNetwork = &network{get: get, attest: attest}
	return func() { testNetwork = prev }
}

func registryRelease(name string) (Release, bool) {
	t, ok := registry[name]
	return t.Release, ok
}

// Install explains name as capability uses it, asks confirm, and on a yes
// installs the pinned release. It is the package-level spelling of
// Default(guard).Install.
func Install(name string, capability Capability, confirm Confirm, guard string) Result {
	return Default(guard).Install(name, capability, confirm)
}

const (
	// installTimeout bounds the download and its checks.
	installTimeout = 15 * time.Minute
	// verifyTimeout bounds the verify command.
	verifyTimeout = 30 * time.Second
	// maxOutput bounds what is kept of a program's output.
	maxOutput = 64 * 1024
	// tailLines is how much of a failure's output the summary carries.
	tailLines = 6
)

// PlanAction is what setup does about one tool.
type PlanAction string

const (
	// UseOwned: abcd's own recorded copy is in place and intact.
	UseOwned PlanAction = "use-abcd-copy"
	// UseExisting: a copy installed some other way is compatible, so it is
	// used and nothing is installed.
	UseExisting PlanAction = "use-existing"
	// InstallOwn: no compatible copy is in place, so abcd offers its own.
	InstallOwn PlanAction = "install"
	// Unavailable: abcd cannot install the tool here, and says why.
	Unavailable PlanAction = "unavailable"
)

// Copy is one copy of a tool on the machine.
type Copy struct {
	// Path is where it is, in the tilde form.
	Path string `json:"path"`
	// Version is the version it reports, or "" when it reported none.
	Version string `json:"version,omitempty"`
	// Compatible reports that Version lies in the range abcd works with.
	Compatible bool `json:"compatible"`
	// Why says why it is not compatible.
	Why string `json:"why,omitempty"`
	// Unjudged reports that its version was not read (Installer.PathOnly).
	Unjudged bool `json:"unjudged,omitempty"`
	abs      string
}

// Download is the checked download an install would make.
type Download struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	// Member is the program's path inside the archive.
	Member string `json:"member"`
	// Dest is where the program is written, in the tilde form.
	Dest string `json:"dest"`
	// Checks is how far the download is checked.
	Checks string `json:"checks"`
	// ChecksumOnly reports a tool whose makers publish no signed build record.
	ChecksumOnly bool `json:"checksum_only"`
	dest         string
}

// Plan is what setup would do about one tool, decided from the disk, the PATH
// and the version the copy already installed reports, with no network: the
// core API a setup flow lays out before it asks anything. It prints nothing.
type Plan struct {
	Tool   string     `json:"tool"`
	Action PlanAction `json:"action"`
	// Pinned is the version abcd's release fixes.
	Pinned string `json:"pinned,omitempty"`
	// Compatible is the range of versions abcd works with, as a person reads
	// it.
	Compatible string `json:"compatible,omitempty"`
	// Download is the checked download, when one is offered.
	Download *Download `json:"download,omitempty"`
	// Existing is the copy PATH finds first, when it is not abcd's own.
	Existing *Copy `json:"existing,omitempty"`
	// Owned is abcd's own recorded copy, when one is in place.
	Owned *Copy `json:"owned,omitempty"`
	// Runs says which copy abcd runs, and which the person's shell runs.
	Runs string `json:"runs,omitempty"`
	// Upgrade reports that the download replaces abcd's own copy, which
	// still works, with the version abcd's release pins.
	Upgrade bool `json:"upgrade,omitempty"`
	// Why says why the tool is unavailable.
	Why string `json:"why,omitempty"`
}

// Plan decides what to do about name. It never downloads, never writes, and
// runs only the verify command of a copy PATH already finds, to read its
// version, and not even that under PathOnly.
//
// abcd's own recorded copy is used only while its version lies in the range
// abcd works with and is the one abcd's release pins. A copy of abcd's at
// another version in the range still runs, and the pin is offered as an
// upgrade over it; one outside the range is not run, and abcd's pinned copy
// is offered in its place. Either way the download replaces only a copy that
// is still the bytes abcd wrote (Install).
func (in *Installer) Plan(name string) Plan {
	p := Plan{Tool: name}
	tool, ok := registry[name]
	rel, rok := in.release(name)
	if !ok || !rok {
		p.Action = Unavailable
		p.Why = "abcd's tool registry has no entry for " + name + ", so it holds no download it trusts; " +
			"it knows " + strings.Join(Names(), ", ")
		return p
	}
	p.Pinned = rel.Version
	p.Compatible = rel.Compatible.String()
	if e, ok := owned(in.Home, name); ok {
		c := Copy{Path: displayIn(in.Home, e.Path), Version: e.Version, Compatible: rel.Compatible.Contains(e.Version), abs: e.Path}
		if !c.Compatible {
			c.Why = "abcd's own copy is version " + e.Version + ", outside the versions abcd works with (" + p.Compatible + ")"
		}
		p.Owned = &c
		if c.Compatible {
			p.Runs = "abcd runs its own copy at " + c.Path + " (" + e.Version + ")"
			if shell := in.shellRuns(name, e.Path); shell != "" {
				p.Runs += "; in your shell, " + name + " runs " + displayIn(in.Home, shell) + ", which comes first on your PATH"
			}
			if e.Version == rel.Version {
				p.Action = UseOwned
				return p
			}
			p.Upgrade = true
		}
	}
	if p.Owned != nil {
		// abcd's own copy decides; another copy on the PATH is not judged.
	} else if prog, err := in.admit(name); err == nil && in.PathOnly {
		lex := prog
		if l, err := in.LookPath(name); err == nil {
			lex = l
		}
		p.Existing = &Copy{Path: displayIn(in.Home, lex), Unjudged: true, abs: prog}
		p.Action = UseExisting
		p.Runs = "abcd runs the copy already installed at " + p.Existing.Path + "; abcd ahoy install checks its version"
		return p
	} else if err == nil {
		c := in.judge(tool, rel, prog)
		if lex, err := in.LookPath(name); err == nil {
			c.Path = displayIn(in.Home, lex) // the spelling the person's PATH gives
		}
		p.Existing = &c
		if c.Compatible {
			p.Action = UseExisting
			p.Runs = "abcd runs the copy already installed at " + c.Path + " (" + c.Version + "), which is compatible (" +
				p.Compatible + "); nothing is installed"
			return p
		}
	}
	asset, ok := rel.assetFor(in.GOOS, in.GOARCH)
	if !ok {
		p.Action = Unavailable
		p.Why = "abcd's release holds no download of " + name + " for " + in.GOOS + "/" + in.GOARCH + "; see " + tool.Homepage
		return p
	}
	dir := userBinDir(in.getenv, in.Home)
	if dir == "" {
		p.Action = Unavailable
		p.Why = "no home directory is resolved, so there is no bin folder of your own to install " + name + " into"
		return p
	}
	dest := filepath.Join(dir, name)
	p.Download = &Download{
		Version: rel.Version, URL: asset.URL, SHA256: asset.SHA256, Member: asset.Member,
		Dest: displayIn(in.Home, dest), Checks: rel.Checks(), ChecksumOnly: rel.Signer == nil, dest: dest,
	}
	if exists(dest) && (p.Owned == nil || p.Owned.abs != dest) {
		// Not abcd's: Owned found no intact recorded copy there.
		p.Action = Unavailable
		p.Why = p.Download.Dest + " already holds a copy of " + name + " that abcd did not install, and abcd never changes another copy; " +
			"move or remove it yourself to let abcd install its own, or set XDG_BIN_HOME to another folder of yours"
		return p
	}
	p.Action = InstallOwn
	if p.Upgrade {
		// The copy that works keeps running unless the person says yes.
		p.Runs += "; after the update abcd runs " + rel.Version + " at " + p.Download.Dest
		return p
	}
	p.Runs = "after the install abcd runs its own copy at " + p.Download.Dest + " whatever comes first on your PATH"
	if p.Existing != nil {
		if in.before(filepath.Dir(p.Existing.abs), dir) {
			p.Runs += "; in your shell, " + name + " still runs " + p.Existing.Path + ", which comes first on your PATH"
		} else {
			p.Runs += "; in your shell, " + name + " runs abcd's copy, which comes before " + p.Existing.Path + " on your PATH"
		}
	} else if !in.onPath(dir) {
		p.Runs += "; " + displayIn(in.Home, dir) + " is not on your PATH, so your shell will not find " + name + " by name until you add it"
	}
	return p
}

// judge reads a copy's version through the tool's verify command and holds it
// to the compatible range.
func (in *Installer) judge(tool Tool, rel Release, prog string) Copy {
	c := Copy{Path: displayIn(in.Home, prog), abs: prog}
	ctx, cancel := context.WithTimeout(context.Background(), verifyTimeout)
	out, err := in.Run(ctx, append([]string{prog}, tool.Verify[1:]...))
	cancel()
	if err != nil {
		c.Why = "running " + strings.Join(tool.Verify, " ") + " failed (" + err.Error() + ")"
		return c
	}
	c.Version = versionIn(firstLine(out))
	switch {
	case c.Version == "":
		c.Why = "it reports no version abcd can read"
	case !rel.Compatible.Contains(c.Version):
		c.Why = "version " + c.Version + " is outside the versions abcd works with (" + rel.Compatible.String() + ")"
	default:
		c.Compatible = true
	}
	return c
}

func (in *Installer) release(name string) (Release, bool) {
	if in.Release != nil {
		return in.Release(name)
	}
	return registryRelease(name)
}

func (in *Installer) getenv(k string) string {
	if in.Getenv == nil {
		return ""
	}
	return in.Getenv(k)
}

// pathDirs is the PATH's entries, in order.
func (in *Installer) pathDirs() []string {
	return filepath.SplitList(in.getenv("PATH"))
}

// sameDir reports whether two directory spellings name one directory, after
// symlinks (a PATH entry under /var names the /private/var a lookup resolves
// to on macOS).
func sameDir(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b) || resolvedOrSelf(a) == resolvedOrSelf(b)
}

func (in *Installer) onPath(dir string) bool {
	for _, d := range in.pathDirs() {
		if sameDir(d, dir) {
			return true
		}
	}
	return false
}

// before reports whether PATH lists a ahead of b (a listed and b not counts).
func (in *Installer) before(a, b string) bool {
	for _, d := range in.pathDirs() {
		switch {
		case sameDir(d, a):
			return true
		case sameDir(d, b):
			return false
		}
	}
	return false
}

// shellRuns is what the person's shell runs for name, as PATH spells it, or
// "" when that is ours (the copy at mine) or nothing.
func (in *Installer) shellRuns(name, mine string) string {
	p, err := in.LookPath(name)
	if err != nil || resolvedOrSelf(p) == resolvedOrSelf(mine) {
		return ""
	}
	return p
}

func resolvedOrSelf(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// Install is the trust boundary. In order, and each refusal writes nothing:
//
//  1. An unknown tool is refused: nothing is composed from a name.
//  2. CI never installs, and is not asked.
//  3. The plan is made: an intact copy of abcd's own, or a compatible copy
//     installed some other way, is used and nothing is asked or installed; a
//     platform with no download, or a bin folder whose program abcd did not
//     install, is refused.
//  4. The caller's Confirm must return yes; nil is a no.
//  5. The bin folder must lie outside the guarded tree.
//  6. The download is fetched whole, bounded, over https from GitHub alone,
//     and its sha256 must equal the one abcd's release pins.
//  7. A tool whose makers sign their builds must carry a signed build record
//     that verifies against the pinned signer.
//  8. The program is read out of the archive by its exact name.
//
// Only then is the program written, atomically, at the account's own rights;
// the verify command must then report the pinned version, and the copy is
// recorded so abcd runs it by its path. A copy that fails the verify, or
// cannot be recorded, is removed again.
func (in *Installer) Install(name string, capability Capability, confirm Confirm) Result {
	tool, known := registry[name]
	e := in.explanation(name, capability)
	r := Result{Tool: name, OnDecline: e.OnDecline}
	if !known {
		r.Declined = true
		r.Why = "abcd's tool registry has no entry for " + name + ", so it holds no download it trusts; " +
			"it knows " + strings.Join(Names(), ", ")
		return r
	}
	r.verify = strings.Join(tool.Verify, " ")
	if reason, ci := cienv.Runner(in.Getenv); ci {
		r.Declined = true
		r.Why = "a CI runner (" + reason + "): abcd never installs a tool in CI"
		return r
	}
	// Stricter than the canonical detector, on purpose: it counts CI=false and
	// CI=0 as no runner, which is right for a check that only skips a warning,
	// but an install changes the machine, so any CI value refuses here.
	if v := in.getenv("CI"); v != "" {
		r.Declined = true
		r.Why = "CI is set: abcd never installs a tool in CI, whatever CI's value"
		return r
	}
	plan := in.Plan(name)
	e.Plan = &plan
	switch plan.Action {
	case UseOwned, UseExisting:
		r.Ready = true
		r.Runs = plan.Runs
		r.OnDecline = ""
		if plan.Owned != nil {
			r.Path, r.Version = plan.Owned.Path, plan.Owned.Version
		} else {
			r.Path, r.Version = plan.Existing.Path, plan.Existing.Version
		}
		return r
	case Unavailable:
		r.Why = plan.Why
		return r
	}
	dl := plan.Download
	r.pinned = dl.Version
	ans := Answer{Why: "no one was asked"}
	if confirm != nil {
		ans = confirm(e)
	}
	if !ans.Yes {
		r.Declined = true
		r.Why = ans.Why
		if r.Why == "" {
			r.Why = "the answer was no"
		}
		if plan.Upgrade {
			// abcd's own copy still works and keeps running.
			r.Ready, r.Runs, r.OnDecline = true, plan.Runs, ""
			r.Path, r.Version = plan.Owned.Path, plan.Owned.Version
		}
		return r
	}
	if in.Guard == "" || !filepath.IsAbs(in.Guard) {
		r.Why = "the bin folder cannot be judged without an absolute directory to hold it outside"
		return r
	}
	if WithinTree(dl.dest, resolvedOrSelf(dl.dest), in.Guard) || WithinTree(filepath.Dir(dl.dest), resolvedOrSelf(filepath.Dir(dl.dest)), in.Guard) {
		r.Why = "the bin folder " + filepath.Dir(dl.Dest) + " lies inside the repository this verb ran from, so a program written there " +
			"would be repository content; set XDG_BIN_HOME to a folder of yours outside it"
		return r
	}
	rel, _ := in.release(name)
	asset, _ := rel.assetFor(in.GOOS, in.GOARCH)
	r.Ran = true
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	archive, err := in.Get(ctx, asset.URL, maxArchiveBytes)
	if err != nil {
		r.Why = "the download from " + asset.URL + " failed: " + err.Error()
		return r
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != asset.SHA256 {
		r.Why = "the download's sha256 is " + got + ", but abcd's release fixes " + asset.SHA256 +
			" for it, so it is not the build abcd names"
		return r
	}
	if rel.Signer != nil {
		if in.Attest == nil {
			r.Why = "no check of its signed build record is configured for this run"
			return r
		}
		if _, err := in.Attest(ctx, *rel.Signer, sum[:]); err != nil {
			r.Why = "its signed build record does not check out: " + err.Error()
			return r
		}
	}
	prog, err := extractMember(archive, asset.Archive, asset.Member)
	if err != nil {
		r.Why = err.Error()
		return r
	}
	dir := filepath.Dir(dl.dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.Why = "could not create your bin folder " + filepath.Dir(dl.Dest) + " (" + err.Error() + "); abcd installs only into a folder " +
			"you own, so set XDG_BIN_HOME to one"
		return r
	}
	// The one file the download may replace is abcd's own recorded copy,
	// judged again here: a copy that is no longer the bytes abcd wrote is
	// someone else's, and is never changed.
	var previous []byte
	if exists(dl.dest) {
		own, mine := owned(in.Home, name)
		if plan.Owned == nil || plan.Owned.abs != dl.dest || !mine || own.Path != dl.dest {
			r.Why = dl.Dest + " appeared, or changed, while abcd was downloading, and abcd never changes a copy it did not install"
			return r
		}
		if previous, err = os.ReadFile(dl.dest); err != nil {
			r.Why = "could not keep abcd's copy at " + dl.Dest + " before replacing it (" + err.Error() + "); nothing was written"
			return r
		}
	}
	// putBack leaves the bin folder as it was: abcd's earlier copy where it
	// replaced one, nothing where it wrote a new one.
	putBack := func() {
		if previous != nil {
			_ = fsutil.WriteFileAtomic(dl.dest, previous, 0o755)
			return
		}
		_ = os.Remove(dl.dest)
	}
	if err := fsutil.WriteFileAtomic(dl.dest, prog, 0o755); err != nil {
		r.Why = "could not write " + dl.Dest + " (" + err.Error() + "); abcd installs only into a folder you own, so set " +
			"XDG_BIN_HOME to one"
		return r
	}
	r.Installed = true
	vctx, vcancel := context.WithTimeout(context.Background(), verifyTimeout)
	vout, err := in.Run(vctx, append([]string{dl.dest}, tool.Verify[1:]...))
	vcancel()
	got := versionIn(firstLine(vout))
	switch {
	case err != nil:
		r.Why = err.Error()
		r.Output = tail(vout)
	case got != dl.Version:
		r.Why = "it reports version " + got + ", not the pinned " + dl.Version
		r.Output = tail(vout)
	}
	if r.Why != "" {
		r.Why += "; abcd removed the copy it wrote"
		putBack()
		return r
	}
	progSum := sha256.Sum256(prog)
	rec, err := readRecord(in.Home)
	if err == nil {
		rec.Tools[name] = Installed{Path: dl.dest, SHA256: hex.EncodeToString(progSum[:]), Version: dl.Version,
			AssetSHA256: asset.SHA256, Checked: dl.Checks}
		err = writeRecord(in.Home, rec)
	}
	if err != nil {
		putBack()
		r.Why = "the copy works, but " + displayIn(in.Home, filepath.Join(in.Home, filepath.FromSlash(recordRel))) +
			" could not record it (" + err.Error() + "), so abcd could neither run it by its path nor remove it on uninstall; abcd removed it again"
		return r
	}
	r.Verified = true
	r.Ready = true
	r.OnDecline = ""
	r.Path = dl.Dest
	r.Version = dl.Version
	r.Checked = dl.Checks
	r.ChecksumOnly = dl.ChecksumOnly
	r.Output = firstLine(vout)
	r.Runs = "abcd runs this copy whatever comes first on your PATH"
	if shell := in.shellRuns(name, dl.dest); shell != "" {
		r.Runs += "; in your shell, " + name + " runs " + displayIn(in.Home, shell) + ", which comes first on your PATH"
	}
	return r
}

var errNotFound = errors.New("not found on PATH")

// admit resolves a bare program name on PATH and refuses a result inside the
// guarded tree. PATH is the operator's environment, which abcd trusts to locate
// git and gh everywhere else; what a checkout CAN reach is a PATH entry that
// points into it, so the result is judged as the gitleaks adapter judges a
// binary: absolute, and outside the guarded tree both lexically and after
// symlink resolution. An empty guard is refused rather than trusted.
func (in *Installer) admit(name string) (string, error) {
	p, err := in.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, errNotFound)
	}
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("%s resolves to %q, which is not an absolute path; nothing was run", name, p)
	}
	if in.Guard == "" || !filepath.IsAbs(in.Guard) {
		return "", fmt.Errorf("%s cannot be judged without an absolute directory to hold it outside; nothing was run", name)
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", fmt.Errorf("%s resolves to %q, which does not resolve; nothing was run", name, p)
	}
	if WithinTree(p, resolved, in.Guard) {
		return "", fmt.Errorf("%s resolves to %q, inside the repository this verb ran from; "+
			"a program there is repository content and is never run as an install step", name, p)
	}
	return resolved, nil
}

// WithinTree reports whether the program at p, which resolves to resolved
// after symlinks, lies inside the tree at guard, judged lexically and after
// symlink resolution on both sides. It is the one in-checkout judgement for a
// program: the installer refuses to run what it holds, and ahoy's presence
// check does not count it as installed. Both p and guard are absolute; the
// caller has already resolved p.
func WithinTree(p, resolved, guard string) bool {
	guards := []string{filepath.Clean(guard)}
	if g, err := filepath.EvalSymlinks(guard); err == nil {
		guards = append(guards, g)
	}
	fold := fsutil.CaseFoldingFS()
	for _, g := range guards {
		for _, c := range []string{filepath.Clean(p), resolved} {
			if fsutil.PathWithin(c, g, fold) {
				return true
			}
		}
	}
	return false
}

// pipeGrace is how long runArgv keeps reading a step's output once the step
// has exited or been killed. It bounds the one wait the group kill cannot end:
// a descendant that left the step's process group still holds the output pipe.
var pipeGrace = 10 * time.Second

// runArgv is the production runner: the argv is executed directly (no shell),
// stdin is the null device so a package manager cannot stop to ask, the working
// directory is the temporary directory rather than the repository, and the
// child leads its own process group so a timeout kills that group through this
// handle and nothing else.
//
// The kill reaches the step's process group, not every process the step
// started: a descendant that moves itself into another group or session
// (setsid, setpgid) is out of its reach and outlives the step. Such a
// descendant can also keep the step's output pipe open, and reading until that
// pipe closes would make the time bound unbounded, so the wait for output ends
// pipeGrace after the step itself exits or is killed (cmd.WaitDelay), and a
// step that exited cleanly while leaving such a process behind is reported as
// a failure that says so.
func runArgv(ctx context.Context, argv []string) ([]byte, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = os.TempDir()
	cmd.Stdin = nil
	var buf bytes.Buffer
	w := &capped{w: &buf, remaining: maxOutput}
	cmd.Stdout = w
	cmd.Stderr = w
	procgroup.Lead(cmd)
	cmd.WaitDelay = pipeGrace
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if errors.Is(err, exec.ErrWaitDelay) {
			err = fmt.Errorf("exited, but a process it started left its process group and still held its output "+
				"%s later; abcd stopped waiting for it, and it may still be running", pipeGrace)
		}
		return buf.Bytes(), err
	case <-ctx.Done():
		// The group is the one this child leads (Setpgid), addressed through
		// the pid this handle holds: never a pattern, never another process.
		_ = procgroup.Kill(cmd)
		<-done
		return buf.Bytes(), fmt.Errorf("did not finish within the time allowed (%w)", ctx.Err())
	}
}

// capped keeps at most remaining bytes and drops the rest without failing the
// writer, so a chatty package manager is never killed for its verbosity.
type capped struct {
	w         *bytes.Buffer
	remaining int
}

func (c *capped) Write(p []byte) (int, error) {
	n := len(p)
	if c.remaining > 0 {
		k := n
		if k > c.remaining {
			k = c.remaining
		}
		c.w.Write(p[:k])
		c.remaining -= k
	}
	return n, nil
}

// outputLines is the one place a program's output becomes result text: its
// non-blank lines, trimmed and passed through termsafe. The output is the
// program's, not abcd's, and Output carries it to every render of the result
// (the ahoy changed: and note: lines, and --json, which escapes no C1 control
// or bidi override), so an escape sequence in it never reaches a terminal.
func outputLines(out []byte) []string {
	var keep []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			keep = append(keep, termsafe.Sanitize(l))
		}
	}
	return keep
}

// tail is the last few non-blank lines of out, joined on " | ".
func tail(out []byte) string {
	keep := outputLines(out)
	if len(keep) > tailLines {
		keep = keep[len(keep)-tailLines:]
	}
	return strings.Join(keep, " | ")
}

// firstLine is the first non-blank line of out.
func firstLine(out []byte) string {
	if keep := outputLines(out); len(keep) > 0 {
		return keep[0]
	}
	return ""
}

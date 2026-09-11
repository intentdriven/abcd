// Package update implements the core of `abcd update` (itd-130 / spc-32): a
// user-invoked fetch/verify/swap for the PATH-installed binary. It is adr-38
// tier 2 — the verb's documented meaning IS the fetch; nothing here runs
// implicitly — and it completes a chosen update (tier 3): the tag is named
// before anything is replaced, the asset is verified against the same
// release's checksums.txt, and the swap is atomic in the target's directory.
// The transport is pinned: no proxy, no CA overrides from the environment,
// redirects only onto the release origin's own hosts, every hop re-checked
// under the urlguard policy. Transport-agnostic (adr-23): everything returns
// a structured Report; the CLI renders it.
package update

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/minio/selfupdate"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/urlguard"
)

// Action is the outcome class a Report carries.
type Action string

const (
	ActionSwapped Action = "swapped"
	ActionCurrent Action = "already-current"
	ActionRefused Action = "refused"
)

// Refusal is a named no: the shape that refused, why, and the remedy. Every
// refusal is loud and names its way out (the itd-130 dispatch contract).
type Refusal struct {
	Shape  string `json:"shape"`
	Detail string `json:"detail"`
	Remedy string `json:"remedy"`
}

// Ownership names the proof that the file about to be replaced is abcd's own.
// There are three, and only the first depends on anything a forge still serves
// — which is why the other two exist (iss-2609012000222546): release objects
// are deletable, and an ownership proof that rests on them stops proving the
// day they are deleted, stranding an install with no way forward.
type Ownership string

const (
	// OwnedByManifest: the file's digest appears in a published release's
	// checksums.txt. The strongest proof and the only one that also dates the
	// file, so it is tried first — but it dies with the release object.
	OwnedByManifest Ownership = "release-manifest"
	// OwnedByRunningExecutable: the file IS the executable this process is
	// running from. Nothing else can be: the code asking the question was
	// loaded out of those very bytes, so the file is abcd by construction and
	// no forge object is consulted to know it.
	OwnedByRunningExecutable Ownership = "running-executable"
	// OwnedByPathEntry: ~/.abcd/path-entry records this exact file as the copy
	// abcd installed, and the bytes still hash to what was recorded. Local,
	// written at install time by every install route, and equally independent
	// of what the forge still serves.
	OwnedByPathEntry Ownership = "path-entry-record"
)

// Prose renders the ownership proof as the half-sentence the receipt prints.
// The words live in the core beside the enum, so the CLI never invents its own
// account of why abcd was allowed to replace the file.
func (o Ownership) Prose() string {
	switch o {
	case OwnedByManifest:
		return "its digest is published in a release's checksums.txt"
	case OwnedByRunningExecutable:
		return "it is the binary running this command, so it is abcd by construction"
	case OwnedByPathEntry:
		return "~/.abcd/path-entry records it as this machine's abcd install"
	default:
		return string(o)
	}
}

// Report is the update receipt: origin, tag, digest, and what happened. It
// prints in both TTY and piped modes — silence is only ever about progress.
type Report struct {
	Origin     string    `json:"origin"`
	Tag        string    `json:"tag,omitempty"`
	Asset      string    `json:"asset,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	TargetPath string    `json:"target_path,omitempty"`
	OldVersion string    `json:"old_version,omitempty"`
	OldDigest  string    `json:"old_digest,omitempty"`
	NewVersion string    `json:"new_version,omitempty"`
	Ownership  Ownership `json:"ownership,omitempty"`
	Action     Action    `json:"action"`
	EnvIgnored []string  `json:"env_ignored,omitempty"`
	Refusal    *Refusal  `json:"refusal,omitempty"`
}

// RemedyPluginUpdate is the way out of every refusal whose cause is a binary
// the plugin install owns: the plugin-root refusal here, and the stale-binary
// note the CLI appends to an unknown command (iss-2608230943088357). One
// string, so the two surfaces cannot drift apart.
const RemedyPluginUpdate = "take a plugin update in the host (e.g. /plugin update abcd)"

// remedyReinstallOverIt is the way out of the unprovenanced-file refusal, and
// it deliberately does not begin with "remove it". The refusal it answers is
// the last thing a user with one abcd sees: deleting the file first destroys
// the only tool that could fetch a replacement, which is exactly the dead end
// iss-2609012000222546 reports. So the remedy names the reinstall route, says
// it overwrites the entry in place, and states outright that the verb cannot
// run once the file is gone.
const remedyReinstallOverIt = "reinstall OVER it rather than removing it — `abcd update` cannot run once this file is gone: " +
	"re-run the install one-liner from the README (" + releaseOrigin + "#install), which overwrites the entry in place and " +
	"records it as this machine's abcd, or run `abcd ahoy install` from a plugin session"

// brewCellarPrefixes are the resolved locations a Homebrew-installed binary
// lives under. A PREFIX test on the resolved path, never a substring match
// (the 2026-08-20 decision-log entry): a plain copy at /usr/local/bin/abcd
// resolves to itself and does not match; a brew install resolves into Cellar.
var brewCellarPrefixes = []string{
	"/opt/homebrew/Cellar/",
	"/usr/local/Cellar/",
	"/home/linuxbrew/.linuxbrew/Cellar/",
}

// osExecutable is os.Executable behind a package var, so a test can pin the
// running executable at a fixture path — the same seam ahoy keeps for the same
// reason. The shipping code never assigns it.
var osExecutable = os.Executable

// caseFoldingFS is the package's view of fsutil.CaseFoldingFS, held as a var so a
// test can provoke the case-folding branch of the Cellar-prefix match on a
// case-sensitive host — the seam launch and lifeboat use for the same reason.
var caseFoldingFS = fsutil.CaseFoldingFS

// Plan maps the resolved PATH target onto the spc-32 dispatch table: only a
// regular file outside a package manager's tree proceeds; every other shape
// is a refusal naming the mechanism that owns it.
func Plan(t ahoy.UpdateTarget) *Refusal {
	// A package-manager install is checked on the RESOLVED path first: a brew
	// install is a symlink into Cellar/, which classifies as a foreign symlink,
	// so the brew remedy has to win before the generic foreign refusal.
	// The refusal detail is a rendered success-adjacent envelope, not an error
	// value the CLI scrub ever sees, so the developer-identity home root in these
	// paths is redacted to ~ at the point it enters the string (a stock install
	// puts the target under ~/.local/bin or a ~-rooted plugin root).
	targetPath := fsutil.RedactHome(t.Path)
	resolvedPath := fsutil.RedactHome(t.ResolvedPath)
	// The prefix match folds case on a case-folding filesystem: macOS/APFS may
	// echo a case-variant Cellar path, which byte-exact would fall through to the
	// generic foreign-binary refusal and hand the operator the wrong remedy. The
	// fold routes through the canonical fsutil.FoldPath key so a case-sensitive
	// host keeps byte-exact semantics (iss-2608270908349399).
	fold := caseFoldingFS()
	for _, p := range brewCellarPrefixes {
		if t.ResolvedPath != "" && strings.HasPrefix(fsutil.FoldPath(t.ResolvedPath, fold), fsutil.FoldPath(p, fold)) {
			return &Refusal{
				Shape:  "package-manager",
				Detail: "the binary resolves into " + resolvedPath + ", which Homebrew owns; a self-update there would fight the package manager",
				Remedy: "run `brew upgrade abcd`",
			}
		}
	}
	switch t.Kind {
	case ahoy.UpdateTargetPluginRoot:
		return &Refusal{
			Shape:  string(t.Kind),
			Detail: "the binary at " + targetPath + " belongs to the plugin install; abcd update never touches a plugin root; take a plugin update in the host",
			Remedy: RemedyPluginUpdate,
		}
	case ahoy.UpdateTargetDevShim:
		return &Refusal{
			Shape:  string(t.Kind),
			Detail: "the entry at " + targetPath + " is the track-latest dev shim: it rebuilds abcd from the source tip on every call, so a release binary would be a downgrade of intent",
			Remedy: "switch modes first: `abcd ahoy install` re-pins the entry",
		}
	case ahoy.UpdateTargetDangling:
		return &Refusal{
			Shape:  string(t.Kind),
			Detail: "the entry at " + targetPath + " is an abcd-owned link whose binary is gone (a plugin update strands it)",
			Remedy: "run `abcd ahoy install` — it repoints the entry at the current plugin binary",
		}
	case ahoy.UpdateTargetForeign:
		detail := "the entry at " + targetPath + " is not something abcd owns, and abcd never clobbers a binary it does not own"
		if t.LaterOwned != "" {
			detail += "; a working abcd install sits shadowed behind it at " + fsutil.RedactHome(t.LaterOwned)
		}
		return &Refusal{Shape: string(t.Kind), Detail: detail, Remedy: "remove or rename the occupant, or see `abcd ahoy` for the install's health"}
	case ahoy.UpdateTargetFile:
		// The one shape abcd may swap: a regular file whose provenance Apply then
		// verifies against the release checksums. Proceed.
		return nil
	case ahoy.UpdateTargetAbsent:
		return &Refusal{
			Shape:  string(t.Kind),
			Detail: "no abcd was found on PATH, so there is nothing to update",
			Remedy: "install first: `abcd ahoy install` from a plugin session, or the install one-liner in the README",
		}
	default:
		// A mutating verb fails closed on an unrecognised target kind rather than
		// falling through to fetch-and-swap (unrecognized-input-never-writes).
		return &Refusal{
			Shape:  "unclassified-target",
			Detail: "the entry at " + targetPath + " is an unrecognised install shape (" + string(t.Kind) + "), and abcd update only replaces a target it can classify",
			Remedy: "see `abcd ahoy` for the install's health",
		}
	}
}

// NextStep renders the one line naming how the classified install is brought
// forward — the answer `abcd update` itself would give. The swappable shape is
// pointed at the verb; every other shape is pointed at the mechanism that owns
// its binary, taken verbatim from Plan's refusal so the remedy text lives in
// exactly one place. This is the canonical "what do I run next" primitive:
// `version --check` and every schema-too-new refusal render through it
// (itd-130 / iss-2609012111168872).
func NextStep(t ahoy.UpdateTarget) string {
	if r := Plan(t); r != nil {
		return r.Remedy
	}
	return "run `abcd update`"
}

// resolveTarget is the disk-only classification NextStepHere consults — a
// package var so a test can pin an install shape without a PATH fixture.
var resolveTarget = ahoy.ResolveUpdateTarget

// NextStepHere classifies the install that answers to `abcd` on this machine
// and renders NextStep for it. Disk only: the walk reads PATH and the plugin
// root and touches no network (adr-38), so it is safe inside an error path.
func NextStepHere() string {
	return NextStep(resolveTarget())
}

// TooNew is the canonical schema-too-new refusal: an artefact stamped with a
// schema this binary does not know yet. The frame is stable ("<what> schema
// vN; this abcd knows up to vM — update abcd: ...") and the tail names the
// command for this install shape, so the message ends in something the user
// can type rather than the verbless "upgrade" it used to end in.
func TooNew(what string, got, want int) error {
	return fmt.Errorf("%s schema v%d; this abcd knows up to v%d — update abcd: %s", what, got, want, NextStepHere())
}

// scrubbedEnv are the transport-override variables the updater refuses to
// honour: one config surface must never supply both the payload and the
// manifest it is verified against (the seam hooks/bootstrap.sh scrubs).
var scrubbedEnv = []string{
	"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY",
	"https_proxy", "http_proxy", "all_proxy",
	"SSL_CERT_FILE", "SSL_CERT_DIR",
}

// tagShape is the accepted release-tag alphabet; a tag travels into a URL
// path, so anything path-shaped refuses before any request is built.
var tagShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)

const (
	maxChecksumsBytes = 1 << 20 // 1 MiB: checksums.txt is a few hundred bytes
	maxAtomBytes      = 1 << 20 // 1 MiB: the releases atom feed
	maxAssetBytes     = 1 << 28 // 256 MiB: far above any abcd binary
	fetchTimeout      = 5 * time.Minute
	maxRedirects      = 8
)

// Updater fetches and applies releases from one pinned origin.
type Updater struct {
	origin       string // e.g. https://github.com/intentdriven/abcd — no trailing slash
	assetName    string // abcd-<goos>-<arch>
	allowHTTP    bool   // test seam: the shipping constructor never sets it
	client       *http.Client
	latest       vintage.ReleaseFetcher // nil: bare ResolveTag("") refuses
	redirectHost map[string]bool
	envIgnored   []string // proxy/CA overrides scrubbed at construction, for the receipt
	provenanceN  int      // how many recent releases to walk to prove a target file
}

// releaseOrigin is the one place releases come from — the same origin
// `version --check` and hooks/bootstrap.sh resolve against, deliberately.
const releaseOrigin = "https://github.com/intentdriven/abcd"

// NewGitHubUpdater builds the updater as it ships: the pinned origin, the
// full urlguard policy, and GitHub's own asset hosts as the only legal
// redirect targets.
func NewGitHubUpdater() *Updater {
	u := newUpdater(releaseOrigin, urlguard.BlockedIP, "abcd-"+runtime.GOOS+"-"+runtime.GOARCH, false, vintage.NewGitHubReleaseFetcher())
	u.redirectHost["objects.githubusercontent.com"] = true
	u.redirectHost["release-assets.githubusercontent.com"] = true
	return u
}

// newUpdater is the test-reachable constructor: explicit origin, address
// policy (nil allows everything — loopback test servers), asset name, and
// scheme policy, mirroring the vintage fetcher's seam.
//
// It scrubs the transport-override environment BEFORE it builds the client and
// its root pool, and before the caller performs any fetch. crypto/x509 reads
// SSL_CERT_FILE/SSL_CERT_DIR once per process (a sync.Once behind the system
// pool), so a scrub that ran after the first handshake would be too late — the
// exact ordering hooks/bootstrap.sh documents scrubbing "before any fetch".
// The root pool is then built explicitly from the scrubbed environment and
// pinned onto the client, so the pool this updater trusts never depends on a
// variable an attacker set.
func newUpdater(origin string, blocked func(net.IP) bool, assetName string, allowHTTP bool, latest vintage.ReleaseFetcher) *Updater {
	ignored := scrubEnv()
	parsed, _ := url.Parse(origin)
	hosts := map[string]bool{}
	if parsed != nil {
		hosts[parsed.Host] = true
	}
	u := &Updater{
		origin:       strings.TrimRight(origin, "/"),
		assetName:    assetName,
		allowHTTP:    allowHTTP,
		latest:       latest,
		redirectHost: hosts,
		envIgnored:   ignored,
		provenanceN:  12,
	}
	dialer := &net.Dialer{Timeout: fetchTimeout}
	if blocked != nil {
		dialer.Control = urlguard.DialControl(blocked)
	}
	// Load the root pool now, from the already-scrubbed environment, and pin it
	// explicitly so verification does not depend on the process-global lazy
	// load. A nil pool (SystemCertPool error) falls back to the platform
	// default, which is still env-free because the scrub already ran.
	var rootCAs *x509.CertPool
	if pool, err := x509.SystemCertPool(); err == nil {
		rootCAs = pool
	}
	u.client = &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			// Proxy is nil BY CONSTRUCTION: the environment never chooses
			// where this client connects.
			Proxy:               nil,
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 30 * time.Second,
			TLSClientConfig:     &tls.Config{RootCAs: rootCAs},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "https" && !u.allowHTTP {
				return errors.New("redirect left https: " + req.URL.Redacted())
			}
			if !u.redirectHost[req.URL.Host] {
				return errors.New("redirect to a host outside the release origin: " + req.URL.Host)
			}
			if blocked != nil {
				if err := urlguard.CheckHostWith(req.URL.Hostname(), blocked); err != nil {
					return fmt.Errorf("redirect refused: %w", err)
				}
			}
			return nil
		},
	}
	return u
}

// ResolveTag validates an explicit tag, or resolves `releases/latest` when
// none is given — the one discovery this verb performs, and only because the
// user typed the verb (adr-38 tier 2).
func (u *Updater) ResolveTag(requested string) (string, error) {
	if requested != "" {
		if !tagShape.MatchString(requested) {
			return "", fmt.Errorf("%q is not a release tag", requested)
		}
		return requested, nil
	}
	if u.latest == nil {
		return "", errors.New("no release resolver is configured; name a tag explicitly")
	}
	tag, err := u.latest.LatestTag()
	if err != nil {
		return "", fmt.Errorf("could not resolve the latest release: %w", err)
	}
	if !tagShape.MatchString(tag) {
		return "", fmt.Errorf("the resolved tag %q is not a release tag", tag)
	}
	return tag, nil
}

// Apply completes the chosen update: prove the target file is abcd's, download
// the tag's asset, verify it against the release's own manifest, and swap
// atomically. It never leaves a partial file: verification happens before
// anything at the target path is touched.
//
// Ownership is proven by whichever of the three proofs answers first (see
// Ownership). The published-digest proof is tried first because it also DATES
// the file — the old version is derived from the release whose manifest matches
// the on-disk bytes, never read from the running binary, which may not be the
// file being replaced. It is not, however, the only proof, and must not be:
// release objects are deletable, and when the 2026-08-30 cleanup removed every
// manifest older than v0.6.9 this refused every install it had previously
// vouched for and sent the user to delete their only abcd
// (iss-2609012000222546). The two local proofs cannot be revoked from off the
// machine; a file they carry has no derivable version, so the receipt names it
// an unpublished build and reports its digest.
func (u *Updater) Apply(target, tag string, progress io.Writer) (Report, error) {
	// TargetPath is rendered in the receipt (text and --json) and relayed by the
	// plugin into agent chat, so the home root is redacted to ~ — the report is a
	// success envelope the CLI error scrub never touches.
	rep := Report{Origin: u.origin, Tag: tag, Asset: u.assetName, TargetPath: fsutil.RedactHome(target)}
	rep.EnvIgnored = u.envIgnored

	sums, found, err := u.fetchChecksums(tag)
	if err != nil {
		return rep, fmt.Errorf("fetching %s checksums: %w", tag, err)
	}
	if !found {
		return rep, fmt.Errorf("release %s has no checksums.txt at %s", tag, u.origin)
	}
	wantHex, ok := sums[u.assetName]
	if !ok {
		return rep, fmt.Errorf("release %s publishes no %s", tag, u.assetName)
	}

	targetHex, err := fileSHA256(target)
	if err != nil {
		return rep, fmt.Errorf("reading %s: %w", target, err)
	}
	if targetHex == wantHex {
		rep.Action = ActionCurrent
		rep.Digest = wantHex
		rep.OldVersion = tag
		rep.NewVersion = tag
		rep.Ownership = OwnedByManifest
		return rep, nil
	}

	oldVer, proven, err := u.deriveTargetVersion(targetHex, tag, sums)
	if err != nil {
		return rep, err
	}
	switch {
	case proven:
		rep.Ownership = OwnedByManifest
		rep.OldVersion = oldVer
	case runsFromTarget(target):
		// The file IS the executable this process runs from, so it is abcd by
		// construction: these bytes are the ones that loaded the code asking
		// the question, and a foreign binary cannot be them. Provenance had one
		// job left — dating the file — and the manifests that would have done
		// it are gone, so the receipt reports an unpublished build by digest
		// and the swap proceeds.
		rep.Ownership = OwnedByRunningExecutable
		rep.OldDigest = targetHex
	case ahoy.IsOwnedPathCopy(target):
		// The verb was invoked from elsewhere (a plugin-root binary, a source
		// checkout) and is acting on the PATH copy: abcd's own install record
		// names this file and its bytes still match, which is an ownership
		// claim abcd wrote itself and no release deletion can revoke.
		rep.Ownership = OwnedByPathEntry
		rep.OldDigest = targetHex
	default:
		rep.Action = ActionRefused
		rep.Refusal = &Refusal{
			Shape: "unprovenanced-file",
			Detail: "the file at " + fsutil.RedactHome(target) + " matches no published release of abcd (digest " + targetHex +
				"), is not the binary running this command, and no ~/.abcd/path-entry record vouches for it, so abcd will not replace it",
			Remedy: remedyReinstallOverIt,
		}
		return rep, nil
	}

	sum, err := hex.DecodeString(wantHex)
	if err != nil {
		return rep, fmt.Errorf("release %s carries a malformed digest for %s", tag, u.assetName)
	}
	body, size, err := u.get(u.assetURL(tag, u.assetName), maxAssetBytes)
	if err != nil {
		return rep, fmt.Errorf("downloading %s %s: %w", tag, u.assetName, err)
	}
	defer body.Close()
	reader := io.Reader(body)
	if progress != nil {
		reader = &progressReader{r: body, total: size, out: progress, label: u.assetName + " " + tag}
	}
	// selfupdate verifies the stream against the checksum BEFORE the rename
	// and restores the target on failure — the atomic swap plus the Windows
	// rename dance, in the one implementation the interview signed off.
	if err := selfupdate.Apply(reader, selfupdate.Options{TargetPath: target, Checksum: sum}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return rep, fmt.Errorf("swap failed AND rollback failed — the binary at %s may be broken: %v (rollback: %v)", target, err, rerr)
		}
		return rep, fmt.Errorf("verifying %s against the release manifest: %w", u.assetName, err)
	}
	if progress != nil {
		fmt.Fprintln(progress)
	}
	rep.Action = ActionSwapped
	rep.NewVersion = tag
	rep.Digest = wantHex
	return rep, nil
}

// runsFromTarget reports whether target is the very file this process was
// executed from. It is the one ownership proof that cannot be revoked by
// anything outside this machine: the bytes at target are the bytes that were
// loaded to run this code, so the file is abcd, and no foreign binary can be
// standing there.
//
// The comparison is os.SameFile, never a string compare: os.Stat follows
// symlinks, so one inode spelled two ways — /usr/local/bin/abcd and the
// ~/.local/bin/abcd a link points at — still matches, where a path compare
// would miss it and refuse the very install this exists to rescue. Anything
// that fails to stat reports false: the refusing side, where the install
// record still gets its turn.
func runsFromTarget(target string) bool {
	exe, err := osExecutable()
	if err != nil || exe == "" || target == "" {
		return false
	}
	efi, err := os.Stat(exe)
	if err != nil {
		return false
	}
	tfi, err := os.Stat(target)
	if err != nil {
		return false
	}
	return os.SameFile(efi, tfi)
}

// deriveTargetVersion establishes that the on-disk target is a published abcd
// binary and returns the release it belongs to. It checks the install tag's
// own manifest first (the target may be another platform's asset of the same
// cut), then walks recent releases newest-first (spc-32) up to provenanceN,
// so a machine one-liner-installed at an OLD release is still proven and its
// old version reported correctly — never the running binary's version. A 404
// on the release list is "nothing to prove against" (not proven); a transport
// failure is a real error surfaced loudly. Not proven is no longer the end of
// the road: Apply falls through to the two local proofs, which is why a
// deleted release now costs the receipt a version rather than costing the user
// their install.
func (u *Updater) deriveTargetVersion(targetHex, installTag string, installSums map[string]string) (string, bool, error) {
	for _, h := range installSums {
		if h == targetHex {
			return installTag, true, nil
		}
	}
	tags, err := u.recentTags(u.provenanceN)
	if err != nil {
		return "", false, fmt.Errorf("resolving recent releases to establish provenance: %w", err)
	}
	for _, tg := range tags {
		if tg == installTag {
			continue
		}
		sums, found, err := u.fetchChecksums(tg)
		if err != nil {
			return "", false, fmt.Errorf("fetching %s checksums to establish provenance: %w", tg, err)
		}
		if !found {
			continue
		}
		for _, h := range sums {
			if h == targetHex {
				return tg, true, nil
			}
		}
	}
	return "", false, nil
}

// tagFromReleaseHref extracts <tag> from a /releases/tag/<tag> atom link.
var tagFromReleaseHref = regexp.MustCompile(`/releases/tag/([^/?#"]+)`)

// recentTags reads the release list off the origin's atom feed, newest-first.
// A missing feed (404) is an empty list, not an error — provenance simply
// cannot be established by discovery then.
func (u *Updater) recentTags(limit int) ([]string, error) {
	body, _, err := u.get(u.origin+"/releases.atom", maxAtomBytes)
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return nil, nil
		}
		return nil, err
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, maxAtomBytes))
	if err != nil {
		return nil, err
	}
	var feed struct {
		Entries []struct {
			Links []struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return nil, fmt.Errorf("parsing the release feed: %w", err)
	}
	var tags []string
	for _, e := range feed.Entries {
		for _, l := range e.Links {
			if m := tagFromReleaseHref.FindStringSubmatch(l.Href); m != nil && tagShape.MatchString(m[1]) {
				tags = append(tags, m[1])
				break
			}
		}
		if len(tags) >= limit {
			break
		}
	}
	return tags, nil
}

// scrubEnv unsets every transport-override variable, returning the names
// that were set so the receipt says what was ignored (loud, never silent).
func scrubEnv() []string {
	var ignored []string
	for _, name := range scrubbedEnv {
		if _, set := os.LookupEnv(name); set {
			ignored = append(ignored, name)
			os.Unsetenv(name)
		}
	}
	return ignored
}

func (u *Updater) assetURL(tag, name string) string {
	return u.origin + "/releases/download/" + url.PathEscape(tag) + "/" + url.PathEscape(name)
}

// fetchChecksums downloads a release's manifest. A missing release (404) is
// found=false, not an error: provenance treats it as "cannot prove", while a
// transport failure is a real error the caller surfaces loudly.
func (u *Updater) fetchChecksums(tag string) (map[string]string, bool, error) {
	body, _, err := u.get(u.assetURL(tag, "checksums.txt"), maxChecksumsBytes)
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, maxChecksumsBytes))
	if err != nil {
		return nil, false, err
	}
	sums, err := parseChecksums(raw)
	if err != nil {
		return nil, false, err
	}
	return sums, true, nil
}

type notFoundError struct{ url string }

func (e *notFoundError) Error() string { return "not found: " + e.url }

// get performs one bounded GET under the pinned transport.
func (u *Updater) get(rawURL string, limit int64) (io.ReadCloser, int64, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, err
	}
	if parsed.Scheme != "https" && !u.allowHTTP {
		return nil, 0, errors.New("the release origin must be https")
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "abcd-update")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, 0, &notFoundError{url: rawURL}
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("GET %s: %s", rawURL, resp.Status)
	}
	if resp.ContentLength > limit {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("GET %s: %d bytes exceeds the %d-byte bound", rawURL, resp.ContentLength, limit)
	}
	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(resp.Body, limit), resp.Body}, resp.ContentLength, nil
}

// parseChecksums reads a sha256sum-format manifest ("<64 hex>  <name>", one
// per line, `*` binary markers tolerated). A malformed manifest is an error:
// silently matching nothing would read as "not provenanced" and refuse for
// the wrong reason.
func parseChecksums(raw []byte) (map[string]string, error) {
	sums := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || len(fields[0]) != 64 {
			return nil, fmt.Errorf("malformed checksums line: %q", line)
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return nil, fmt.Errorf("malformed digest in checksums line: %q", line)
		}
		sums[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
	}
	if len(sums) == 0 {
		return nil, errors.New("the checksums manifest names no assets")
	}
	return sums, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// progressReader reports download progress on a TTY. Hand-rolled on purpose:
// a percentage line is not worth a dependency.
type progressReader struct {
	r     io.Reader
	total int64
	done  int64
	last  int
	out   io.Writer
	label string
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.total > 0 {
		pct := int(p.done * 100 / p.total)
		if pct != p.last {
			p.last = pct
			fmt.Fprintf(p.out, "\r  downloading %s: %3d%%", p.label, pct)
		}
	}
	return n, err
}

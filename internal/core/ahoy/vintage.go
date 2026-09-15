package ahoy

import (
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// currentVintage is the seam for reading the running binary's build vintage. A
// package var so the refusal, notice, and render paths can be exercised with an
// explicit vintage without rebuilding the test binary; production reads the real
// embedded build info.
var currentVintage = vintage.CurrentBuildVintage

// SetCurrentVintageForTest overrides the build-vintage seam and returns a
// restore func. It is a test-only seam (in the manner of this package's history
// hooks): a go-test binary is itself unstamped, so without an override every
// install path in a front-door test package would hit the itd-111
// unknown-vintage refusal. Front-door test packages, which cannot reach the
// unexported seam, install a deterministic vintage through this. Production never
// calls it — it reads the embedded build info.
func SetCurrentVintageForTest(f func() vintage.Current) (restore func()) {
	prev := currentVintage
	currentVintage = f
	return func() { currentVintage = prev }
}

// VintageSourceCheckoutTip is the VintageStatus.Source a dogfood comparison
// reports — the one reference that is ancestry-guarded and so the one whose
// staleness may claim a direction (see Staleness).
const VintageSourceCheckoutTip = "checkout tip"

// VintageStatus is the assembled vintage picture for a repo: the install mode,
// the comparator's report, and the human name of the reference compared against.
// It is the single source the `version` and `ahoy` renders, the session-start
// notice, and the install refusal all consume — one comparison, many renders.
type VintageStatus struct {
	Mode     string         // install mode ("dev (tip build)" / "pinned" / "" / shadowed)
	Report   vintage.Report // outcome + current + expected
	Source   string         // reference name: "checkout tip" / "plugin manifest pin"; "" when the current vintage is undeterminable
	RepoRoot string         // the source checkout root, when the tip was the reference
}

// Vintage assembles the running binary's vintage picture for cwd. It resolves the
// install mode and the plugin root, reads the binary's own build vintage, and
// picks the applicable disk reference: the source checkout tip when cwd is
// abcd's own checkout (the dogfood case), otherwise the plugin-cache manifest
// pin. It never touches the network.
func Vintage(cwd string) VintageStatus {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}
	pluginRoot, pluginOK := resolvePluginRoot()
	mode := detectInstallMode(pluginRoot, pluginOK)
	pinTag := ""
	if pluginOK {
		pinTag = readPinnedTag(pluginRoot)
	}
	return vintageFrom(currentVintage(), mode, abs, core.Version, pinTag)
}

// vintageFrom is the pure assembly, split from Vintage so its branch selection
// can be exercised with an explicit current vintage, a fixture checkout, and an
// explicit version/pin — no rebuild of the test binary required.
func vintageFrom(cur vintage.Current, mode, cwd, version, pinTag string) VintageStatus {
	// An undeterminable current vintage is terminal: no reference can make an
	// unknown binary fresh, so report it directly. The rebuild fix — not a disk
	// reference — is the out, so no Source is named.
	if !cur.Known {
		return VintageStatus{Mode: mode, Report: vintage.Report{Outcome: vintage.Unknown, Current: cur.Revision}}
	}
	// Dogfood: the embedded revision against the source checkout tip. The
	// provider self-verifies (by ancestry) that cwd is abcd's own checkout, so a
	// non-dogfood cwd yields Unknown here and falls through to the pinned
	// comparison below rather than reporting a spurious stale.
	if rep := vintage.Compare(cur, vintage.CheckoutTip(cwd, cur.Revision)); rep.Outcome != vintage.Unknown {
		return VintageStatus{Mode: mode, Report: rep, Source: VintageSourceCheckoutTip, RepoRoot: cwd}
	}
	// Everywhere else: the stamped version against the plugin-cache manifest pin.
	// A "dev"/empty version is itself undeterminable as a pinned vintage.
	pinCur := vintage.Current{Revision: version, Known: version != "" && version != "dev"}
	return VintageStatus{
		Mode:   mode,
		Report: vintage.Compare(pinCur, vintage.PinnedVersion(pinTag)),
		Source: "plugin manifest pin",
	}
}

// DogfoodReport is the source-checkout-tip comparison the session-start notice
// consumes. It is scoped to the dogfood case on purpose: a pinned install in a
// user's own repo must never be nagged, so IsDogfood gates the whole notice.
type DogfoodReport struct {
	IsDogfood bool            // cwd is abcd's own source checkout (the binary was built from it)
	Outcome   vintage.Outcome // Fresh (silent) / Stale / Unknown (a dirty rebuild)
	Revision  string          // the binary's embedded revision (full)
	Tip       string          // the checkout tip it should match (full)
}

// DogfoodStaleness reports whether the running binary trails the tip of the
// source checkout it was built from, for the session-start notice. It reads the
// binary's own build vintage and compares against cwd's HEAD.
func DogfoodStaleness(cwd string) DogfoodReport {
	return dogfoodStalenessFrom(currentVintage(), cwd)
}

// dogfoodStalenessFrom is the pure core, split so the fresh/stale/unknown and
// not-dogfood branches can be exercised with an explicit current vintage against
// a fixture checkout. IsDogfood is true only when the binary's embedded revision
// is contained in cwd's history — proof cwd is abcd's own checkout — so a pinned
// install elsewhere yields no notice.
func dogfoodStalenessFrom(cur vintage.Current, cwd string) DogfoodReport {
	if cur.Revision == "" {
		return DogfoodReport{} // no embedded revision: nothing to locate against a tip
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}
	head, err := gitutil.Run(abs, "rev-parse", "HEAD")
	if err != nil || head == "" {
		return DogfoodReport{}
	}
	// The binary's revision must be a commit in this checkout for the comparison
	// to be about THIS checkout at all; --is-ancestor also fixes the direction.
	if _, err := gitutil.Run(abs, "merge-base", "--is-ancestor", cur.Revision, head); err != nil {
		return DogfoodReport{}
	}
	r := DogfoodReport{IsDogfood: true, Revision: cur.Revision, Tip: head}
	switch {
	case !cur.Known:
		// A dirty rebuild atop a real commit: the revision no longer identifies
		// what is running, so the vintage is unknown and worth surfacing.
		r.Outcome = vintage.Unknown
	case cur.Revision == head:
		r.Outcome = vintage.Fresh
	default:
		r.Outcome = vintage.Stale
	}
	return r
}

// staleBinaryRefusal returns the reason `ahoy install` must refuse to run
// through this binary, or "" to proceed. It refuses in two cases (AC2 and
// design decision 7): a build vintage that cannot be determined (an unstamped
// or modified/dirty build), and a dogfood binary behind its own source tip.
// Everything else — a determinable vintage that is not a dogfood-stale binary —
// proceeds. The check is disk-only; it never touches the network.
func staleBinaryRefusal(cur vintage.Current, cwd string) string {
	if !cur.Known {
		return "the running abcd binary's vintage cannot be determined (an unstamped or modified/dirty build), so it may be applying stale install logic. Rebuild it with `make build`, or re-run with --allow-stale-binary to proceed anyway."
	}
	if rep := dogfoodStalenessFrom(cur, cwd); rep.IsDogfood && rep.Outcome == vintage.Stale {
		return fmt.Sprintf("the running abcd binary was built from commit %s but this checkout is at %s — it is behind its own source and may apply stale install logic. Rebuild it with `make build`, or re-run with --allow-stale-binary to proceed anyway.",
			shortRev(rep.Revision), shortRev(rep.Tip))
	}
	return ""
}

// DisplayVintage renders the vintage for a one-line surface: a short revision
// for a checkout-tip comparison, the version verbatim for a pinned one, and
// "unknown" when it could not be determined.
func (v VintageStatus) DisplayVintage() string {
	if v.Report.Outcome == vintage.Unknown && v.Report.Current == "" {
		return "unknown"
	}
	if v.Report.Current == "" {
		return "unknown"
	}
	if isHexSHA(v.Report.Current) {
		return shortRev(v.Report.Current)
	}
	return v.Report.Current
}

// Staleness renders the comparator verdict in the words a surface prints.
func (v VintageStatus) Staleness() string {
	switch v.Report.Outcome {
	case vintage.Fresh:
		return "up to date"
	case vintage.Stale:
		ref := v.Report.Expected
		if isHexSHA(ref) {
			ref = shortRev(ref)
		} else {
			// A version/tag reference comes from the plugin-cache manifest, which
			// is not shape-checked upstream; sanitise it like every other
			// untrusted string this repo renders to a terminal (as skew.go does).
			ref = termsafe.Sanitize(ref)
		}
		// Only the checkout-tip comparison is ancestry-guarded, so only it may
		// claim a direction. A version/pin comparison is string equality — a
		// binary newer than its pin is the same inequality read the other way —
		// so it stays non-directional ("differs from"), the caution skew.go and
		// VersionTransition already take.
		if v.Source == VintageSourceCheckoutTip {
			return "stale — behind the checkout tip (" + ref + ")"
		}
		src := v.Source
		if src == "" {
			src = "reference"
		}
		return "stale — differs from the " + src + " (" + ref + ")"
	default:
		return "unknown"
	}
}

// VersionTransition reports a version change performed since this repo was last
// set up: the last-recorded setup_version (written into .abcd/config.json by
// install) against the running binary's version. It is AC6's report only — the
// fetch that changed the binary is provisioning's job (itd-105/108), out of this
// intent. changed is false when either side is undeterminable or they match; the
// report is direction-neutral, since a repo set up by a newer binary and now run
// through an older one is the same inequality read backwards.
func VersionTransition(cwd string) (recorded, running string, changed bool) {
	return versionTransitionFrom(recordedSetupVersion(cwd), core.Version)
}

// versionTransitionFrom is the pure comparison, split so the change/no-change
// branches are testable without a fixture config or a re-stamped core.Version.
func versionTransitionFrom(recorded, running string) (from, to string, changed bool) {
	if running == "" || running == "dev" || recorded == "" {
		return recorded, running, false
	}
	return recorded, running, recorded != running
}

// recordedSetupVersion reads meta.setup_version from the repo config, or "" when
// it is absent or unreadable.
func recordedSetupVersion(cwd string) string {
	cfg, err := readConfig(cwd)
	if err != nil || cfg == nil {
		return ""
	}
	v, _ := subMap(cfg, "meta")["setup_version"].(string)
	return v
}

// readPinnedTag reads the release tag the bootstrap's provenance record pins,
// or "" when no record answers. The root-local .binary-meta wins when present —
// it is written only by the degraded per-root fetch, so it describes exactly
// this root's binary (a migrated pre-cache root included, whose binary stays
// put while the shared cache moves on). A cache-provisioned root carries no
// root-local record; for it the tag comes from the persistent data dir's
// cache/binary-meta (spc-35), reached through the hook's environment or the
// root's .data-dir stamp (pluginDataDir). The same precedence lives in
// internal/surface/cli/skew.go's readSkewMeta; the two readers should be
// consolidated if either record changes shape again.
func readPinnedTag(pluginRoot string) string {
	if tag := metaReleaseTag(filepath.Join(pluginRoot, ".binary-meta")); tag != "" {
		return tag
	}
	if data := pluginDataDir(pluginRoot).dir; data != "" {
		return metaReleaseTag(filepath.Join(data, "cache", "binary-meta"))
	}
	return ""
}

// metaReleaseTag extracts release_tag from one bootstrap-written key=value
// record, or "" when the file is absent, unreadable, or carries no tag.
func metaReleaseTag(path string) string {
	return metaField(path, "release_tag")
}

// isHexSHA reports whether s is a full 40-character hex commit SHA — the shape
// the checkout-tip comparison keys on, distinct from a version string.
func isHexSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// shortRev abbreviates a commit SHA for a one-line surface.
func shortRev(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

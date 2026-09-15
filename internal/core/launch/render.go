package launch

// render.go — the single writer of a plugin version (adr-19, adr-20).
//
// The version is an OUTPUT of the release cut, not a fact about the source. The
// working tree's `.claude-plugin/*.json` therefore stay version-absent forever,
// and the derived version is stamped into the payload's COPIES of them, here,
// at ship time. Nothing else in the repository writes a version into a manifest;
// the lockstep checker beside this file only reads.
//
// The two polarities that make that safe are encoded in lockstep.go, and each is
// asserted where it belongs. The render proves TreePublic (keys present and
// agreeing) over its OWN OUTPUT, finishing with the public check, so a stamp
// that missed a pinned location is a refusal rather than a published half-state.
// TreeDev (keys absent) over the SOURCE tree is not the render's assertion: it
// is the launch gates' — DryRun and Ship run it on every preview — and
// lockstep_repo_test.go pins it over this repository's committed manifests.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// bumpTiers are the tiers adr-20's marketplace changelog entry admits. The
// render validates against this set rather than accepting any string, because a
// misspelt tier lands in a published manifest that no later check inspects.
var bumpTiers = map[string]struct{}{"patch": {}, "minor": {}, "major": {}}

// ChangelogEntry is the marketplace plugin entry's per-release record (adr-20
// R3), minus its version.
//
// The version is deliberately NOT a field: the render writes the one derived
// version into this entry and into both version pointers from a single input, so
// the entry cannot disagree with the manifest it travels in. Everything here is
// supplied by the caller — a core that read the clock or the git HEAD itself
// would put two unpinnable inputs inside a durable release artefact.
type ChangelogEntry struct {
	// Tier is the SemVer component the cut moved: patch, minor or major.
	Tier string
	// Reason is the human sentence explaining the bump, e.g. "additive itd-67 shipped".
	Reason string
	// Date is the release date; only its calendar day is recorded.
	Date time.Time
	// SourceSHA is the commit the release was cut from.
	SourceSHA string
}

// PayloadRenderRequest is the input to RenderPayload.
type PayloadRenderRequest struct {
	// RepoRoot is the source tree the payload is cut from. It is read only.
	RepoRoot string
	// Dest is the staging directory the payload is written to. It must live
	// OUTSIDE RepoRoot and be empty or absent.
	Dest string
	// Version is the derived release version, strict SemVer with no leading "v".
	// It is an input rather than something read back out of the tree — the tree
	// has no version to read, which is the whole point of adr-19.
	Version string
	// Entry is the marketplace changelog record for this release.
	Entry ChangelogEntry
}

// PayloadRenderResult is a completed render.
type PayloadRenderResult struct {
	// Dest is the staging directory the payload was written to.
	Dest string `json:"dest"`
	// Version is the version stamped at every pinned location.
	Version string `json:"version"`
	// Bundle is the resolution the payload was written from, so a caller can
	// render the same file classification a dry-run would.
	Bundle Bundle `json:"bundle"`
	// Files counts the payload files written.
	Files int `json:"files"`
	// Manifests names the manifests that were version-stamped, repo-relative.
	Manifests []string `json:"manifests"`
	// Lockstep is the public check over the rendered payload — the proof that
	// what was written is internally consistent.
	Lockstep LockstepResult `json:"lockstep"`
	// Smoke is the light installability check over the rendered payload — the
	// proof that what was written would actually install.
	Smoke SmokeReport `json:"smoke"`
}

// ErrPayloadDrift reports that the rendered payload failed the public lockstep
// check. It is its own error because it means the render itself is wrong (a
// pinned location was missed), not that the caller's input was.
var ErrPayloadDrift = errors.New("the rendered payload failed the public lockstep check")

// ErrPayloadScanRefused reports that the payload the render would materialise
// carries a secret/PII hard-fail, an unscannable coverage gap, or was scanned by
// an unavailable scanner — the fail-closed secret/PII gate the materialising path
// must pass before any write (gh-328). It is the render's twin of the ship gate's
// scan refusal: DryRun runs the scan advisory-only and Ship (the gated path) has
// no production caller, so the live `launch ship --payload-dir` render was the
// sole materialising door that never invoked the scanner.
var ErrPayloadScanRefused = errors.New("the payload failed the secret/PII scan gate")

// ErrPayloadUninstallable reports that the rendered payload declares a surface
// it does not carry. It is distinct from ErrPayloadDrift because the fault is
// the payload's CONTENTS, not the version stamp: the manifests agree perfectly
// and the plugin still would not install.
var ErrPayloadUninstallable = errors.New("the rendered payload failed the installability smoke")

// payloadLockstep is the render's self-check over its own output. It is a
// package var for the same reason ignoreChecker is: the drift branch guards
// against a PINNED LOCATION THE RENDER FORGOT TO STAMP, and while the render and
// the checker read the same adr-20 list that state cannot be provoked from the
// outside — so substituting a checker that reports drift is the only way to
// prove the backstop refuses rather than publishes.
var payloadLockstep = CheckLockstep

// caseFoldsPaths reports whether the payload-destination gate compares paths
// case-insensitively. It is a package var for the same reason payloadLockstep
// is: the gate's fail-closed direction cannot be provoked on a case-SENSITIVE
// host, so substituting the predicate is the only way to prove the refusal
// happens — and on a case-folding host, the only way to prove two genuinely
// distinct directories are still accepted.
var caseFoldsPaths = fsutil.CaseFoldingFS

// PayloadPrecheck is everything a render refuses on WITHOUT knowing the version:
// the resolved locations, the payload the render would write, and the light
// installability verdict over it.
//
// It exists as a named result because the ship verb writes a durable release
// record (the dated CHANGELOG heading) before it has a version to stamp, and a
// render that refuses after that write leaves a release in flight that can never
// be retried. Separating the version-free half lets the caller prove the render
// will be ACCEPTED before it writes anything.
type PayloadPrecheck struct {
	// Root is the symlink-resolved source tree.
	Root string
	// Dest is the symlink-resolved staging directory. Nothing is created here.
	Dest string
	// VersionLocationPath is the absolute path to the adr-19 contract.
	VersionLocationPath string
	// PrimaryPath and PrimaryPointer are the pinned primary version location the
	// contract selected, repo-relative and RFC-6901.
	PrimaryPath, PrimaryPointer string
	// Bundle is the payload resolution the render would write from.
	Bundle Bundle
	// Smoke is the light installability tier over the RESOLVED bundle — the same
	// assertions the render makes over its written output, made early enough to
	// refuse before any durable write.
	Smoke SmokeReport
}

// PrecheckPayload resolves the release payload and runs every refusal a render
// makes that does not depend on the version.
//
// It performs ZERO writes — not even the destination directory — so a caller may
// run it speculatively and a refused cut leaves the filesystem exactly as it
// found it. RenderPayload runs it as its own first step, so the two can never
// disagree about what is refusable.
//
// It refuses when: dest overlaps repoRoot or is already populated; the
// version-location contract is unreadable or blocked (adr-19 — a blocked
// decision has no schema-valid place to write); the bundle carries a violation;
// either manifest is not in the payload at all; or the payload declares a
// surface it does not carry.
func PrecheckPayload(repoRoot, dest string) (PayloadPrecheck, error) {
	var pre PayloadPrecheck

	pre.Root = fsutil.RealExistingPath(repoRoot)
	pre.Dest = fsutil.RealExistingPath(dest)
	if pre.Dest == "" || pre.Root == "" {
		return pre, errors.New("both the repository root and the payload destination must be named")
	}
	// Containment is compared through the canonical primitive, case-folded on a
	// filesystem that folds case: RealExistingPath does not case-canonicalise
	// (Go's darwin EvalSymlinks keeps the caller's spelling for every non-symlink
	// component), so a byte-sensitive prefix test reads a case-variant
	// --payload-dir — or a case-variant $PWD-derived root — as out-of-tree and
	// lets the render stage the payload inside the working tree.
	fold := caseFoldsPaths()
	if fsutil.PathWithin(pre.Dest, pre.Root, fold) {
		return pre, errors.New("the payload destination is inside the repository — the payload is an output, not a tracked directory")
	}
	if fsutil.PathWithin(pre.Root, pre.Dest, fold) {
		return pre, errors.New("the payload destination contains the repository")
	}
	// An existing NON-directory is refused explicitly: DirHasEntries reads it as
	// "no entries", and the render's own MkdirAll would then fail after the
	// caller had already written the release record.
	if info, err := os.Lstat(pre.Dest); err == nil && !info.IsDir() {
		return pre, errors.New("the payload destination exists and is not a directory")
	}
	populated, err := fsutil.DirHasEntries(pre.Dest)
	if err != nil {
		return pre, err
	}
	if populated {
		return pre, errors.New("the payload destination is not empty — refusing to render over existing content")
	}

	// The contract says WHERE the version goes. It is read from the source tree
	// (it is a decision artefact, never shipped) and it is the only thing that
	// tells the render which manifest and pointer adr-19 selected.
	pre.VersionLocationPath = filepath.Join(pre.Root, versionLocationRelPath)
	decision, err := loadJSON(pre.VersionLocationPath)
	if err != nil {
		return pre, fmt.Errorf("version-location.json not readable: %w", err)
	}
	primaryPath, primaryPtr, verr := validateVersionLocation(decision)
	if verr != "" {
		return pre, errors.New(verr)
	}
	pre.PrimaryPath, pre.PrimaryPointer = primaryPath, primaryPtr

	bundle, err := ResolveBundle(pre.Root, nil)
	if err != nil {
		return pre, err
	}
	pre.Bundle = bundle
	if bundle.HasViolation() {
		return pre, fmt.Errorf("the payload carries %d rejected file(s); resolve them before rendering a release", len(bundle.Rejected))
	}

	// The secret/PII scan gate, run on the MATERIALISING path so the render fails
	// closed BEFORE any write — the design record pins the scan as "hard-fail
	// (absent/fail-closed, never a silent skip)", but until gh-328 the scan lived
	// only in DryRun (advisory, exit-0) and Ship (no production caller), so the
	// live `launch ship --payload-dir` render staged the payload unscanned. This
	// runs the SAME scanner over the SAME resolved bundle and refuses on exactly
	// the scan verdict wouldRefuseOn reports — no second scanner.
	scan := scanBundle(pre.Root, bundle)
	if reasons := scanRefusals(scan); len(reasons) > 0 {
		return pre, fmt.Errorf("%w: %s", ErrPayloadScanRefused, strings.Join(reasons, "; "))
	}

	included := make(map[string]struct{}, len(bundle.Included))
	for _, f := range bundle.Included {
		included[f.LogicalPath] = struct{}{}
	}
	for _, rel := range []string{primaryPath, marketplaceFile} {
		if _, ok := included[rel]; !ok {
			return pre, fmt.Errorf("%s is not in the payload — a version stamped there would never ship", rel)
		}
	}

	// The smoke over the resolved bundle asserts the same declared surface the
	// render's post-write smoke does, minus the version stamp — which adds keys
	// and names no path. So a payload that would fail installability fails HERE,
	// before a release record exists.
	pre.Smoke = SmokeLight(NewBundleTree(bundle))
	if !pre.Smoke.OK {
		return pre, fmt.Errorf("%w: %s", ErrPayloadUninstallable, strings.Join(smokeDetails(pre.Smoke), "; "))
	}
	return pre, nil
}

// RenderPayload writes the release payload to req.Dest with the derived version
// stamped into its manifests, and proves the result with CheckLockstep.
//
// It resolves the payload through the SAME ResolveBundle/LoadIncludes machinery
// the dry-run and ship gates use, so there is exactly one notion of "what
// ships"; the render adds only the version stamp on the way out. It never
// touches the source tree: every write lands under Dest.
//
// Everything refusable that does not need the version is delegated to
// PrecheckPayload, so a caller may run those refusals FIRST; on top of them this
// refuses when the version is not strict SemVer or the changelog entry is
// incomplete.
func RenderPayload(req PayloadRenderRequest) (PayloadRenderResult, error) {
	res := PayloadRenderResult{Version: req.Version}

	if !IsStrictSemver(req.Version) {
		return res, fmt.Errorf("the release version %q is not strict SemVer (major.minor.patch, no leading v)", req.Version)
	}
	if _, ok := bumpTiers[req.Entry.Tier]; !ok {
		return res, fmt.Errorf("the changelog entry's bump tier %q is not one of patch, minor, major", req.Entry.Tier)
	}
	if req.Entry.Reason == "" || req.Entry.SourceSHA == "" || req.Entry.Date.IsZero() {
		return res, errors.New("the changelog entry needs a reason, a source SHA and a date")
	}

	pre, err := PrecheckPayload(req.RepoRoot, req.Dest)
	if err != nil {
		return res, err
	}
	dest, vlPath := pre.Dest, pre.VersionLocationPath
	primaryPath, primaryPtr := pre.PrimaryPath, pre.PrimaryPointer
	bundle := pre.Bundle
	res.Dest = dest
	res.Bundle = bundle

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return res, err
	}

	for _, f := range bundle.Included {
		if err := copyPayloadFile(dest, f); err != nil {
			return res, err
		}
	}
	res.Files = len(bundle.Included)

	if err := stampPointer(dest, primaryPath, primaryPtr, req.Version); err != nil {
		return res, err
	}
	if err := stampMarketplace(dest, req); err != nil {
		return res, err
	}
	res.Manifests = []string{primaryPath, marketplaceFile}

	res.Lockstep = payloadLockstep(TreePublic, dest, vlPath)
	if !res.Lockstep.OK {
		return res, fmt.Errorf("%w: %s", ErrPayloadDrift, strings.Join(lockstepDetail(res.Lockstep), "; "))
	}

	// The render is the only step that MATERIALISES a release artefact, so it is
	// where "a missing declared path FAILS" has to bite: a dry-run previews, and
	// nothing downstream of here re-reads the payload before it ships. The smoke
	// runs over the WRITTEN payload rather than the resolved bundle so it judges
	// the exact bytes a user would install, stamped manifests included.
	res.Smoke = SmokeLight(NewDirTree(dest))
	if !res.Smoke.OK {
		return res, fmt.Errorf("%w: %s", ErrPayloadUninstallable, strings.Join(smokeDetails(res.Smoke), "; "))
	}
	return res, nil
}

// smokeDetails flattens a failed smoke into the lines a refusal quotes, so an
// operator sees WHICH declared path is missing rather than a count.
func smokeDetails(report SmokeReport) []string {
	out := make([]string, 0, len(report.Findings))
	for _, f := range report.Findings {
		out = append(out, f.Detail)
	}
	return out
}

// lockstepDetail flattens a failing lockstep result into the lines a refusal
// quotes, so an operator sees WHICH pinned location disagreed.
func lockstepDetail(res LockstepResult) []string {
	if res.Unreadable {
		return []string{res.Detail}
	}
	return res.Drifts
}

// copyPayloadFile writes one resolved bundle file under dest, carrying the git
// mode the bundle recorded so an executable hook stays executable in the
// artefact.
func copyPayloadFile(dest string, f IncludedFile) error {
	data, err := os.ReadFile(f.ResolvedPath)
	if err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if f.GitMode == "100755" {
		perm = 0o755
	}
	return fsutil.WriteFileAtomic(filepath.Join(dest, filepath.FromSlash(f.LogicalPath)), data, perm)
}

// stampMarketplace writes the two pinned marketplace locations — the published
// version and the changelog entry — from the one version input, so they cannot
// disagree with the primary or with each other.
func stampMarketplace(dest string, req PayloadRenderRequest) error {
	return editManifest(dest, marketplaceFile, func(doc any) error {
		if err := setPointer(doc, secondaryVersionPointer, req.Version); err != nil {
			return err
		}
		return setPointer(doc, secondaryChangelogPtr, map[string]any{
			"version":    req.Version,
			"tier":       req.Entry.Tier,
			"reason":     req.Entry.Reason,
			"date":       req.Entry.Date.UTC().Format("2006-01-02"),
			"source_sha": req.Entry.SourceSHA,
		})
	})
}

// stampPointer writes value at one RFC-6901 pointer in a payload manifest.
func stampPointer(dest, rel, pointer string, value any) error {
	return editManifest(dest, rel, func(doc any) error { return setPointer(doc, pointer, value) })
}

// editManifest reads a manifest from the PAYLOAD (never the source tree),
// applies mutate to the decoded document and writes it back indented. Reading
// the payload copy rather than the original is what makes "the source tree is
// never mutated" structural rather than a convention a later edit could break.
func editManifest(dest, rel string, mutate func(any) error) error {
	// rel is a payload-relative manifest path that gets WRITTEN back. Contain it
	// lexically before the join so an absolute path or a ".." traversal can never
	// stamp a file outside dest, whatever its source (gh-488).
	if !fsutil.ValidRelPath(rel) {
		return fmt.Errorf("%s: manifest path is not a contained payload-relative path", rel)
	}
	path := filepath.Join(dest, filepath.FromSlash(rel))
	doc, err := loadJSON(path)
	if err != nil {
		return fmt.Errorf("%s not readable in the payload: %w", rel, err)
	}
	if err := mutate(doc); err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o644)
}

// setPointer writes value at an RFC-6901 pointer over a decoded JSON document.
//
// It is deliberately NOT a creating writer: every container on the way to the
// final token must already exist, and an array index must already be in range.
// A manifest that lacks the shape the pinned pointers describe is a manifest
// this render does not understand, and inventing the missing structure would
// publish a plausible-looking artefact no harness reads.
func setPointer(doc any, pointer string, value any) error {
	if !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("%q is not an RFC-6901 pointer", pointer)
	}
	tokens := strings.Split(pointer, "/")[1:]
	cur := doc
	for i, raw := range tokens {
		tok := unescapePointerToken(raw)
		last := i == len(tokens)-1
		switch c := cur.(type) {
		case map[string]any:
			if last {
				c[tok] = value
				return nil
			}
			next, ok := c[tok]
			if !ok {
				return fmt.Errorf("pointer %s: no %q to descend into", pointer, tok)
			}
			cur = next
		case []any:
			idx, ok := atoiIndex(tok)
			if !ok || idx >= len(c) {
				return fmt.Errorf("pointer %s: index %q is out of range", pointer, tok)
			}
			if last {
				c[idx] = value
				return nil
			}
			cur = c[idx]
		default:
			return fmt.Errorf("pointer %s: %q is not inside an object or array", pointer, tok)
		}
	}
	return fmt.Errorf("pointer %s resolves to nothing writable", pointer)
}

package launch

// parity.go — the file-level parity diff between the payload a launch would
// publish and the payload the previous release published (itd-66 AC3, AC7;
// spc-2609201955277614 piece 1).
//
// The diff answers the operator's question "what exactly does this snapshot
// change?" per file: every path added, changed or removed, each with its
// SHA-256, so the delta is visible before promotion rather than discovered
// after it.
//
// # The baseline
//
// The baseline is the previous release's payload, read one of two ways:
//
//   - a FRESH RENDER AT THE TAG, the default: the tag is checked out into a
//     private temporary clone and resolved through the same ResolveBundle the
//     render uses, with the tag's own include config. It reads only this
//     checkout's git objects, so it is disk-only (adr-38 tier 1).
//   - the tag's RELEASE ASSET, only when the operator asks for it (adr-38 tier
//     2: the network answers an explicit ask). The published plugin archive is
//     fetched with the release's own checksums.txt and refused on any digest
//     mismatch; a release that publishes no verifiable archive falls back to the
//     render at the tag, and the report says so.
//
// No previous release is a first launch: every path is added, and the report
// says why. A baseline that cannot be read is a named refusal, never an empty
// diff, because an empty diff is the one answer that reads as "nothing changed".
//
// # What a digest is taken over
//
// Each file's digest is the SHA-256 of its bytes, with one exception: the two
// manifests the release stamps (the primary version location and the
// marketplace catalog) are compared as canonical JSON with the stamped version
// keys removed. The render re-marshals those two files when it stamps them, so
// neither their source formatting nor the version they carry is part of what a
// release changes by content; the version bump is the cut's own report. The
// digest shown for them is that normalised digest, and the report names them.
// The release archive leaves the catalog out by construction (it names the
// archive's digest), so against an archive baseline the catalog is named as not
// compared rather than reported as added.

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// ParityChange is how one path differs between the baseline and the payload.
type ParityChange string

const (
	ParityAdded   ParityChange = "added"
	ParityChanged ParityChange = "changed"
	ParityRemoved ParityChange = "removed"
)

// ParitySource names where the baseline was read from.
type ParitySource string

const (
	// ParitySourceNone — there is no previous release; every path is added.
	ParitySourceNone ParitySource = "none"
	// ParitySourceRenderAtTag — a fresh render of the payload at the tag.
	ParitySourceRenderAtTag ParitySource = "render-at-tag"
	// ParitySourceReleaseAsset — the tag's published plugin archive, verified.
	ParitySourceReleaseAsset ParitySource = "release-asset"
)

// ErrParityBaselineUnreadable is the precheck's refusal when the baseline the
// diff is measured against cannot be read.
var ErrParityBaselineUnreadable = errors.New("the payload parity baseline could not be read")

// ParityEntry is one path that differs.
type ParityEntry struct {
	Path   string       `json:"path"`
	Change ParityChange `json:"change"`
	// Digest is the payload's SHA-256 for the path; empty when removed.
	Digest string `json:"digest,omitempty"`
	// BaselineDigest is the baseline's SHA-256; empty when added.
	BaselineDigest string `json:"baseline_digest,omitempty"`
}

// ParityReport is one parity diff.
type ParityReport struct {
	// Baseline is the release tag measured against; empty for a first launch.
	Baseline string       `json:"baseline"`
	Source   ParitySource `json:"source"`
	// Fetched names every URL fetched to read the baseline, in order, so a
	// network read is never silent.
	Fetched []string `json:"fetched,omitempty"`
	// Note says why the source is what it is: a first launch, a baseline that
	// shipped no payload, or a release asset that was absent.
	Note string `json:"note,omitempty"`
	// Refused is set when the baseline could not be read; the diff is then
	// empty and RefusalReason names why.
	Refused       bool   `json:"refused"`
	RefusalReason string `json:"refusal_reason,omitempty"`
	// Entries are the paths that differ, sorted by path.
	Entries   []ParityEntry `json:"entries"`
	Added     int           `json:"added"`
	Changed   int           `json:"changed"`
	Removed   int           `json:"removed"`
	Unchanged int           `json:"unchanged"`
	// Normalised names the manifests compared with their version stamps removed.
	Normalised []string `json:"normalised,omitempty"`
	// NotCompared names payload paths the baseline cannot carry by construction.
	NotCompared []string `json:"not_compared,omitempty"`
}

// ReleaseAssetFetcher reads one named asset of a release. found is false, with
// no error, when the release or the asset does not exist; url is what was
// requested, for the report. The production fetcher lives at the front door,
// so this package never opens a connection and a test never needs one.
type ReleaseAssetFetcher interface {
	FetchReleaseAsset(tag, name string) (data []byte, url string, found bool, err error)
}

// ParityInput is the caller's half of a parity diff.
type ParityInput struct {
	// Baseline is the previous release's tag; empty means there is none.
	Baseline string
	// BaselineError is set when the caller could not resolve the baseline at
	// all (the tags could not be listed): the diff refuses with it.
	BaselineError string
	// Fetch, when set, reads the baseline from the tag's release asset first.
	// Nil keeps the diff disk-only.
	Fetch ReleaseAssetFetcher
}

// checksumsAsset is the release's own digest manifest.
const checksumsAsset = "checksums.txt"

// maxBaselineArchiveEntryBytes bounds one archive entry read into memory.
const maxBaselineArchiveEntryBytes = 64 << 20

// ValidateBaselineTag refuses a configured baseline that is not a strict
// v-prefixed release tag present in this checkout. It is the check a front
// door makes on an operator-named baseline before anything runs, so a wrong
// one errors rather than reading as a first launch.
func ValidateBaselineTag(repoRoot, tag string) error {
	if !IsStrictSemver(strings.TrimPrefix(tag, "v")) || !strings.HasPrefix(tag, "v") {
		return fmt.Errorf("the baseline %q is not a release tag (vMAJOR.MINOR.PATCH)", tag)
	}
	if _, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", "refs/tags/"+tag+"^{commit}"); err != nil {
		return fmt.Errorf("the baseline %s is not a tag in this checkout (a shallow or tagless clone holds only the tags it fetched)", tag)
	}
	return nil
}

// PayloadParity diffs the resolved payload against the previous release's.
// It never returns an error: an unreadable baseline is a refusal inside the
// report, so the preview always has a report to render.
func PayloadParity(repoRoot string, bundle Bundle, in ParityInput) ParityReport {
	rep := ParityReport{Baseline: in.Baseline, Entries: []ParityEntry{}}
	refuse := func(reason string) ParityReport {
		rep.Refused, rep.RefusalReason = true, reason
		rep.Entries = []ParityEntry{}
		rep.Added, rep.Changed, rep.Removed, rep.Unchanged = 0, 0, 0, 0
		return rep
	}
	if in.BaselineError != "" {
		return refuse("the previous release could not be resolved: " + in.BaselineError)
	}

	stamped := stampedManifests(repoRoot)
	current, err := bundleDigests(bundle, stamped)
	if err != nil {
		return refuse("the payload could not be read for the diff: " + err.Error())
	}
	for _, rel := range stampedPaths(stamped) {
		if _, ok := current[rel]; ok {
			rep.Normalised = append(rep.Normalised, rel)
		}
	}

	if in.Baseline == "" {
		rep.Source = ParitySourceNone
		rep.Note = "no previous release tag, so this is a first launch and every payload path is added"
		rep.fill(current, map[string]string{})
		return rep
	}
	if err := ValidateBaselineTag(repoRoot, in.Baseline); err != nil {
		return refuse(err.Error())
	}

	if in.Fetch != nil {
		baseline, fetched, note, err := assetDigests(repoRoot, in.Baseline, in.Fetch, stamped)
		rep.Fetched = fetched
		if err != nil {
			return refuse(fmt.Sprintf("the release asset of %s could not be used: %v", in.Baseline, err))
		}
		if baseline != nil {
			rep.Source = ParitySourceReleaseAsset
			// The archive omits the catalog by construction, so the catalog is
			// named rather than read as added.
			if _, ok := current[marketplaceFile]; ok {
				delete(current, marketplaceFile)
				rep.NotCompared = append(rep.NotCompared, marketplaceFile)
			}
			rep.fill(current, baseline)
			return rep
		}
		rep.Note = note + "; the baseline is a fresh render at the tag"
	}

	baseline, noPayload, err := renderAtTagDigests(repoRoot, in.Baseline, stamped)
	if err != nil {
		return refuse(fmt.Sprintf("the payload at %s could not be rendered: %v", in.Baseline, err))
	}
	rep.Source = ParitySourceRenderAtTag
	if noPayload {
		note := in.Baseline + " declared no launch payload, so the previous release published none and every payload path is added"
		if rep.Note != "" {
			note = rep.Note + "; " + note
		}
		rep.Note = note
	}
	rep.fill(current, baseline)
	return rep
}

// fill classifies every path of the two digest maps.
func (rep *ParityReport) fill(current, baseline map[string]string) {
	paths := map[string]struct{}{}
	for p := range current {
		paths[p] = struct{}{}
	}
	for p := range baseline {
		paths[p] = struct{}{}
	}
	for _, p := range sortedKeys(paths) {
		now, inNow := current[p]
		was, inWas := baseline[p]
		switch {
		case inNow && !inWas:
			rep.Entries = append(rep.Entries, ParityEntry{Path: p, Change: ParityAdded, Digest: now})
			rep.Added++
		case !inNow && inWas:
			rep.Entries = append(rep.Entries, ParityEntry{Path: p, Change: ParityRemoved, BaselineDigest: was})
			rep.Removed++
		case now != was:
			rep.Entries = append(rep.Entries, ParityEntry{Path: p, Change: ParityChanged, Digest: now, BaselineDigest: was})
			rep.Changed++
		default:
			rep.Unchanged++
		}
	}
}

// stampedManifests maps each manifest the release stamps to the pointers it
// stamps there. The primary location comes from the version-location contract;
// an unreadable contract leaves only the catalog, whose pointers are fixed.
func stampedManifests(repoRoot string) map[string][]string {
	out := map[string][]string{marketplaceFile: {secondaryVersionPointer, secondaryChangelogPtr}}
	if decision, err := loadJSON(filepath.Join(repoRoot, versionLocationRelPath)); err == nil {
		if primary, ptr, verr := validateVersionLocation(decision); verr == "" {
			out[primary] = append(out[primary], ptr)
		}
	}
	return out
}

// bundleDigests digests every file the resolved bundle ships.
func bundleDigests(bundle Bundle, stamped map[string][]string) (map[string]string, error) {
	out := make(map[string]string, len(bundle.Included))
	for _, f := range bundle.Included {
		data, err := os.ReadFile(f.ResolvedPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.LogicalPath, pathFreeError(err))
		}
		out[f.LogicalPath] = payloadDigest(f.LogicalPath, data, stamped)
	}
	return out, nil
}

// payloadDigest is the SHA-256 the diff compares for one file: its bytes, or,
// for a stamped manifest that parses, its canonical JSON with the stamps
// removed.
func payloadDigest(rel string, data []byte, stamped map[string][]string) string {
	if ptrs, ok := stamped[rel]; ok {
		var doc any
		if err := json.Unmarshal(data, &doc); err == nil {
			for _, p := range ptrs {
				deletePointer(doc, p)
			}
			if canon, err := json.Marshal(doc); err == nil {
				data = canon
			}
		}
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// deletePointer removes the member an RFC-6901 pointer names, when its parent
// is an object that holds it; anything else is left as it is.
func deletePointer(doc any, pointer string) {
	if !strings.HasPrefix(pointer, "/") {
		return
	}
	tokens := strings.Split(pointer, "/")[1:]
	cur := doc
	for i, raw := range tokens {
		tok := unescapePointerToken(raw)
		last := i == len(tokens)-1
		switch c := cur.(type) {
		case map[string]any:
			if last {
				delete(c, tok)
				return
			}
			cur = c[tok]
		case []any:
			idx, ok := atoiIndex(tok)
			if !ok || idx >= len(c) || last {
				return
			}
			cur = c[idx]
		default:
			return
		}
	}
}

// renderAtTagDigests renders the payload as it stood at tag, in a private
// temporary clone it removes, and digests it. noPayload reports a tag that
// declared no launch payload, which is an empty previous release.
func renderAtTagDigests(repoRoot, tag string, stamped map[string][]string) (map[string]string, bool, error) {
	tmp, err := os.MkdirTemp("", "abcd-parity-")
	if err != nil {
		return nil, false, pathFreeError(err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	tree := filepath.Join(tmp, "tree")
	// --shared reads this checkout's objects through an alternate and writes
	// nothing into it; --template= keeps a template directory's hooks out of the
	// clone, and the isolated git forces hooks off for the checkout besides.
	if _, err := gitutil.Run(tmp, "clone", "--quiet", "--shared", "--no-checkout", "--template=", "--", repoRoot, "tree"); err != nil {
		return nil, false, errors.New("a private clone of this checkout could not be made: " + scrubTemp(err, tmp))
	}
	if _, err := gitutil.Run(tree, "-c", "advice.detachedHead=false", "checkout", "--quiet", "--detach", "refs/tags/"+tag+"^{commit}"); err != nil {
		return nil, false, errors.New("the tag could not be checked out: " + scrubTemp(err, tmp))
	}
	bundle, err := ResolveBundle(tree, nil)
	if errors.Is(err, ErrNoLaunchPayload) {
		return map[string]string{}, true, nil
	}
	if err != nil {
		return nil, false, errors.New(scrubTemp(err, tmp))
	}
	digests, err := bundleDigests(bundle, stamped)
	return digests, false, err
}

// scrubTemp keeps a private temporary path out of a reason an operator reads.
func scrubTemp(err error, tmp string) string {
	return strings.ReplaceAll(err.Error(), tmp, "<temp>")
}

// assetDigests reads the baseline from the tag's published plugin archive,
// verified against the release's checksums.txt. A nil map with a nil error
// means the release publishes no verifiable archive, and note says so; an error
// means the release claims an archive that cannot be read or verified.
func assetDigests(repoRoot, tag string, fetch ReleaseAssetFetcher, stamped map[string][]string) (map[string]string, []string, string, error) {
	var fetched []string
	name, err := pluginName(repoRoot)
	if err != nil {
		return nil, nil, "", err
	}
	archive := PluginArchiveName(name, strings.TrimPrefix(tag, "v"))

	raw, url, found, err := fetch.FetchReleaseAsset(tag, checksumsAsset)
	fetched = append(fetched, url)
	if err != nil {
		return nil, fetched, "", fmt.Errorf("fetching %s: %w", checksumsAsset, err)
	}
	if !found {
		return nil, fetched, tag + " publishes no " + checksumsAsset + ", so it has no verifiable payload archive", nil
	}
	sums, err := parseChecksumManifest(raw)
	if err != nil {
		return nil, fetched, "", err
	}
	want, ok := sums[archive]
	if !ok {
		return nil, fetched, tag + "'s " + checksumsAsset + " names no " + archive, nil
	}

	data, url, found, err := fetch.FetchReleaseAsset(tag, archive)
	fetched = append(fetched, url)
	if err != nil {
		return nil, fetched, "", fmt.Errorf("fetching %s: %w", archive, err)
	}
	if !found {
		return nil, fetched, "", fmt.Errorf("%s names %s, and the release does not serve it", checksumsAsset, archive)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != want {
		return nil, fetched, "", fmt.Errorf("%s fails its checksum: the release publishes %s, the fetched bytes are %s", archive, want, got)
	}
	digests, err := archiveDigests(data, stamped)
	if err != nil {
		return nil, fetched, "", fmt.Errorf("%s: %w", archive, err)
	}
	return digests, fetched, "", nil
}

// archiveDigests digests every file entry of a verified release archive.
func archiveDigests(data []byte, stamped map[string][]string) (map[string]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("not a readable zip: %w", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if !fsutil.ValidRelPath(f.Name) {
			return nil, fmt.Errorf("the archive holds an entry outside the payload: %q", f.Name)
		}
		if f.UncompressedSize64 > maxBaselineArchiveEntryBytes {
			return nil, fmt.Errorf("the archive entry %s is larger than %d bytes", f.Name, maxBaselineArchiveEntryBytes)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		body, err := io.ReadAll(io.LimitReader(rc, maxBaselineArchiveEntryBytes+1))
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		if len(body) > maxBaselineArchiveEntryBytes {
			return nil, fmt.Errorf("the archive entry %s is larger than %d bytes", f.Name, maxBaselineArchiveEntryBytes)
		}
		if _, dup := out[f.Name]; dup {
			return nil, fmt.Errorf("the archive holds %s twice", f.Name)
		}
		out[f.Name] = payloadDigest(f.Name, body, stamped)
	}
	return out, nil
}

// parseChecksumManifest reads a sha256sum-format manifest ("<64 hex>  <name>"
// per line, a `*` binary marker tolerated). A malformed manifest is an error:
// matching nothing would read as "no archive published" and fall back quietly.
func parseChecksumManifest(raw []byte) (map[string]string, error) {
	sums := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !sha256HexRe.MatchString(fields[0]) {
			return nil, fmt.Errorf("%s holds a malformed line: %q", checksumsAsset, line)
		}
		sums[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
	}
	if len(sums) == 0 {
		return nil, fmt.Errorf("%s names no assets", checksumsAsset)
	}
	return sums, nil
}

// stampedPaths returns the stamped manifests' paths in order.
func stampedPaths(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// parityRefusals is the parity slice of what a launch would refuse on.
func parityRefusals(rep *ParityReport) []string {
	if rep == nil || !rep.Refused {
		return nil
	}
	return []string{"payload parity: " + rep.RefusalReason}
}

// parityGate is the parity diff's gate row.
func parityGate(rep *ParityReport) GateSummary {
	g := GateSummary{Name: "payload-parity", Status: "ran", Tier: TierHardFail}
	switch {
	case rep.Refused:
		g.Detail = "refused: " + rep.RefusalReason
	case rep.Source == ParitySourceNone:
		g.Detail = fmt.Sprintf("first launch: %d added", rep.Added)
	default:
		g.Detail = fmt.Sprintf("against %s (%s): %d added, %d changed, %d removed, %d unchanged",
			rep.Baseline, rep.Source, rep.Added, rep.Changed, rep.Removed, rep.Unchanged)
	}
	return g
}

// Markdown renders the diff as the pre-flight report's parity section: the
// baseline and where it came from, then every path that differs with its
// digests.
func (rep ParityReport) Markdown() string {
	var b strings.Builder
	b.WriteString("\n## Payload parity\n\n")
	baseline := rep.Baseline
	if baseline == "" {
		baseline = "(none)"
	}
	fmt.Fprintf(&b, "- baseline: %s (%s)\n", baseline, rep.Source)
	for _, u := range rep.Fetched {
		fmt.Fprintf(&b, "- fetched: %s\n", u)
	}
	if rep.Note != "" {
		fmt.Fprintf(&b, "- note: %s\n", rep.Note)
	}
	if rep.Refused {
		fmt.Fprintf(&b, "- **refused**: %s\n", rep.RefusalReason)
		return b.String()
	}
	fmt.Fprintf(&b, "- %d added, %d changed, %d removed, %d unchanged\n", rep.Added, rep.Changed, rep.Removed, rep.Unchanged)
	if len(rep.Normalised) > 0 {
		fmt.Fprintf(&b, "- compared with their version stamps removed: %s\n", strings.Join(rep.Normalised, ", "))
	}
	if len(rep.NotCompared) > 0 {
		fmt.Fprintf(&b, "- not compared (the release archive omits them by construction): %s\n", strings.Join(rep.NotCompared, ", "))
	}
	if len(rep.Entries) == 0 {
		return b.String()
	}
	b.WriteString("\n| Change | Path | SHA-256 | Baseline SHA-256 |\n|---|---|---|---|\n")
	for _, e := range rep.Entries {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", e.Change, strings.ReplaceAll(e.Path, "|", "\\|"), orDash(e.Digest), orDash(e.BaselineDigest))
	}
	return b.String()
}

package lifeboat

// synthesis_review.go is the LIFEBOAT-REVIEW synthesis seam (Agent A2, M6/itd-88):
// `disembark review <lifeboat-dir> <source-repo>`. It audits an already-PACKED
// lifeboat and writes a registered verdict + cited findings to
// review/review-<manifest12>.json (+ .md), a post-pack mutable artifact kept out of
// manifest_sha256 (mirroring graveyard/lessons.json).
//
// Dual-mode single entrypoint (mirroring IngestLessons):
//
//   - DETERMINISTIC (raw == nil): the verdict is a mechanical, pure mapping over
//     VerifyManifest + the packed coverage summary — no model, no wall-clock, and
//     the source repo's CONTENT is never read (it is gated as a real dir only, so
//     the audit stays deterministic and safe even when the source is gone). The
//     inputs are the lifeboat's own sealed files.
//   - DELEGATED (raw != nil): untrusted host/model JSON is validated behind the
//     same guards as an intent verdict — DisallowUnknownFields, a three-branch
//     schema gate, a mode gate, a semver prompt_version, an enum-membership verdict
//     gate (an out-of-enum verdict is a whole-payload refusal, like intent's
//     out-of-enum criterion), and per-finding cite-or-be-dropped over the packed
//     path set. The trusted attestation fields (source_name, manifest_sha256,
//     manifest_verified, coverage) are always stamped by the core, never taken from
//     the payload.
//
// The artefact filename is manifest-derived — review-<manifest12>.json where
// manifest12 = shortHex(prov.ManifestSHA256) — so a deterministic run then a
// delegated run write the SAME file (clean replacement, no timestamp twin, no
// wall-clock anywhere).
//
// A false ManifestVerified is a MAJOR_RETHINK verdict INPUT, never a fatal error:
// a tampered lifeboat still yields a written audit recording the failure (exit 0).
//
// See the M6 design record (§4.3, §5, §6) and synthesis_types.go for the shapes.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// reviewArtefactDir is where the verdict artefact lives; legacyReviewDir is the
// pre-spc-30 home, swept on re-run (see removeLegacyReviewArtefact).
const (
	reviewArtefactDir = "review"
	legacyReviewDir   = "audit"
)

// removeLegacyReviewArtefact deletes the pre-spc-30 audit/oracle-<manifest12>
// pair for exactly this manifest, then removes audit/ if it is left empty.
//
// The scoping comes from the ROOT, not from the name construction. manifest12
// is 12-hex so the two names cannot match another manifest's files, and
// Root.Remove refuses a non-empty directory so an unrelated file keeps audit/
// alive — but neither fact contains the DELETE: a lifeboat is untrusted input,
// and a symlinked audit/ redirects a bare os.Remove anywhere the invoking user
// can write. Every other mutation in this file already goes through the
// lifeboat's os.Root; this one must too, or it is the one unlink that escapes
// the boundary the rest of the package defends.
//
// Errors are deliberately swallowed: the review artefact is already written and
// durable, the removal is hygiene, and a read-only or absent legacy dir is the
// normal case — failing the run over it would be worse.
func removeLegacyReviewArtefact(root *os.Root, manifest12 string) {
	for _, ext := range []string{".json", ".md"} {
		_ = root.Remove(path.Join(legacyReviewDir, "oracle-"+manifest12+ext))
	}
	_ = root.Remove(legacyReviewDir) // refuses a non-empty directory
}

// ReviewLifeboat audits the packed lifeboat at lifeboatDir against sourceRepo. When
// raw is nil it runs the deterministic mapping; when raw carries a payload it
// validates the delegated audit. It is transport-agnostic (returns an ReviewResult,
// never prints) and fails closed on structural problems; a manifest failure is a
// verdict input, not an error. The audit is ALWAYS written on success.
func ReviewLifeboat(lifeboatDir, sourceRepo string, raw []byte) (ReviewResult, error) {
	// 1. Gate the lifeboat (real dir + parseable header + schema) and read the
	//    immutable provenance header — needed to name the audit file.
	abs, prov, err := gateSynthLifeboat(lifeboatDir)
	if err != nil {
		return ReviewResult{}, err
	}
	// Gate the source repo as a real dir. Its CONTENT is never read — this keeps the
	// audit deterministic and safe when the source is gone. A symlinked or absent
	// source is structural (mirrors embark's target gate).
	srcAbs, err := filepath.Abs(sourceRepo)
	if err != nil {
		return ReviewResult{}, err
	}
	if !fsutil.IsRealDir(srcAbs) {
		return ReviewResult{}, fmt.Errorf("source %s is not a directory", filepath.Base(srcAbs))
	}
	sourceName := sanitize(filepath.Base(srcAbs))

	// 2. Manifest attestation and packed coverage summary — the trusted inputs the
	//    core owns in BOTH modes. VerifyManifest is the seal check; a false result
	//    is a verdict input, never fatal.
	manifestVerified := VerifyManifest(abs) == nil
	cov := readPackedCoverage(abs)
	coveragePresent := cov.Present && !cov.Degraded

	manifest12 := shortHex(prov.ManifestSHA256)
	audit := ReviewArtefact{
		SchemaVersion: ReviewSchemaVersion,
		SourceName:    sourceName,
		// manifest_sha256 comes verbatim from the reviewed lifeboat's untrusted
		// _provenance.json (readProvenance is a bare Unmarshal) and is stamped into
		// the durable review .md. sanitize masks terminal-escape runes but leaves a
		// CommonMark HTML-block opener ('<script', '<!--', …) intact, so a hostile
		// header could hide or forge the attestation when the record is rendered.
		// Route it through the file-write cleaner, as the delegated findings already
		// are (GH #325); a legitimate hex sha has no '<' and is unchanged.
		ManifestSHA256:   cleanSynthProse(prov.ManifestSHA256),
		ManifestVerified: manifestVerified,
		Coverage:         cov.Summary,
	}

	res := ReviewResult{LifeboatDir: abs}

	if raw == nil {
		// --- deterministic mode -------------------------------------------------
		audit.Mode = ModeDeterministic
		audit.Verdict = deterministicVerdict(manifestVerified, coveragePresent, cov.Summary)
		audit.Findings = deterministicFindings(manifestVerified, coveragePresent, cov.Summary, prov.SourceName, sourceName)
		res.Mode = ModeDeterministic
	} else {
		// --- delegated mode -----------------------------------------------------
		verdict, findings, drops, err := validateReview(abs, raw)
		if err != nil {
			return ReviewResult{}, err
		}
		audit.Mode = ModeDelegated
		audit.PromptVersion = drops.promptVersion
		audit.Verdict = verdict
		audit.Findings = findings
		res.Mode = ModeDelegated
		res.Dropped = drops.count
		res.Drops = drops.drops
	}

	// 3. Write the audit json + md into the contained lifeboat. The filename is the
	//    validated 12-hex manifest prefix (shortHex filters to [0-9a-f]), so it can
	//    never build a path escaping audit/. Same filename on every re-run → clean
	//    replacement, no stale twin.
	data, err := marshalSynth(audit)
	if err != nil {
		return ReviewResult{}, err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return ReviewResult{}, err
	}
	defer root.Close()

	jsonRel := path.Join(reviewArtefactDir, "review-"+manifest12+".json")
	mdRel := path.Join(reviewArtefactDir, "review-"+manifest12+".md")
	if err := writeIntoLifeboat(root, abs, jsonRel, data); err != nil {
		return ReviewResult{}, err
	}
	if err := writeIntoLifeboat(root, abs, mdRel, []byte(renderReviewMD(audit))); err != nil {
		return ReviewResult{}, err
	}
	// Clean replacement across the spc-30 rename: a lifeboat reviewed before it
	// carries the pre-rename pair under audit/. Remove THIS manifest's stale
	// pair (never another manifest's, and never an unrelated file), then prune
	// audit/ if that emptied it. Best-effort by design — the review is written
	// and valid either way, so a removal failure must not fail the run.
	removeLegacyReviewArtefact(root, manifest12)

	res.Verdict = audit.Verdict
	res.Written = len(audit.Findings)
	res.ReviewPath = jsonRel
	res.RenderPath = mdRel
	return res, nil
}

// deterministicVerdict is the pinned, pure mapping (design §4.3): a failed manifest
// dominates (not shippable at all); else absent/unusable coverage cannot be
// attested; else more blank than grounded sections is too thin to ship; else SHIP.
func deterministicVerdict(manifestVerified, coveragePresent bool, s Summary) ReviewVerdict {
	switch {
	case !manifestVerified:
		return VerdictMajorRethink
	case !coveragePresent:
		return VerdictNeedsWork
	case s.Blank > s.Grounded:
		return VerdictNeedsWork
	default:
		return VerdictShip
	}
}

// deterministicFindings emits the fixed-order, evidence-only findings that back the
// deterministic verdict. Each cites a packed lifeboat path (or the provenance
// header for the manifest/identity findings); deterministic entries are trusted
// descriptive pointers and are never dropped. Bounded at maxReviewFindings as
// defence in depth shared with the delegated path.
func deterministicFindings(manifestVerified, coveragePresent bool, s Summary, provName, sourceName string) []ReviewFinding {
	var out []ReviewFinding
	if !manifestVerified {
		out = append(out, ReviewFinding{
			ID: "fnd-manifest", Severity: "blocker",
			Finding:  "the on-disk tree does not match the sealed manifest; the lifeboat does not faithfully carry the record",
			Evidence: []string{ProvenanceName},
		})
	}
	if !coveragePresent {
		out = append(out, ReviewFinding{
			ID: "fnd-coverage-missing", Severity: "warning",
			Finding:  "coverage.json is absent or unreadable; the lifeboat's brief coverage cannot be attested",
			Evidence: []string{"coverage.json"},
		})
	} else if s.Blank > s.Grounded {
		out = append(out, ReviewFinding{
			ID: "fnd-coverage-thin", Severity: "warning",
			Finding:  fmt.Sprintf("%d sections are blank against %d grounded; the record is thin", s.Blank, s.Grounded),
			Evidence: []string{"coverage.json"},
		})
	}
	if provName != sourceName {
		out = append(out, ReviewFinding{
			ID: "fnd-source-name", Severity: "info",
			// provName is the untrusted _provenance.json source_name, embedded raw in
			// deterministic finding prose that lands in the durable review .md. Its
			// delegated siblings are cleaned with cleanSynthProse; this one used
			// sanitize, which leaves a CommonMark HTML-block opener intact (GH #325).
			// %q-quoting does not defang '<script>'. Route it through the same
			// file-write cleaner so it can neither open nor close an HTML construct.
			Finding: fmt.Sprintf("the lifeboat's recorded source name %q differs from the audited source %q",
				cleanSynthProse(provName), sourceName),
			Evidence: []string{ProvenanceName},
		})
	}
	if len(out) > maxReviewFindings {
		out = out[:maxReviewFindings]
	}
	return out
}

// reviewDropReport carries the delegated validator's per-entry outcome back to the
// result without widening the function signature.
type reviewDropReport struct {
	count         int
	drops         []ReviewFindingDrop
	promptVersion string
}

// validateReview parses and fully validates an untrusted delegated audit payload
// (design §6). It fails closed on any structural problem (returns an error, nothing
// written) and drops — never fatally — a per-finding problem. It returns the
// membership-checked verdict, the surviving findings (sorted by id), and the drop
// report. The verdict is the headline of the audit: an out-of-enum verdict is a
// whole-payload refusal, mirroring intent's out-of-enum criterion.
func validateReview(abs string, raw []byte) (ReviewVerdict, []ReviewFinding, reviewDropReport, error) {
	var rep reviewDropReport
	if len(raw) > maxSynthesisBytes {
		return "", nil, rep, fmt.Errorf("review payload exceeds the %d-byte cap", maxSynthesisBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields() // reject smuggled extra fields
	var in ReviewArtefact
	if err := dec.Decode(&in); err != nil {
		return "", nil, rep, fmt.Errorf("malformed review JSON: %v", err)
	}
	// Three-branch schema gate (mirrors IngestLessons).
	if in.SchemaVersion == 0 {
		return "", nil, rep, errors.New("review payload is missing schema_version")
	}
	if in.SchemaVersion > ReviewSchemaVersion {
		return "", nil, rep, update.TooNew("review",
			in.SchemaVersion, ReviewSchemaVersion)
	}
	if in.SchemaVersion != ReviewSchemaVersion {
		return "", nil, rep, fmt.Errorf("unsupported review schema_version %d", in.SchemaVersion)
	}
	// Mode gate: a delegated payload must not claim deterministic. An absent mode is
	// allowed (the core stamps delegated on write regardless).
	if in.Mode != "" && in.Mode != ModeDelegated {
		return "", nil, rep, fmt.Errorf("a delegated payload must not claim mode %q", in.Mode)
	}
	// prompt_version gate: required and semver-shaped in delegated mode.
	if !promptVersionRe.MatchString(in.PromptVersion) {
		return "", nil, rep, fmt.Errorf("prompt_version %q is not semver-shaped", in.PromptVersion)
	}
	rep.promptVersion = in.PromptVersion
	// Whole-payload verdict membership gate.
	if !in.Verdict.Valid() {
		return "", nil, rep, fmt.Errorf("out-of-enum verdict %q", in.Verdict)
	}
	if len(in.Findings) > maxReviewFindings {
		return "", nil, rep, fmt.Errorf("too many findings (%d > %d)", len(in.Findings), maxReviewFindings)
	}

	// Build the packed path set once — the live set a review finding must cite.
	root, err := os.OpenRoot(abs)
	if err != nil {
		return "", nil, rep, err
	}
	defer root.Close()
	paths, err := buildLifeboatPathSet(root, reviewOwnOutput)
	if err != nil {
		return "", nil, rep, err
	}

	// Per-finding validation, drop-not-fatal.
	var out []ReviewFinding
	seen := map[string]bool{}
	for _, f := range in.Findings {
		drop := func(reason string) {
			rep.count++
			rep.drops = append(rep.drops, ReviewFindingDrop{ID: f.ID, Reason: reason})
		}
		if len(f.ID) > maxSynthIDLen || !fndIDRe.MatchString(f.ID) {
			drop("malformed finding id")
			continue
		}
		if seen[f.ID] {
			drop("duplicate finding id")
			continue
		}
		refs := filterSynthEvidence(f.Evidence, paths)
		if len(refs) == 0 {
			drop("no valid evidence refs")
			continue
		}
		clean := cleanSynthProse(f.Finding)
		if clean == "" {
			drop("empty finding prose")
			continue
		}
		// First-wins dedup marks the id seen only after the entry SURVIVES every
		// check (the IngestLessons note), so a dropped first occurrence cannot poison
		// a later fully-valid duplicate.
		seen[f.ID] = true
		out = append(out, ReviewFinding{
			ID:       f.ID,
			Severity: cleanSynthProse(f.Severity), // optional, never a drop reason
			Finding:  clean,
			Evidence: refs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return in.Verdict, out, rep, nil
}

// Render is the deterministic, sanitised human summary of an audit (the transport
// render; the surface prints it).
func (r ReviewResult) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "review of %s\n", sanitize(r.LifeboatDir))
	fmt.Fprintf(&b, "  verdict:  %s\n", r.Verdict)
	fmt.Fprintf(&b, "  mode:     %s\n", r.Mode)
	fmt.Fprintf(&b, "  findings: %d\n", r.Written)
	fmt.Fprintf(&b, "  dropped:  %d\n", r.Dropped)
	if r.ReviewPath != "" {
		fmt.Fprintf(&b, "  review:   %s\n", sanitize(r.ReviewPath))
	}
	for _, d := range r.Drops {
		fmt.Fprintf(&b, "    - %s (%s)\n", sanitize(d.ID), sanitize(d.Reason))
	}
	return b.String()
}

// renderReviewMD renders the on-disk review/review-<manifest12>.md — a
// deterministic, sanitised human view of the audit (no wall-clock, fixed order).
func renderReviewMD(a ReviewArtefact) string {
	var b strings.Builder
	b.WriteString("# Lifeboat review\n\n")
	fmt.Fprintf(&b, "- verdict: %s\n", a.Verdict)
	fmt.Fprintf(&b, "- mode: %s\n", a.Mode)
	if a.PromptVersion != "" {
		fmt.Fprintf(&b, "- prompt_version: %s\n", sanitize(a.PromptVersion))
	}
	fmt.Fprintf(&b, "- source: %s\n", sanitize(a.SourceName))
	verified := "not verified"
	if a.ManifestVerified {
		verified = "verified"
	}
	fmt.Fprintf(&b, "- manifest: %s (%s)\n", sanitize(a.ManifestSHA256), verified)
	fmt.Fprintf(&b, "- coverage: grounded %d · partial %d · blank %d\n",
		a.Coverage.Grounded, a.Coverage.Partial, a.Coverage.Blank)
	b.WriteString("\n## Findings\n\n")
	if len(a.Findings) == 0 {
		b.WriteString("(none)\n")
		return b.String()
	}
	for _, f := range a.Findings {
		sev := ""
		if f.Severity != "" {
			sev = "[" + sanitize(f.Severity) + "] "
		}
		fmt.Fprintf(&b, "- %s%s — %s", sev, sanitize(f.ID), sanitize(f.Finding))
		if len(f.Evidence) > 0 {
			fmt.Fprintf(&b, " (evidence: %s)", strings.Join(sanitizeAll(f.Evidence), ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

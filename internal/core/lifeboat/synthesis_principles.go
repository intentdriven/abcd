package lifeboat

// synthesis_principles.go — the PRINCIPLES seam of M6 (itd-88), plus the shared
// synthesis helpers Agent A2's oracle reuses (same package, a build-order dep).
//
// SynthesizePrinciples operates POST-PACK on an already-sealed lifeboat. It has
// two modes on one entrypoint (mirroring IngestLessons):
//
//   - Deterministic (raw == nil): an evidence-only fallback built from the packed
//     lifeboat's own ADRs. Each ADR that carries a Decision/Consequences section
//     surfaces its first bullet as a cited principle (prn-<adr-id>), confidence
//     high — the record's own asserted words, never interpretation. Byte-identical
//     across re-runs of an unchanged lifeboat.
//   - Delegated (raw != nil): validated untrusted model output. The payload is read
//     behind the same guards as a lessons ingest (size cap, DisallowUnknownFields,
//     schema gate) and every entry is cite-or-be-dropped against the live
//     record/finding/path sets, its prose sanitised and marker-neutralised.
//
// principles.json + principles.md are ALWAYS written (even empty), a deliberate
// divergence from lessons: they are standalone top-level artifacts a consumer opens
// unconditionally, so a mode-stamped possibly-empty file keeps the file set stable.
// They are kept OUT of manifest_sha256 (see embark_types.go's exclusion sets).

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

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SynthesizePrinciples builds or ingests principles.json (+ principles.md) for a
// packed lifeboat. raw == nil selects the deterministic evidence-only fallback;
// raw != nil validates the untrusted delegated payload. It is transport-agnostic:
// it returns a PrinciplesResult and never prints. Structural faults fail closed
// (an error, nothing written); per-entry faults drop that entry (recorded), never
// the batch. The file is always written on success — an all-drop ingest writes
// "principles": [] and reports the drops, exit 0.
func SynthesizePrinciples(lifeboatDir string, raw []byte) (PrinciplesResult, error) {
	abs, _, err := gateSynthLifeboat(lifeboatDir)
	if err != nil {
		return PrinciplesResult{}, err
	}

	res := PrinciplesResult{LifeboatDir: abs}
	var (
		principles    []Principle
		mode          SynthesisMode
		promptVersion string
	)

	if raw == nil {
		// Deterministic order is gvEachADR order (home then sorted name) — already
		// byte-stable, so it is NOT re-sorted by id (which would misorder adr-2 vs
		// adr-10).
		mode = ModeDeterministic
		var drops []PrincipleDrop
		principles, drops, err = deterministicPrinciples(abs)
		if err != nil {
			return PrinciplesResult{}, err
		}
		res.Dropped = len(drops)
		res.Drops = drops
	} else {
		mode = ModeDelegated
		var drops []PrincipleDrop
		principles, promptVersion, drops, err = validateDelegatedPrinciples(abs, raw)
		if err != nil {
			return PrinciplesResult{}, err
		}
		res.Dropped = len(drops)
		res.Drops = drops
		// Delegated survivors are sorted by id before write, so a re-ingest of the
		// same payload writes byte-identical files (mirrors IngestLessons).
		sort.Slice(principles, func(i, j int) bool { return principles[i].ID < principles[j].ID })
	}

	if principles == nil {
		principles = []Principle{} // marshal "principles": [], never null
	}

	file := PrinciplesFile{
		SchemaVersion: PrinciplesSchemaVersion,
		Mode:          mode,
		PromptVersion: promptVersion,
		Principles:    principles,
	}
	data, err := marshalSynth(file)
	if err != nil {
		return PrinciplesResult{}, err
	}

	root, err := os.OpenRoot(abs)
	if err != nil {
		return PrinciplesResult{}, err
	}
	defer root.Close()

	// Full replacement: a single-file artifact, so the atomic overwrite IS the
	// replacement — no accretion across runs, no stale twin.
	if err := writeIntoLifeboat(root, abs, "principles.json", data); err != nil {
		return PrinciplesResult{}, err
	}
	if err := writeIntoLifeboat(root, abs, "principles.md", []byte(renderPrinciplesMarkdown(file))); err != nil {
		return PrinciplesResult{}, err
	}

	res.Mode = mode
	res.Written = len(principles)
	res.PrinciplesPath = "principles.json"
	res.RenderPath = "principles.md"
	return res, nil
}

// deterministicPrinciples surfaces each packed ADR's own stated decision as a
// cited principle. It reads only the lifeboat's own files through the contained
// SourceContext (docs/adrs/ is a conventional ADR home gvEachADR already scans);
// it quotes the record and never invents. An ADR with no Decision/Consequences
// section, or an empty first bullet, contributes nothing — an empty result is a
// first-class "principles": []. Order is gvEachADR order (home then sorted name).
func deterministicPrinciples(abs string) ([]Principle, []PrincipleDrop, error) {
	ctx, err := newSourceContext(abs)
	if err != nil {
		return nil, nil, err
	}
	defer ctx.Close()

	var out []Principle
	var drops []PrincipleDrop
	seen := map[string]bool{}
	gvEachADR(ctx, func(name, p string, fields map[string]frontmatter.Field) {
		if len(out) >= maxPrinciples {
			return
		}
		adrID := gvADRID(fields, name)
		if adrID == "" {
			return
		}
		data, ok := ctx.ReadFile(p)
		if !ok {
			return
		}
		// Evaluate each keyword's own first-matching section independently:
		// gvSectionBullets matches only the first heading in document order, so a
		// combined ("consequences","decision") call returns the earlier Decision
		// section — which, when it is prose or a numbered list, yields zero bullets
		// and silently drops an ADR whose Consequences below IS bulleted. Prefer
		// Consequences, fall back to Decision.
		bullets, found := gvSectionBullets(data, "consequences")
		if !found || len(bullets) == 0 {
			bullets, found = gvSectionBullets(data, "decision")
		}
		if !found || len(bullets) == 0 {
			return
		}
		principle := cleanSynthProse(bulletText(bullets[0]))
		if principle == "" {
			return
		}
		prnID := "prn-" + adrID
		if !prnIDRe.MatchString(prnID) {
			return
		}
		// The dedup is tested only once the ADR has a principle to contribute: a
		// same-numbered ADR that yields no bullet is not a lost principle and must
		// not inflate the drop count. When a genuine ADR-number collision DOES drop a
		// distilled principle, the drop is announced — matching the delegated / review
		// / lessons ingests rather than the silent first-wins map this replaced
		// (iss-2608270908344367).
		if seen[adrID] {
			drops = append(drops, PrincipleDrop{
				ID:     prnID,
				Reason: "duplicate ADR id (shadowed by an earlier ADR of the same number)",
			})
			return
		}
		seen[adrID] = true
		out = append(out, Principle{
			ID:         prnID,
			Principle:  principle,
			Confidence: ConfidenceHigh,
			Evidence:   []string{adrID, p},
		})
	})
	return out, drops, nil
}

// validateDelegatedPrinciples reads the untrusted payload behind the lessons-ingest
// guards, then applies cite-or-be-dropped per entry against the live R∪F∪P sets.
// A structural fault (oversize, unknown field, schema/mode/prompt_version) is
// fatal (nothing written); a per-entry fault drops that entry. It returns the
// survivors, the recorded prompt_version, and the drops.
func validateDelegatedPrinciples(abs string, raw []byte) ([]Principle, string, []PrincipleDrop, error) {
	if len(raw) > maxSynthesisBytes {
		return nil, "", nil, fmt.Errorf("principles payload exceeds the %d-byte cap", maxSynthesisBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var pf PrinciplesFile
	if err := dec.Decode(&pf); err != nil {
		return nil, "", nil, fmt.Errorf("malformed principles JSON: %v", err)
	}
	if err := synthSchemaGate("principles", pf.SchemaVersion, PrinciplesSchemaVersion); err != nil {
		return nil, "", nil, err
	}
	if err := synthModeGate(pf.Mode); err != nil {
		return nil, "", nil, err
	}
	if !promptVersionRe.MatchString(pf.PromptVersion) {
		return nil, "", nil, errors.New("delegated principles payload is missing a semver prompt_version")
	}
	if len(pf.Principles) > maxPrinciples {
		return nil, "", nil, fmt.Errorf("too many principles (%d > %d)", len(pf.Principles), maxPrinciples)
	}

	valid, err := buildPrincipleEvidenceSet(abs)
	if err != nil {
		return nil, "", nil, err
	}

	var survivors []Principle
	var drops []PrincipleDrop
	seen := map[string]bool{}
	for _, in := range pf.Principles {
		drop := func(reason string) { drops = append(drops, PrincipleDrop{ID: in.ID, Reason: reason}) }
		if len(in.ID) > maxSynthIDLen || !prnIDRe.MatchString(in.ID) {
			drop("malformed principle id")
			continue
		}
		if seen[in.ID] {
			drop("duplicate principle id")
			continue
		}
		if in.Confidence != ConfidenceHigh && in.Confidence != ConfidenceMedium && in.Confidence != ConfidenceLow {
			drop("unknown confidence")
			continue
		}
		refs := filterSynthEvidence(in.Evidence, valid)
		if len(refs) == 0 {
			drop("no valid evidence refs")
			continue
		}
		clean := cleanSynthProse(in.Principle)
		if clean == "" {
			drop("empty principle prose")
			continue
		}
		// First-wins dedup marks the id seen only after full survival (the
		// IngestLessons note), so a dropped first occurrence cannot poison a later
		// fully-citable duplicate.
		seen[in.ID] = true
		survivors = append(survivors, Principle{
			ID:         in.ID,
			Principle:  clean,
			Confidence: in.Confidence,
			Evidence:   refs,
		})
	}
	return survivors, pf.PromptVersion, drops, nil
}

// buildPrincipleEvidenceSet is the union R∪F∪P a delegated principle's evidence
// must hit: live record ids (adr/itd/iss), live graveyard finding ids, and every
// packed lifeboat path. A principle survives iff ≥1 of its refs is a member.
func buildPrincipleEvidenceSet(abs string) (map[string]bool, error) {
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	paths, err := buildLifeboatPathSet(root, principlesOwnOutput)
	if err != nil {
		return nil, err
	}
	findings, err := collectLiveFindingIDs(abs)
	if err != nil {
		return nil, err
	}
	records, err := collectLiveRecordIDs(abs, paths)
	if err != nil {
		return nil, err
	}
	valid := make(map[string]bool, len(paths)+len(findings)+len(records))
	for k := range paths {
		valid[k] = true
	}
	for k := range findings {
		valid[k] = true
	}
	for k := range records {
		valid[k] = true
	}
	return valid, nil
}

// Render is the deterministic, sanitised human summary of a principles run.
func (r PrinciplesResult) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "principles for %s (%s)\n", sanitize(r.LifeboatDir), sanitize(string(r.Mode)))
	fmt.Fprintf(&b, "  written: %d  (principles.json)\n", r.Written)
	fmt.Fprintf(&b, "  dropped: %d\n", r.Dropped)
	for _, d := range r.Drops {
		fmt.Fprintf(&b, "    - %s (%s)\n", sanitize(d.ID), sanitize(d.Reason))
	}
	return b.String()
}

// renderPrinciplesMarkdown is the sanitised, deterministic principles.md written
// into the lifeboat. It is a render of the file content, distinct from
// PrinciplesResult.Render (the CLI summary).
func renderPrinciplesMarkdown(f PrinciplesFile) string {
	var b strings.Builder
	b.WriteString("# Principles\n\n")
	fmt.Fprintf(&b, "_mode: %s", sanitize(string(f.Mode)))
	if f.PromptVersion != "" {
		fmt.Fprintf(&b, "; prompt %s", sanitize(f.PromptVersion))
	}
	b.WriteString("_\n\n")
	if len(f.Principles) == 0 {
		b.WriteString("_No principles distilled._\n")
		return b.String()
	}
	for _, p := range f.Principles {
		fmt.Fprintf(&b, "## %s (%s)\n\n", sanitize(p.ID), sanitize(string(p.Confidence)))
		fmt.Fprintf(&b, "%s\n\n", sanitize(p.Principle))
		if len(p.Evidence) > 0 {
			fmt.Fprintf(&b, "Evidence: %s\n\n", strings.Join(sanitizeAll(p.Evidence), ", "))
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Shared synthesis helpers. A1 lands these; A2's oracle reuses them.
// ---------------------------------------------------------------------------

// gateSynthLifeboat is the shared entry gate for every synthesis verb: a real
// directory carrying a parseable _provenance.json whose schema this abcd
// understands. It mirrors the IngestLessons / runPlanner gate but does NOT verify
// the manifest (a false manifest is a verdict input for the oracle, never a gate
// for principles/press-release). It returns the absolute lifeboat path and the
// parsed provenance header.
func gateSynthLifeboat(lifeboatDir string) (string, Provenance, error) {
	abs, err := filepath.Abs(lifeboatDir)
	if err != nil {
		return "", Provenance{}, err
	}
	if !fsutil.IsRealDir(abs) {
		return "", Provenance{}, fmt.Errorf("lifeboat %s is not a directory", filepath.Base(abs))
	}
	if !isAbcdLifeboat(abs) {
		return "", Provenance{}, fmt.Errorf("%s is not an abcd lifeboat (no parseable %s)", filepath.Base(abs), ProvenanceName)
	}
	prov, err := readProvenance(abs)
	if err != nil {
		return "", Provenance{}, err
	}
	if prov.SchemaVersion > SchemaVersion {
		return "", Provenance{}, update.TooNew("lifeboat",
			prov.SchemaVersion, SchemaVersion)
	}
	return abs, prov, nil
}

// buildLifeboatPathSet is the packed-path membership set P: every regular file's
// lifeboat-relative POSIX path, from the same sorted, symlink-refusing, bounded
// walk VerifyManifest uses. A delegated ref that names a packed path is a valid
// citation.
func buildLifeboatPathSet(root *os.Root, ownOutput func(string) bool) (map[string]bool, error) {
	rels, err := walkLifeboatFiles(root)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(rels))
	for _, r := range rels {
		// A verb's OWN post-pack output must never enter its citation set, or a
		// payload citing the verb's own not-yet-written artifact would be dropped
		// on the first run and survive on the second — a self-referential re-run
		// non-determinism. Another verb's output is a legitimate citation (a press
		// release cites principles.json), so the exclusion is per-verb, not global.
		if ownOutput != nil && ownOutput(r) {
			continue
		}
		set[r] = true
	}
	return set, nil
}

// principlesOwnOutput matches the principle verb's own post-pack artifacts.
func principlesOwnOutput(rel string) bool {
	return rel == "principles.json" || rel == "principles.md"
}

// pressReleaseOwnOutput matches the press-release verb's own post-pack artifacts.
func pressReleaseOwnOutput(rel string) bool {
	return rel == "press-release.json" || rel == "press-release.md"
}

// reviewOwnOutput matches the review verb's own post-pack artefacts, including
// the pre-spc-30 audit/ home so an older pack's artefact is still recognised as
// the verb's own output rather than packed evidence.
func reviewOwnOutput(rel string) bool {
	return strings.HasPrefix(rel, reviewArtefactDir+"/") || strings.HasPrefix(rel, legacyReviewDir+"/")
}

// cleanSynthProse sanitises untrusted synthesis prose (principle, finding,
// headline, subhead, quote): it collapses newlines, neutralises HTML-comment
// delimiters (so the prose can neither break its line nor forge an abcd marker),
// strips control characters, then caps the length. An empty result signals the
// entry should be dropped. It mirrors cleanLessonProse — both route through
// termsafe.CleanProse, differing only in their byte cap.
func cleanSynthProse(s string) string { return cleanSynthProseN(s, maxSynthProseBytes) }

// cleanSynthProseN is cleanSynthProse with an explicit byte cap, so a longer-form
// field (the press-release body) can share one sanitiser at its own ceiling. The
// cleaning itself is termsafe's — the canonical home this seam and the release
// ingest seam both route through, rather than keeping divergent copies.
func cleanSynthProseN(s string, capBytes int) string {
	return termsafe.CleanProse(s, capBytes)
}

// filterSynthEvidence keeps only the refs that are members of valid, deduped and
// order-preserving, reading at most maxSynthEvidenceRefs entries. It mirrors
// filterEvidence.
func filterSynthEvidence(refs []string, valid map[string]bool) []string {
	n := len(refs)
	if n > maxSynthEvidenceRefs {
		n = maxSynthEvidenceRefs
	}
	var out []string
	seen := map[string]bool{}
	for _, r := range refs[:n] {
		if valid[r] && !seen[r] {
			out = append(out, r)
			seen[r] = true
		}
	}
	return out
}

// marshalSynth renders a synthesis artifact deterministically (indented, trailing
// newline), matching the marshalLessonsFile / plan.go convention.
func marshalSynth(v any) ([]byte, error) {
	j, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(j, '\n'), nil
}

// synthSchemaGate is the shared three-branch schema check for a delegated payload:
// a missing (0), too-new, or unsupported schema_version is a structural refusal.
func synthSchemaGate(what string, got, want int) error {
	if got == 0 {
		return fmt.Errorf("%s payload is missing schema_version", what)
	}
	if got > want {
		return update.TooNew(what, got, want)
	}
	if got != want {
		return fmt.Errorf("unsupported %s schema_version %d", what, got)
	}
	return nil
}

// synthModeGate refuses a delegated payload that claims deterministic. An empty
// mode is allowed (the core stamps ModeDelegated on write regardless).
func synthModeGate(mode SynthesisMode) error {
	if mode != "" && mode != ModeDelegated {
		return errors.New("a delegated payload must not claim mode " + string(mode))
	}
	return nil
}

// collectLiveFindingIDs builds the live graveyard finding-id set F from the sealed
// layer-1/2 files. Absent graveyard files contribute nothing (a lifeboat need not
// carry a graveyard); a present-but-unreadable file (symlink, oversize, corrupt)
// is fatal, matching the guarded-read taxonomy.
func collectLiveFindingIDs(abs string) (map[string]bool, error) {
	var groups [][]Finding
	for _, rel := range []string{path.Join("graveyard", "archaeology.json"), path.Join("graveyard", "abandoned.json")} {
		if _, err := os.Lstat(filepath.Join(abs, rel)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("graveyard: stat %s: %w", rel, err)
		}
		switch filepath.Base(rel) {
		case "archaeology.json":
			arch, err := readGraveyardFile[Archaeology](abs, rel)
			if err != nil {
				return nil, err
			}
			groups = append(groups, arch.Findings)
		case "abandoned.json":
			aband, err := readGraveyardFile[Abandoned](abs, rel)
			if err != nil {
				return nil, err
			}
			groups = append(groups, aband.Findings)
		}
	}
	return collectFindingIDs(groups...), nil
}

// collectLiveRecordIDs builds the live record-id set R: adr-N from the packed ADRs
// (gvEachADR + gvADRID), itd-N from rescue/intents/**, iss-N from
// activity/issues/** (each a frontmatter id of the expected shape). It reads only
// through the contained SourceContext. Membership is a set, so map-iteration order
// does not affect the result.
func collectLiveRecordIDs(abs string, paths map[string]bool) (map[string]bool, error) {
	ctx, err := newSourceContext(abs)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()
	ids := map[string]bool{}
	gvEachADR(ctx, func(name, p string, fields map[string]frontmatter.Field) {
		if id := gvADRID(fields, name); id != "" {
			ids[id] = true
		}
	})
	for p := range paths {
		if !isMarkdownPath(p) {
			continue
		}
		switch {
		case strings.HasPrefix(p, "rescue/intents/"):
			if fields, ok := gvFields(ctx, p); ok {
				if id := gvUnquote(fields["id"].Value); gvIntentIDRe.MatchString(id) {
					ids[id] = true
				}
			}
		case strings.HasPrefix(p, "activity/issues/"):
			if fields, ok := gvFields(ctx, p); ok {
				if id := gvUnquote(fields["id"].Value); gvIssueIDRe.MatchString(id) {
					ids[id] = true
				}
			}
		}
	}
	return ids, nil
}

// bulletText strips a leading Markdown bullet marker from a trimmed section bullet
// so a distilled principle reads as prose, not as a list item.
func bulletText(b string) string {
	b = strings.TrimSpace(b)
	b = strings.TrimPrefix(b, "- ")
	b = strings.TrimPrefix(b, "* ")
	return strings.TrimSpace(b)
}

// isMarkdownPath reports whether a packed path is a Markdown record.
func isMarkdownPath(p string) bool {
	low := strings.ToLower(p)
	return strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".markdown")
}

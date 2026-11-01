package lifeboat

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
	"syscall"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// IngestLessons validates host-produced lesson JSON against a PACKED lifeboat and
// writes the survivors into it. It is transport-agnostic: it returns a
// LessonsResult and never prints. It fails closed on structural problems (returns
// an error); per-entry problems drop that entry (recorded in the result), never
// the batch. Output ordering is deterministic (lessons sorted by id), so a
// re-ingest of the same payload writes byte-identical files.
//
// Each ingest FULLY REPLACES layer 3: the prior graveyard/lessons.json and
// graveyard/low-confidence/ are cleared before the current survivors are written,
// so re-ingesting is the current interpretation, not an accretion of every past
// run — a lesson promoted low->high leaves no stale low-confidence file, and a
// lesson dropped from a later payload does not persist.
//
// The written lessons files are DELIBERATELY NOT part of manifest_sha256. That
// hash is pinned at pack time (adr-35) over the deterministic layer-1/2
// extraction and the rest of the lifeboat; it is the integrity seal
// _provenance.json records and the key to embark's re-pack closure. Layer 3 is a
// later, mutable, host-delegated interpretation written into an already-sealed
// lifeboat. Folding it into the manifest would either force a re-seal (breaking
// "pinned at pack time") or leave a hash that no longer matches _provenance.json.
// The graveyard interpretation's integrity is the per-entry cite-or-be-dropped
// rule, not the manifest seal — by design.
//
// The payload is secret-scanned before it is written, as pack's planned bytes
// are: every file the call would write is handed to scan before anything is
// cleared or written, and a non-nil error refuses the whole call, leaving the
// lifeboat as it was. Refuse, never redact. A nil scan fails closed.
//
// It is PlanIngestLessons followed by the plan's Apply, so a practice run
// (`--dry-run`) and the real write share every check.
func IngestLessons(lifeboatDir string, raw []byte, scan SecretScan) (LessonsResult, error) {
	p, err := PlanIngestLessons(lifeboatDir, raw, scan)
	if err != nil {
		return LessonsResult{}, err
	}
	return p.Apply()
}

// LessonsPlan is what IngestLessons would do: the result it would report, and
// the files it would write and clear.
type LessonsPlan struct {
	Result LessonsResult
	SynthPlan
}

// Apply clears the prior layer 3 and writes the planned survivors.
func (p LessonsPlan) Apply() (LessonsResult, error) {
	if err := p.apply(); err != nil {
		return LessonsResult{}, err
	}
	return p.Result, nil
}

// PlanIngestLessons does everything IngestLessons does before its first
// write: the lifeboat gate, the payload's decode and per-entry validation, the
// composition of every survivor file, and the secret scan over those bytes. A
// refused payload is refused here, in the same words; nothing is written.
func PlanIngestLessons(lifeboatDir string, raw []byte, scan SecretScan) (LessonsPlan, error) {
	if scan == nil {
		return LessonsPlan{}, errors.New("lessons: a secret scan is required")
	}
	abs, err := filepath.Abs(lifeboatDir)
	if err != nil {
		return LessonsPlan{}, err
	}
	// 1. Gate the lifeboat: a real directory, reached through no symlinked
	// ancestor inside a checkout, carrying a parseable _provenance.json.
	if err := proveOperand("lifeboat", abs); err != nil {
		return LessonsPlan{}, err
	}
	if !fsutil.IsRealDir(abs) {
		return LessonsPlan{}, fmt.Errorf("lifeboat %s is not a directory", filepath.Base(abs))
	}
	if !isAbcdLifeboat(abs) {
		return LessonsPlan{}, fmt.Errorf("%s is not an abcd lifeboat (no parseable %s)", filepath.Base(abs), ProvenanceName)
	}
	// graveyard/ must be a real directory — never a symlink we would read or write
	// through. This is one belt of the belt-and-suspenders containment.
	if !fsutil.IsRealDir(filepath.Join(abs, "graveyard")) {
		return LessonsPlan{}, errors.New("lifeboat graveyard/ is missing or not a real directory")
	}

	// 2. Read layers 1 + 2 behind the read guards and build the live id set.
	arch, err := readGraveyardFile[Archaeology](abs, path.Join("graveyard", "archaeology.json"))
	if err != nil {
		return LessonsPlan{}, err
	}
	aband, err := readGraveyardFile[Abandoned](abs, path.Join("graveyard", "abandoned.json"))
	if err != nil {
		return LessonsPlan{}, err
	}
	ids := collectFindingIDs(arch.Findings, aband.Findings)

	// 3. Parse the untrusted lessons payload, fail-closed on structure.
	if len(raw) > maxLessonsBytes {
		return LessonsPlan{}, fmt.Errorf("lessons payload exceeds the %d-byte cap", maxLessonsBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields() // reject smuggled extra fields
	var lf LessonsFile
	if err := dec.Decode(&lf); err != nil {
		// The decoder names an undeclared field by the payload's own key:
		// redacted, never raw (iss-2609290218032954).
		return LessonsPlan{}, fmt.Errorf("malformed lessons JSON: %s", scanner.RedactRefusal(abs, err.Error()))
	}
	if lf.SchemaVersion == 0 {
		return LessonsPlan{}, errors.New("lessons payload is missing schema_version")
	}
	if lf.SchemaVersion > LessonsSchemaVersion {
		return LessonsPlan{}, update.TooNew("lessons",
			lf.SchemaVersion, LessonsSchemaVersion)
	}
	if lf.SchemaVersion != LessonsSchemaVersion {
		return LessonsPlan{}, fmt.Errorf("unsupported lessons schema_version %d", lf.SchemaVersion)
	}
	if len(lf.Lessons) > maxLessons {
		return LessonsPlan{}, fmt.Errorf("too many lessons (%d > %d)", len(lf.Lessons), maxLessons)
	}

	// 4. Per-entry validation, drop-not-fatal.
	res := LessonsResult{LifeboatDir: fsutil.RedactHome(abs)}
	seen := map[string]bool{}
	var mainLessons, lowLessons []Lesson
	for _, in := range lf.Lessons {
		drop := func(reason string) {
			res.Dropped++
			res.Drops = append(res.Drops, LessonDrop{ID: in.ID, Reason: reason})
		}
		if len(in.ID) > maxLessonIDLen || !lessonIDRe.MatchString(in.ID) {
			drop("malformed lesson id")
			continue
		}
		if seen[in.ID] {
			drop("duplicate lesson id")
			continue
		}
		if in.Confidence != ConfidenceHigh && in.Confidence != ConfidenceMedium && in.Confidence != ConfidenceLow {
			drop("unknown confidence")
			continue
		}
		refs := filterEvidence(in.Evidence, ids)
		if len(refs) == 0 {
			drop("no valid evidence refs")
			continue
		}
		clean := cleanLessonProse(in.Lesson)
		if clean == "" {
			drop("empty lesson prose")
			continue
		}
		// First-wins dedup marks the id seen only once the entry SURVIVES every
		// per-entry check. Marking earlier would let a dropped first occurrence
		// poison the id, so a later fully-citable duplicate is refused as a
		// "duplicate" of an entry that was never written.
		seen[in.ID] = true
		l := Lesson{ID: in.ID, Lesson: clean, Confidence: in.Confidence, Evidence: refs}
		if in.Confidence == ConfidenceLow {
			lowLessons = append(lowLessons, l)
		} else {
			mainLessons = append(mainLessons, l)
		}
	}

	// 5. Plan every file this ingest writes, then secret-scan the planned bytes
	//    BEFORE the prior layer 3 is cleared or anything is written.
	var planned []PlannedFile
	if len(mainLessons) > 0 {
		sort.Slice(mainLessons, func(i, j int) bool { return mainLessons[i].ID < mainLessons[j].ID })
		data, err := marshalLessonsFile(mainLessons)
		if err != nil {
			return LessonsPlan{}, err
		}
		planned = append(planned, plannedFile(path.Join("graveyard", "lessons.json"), data))
	}
	sort.Slice(lowLessons, func(i, j int) bool { return lowLessons[i].ID < lowLessons[j].ID })
	for _, l := range lowLessons {
		data, err := marshalLessonsFile([]Lesson{l})
		if err != nil {
			return LessonsPlan{}, err
		}
		// The filename is built from the VALIDATED lesson id (lessonIDRe +
		// maxLessonIDLen) — that regex is the path-traversal defence: the id can
		// carry no separator, "..", or control character, so the target can never
		// escape graveyard/low-confidence/.
		planned = append(planned, plannedFile(path.Join("graveyard", "low-confidence", l.ID+".json"), data))
	}
	if err := scan(planned); err != nil {
		return LessonsPlan{}, err
	}

	// 6. The writes, laid by Apply. os.Root is an independent containment
	//    assertion over the relative target; the durable write itself runs
	//    through the canonical fsutil.WriteFileAtomic. Two layers, so a bug in
	//    one is not an escape.
	//
	// Each ingest is a FULL REPLACEMENT of layer 3: the prior interpretation is
	// cleared before the current survivors are written, so a lesson promoted
	// low->high leaves no stale low-confidence/<id>.json and a lesson dropped
	// from a later payload does not persist. Removal runs through the contained
	// os.Root; a missing path is not an error. The plan names every file the
	// clear would remove that the survivors do not write again.
	sp := newSynthPlan(abs, planned)
	sp.clearLayer3 = true
	sp.planRemove(layer3Files(abs))
	res.Written = len(mainLessons)
	res.LowConfidence = len(lowLessons)
	return LessonsPlan{Result: res, SynthPlan: sp}, nil
}

// plannedFile is one file a post-pack verb is about to write into the
// lifeboat, in the shape pack's secret scan reads.
func plannedFile(rel string, data []byte) PlannedFile {
	return PlannedFile{Path: rel, Content: data, Bytes: len(data)}
}

// writePlanned writes each planned file, in order, into the contained lifeboat.
// The caller has already secret-scanned exactly these bytes.
func writePlanned(root *os.Root, abs string, files []PlannedFile) error {
	for _, f := range files {
		if err := writeIntoLifeboat(root, abs, f.Path, f.Content); err != nil {
			return err
		}
	}
	return nil
}

// Render is the deterministic, sanitised human summary of an ingest.
func (r LessonsResult) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "graveyard lessons for %s\n", sanitize(r.LifeboatDir))
	fmt.Fprintf(&b, "  written:        %d  (graveyard/lessons.json)\n", r.Written)
	fmt.Fprintf(&b, "  low-confidence: %d  (graveyard/low-confidence/)\n", r.LowConfidence)
	fmt.Fprintf(&b, "  dropped:        %d\n", r.Dropped)
	for _, d := range r.Drops {
		fmt.Fprintf(&b, "    - %s (%s)\n", sanitize(d.ID), sanitize(d.Reason))
	}
	return b.String()
}

// readGraveyardFile reads and JSON-decodes one packed layer-1/2 file behind the
// trust guards (no symlink, regular file, size cap). A packed lifeboat always
// carries both files; missing/oversize/symlink/unparseable is fatal.
func readGraveyardFile[T any](abs, rel string) (T, error) {
	var zero T
	p := filepath.Join(abs, rel)
	// Guarded read closes the lstat→read TOCTOU: the previous lstat (symlink +
	// size check) followed by a separate os.ReadFile left a swap window — a
	// regular in-cap file could pass the checks and then be swapped for a
	// symlink-to-/dev/zero or a grown file that os.ReadFile would follow/read
	// unbounded. ReadGuarded refuses symlinks on the open fd (O_NOFOLLOW) and
	// bounds the read to the cap in one call, matching classifyEmbark/pack. The
	// source is a packed lifeboat from an arbitrary target dir — a trust boundary.
	data, err := fsutil.ReadGuarded(p, maxGraveyardFileBytes)
	if err != nil {
		switch {
		case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
			return zero, fmt.Errorf("graveyard: %s is not a regular file (a symlink or non-regular target is refused)", rel)
		case errors.Is(err, fsutil.ErrTooBig):
			return zero, fmt.Errorf("graveyard: %s exceeds the %d-byte cap", rel, maxGraveyardFileBytes)
		default:
			return zero, fmt.Errorf("graveyard: read %s: %w", rel, err)
		}
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, fmt.Errorf("graveyard: %s is not valid JSON: %w", rel, err)
	}
	return v, nil
}

// clearLayer3 removes the prior layer-3 interpretation through the containment
// root — graveyard/lessons.json and the whole graveyard/low-confidence/ directory
// — so a re-ingest is a full replacement rather than an accreting merge. Removing
// a path that does not exist is not an error (os.Root.RemoveAll matches
// os.RemoveAll here). The low-confidence directory is recreated by
// writeIntoLifeboat's contained MkdirAll when the current payload has survivors
// routed to it.
func clearLayer3(root *os.Root) error {
	for _, rel := range []string{
		path.Join("graveyard", "lessons.json"),
		path.Join("graveyard", "low-confidence"),
	} {
		if err := root.RemoveAll(rel); err != nil {
			return err
		}
	}
	return nil
}

// writeIntoLifeboat durably writes data at the relative target inside the
// lifeboat. It first asserts, via the os.Root, that no existing parent component
// of rel is a symlink (belt-and-suspenders — os.Root also refuses symlink
// traversal on its own), creates the missing directories contained, then commits
// the bytes through the canonical fsutil.WriteFileAtomic. The os.Root check to
// WriteFileAtomic gap is a benign TOCTOU under the trusted-worktree model, the
// same note readVerdictFile carries.
func writeIntoLifeboat(root *os.Root, abs, rel string, data []byte) error {
	if err := ensureContainedDir(root, rel); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(abs, rel), data, 0o644)
}

// createIntoLifeboat is writeIntoLifeboat for a file that must not exist yet:
// the same contained directory walk, then an exclusive create, so a file that
// landed at rel after the caller judged it absent fails the write loudly
// (os.ErrExist) instead of being replaced by a rename over it.
func createIntoLifeboat(root *os.Root, rel string, data []byte) error {
	if err := ensureContainedDir(root, rel); err != nil {
		return err
	}
	return fsutil.CreateExclusiveIn(root, rel, data, 0o644)
}

// ensureContainedDir asserts that no existing parent component of rel is a
// symlink and creates the missing ones through the root.
func ensureContainedDir(root *os.Root, rel string) error {
	dir := path.Dir(rel)
	if dir != "." {
		cur := ""
		for _, seg := range strings.Split(dir, "/") {
			if cur == "" {
				cur = seg
			} else {
				cur = cur + "/" + seg
			}
			fi, err := root.Lstat(cur)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					break // this and every deeper component is created by MkdirAll
				}
				return err
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlinked component %q inside the lifeboat", cur)
			}
		}
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// marshalLessonsFile renders a LessonsFile deterministically (indented, trailing
// newline), matching the plan.go convention for machine-and-human JSON.
func marshalLessonsFile(lessons []Lesson) ([]byte, error) {
	j, err := json.MarshalIndent(LessonsFile{SchemaVersion: LessonsSchemaVersion, Lessons: lessons}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(j, '\n'), nil
}

// filterEvidence keeps only the refs that resolve to a live layer-1/2 finding id,
// deduped and order-preserving, reading at most maxLessonEvidenceRefs entries.
func filterEvidence(refs []string, ids map[string]bool) []string {
	n := len(refs)
	if n > maxLessonEvidenceRefs {
		n = maxLessonEvidenceRefs
	}
	var out []string
	seen := map[string]bool{}
	for _, r := range refs[:n] {
		if ids[r] && !seen[r] {
			out = append(out, r)
			seen[r] = true
		}
	}
	return out
}

// cleanLessonProse sanitises untrusted lesson prose: it collapses newlines,
// neutralises HTML openers (so the prose can neither break its line nor forge an
// abcd marker), strips control characters, then caps the length. An empty result
// signals the entry should be dropped. The cleaning itself is termsafe's — the
// canonical home this seam and cleanSynthProse both route through, rather than
// keeping divergent copies.
func cleanLessonProse(s string) string {
	return termsafe.CleanProse(s, maxLessonProseBytes)
}

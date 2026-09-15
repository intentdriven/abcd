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
func IngestLessons(lifeboatDir string, raw []byte) (LessonsResult, error) {
	abs, err := filepath.Abs(lifeboatDir)
	if err != nil {
		return LessonsResult{}, err
	}
	// 1. Gate the lifeboat: a real directory carrying a parseable _provenance.json.
	if !fsutil.IsRealDir(abs) {
		return LessonsResult{}, fmt.Errorf("lifeboat %s is not a directory", filepath.Base(abs))
	}
	if !isAbcdLifeboat(abs) {
		return LessonsResult{}, fmt.Errorf("%s is not an abcd lifeboat (no parseable %s)", filepath.Base(abs), ProvenanceName)
	}
	// graveyard/ must be a real directory — never a symlink we would read or write
	// through. This is one belt of the belt-and-suspenders containment.
	if !fsutil.IsRealDir(filepath.Join(abs, "graveyard")) {
		return LessonsResult{}, errors.New("lifeboat graveyard/ is missing or not a real directory")
	}

	// 2. Read layers 1 + 2 behind the read guards and build the live id set.
	arch, err := readGraveyardFile[Archaeology](abs, path.Join("graveyard", "archaeology.json"))
	if err != nil {
		return LessonsResult{}, err
	}
	aband, err := readGraveyardFile[Abandoned](abs, path.Join("graveyard", "abandoned.json"))
	if err != nil {
		return LessonsResult{}, err
	}
	ids := collectFindingIDs(arch.Findings, aband.Findings)

	// 3. Parse the untrusted lessons payload, fail-closed on structure.
	if len(raw) > maxLessonsBytes {
		return LessonsResult{}, fmt.Errorf("lessons payload exceeds the %d-byte cap", maxLessonsBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields() // reject smuggled extra fields
	var lf LessonsFile
	if err := dec.Decode(&lf); err != nil {
		return LessonsResult{}, fmt.Errorf("malformed lessons JSON: %v", err)
	}
	if lf.SchemaVersion == 0 {
		return LessonsResult{}, errors.New("lessons payload is missing schema_version")
	}
	if lf.SchemaVersion > LessonsSchemaVersion {
		return LessonsResult{}, update.TooNew("lessons",
			lf.SchemaVersion, LessonsSchemaVersion)
	}
	if lf.SchemaVersion != LessonsSchemaVersion {
		return LessonsResult{}, fmt.Errorf("unsupported lessons schema_version %d", lf.SchemaVersion)
	}
	if len(lf.Lessons) > maxLessons {
		return LessonsResult{}, fmt.Errorf("too many lessons (%d > %d)", len(lf.Lessons), maxLessons)
	}

	// 4. Per-entry validation, drop-not-fatal.
	res := LessonsResult{LifeboatDir: abs}
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

	// 5. Write survivors. os.Root is an independent containment assertion over the
	//    relative target; the durable write itself runs through the canonical
	//    fsutil.WriteFileAtomic. Two layers, so a bug in one is not an escape.
	root, err := os.OpenRoot(abs)
	if err != nil {
		return LessonsResult{}, err
	}
	defer root.Close()

	// Each ingest is a FULL REPLACEMENT of layer 3: clear the prior interpretation
	// before writing the current survivors, so a lesson promoted low->high leaves
	// no stale low-confidence/<id>.json and a lesson dropped from a later payload
	// does not persist. Removal runs through the contained os.Root; a missing
	// path is not an error.
	if err := clearLayer3(root); err != nil {
		return LessonsResult{}, err
	}

	if len(mainLessons) > 0 {
		sort.Slice(mainLessons, func(i, j int) bool { return mainLessons[i].ID < mainLessons[j].ID })
		data, err := marshalLessonsFile(mainLessons)
		if err != nil {
			return LessonsResult{}, err
		}
		if err := writeIntoLifeboat(root, abs, path.Join("graveyard", "lessons.json"), data); err != nil {
			return LessonsResult{}, err
		}
		res.Written = len(mainLessons)
	}

	sort.Slice(lowLessons, func(i, j int) bool { return lowLessons[i].ID < lowLessons[j].ID })
	for _, l := range lowLessons {
		data, err := marshalLessonsFile([]Lesson{l})
		if err != nil {
			return LessonsResult{}, err
		}
		// The filename is built from the VALIDATED lesson id (lessonIDRe +
		// maxLessonIDLen) — that regex is the path-traversal defence: the id can
		// carry no separator, "..", or control character, so the target can never
		// escape graveyard/low-confidence/.
		rel := path.Join("graveyard", "low-confidence", l.ID+".json")
		if err := writeIntoLifeboat(root, abs, rel, data); err != nil {
			return LessonsResult{}, err
		}
		res.LowConfidence++
	}

	return res, nil
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
	return fsutil.WriteFileAtomic(filepath.Join(abs, rel), data, 0o644)
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

package lifeboat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- fixtures -------------------------------------------------------------

func marshalIndent(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return append(b, '\n')
}

func writeFile(t *testing.T, p string, data []byte) {
	t.Helper()
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

// writeLifeboatFixture hand-builds a packed lifeboat: a parseable
// _provenance.json (so isAbcdLifeboat accepts it) plus the two layer-1/2 files.
// Plan does not yet emit the graveyard, so layer-3 tests build the lifeboat by
// hand and are independent of the parallel archaeology/abandoned work.
func writeLifeboatFixture(t *testing.T, arch Archaeology, aband Abandoned) string {
	t.Helper()
	dir := t.TempDir()
	gy := filepath.Join(dir, "graveyard")
	if err := os.MkdirAll(gy, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gy, "archaeology.json"), marshalIndent(t, arch))
	writeFile(t, filepath.Join(gy, "abandoned.json"), marshalIndent(t, aband))
	prov := fmt.Sprintf(`{"schema_version":1,"generator":"test","source_name":"fix","manifest_sha256":%q}`,
		strings.Repeat("a", 64))
	writeFile(t, filepath.Join(dir, ProvenanceName), []byte(prov))
	return dir
}

func stdArch() Archaeology {
	return Archaeology{SchemaVersion: 1, Findings: []Finding{
		{ID: "rev-9f3a1c2d4e5b", Signal: SignalRevert, Summary: "reverted commit",
			Evidence: []string{`Revert "add cache"`}},
		{ID: "del-src/engine-v1", Signal: SignalDeletedPath, Summary: "deleted path",
			Evidence: []string{"deleted; 47 commits touched it"}},
	}}
}

func stdAband() Abandoned {
	return Abandoned{SchemaVersion: 1, Findings: []Finding{
		{ID: "adr-12", Signal: SignalSupersededADR, Summary: "ADR superseded",
			Evidence: []string{"superseded_by: adr-31"}},
	}}
}

func stdFixture(t *testing.T) string { return writeLifeboatFixture(t, stdArch(), stdAband()) }

func payload(t *testing.T, lessons ...Lesson) []byte {
	t.Helper()
	return marshalIndent(t, LessonsFile{SchemaVersion: LessonsSchemaVersion, Lessons: lessons})
}

func readWrittenLessons(t *testing.T, dir string) LessonsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "graveyard", "lessons.json"))
	if err != nil {
		t.Fatalf("read lessons.json: %v", err)
	}
	var lf LessonsFile
	if err := json.Unmarshal(data, &lf); err != nil {
		t.Fatalf("lessons.json is not valid JSON: %v", err)
	}
	return lf
}

func lessonsExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "graveyard", "lessons.json"))
	return err == nil
}

// --- tests ----------------------------------------------------------------

// TestIngestLessonsCiteOrDropped is the cite-or-be-dropped core: a lesson that
// cites a live finding id is written; one with zero valid refs is dropped
// (recorded, not fatal) and the run still succeeds.
func TestIngestLessonsCiteOrDropped(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t,
		Lesson{ID: "les-engine-v1", Lesson: "engine v1 was retired", Confidence: ConfidenceHigh,
			Evidence: []string{"rev-9f3a1c2d4e5b"}},
		Lesson{ID: "les-guess-cache", Lesson: "maybe caching mattered", Confidence: ConfidenceHigh,
			Evidence: []string{"no-such-id"}},
	)
	res, err := IngestLessons(dir, raw)
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if res.Written != 1 {
		t.Errorf("Written = %d, want 1", res.Written)
	}
	if res.Dropped != 1 || len(res.Drops) != 1 {
		t.Fatalf("Dropped = %d, Drops = %v; want 1", res.Dropped, res.Drops)
	}
	if res.Drops[0].ID != "les-guess-cache" || !strings.Contains(res.Drops[0].Reason, "evidence") {
		t.Errorf("unexpected drop: %+v", res.Drops[0])
	}
	lf := readWrittenLessons(t, dir)
	if len(lf.Lessons) != 1 || lf.Lessons[0].ID != "les-engine-v1" {
		t.Errorf("lessons.json = %+v", lf.Lessons)
	}
}

// TestIngestLessonsDeadRefFiltered keeps only the live refs: a lesson citing one
// live and one dead id is written with the live ref alone.
func TestIngestLessonsDeadRefFiltered(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t, Lesson{ID: "les-mixed", Lesson: "mixed refs", Confidence: ConfidenceMedium,
		Evidence: []string{"rev-9f3a1c2d4e5b", "ghost-ref"}})
	res, err := IngestLessons(dir, raw)
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if res.Written != 1 {
		t.Fatalf("Written = %d, want 1", res.Written)
	}
	lf := readWrittenLessons(t, dir)
	if len(lf.Lessons) != 1 || len(lf.Lessons[0].Evidence) != 1 || lf.Lessons[0].Evidence[0] != "rev-9f3a1c2d4e5b" {
		t.Errorf("evidence not filtered to live refs: %+v", lf.Lessons)
	}
}

// TestIngestLessonsLowConfidenceRouted routes a low-confidence entry to
// graveyard/low-confidence/<id>.json and keeps it out of lessons.json.
func TestIngestLessonsLowConfidenceRouted(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t, Lesson{ID: "les-maybe-cache", Lesson: "perhaps", Confidence: ConfidenceLow,
		Evidence: []string{"adr-12"}})
	res, err := IngestLessons(dir, raw)
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if res.Written != 0 || res.LowConfidence != 1 {
		t.Errorf("Written = %d, LowConfidence = %d; want 0, 1", res.Written, res.LowConfidence)
	}
	if lessonsExists(dir) {
		t.Error("lessons.json written for a low-confidence-only batch")
	}
	p := filepath.Join(dir, "graveyard", "low-confidence", "les-maybe-cache.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("low-confidence file not written: %v", err)
	}
	var lf LessonsFile
	if err := json.Unmarshal(data, &lf); err != nil || len(lf.Lessons) != 1 {
		t.Errorf("low-confidence file malformed: %v %+v", err, lf)
	}
}

// TestIngestLessonsDisallowUnknownFields fails closed on a smuggled extra field.
func TestIngestLessonsDisallowUnknownFields(t *testing.T) {
	dir := stdFixture(t)
	raw := []byte(`{"schema_version":1,"lessons":[{"id":"les-x","lesson":"y","confidence":"high","evidence":["adr-12"],"notes":"smuggled"}]}`)
	if _, err := IngestLessons(dir, raw); err == nil {
		t.Fatal("an unknown field must be a fatal error")
	}
}

// TestIngestLessonsRawTooLarge refuses an oversize payload.
func TestIngestLessonsRawTooLarge(t *testing.T) {
	dir := stdFixture(t)
	raw := make([]byte, maxLessonsBytes+1)
	if _, err := IngestLessons(dir, raw); err == nil {
		t.Fatal("an oversize payload must be a fatal error")
	}
}

// TestIngestLessonsSchemaVersion refuses a missing/zero schema and gives an
// upgrade message for a future schema.
func TestIngestLessonsSchemaVersion(t *testing.T) {
	dir := stdFixture(t)
	if _, err := IngestLessons(dir, []byte(`{"lessons":[]}`)); err == nil {
		t.Error("missing schema_version must fail")
	}
	if _, err := IngestLessons(dir, []byte(`{"schema_version":0,"lessons":[]}`)); err == nil {
		t.Error("schema_version 0 must fail")
	}
	_, err := IngestLessons(dir, []byte(`{"schema_version":2,"lessons":[]}`))
	if err == nil || !strings.Contains(err.Error(), "update abcd:") {
		t.Errorf("a future schema must ask for an updated abcd, got: %v", err)
	}
}

// TestIngestLessonsMalformedIDNoTraversal is the path-traversal guard: a lesson
// id that is not a validated kebab token is dropped and creates NO file — not in
// low-confidence, not anywhere the crafted id might have steered a write.
func TestIngestLessonsMalformedIDNoTraversal(t *testing.T) {
	dir := stdFixture(t)
	for _, badID := range []string{"../../escape", "les_bad_underscore", "Les-Caps", strings.Repeat("les-", 40)} {
		raw := payload(t, Lesson{ID: badID, Lesson: "x", Confidence: ConfidenceLow,
			Evidence: []string{"adr-12"}})
		res, err := IngestLessons(dir, raw)
		if err != nil {
			t.Fatalf("IngestLessons(%q): %v", badID, err)
		}
		if res.Dropped != 1 || res.Written != 0 || res.LowConfidence != 0 {
			t.Errorf("id %q: res = %+v; want a single drop, no write", badID, res)
		}
		if res.Drops[0].Reason != "malformed lesson id" {
			t.Errorf("id %q: drop reason = %q", badID, res.Drops[0].Reason)
		}
	}
	if lessonsExists(dir) {
		t.Error("a malformed-id-only batch must write no lessons.json")
	}
	// The parent of the lifeboat must gain nothing from a "../../escape" id.
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.json")); err == nil {
		t.Error("a crafted id escaped the lifeboat")
	}
	lcDir := filepath.Join(dir, "graveyard", "low-confidence")
	if entries, err := os.ReadDir(lcDir); err == nil && len(entries) > 0 {
		t.Errorf("low-confidence dir has unexpected files: %v", entries)
	}
}

// TestIngestLessonsDuplicateID keeps the first entry for an id and drops the rest.
func TestIngestLessonsDuplicateID(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t,
		Lesson{ID: "les-dup", Lesson: "first", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}},
		Lesson{ID: "les-dup", Lesson: "second", Confidence: ConfidenceHigh, Evidence: []string{"rev-9f3a1c2d4e5b"}},
	)
	res, err := IngestLessons(dir, raw)
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if res.Written != 1 || res.Dropped != 1 {
		t.Fatalf("res = %+v; want 1 written, 1 dropped", res)
	}
	lf := readWrittenLessons(t, dir)
	if len(lf.Lessons) != 1 || lf.Lessons[0].Lesson != "first" {
		t.Errorf("first-wins violated: %+v", lf.Lessons)
	}
	if res.Drops[0].Reason != "duplicate lesson id" {
		t.Errorf("drop reason = %q", res.Drops[0].Reason)
	}
}

// TestIngestLessonsDroppedFirstDoesNotPoisonID proves a dropped FIRST occurrence
// of an id does not poison it: a later, fully-citable duplicate still wins. The
// first entry cites only a dead ref (dropped for "no valid evidence refs"); the
// second cites a live id and must be written.
func TestIngestLessonsDroppedFirstDoesNotPoisonID(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t,
		Lesson{ID: "les-x", Lesson: "first, uncitable", Confidence: ConfidenceHigh, Evidence: []string{"dead-ref"}},
		Lesson{ID: "les-x", Lesson: "second, cited", Confidence: ConfidenceHigh, Evidence: []string{"rev-9f3a1c2d4e5b"}},
	)
	res, err := IngestLessons(dir, raw)
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if res.Written != 1 {
		t.Fatalf("Written = %d, want 1 (the citable duplicate must survive)", res.Written)
	}
	lf := readWrittenLessons(t, dir)
	if len(lf.Lessons) != 1 || lf.Lessons[0].Lesson != "second, cited" {
		t.Errorf("citable duplicate not written: %+v", lf.Lessons)
	}
	// The first entry is dropped as uncitable, NOT as a duplicate.
	if res.Dropped != 1 || res.Drops[0].Reason != "no valid evidence refs" {
		t.Errorf("first entry drop = %+v; want a single 'no valid evidence refs' drop", res.Drops)
	}
}

// TestIngestLessonsFullReplacement proves a re-ingest fully replaces layer 3: a
// lesson promoted low->high must not leave its stale low-confidence file behind.
func TestIngestLessonsFullReplacement(t *testing.T) {
	dir := stdFixture(t)
	rawA := payload(t, Lesson{ID: "les-a", Lesson: "maybe", Confidence: ConfidenceLow, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, rawA); err != nil {
		t.Fatalf("ingest A: %v", err)
	}
	lcPath := filepath.Join(dir, "graveyard", "low-confidence", "les-a.json")
	if _, err := os.Stat(lcPath); err != nil {
		t.Fatalf("ingest A did not write the low-confidence file: %v", err)
	}

	rawB := payload(t, Lesson{ID: "les-a", Lesson: "certain", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, rawB); err != nil {
		t.Fatalf("ingest B: %v", err)
	}
	if _, err := os.Stat(lcPath); !os.IsNotExist(err) {
		t.Errorf("stale low-confidence file survived a promoting re-ingest: err=%v", err)
	}
	lf := readWrittenLessons(t, dir)
	if len(lf.Lessons) != 1 || lf.Lessons[0].ID != "les-a" {
		t.Errorf("promoted lesson not in lessons.json: %+v", lf.Lessons)
	}
}

// TestIngestLessonsAllDroppedClearsPrior proves an ingest whose entries are all
// dropped removes a prior lessons.json — the coherent empty state, not a stale
// interpretation left standing.
func TestIngestLessonsAllDroppedClearsPrior(t *testing.T) {
	dir := stdFixture(t)
	rawA := payload(t, Lesson{ID: "les-a", Lesson: "certain", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, rawA); err != nil {
		t.Fatalf("ingest A: %v", err)
	}
	if !lessonsExists(dir) {
		t.Fatal("ingest A did not write lessons.json")
	}

	rawB := payload(t, Lesson{ID: "les-b", Lesson: "x", Confidence: ConfidenceHigh, Evidence: []string{"no-such-id"}})
	res, err := IngestLessons(dir, rawB)
	if err != nil {
		t.Fatalf("ingest B: %v", err)
	}
	if res.Written != 0 {
		t.Fatalf("Written = %d, want 0 (all dropped)", res.Written)
	}
	if lessonsExists(dir) {
		t.Error("prior lessons.json survived an ingest whose entries were all dropped")
	}
}

// TestIngestLessonsSanitisesProse neutralises comment markers, ANSI escapes and
// newlines in the written prose so it cannot forge a marker or break its line.
func TestIngestLessonsSanitisesProse(t *testing.T) {
	dir := stdFixture(t)
	dirty := "line one\n<!-- abcd-review: X -->\x1b[31mred\x1b[0m end"
	raw := payload(t, Lesson{ID: "les-dirty", Lesson: dirty, Confidence: ConfidenceHigh,
		Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw); err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	lf := readWrittenLessons(t, dir)
	got := lf.Lessons[0].Lesson
	if strings.Contains(got, "<!--") || strings.Contains(got, "-->") {
		t.Errorf("comment marker survived: %q", got)
	}
	if strings.ContainsRune(got, '\n') || strings.ContainsRune(got, '\x1b') {
		t.Errorf("newline/ANSI survived: %q", got)
	}
}

// TestIngestLessonsNeutralisesRawHTMLOpener pins the wider neutralisation the
// termsafe seam carries: a bare HTML opener (not just a comment marker) is broken
// with a space, so packed prose cannot smuggle a tag into a rendered lesson.
func TestIngestLessonsNeutralisesRawHTMLOpener(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t, Lesson{ID: "les-tag", Lesson: "before <script>alert(1)</script> after",
		Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw); err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	got := readWrittenLessons(t, dir).Lessons[0].Lesson
	if !strings.Contains(got, "< script") {
		t.Errorf("raw HTML opener not neutralised: %q", got)
	}
}

// TestIngestLessonsNeutralisesAfterSanitise pins the ORDER of the two steps:
// Sanitize substitutes '?' for a masked control byte, so a '<' followed by an
// escape becomes '<?' — a processing instruction — unless the neutralisation runs
// after the sanitiser.
func TestIngestLessonsNeutralisesAfterSanitise(t *testing.T) {
	dir := stdFixture(t)
	raw := payload(t, Lesson{ID: "les-order", Lesson: "a <" + "\x1b" + " b",
		Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw); err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	got := readWrittenLessons(t, dir).Lessons[0].Lesson
	if strings.Contains(got, "<?") {
		t.Errorf("masked control byte forged a processing instruction: %q", got)
	}
}

// TestIngestLessonsLifeboatGate refuses a directory that is not an abcd lifeboat.
func TestIngestLessonsLifeboatGate(t *testing.T) {
	dir := t.TempDir() // no _provenance.json
	if _, err := IngestLessons(dir, payload(t)); err == nil {
		t.Fatal("a non-lifeboat directory must be refused")
	}
}

// TestIngestLessonsSymlinkedGraveyardRefused refuses a lifeboat whose graveyard/
// is a symlink rather than a real directory.
func TestIngestLessonsSymlinkedGraveyardRefused(t *testing.T) {
	dir := t.TempDir()
	real := t.TempDir()
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(real, "archaeology.json"), marshalIndent(t, stdArch()))
	writeFile(t, filepath.Join(real, "abandoned.json"), marshalIndent(t, stdAband()))
	if err := os.Symlink(real, filepath.Join(dir, "graveyard")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	prov := fmt.Sprintf(`{"schema_version":1,"generator":"test","source_name":"fix","manifest_sha256":%q}`,
		strings.Repeat("a", 64))
	writeFile(t, filepath.Join(dir, ProvenanceName), []byte(prov))
	raw := payload(t, Lesson{ID: "les-x", Lesson: "y", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw); err == nil {
		t.Fatal("a symlinked graveyard/ must be refused")
	}
}

// TestIngestLessonsSymlinkedLayerFileRefused pins the per-file read guard: a
// packed layer-1/2 file (graveyard/archaeology.json) that is a symlink must be
// refused, never followed. This is the file-level guard that readGraveyardFile's
// ReadGuarded conversion enforces on the open fd (O_NOFOLLOW), closing the
// lstat→read TOCTOU — distinct from the graveyard/-directory symlink guard.
func TestIngestLessonsSymlinkedLayerFileRefused(t *testing.T) {
	dir := stdFixture(t)
	archPath := filepath.Join(dir, "graveyard", "archaeology.json")
	// Replace the real layer file with a symlink to an out-of-lifeboat target.
	outside := filepath.Join(t.TempDir(), "attacker.json")
	writeFile(t, outside, marshalIndent(t, stdArch()))
	if err := os.Remove(archPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, archPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	raw := payload(t, Lesson{ID: "les-x", Lesson: "y", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	_, err := IngestLessons(dir, raw)
	if err == nil {
		t.Fatal("a symlinked graveyard layer file must be refused, not followed")
	}
	if strings.Contains(err.Error(), "too many levels of symbolic links") {
		t.Errorf("symlink refusal leaked the raw ELOOP syscall detail: %v", err)
	}
}

// TestIngestLessonsDeterministic proves two ingests of the same payload write a
// byte-identical lessons.json, with entries sorted by id.
func TestIngestLessonsDeterministic(t *testing.T) {
	build := func() []byte {
		dir := stdFixture(t)
		raw := payload(t,
			Lesson{ID: "les-zeta", Lesson: "z", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}},
			Lesson{ID: "les-alpha", Lesson: "a", Confidence: ConfidenceHigh, Evidence: []string{"rev-9f3a1c2d4e5b"}},
		)
		if _, err := IngestLessons(dir, raw); err != nil {
			t.Fatalf("IngestLessons: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "graveyard", "lessons.json"))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	a, b := build(), build()
	if !bytes.Equal(a, b) {
		t.Errorf("lessons.json not deterministic:\n%s\n---\n%s", a, b)
	}
	// Sorted by id: les-alpha precedes les-zeta.
	if i, j := bytes.Index(a, []byte("les-alpha")), bytes.Index(a, []byte("les-zeta")); i < 0 || j < 0 || i > j {
		t.Errorf("lessons not sorted by id:\n%s", a)
	}
}

// TestIngestLessonsManifestNotPerturbed proves the layer-3 write does not perturb
// the pinned manifest: the manifest hash over the layer-1/2 files is stable, and
// the two files are byte-identical after the lessons write (they are never in the
// hash, and they are never touched).
func TestIngestLessonsManifestNotPerturbed(t *testing.T) {
	dir := stdFixture(t)
	archPath := filepath.Join(dir, "graveyard", "archaeology.json")
	abandPath := filepath.Join(dir, "graveyard", "abandoned.json")
	archBefore, _ := os.ReadFile(archPath)
	abandBefore, _ := os.ReadFile(abandPath)

	planned := []PlannedFile{
		{Path: "graveyard/archaeology.json", Content: archBefore},
		{Path: "graveyard/abandoned.json", Content: abandBefore},
	}
	h1 := ManifestSHA256(planned)

	raw := payload(t, Lesson{ID: "les-x", Lesson: "y", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw); err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if !lessonsExists(dir) {
		t.Fatal("expected lessons.json to be written")
	}

	archAfter, _ := os.ReadFile(archPath)
	abandAfter, _ := os.ReadFile(abandPath)
	if !bytes.Equal(archBefore, archAfter) || !bytes.Equal(abandBefore, abandAfter) {
		t.Fatal("the lessons write mutated a layer-1/2 file")
	}
	h2 := ManifestSHA256([]PlannedFile{
		{Path: "graveyard/archaeology.json", Content: archAfter},
		{Path: "graveyard/abandoned.json", Content: abandAfter},
	})
	if h1 != h2 {
		t.Errorf("manifest perturbed: %s != %s", h1, h2)
	}
}

// TestLessonsResultRender is the deterministic text render.
func TestLessonsResultRender(t *testing.T) {
	r := LessonsResult{LifeboatDir: "/lb", Written: 3, LowConfidence: 1, Dropped: 2,
		Drops: []LessonDrop{{ID: "les-guess", Reason: "no valid evidence refs"}, {ID: "les-bad", Reason: "malformed lesson id"}}}
	out := r.Render()
	for _, want := range []string{"graveyard lessons for /lb", "written:", "3", "low-confidence:", "dropped:", "les-guess (no valid evidence refs)"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
}

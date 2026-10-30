package lifeboat

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The four verbs that write host-produced text into an already-packed lifeboat
// (graveyard lessons, principles, press release, review) secret-scan the bytes
// they plan to write before any write or removal, as pack's planned bytes are,
// and refuse the whole call on a finding: refuse, never redact. These tests
// drive the core with a stub scan keyed on a marker string, so they do not
// depend on the scanner adapter; the CLI test proves the real scanner end to end.

// plantedMarker stands in for a secret in a host payload. It is plain
// alphanumeric so every prose cleaner on the way leaves it intact.
const plantedMarker = "plantedmarker7Q4z"

var errPlanted = errors.New("planted marker found in planned content")

// markerScan refuses when any planned file's content carries plantedMarker.
func markerScan(files []PlannedFile) error {
	for _, f := range files {
		if bytes.Contains(f.Content, []byte(plantedMarker)) {
			return errPlanted
		}
	}
	return nil
}

// recordingScan records the planned paths it was handed and finds nothing.
func recordingScan(seen *[]string) SecretScan {
	return func(files []PlannedFile) error {
		for _, f := range files {
			*seen = append(*seen, f.Path)
		}
		return nil
	}
}

// snapshotTree returns every regular file under dir by slash path with its
// bytes, plus every directory, so a refused call can be proved to have changed
// nothing at all (no write, no removal, no new directory).
func snapshotTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			out[rel+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", dir, err)
	}
	return out
}

func assertTreeUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	var diffs []string
	for k, v := range before {
		if w, ok := after[k]; !ok {
			diffs = append(diffs, "removed "+k)
		} else if w != v {
			diffs = append(diffs, "changed "+k)
		}
	}
	for k := range after {
		if _, ok := before[k]; !ok {
			diffs = append(diffs, "added "+k)
		}
	}
	sort.Strings(diffs)
	if len(diffs) > 0 {
		t.Errorf("a refused call changed the lifeboat:\n  %s", strings.Join(diffs, "\n  "))
	}
}

// --- graveyard lessons ------------------------------------------------------

func TestIngestLessonsScanRefusesPlantedToken(t *testing.T) {
	for _, tc := range []struct {
		name       string
		confidence Confidence
	}{
		{"main lessons file", ConfidenceHigh},
		{"low-confidence file", ConfidenceLow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := stdFixture(t)
			// A prior interpretation, high and low, which a refused ingest must
			// NOT clear.
			prior := payload(t,
				Lesson{ID: "les-prior", Lesson: "the prior lesson", Confidence: ConfidenceHigh,
					Evidence: []string{"rev-9f3a1c2d4e5b"}},
				Lesson{ID: "les-prior-low", Lesson: "the prior low lesson", Confidence: ConfidenceLow,
					Evidence: []string{"adr-12"}})
			if _, err := IngestLessons(dir, prior, okScan); err != nil {
				t.Fatalf("seed ingest: %v", err)
			}
			before := snapshotTree(t, dir)

			raw := payload(t,
				Lesson{ID: "les-clean", Lesson: "a clean lesson", Confidence: ConfidenceHigh,
					Evidence: []string{"rev-9f3a1c2d4e5b"}},
				Lesson{ID: "les-carrier", Lesson: "carries " + plantedMarker + " in its prose",
					Confidence: tc.confidence, Evidence: []string{"adr-12"}})
			_, err := IngestLessons(dir, raw, markerScan)
			if !errors.Is(err, errPlanted) {
				t.Fatalf("IngestLessons err = %v, want the scan's refusal", err)
			}
			assertTreeUnchanged(t, before, snapshotTree(t, dir))
		})
	}
}

func TestIngestLessonsScanRefusalOnFreshLifeboatWritesNothing(t *testing.T) {
	dir := stdFixture(t)
	before := snapshotTree(t, dir)
	raw := payload(t, Lesson{ID: "les-carrier", Lesson: "carries " + plantedMarker,
		Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw, markerScan); !errors.Is(err, errPlanted) {
		t.Fatalf("IngestLessons err = %v, want the scan's refusal", err)
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
	if lessonsExists(dir) {
		t.Error("lessons.json written despite the refusal")
	}
}

func TestIngestLessonsScanSeesEveryPlannedFile(t *testing.T) {
	dir := stdFixture(t)
	var seen []string
	raw := payload(t,
		Lesson{ID: "les-high", Lesson: "high", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}},
		Lesson{ID: "les-low", Lesson: "low", Confidence: ConfidenceLow, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw, recordingScan(&seen)); err != nil {
		t.Fatal(err)
	}
	sort.Strings(seen)
	want := []string{"graveyard/lessons.json", "graveyard/low-confidence/les-low.json"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Errorf("scanned %v, want %v", seen, want)
	}
}

func TestIngestLessonsNilScanRefused(t *testing.T) {
	dir := stdFixture(t)
	before := snapshotTree(t, dir)
	raw := payload(t, Lesson{ID: "les-x", Lesson: "x", Confidence: ConfidenceHigh, Evidence: []string{"adr-12"}})
	if _, err := IngestLessons(dir, raw, nil); err == nil {
		t.Fatal("a nil scan must be refused")
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

// --- principles -------------------------------------------------------------

func TestSynthesizePrinciplesScanRefusesPlantedToken(t *testing.T) {
	dir := adrLifeboat(t)
	if _, err := SynthesizePrinciples(dir, nil, okScan); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	before := snapshotTree(t, dir)
	pay := principlesPayload(t, "0.1.0",
		Principle{ID: "prn-carrier", Principle: "Carries " + plantedMarker + " in its prose.",
			Confidence: ConfidenceHigh, Evidence: []string{"adr-24"}})
	if _, err := SynthesizePrinciples(dir, pay, markerScan); !errors.Is(err, errPlanted) {
		t.Fatalf("SynthesizePrinciples err = %v, want the scan's refusal", err)
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

func TestSynthesizePrinciplesScanSeesBothFiles(t *testing.T) {
	dir := adrLifeboat(t)
	var seen []string
	if _, err := SynthesizePrinciples(dir, nil, recordingScan(&seen)); err != nil {
		t.Fatal(err)
	}
	sort.Strings(seen)
	if strings.Join(seen, ",") != "principles.json,principles.md" {
		t.Errorf("scanned %v, want principles.json and principles.md", seen)
	}
}

func TestSynthesizePrinciplesNilScanRefused(t *testing.T) {
	dir := adrLifeboat(t)
	before := snapshotTree(t, dir)
	if _, err := SynthesizePrinciples(dir, nil, nil); err == nil {
		t.Fatal("a nil scan must be refused")
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

// --- press release ----------------------------------------------------------

func TestComposePressReleaseScanRefusesPlantedToken(t *testing.T) {
	dir := synthLifeboat(t, map[string]string{briefPressReleasePath: briefPR})
	if _, err := ComposePressRelease(dir, nil, okScan); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	before := snapshotTree(t, dir)
	pay := pressReleasePayload(t, PressReleaseFile{
		Mode:          ModeDelegated,
		PromptVersion: "0.1.0",
		Headline:      "A composed headline.",
		Body:          "A body that carries " + plantedMarker + " in it.",
		Evidence:      []string{briefPressReleasePath},
	})
	if _, err := ComposePressRelease(dir, pay, markerScan); !errors.Is(err, errPlanted) {
		t.Fatalf("ComposePressRelease err = %v, want the scan's refusal", err)
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

func TestComposePressReleaseScanSeesBothFiles(t *testing.T) {
	dir := synthLifeboat(t, map[string]string{briefPressReleasePath: briefPR})
	var seen []string
	if _, err := ComposePressRelease(dir, nil, recordingScan(&seen)); err != nil {
		t.Fatal(err)
	}
	sort.Strings(seen)
	if strings.Join(seen, ",") != "press-release.json,press-release.md" {
		t.Errorf("scanned %v, want press-release.json and press-release.md", seen)
	}
}

func TestComposePressReleaseNilScanRefused(t *testing.T) {
	dir := synthLifeboat(t, map[string]string{briefPressReleasePath: briefPR})
	before := snapshotTree(t, dir)
	if _, err := ComposePressRelease(dir, nil, nil); err == nil {
		t.Fatal("a nil scan must be refused")
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

// --- review -----------------------------------------------------------------

func TestReviewLifeboatScanRefusesPlantedToken(t *testing.T) {
	dir := reviewFixture(t, "001", &Summary{Grounded: 5, Partial: 1, Blank: 2})
	if _, err := ReviewLifeboat(dir, realSourceDir(t), nil, okScan); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	before := snapshotTree(t, dir)
	pay := reviewPayloadJSON("SHIP",
		`{"id":"fnd-carrier","finding":"carries `+plantedMarker+` in it","evidence":["coverage.json"]}`)
	if _, err := ReviewLifeboat(dir, realSourceDir(t), pay, markerScan); !errors.Is(err, errPlanted) {
		t.Fatalf("ReviewLifeboat err = %v, want the scan's refusal", err)
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

func TestReviewLifeboatScanSeesBothFiles(t *testing.T) {
	dir := reviewFixture(t, "001", &Summary{Grounded: 5, Partial: 1, Blank: 2})
	var seen []string
	res, err := ReviewLifeboat(dir, realSourceDir(t), nil, recordingScan(&seen))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(seen)
	if strings.Join(seen, ",") != res.ReviewPath+","+res.RenderPath {
		t.Errorf("scanned %v, want %s and %s", seen, res.ReviewPath, res.RenderPath)
	}
}

func TestReviewLifeboatNilScanRefused(t *testing.T) {
	dir := reviewFixture(t, "001", &Summary{Grounded: 5, Partial: 1, Blank: 2})
	before := snapshotTree(t, dir)
	if _, err := ReviewLifeboat(dir, realSourceDir(t), nil, nil); err == nil {
		t.Fatal("a nil scan must be refused")
	}
	assertTreeUnchanged(t, before, snapshotTree(t, dir))
}

package lifeboat

import (
	"os"
	"path/filepath"
	"testing"
)

// Every lifeboat verb's report travels into --json, and machine output never
// carries an absolute developer-identity path (iss-81). A lifeboat, a target
// or a destination under the home directory is reported with the home redacted
// to "~" (iss-2609261848326365).
func TestLifeboatReportsNameDirectoriesWithoutTheHomePath(t *testing.T) {
	src := embarkableSourceFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	lb := filepath.Join(home, "lifeboat")
	target := filepath.Join(home, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	check := func(what, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", what, got, want)
		}
	}

	pack, err := Pack(src, lb, okScan)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	check("PackResult.Dest", pack.Dest, "~/lifeboat")

	plan, err := EmbarkProbe(lb, target)
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	check("EmbarkPlan.LifeboatDir", plan.LifeboatDir, "~/lifeboat")
	check("EmbarkPlan.TargetDir", plan.TargetDir, "~/target")

	res, err := EmbarkFrom(lb, target)
	if err != nil {
		t.Fatalf("EmbarkFrom: %v", err)
	}
	check("EmbarkResult.LifeboatDir", res.LifeboatDir, "~/lifeboat")
	check("EmbarkResult.TargetDir", res.TargetDir, "~/target")

	pr, err := SynthesizePrinciples(lb, nil)
	if err != nil {
		t.Fatalf("SynthesizePrinciples: %v", err)
	}
	check("PrinciplesResult.LifeboatDir", pr.LifeboatDir, "~/lifeboat")

	press, err := ComposePressRelease(lb, nil)
	if err != nil {
		t.Fatalf("ComposePressRelease: %v", err)
	}
	check("PressReleaseResult.LifeboatDir", press.LifeboatDir, "~/lifeboat")

	rev, err := ReviewLifeboat(lb, src, nil)
	if err != nil {
		t.Fatalf("ReviewLifeboat: %v", err)
	}
	check("ReviewResult.LifeboatDir", rev.LifeboatDir, "~/lifeboat")

	grave := filepath.Join(home, "graveyard-lifeboat")
	copyTree(t, stdFixture(t), grave)
	les, err := IngestLessons(grave, payload(t, Lesson{ID: "les-engine-v1", Lesson: "engine v1 was retired",
		Confidence: ConfidenceHigh, Evidence: []string{"rev-9f3a1c2d4e5b"}}))
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	check("LessonsResult.LifeboatDir", les.LifeboatDir, "~/graveyard-lifeboat")
}

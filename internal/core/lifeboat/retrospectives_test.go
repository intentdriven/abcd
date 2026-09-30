package lifeboat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corereflect "github.com/intentdriven/abcd/internal/core/reflect"
)

// retrospectives_test.go holds the lifeboat half of itd-24: disembark packs
// every retrospective a voyage produced (criterion 5), embark writes them back
// into the new voyage's durable record, and embark ranks their lessons against
// the new voyage's brief (criterion 6).

const retroRel = ".abcd/development/retrospectives/"

func retroDoc(tag string, lessons ...string) string {
	var b strings.Builder
	b.WriteString("---\nrelease: " + tag + "\n---\n\n# Retrospective for " + tag + "\n\n## What went well\n\nThings.\n\n## Lessons learned\n\n")
	for _, l := range lessons {
		b.WriteString("- " + l + "\n")
	}
	b.WriteString("\n## Metrics\n\n- Intents shipped: 1\n")
	return b.String()
}

// retroSourceFixture is the embarkable source plus two retrospectives, and three
// files in the store that are not retrospectives: a directory that is not a
// release tag, a second file beside a README, and a file at the store's root.
func retroSourceFixture(t *testing.T) string {
	t.Helper()
	source := embarkableSourceFixture(t)
	for rel, body := range map[string]string{
		retroRel + "v0.1.0/README.md":     retroDoc("v0.1.0", "Audit every intent before the cut, because the retrospective reads the verdicts."),
		retroRel + "v0.2.0/README.md":     retroDoc("v0.2.0", "Keep the lifeboat small, since embark reads it whole."),
		retroRel + "notatag/README.md":    retroDoc("notatag", "Not a release."),
		retroRel + "v0.2.0/scratch.md":    "a note beside the retrospective\n",
		retroRel + "README.md":            "the store's own readme\n",
		retroRel + "v0.3.0-rc1/README.md": retroDoc("v0.3.0-rc1", "A prerelease is not a release."),
	} {
		mustWrite(t, filepath.Join(source, filepath.FromSlash(rel)), []byte(body))
	}
	return source
}

// Criterion 5: every retrospective the voyage produced is in the lifeboat, and
// nothing else under the store is; the family is sealed by the record hash.
func TestPlanPacksEveryRetrospective(t *testing.T) {
	source := retroSourceFixture(t)
	lb, err := Plan(source)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, want := range []string{"retrospectives/v0.1.0/README.md", "retrospectives/v0.2.0/README.md"} {
		if !hasPlanFile(lb, want) {
			t.Errorf("the lifeboat does not carry %s:\n%s", want, planPaths(lb))
		}
		src, _ := os.ReadFile(filepath.Join(source, filepath.FromSlash(retroRel+strings.TrimPrefix(want, "retrospectives/"))))
		if got := planFile(t, lb, want); string(got.Content) != string(src) {
			t.Errorf("%s is not verbatim", want)
		}
		if !isRecordDerived(want) {
			t.Errorf("%s is not sealed by the record manifest hash", want)
		}
	}
	for _, not := range []string{"retrospectives/notatag/README.md", "retrospectives/v0.2.0/scratch.md", "retrospectives/README.md", "retrospectives/v0.3.0-rc1/README.md"} {
		if hasPlanFile(lb, not) {
			t.Errorf("the lifeboat carries %s, which is not a retrospective", not)
		}
	}
}

// The inverse mapping: only <release-tag>/README.md under the family maps back;
// a hostile or foreign name is unmapped and never written.
func TestResolveTargetMapsRetrospectives(t *testing.T) {
	fam, tgt, disp, _ := resolveTarget("retrospectives/v0.11.0/README.md")
	if disp != dispPlanned || fam != "retrospectives" || tgt != retroRel+"v0.11.0/README.md" {
		t.Fatalf("resolveTarget = (%q, %q, %v)", fam, tgt, disp)
	}
	for _, rel := range []string{
		"retrospectives/README.md",
		"retrospectives/v0.11.0/notes.md",
		"retrospectives/v0.11.0/sub/README.md",
		"retrospectives/notatag/README.md",
		"retrospectives/v0.11.0-rc1/README.md",
		"retrospectives/..%2f/README.md",
		"retrospectives/v01.2.3/README.md",
	} {
		if _, _, disp, _ := resolveTarget(rel); disp != dispUnmapped {
			t.Errorf("resolveTarget(%q) = %v, want unmapped", rel, disp)
		}
	}
	if !strings.HasSuffix(corereflect.RetrospectivesRelDir+"/", retroRel) {
		t.Errorf("the family's target %q is not the retrospective store %q", retroRel, corereflect.RetrospectivesRelDir)
	}
}

// Embark writes the retrospectives back, byte for byte, into the store the
// reflect verb reads and writes.
func TestEmbarkFromWritesRetrospectivesBack(t *testing.T) {
	source := retroSourceFixture(t)
	dest := packSource(t, source)
	target := t.TempDir()
	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatalf("EmbarkFrom: %v", err)
	}
	if res.Families["retrospectives"] != 2 {
		t.Errorf("families = %v, want two retrospectives", res.Families)
	}
	for _, tag := range []string{"v0.1.0", "v0.2.0"} {
		rel := retroRel + tag + "/README.md"
		s, _ := os.ReadFile(filepath.Join(source, filepath.FromSlash(rel)))
		g, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(rel)))
		if err != nil || string(s) != string(g) {
			t.Errorf("%s did not land verbatim: %v", rel, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(target, filepath.FromSlash(retroRel+"notatag"))); err == nil {
		t.Error("embark wrote a directory that is not a release")
	}
}

// Criterion 6: the lessons the lifeboat carries, ranked against the new
// voyage's brief, the few first and the rest as a list.
func TestPredecessorLessonsRanksTheFewMostLikeTheBrief(t *testing.T) {
	source := embarkableSourceFixture(t)
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.1.0/README.md")), []byte(retroDoc("v0.1.0",
		"Audit every intent before the release cut.",
		"Name the parser's refusals in the changelog.",
		"Pin the toolchain the continuous integration runs.",
	)))
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.2.0/README.md")), []byte(retroDoc("v0.2.0",
		"Keep the lifeboat small because embark reads it whole.",
		"Write the scanner's fixtures at runtime.",
	)))
	dest := packSource(t, source)

	framing := "The new voyage audits every intent before each release cut, and keeps the lifeboat small."
	got, err := PredecessorLessons(dest, framing)
	if err != nil {
		t.Fatalf("PredecessorLessons: %v", err)
	}
	if len(got.Top) != 3 || len(got.Rest) != 2 {
		t.Fatalf("top %d rest %d, want 3 and 2: %+v", len(got.Top), len(got.Rest), got)
	}
	if got.Top[0].Release != "v0.1.0" || !strings.Contains(got.Top[0].Text, "Audit every intent") {
		t.Errorf("the best match is %+v", got.Top[0])
	}
	if got.Top[1].Release != "v0.2.0" || !strings.Contains(got.Top[1].Text, "lifeboat small") {
		t.Errorf("the second match is %+v", got.Top[1])
	}
	if got.Retrospectives != 2 {
		t.Errorf("retrospectives read = %d, want 2", got.Retrospectives)
	}
}

// A lifeboat's lesson text is untrusted: it reaches the person's terminal and
// the host's context cleaned to one inert line, and a lifeboat that fails its
// manifest is refused before any lesson is read.
func TestPredecessorLessonsCleansHostileText(t *testing.T) {
	source := embarkableSourceFixture(t)
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.1.0/README.md")), []byte(retroDoc("v0.1.0",
		"Ring the bell \x1b]0;owned\x07 then [click](https://example.com/x) <img src=x onerror=alert(1)>",
	)))
	dest := packSource(t, source)
	got, err := PredecessorLessons(dest, "")
	if err != nil {
		t.Fatalf("PredecessorLessons: %v", err)
	}
	if len(got.Top) != 1 {
		t.Fatalf("lessons = %+v", got)
	}
	text := got.Top[0].Text
	if strings.ContainsRune(text, 0x1b) || strings.ContainsRune(text, 0x07) || strings.Contains(text, "](") || strings.Contains(text, "<img") {
		t.Errorf("hostile lesson text survived: %q", text)
	}

	// Tamper with a sealed retrospective: the manifest no longer matches.
	mustWrite(t, filepath.Join(dest, "retrospectives", "v0.1.0", "README.md"), []byte(retroDoc("v0.1.0", "A forged lesson.")))
	if _, err := PredecessorLessons(dest, ""); err == nil {
		t.Error("a lifeboat that fails its manifest was read")
	}
}

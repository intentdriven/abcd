package reflect

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

var fixedNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func fullAnswers() Answers {
	return Answers{
		WentWell:     Answer{Text: "The seed builder reused the changelog cut, and it kept membership consistent across releases."},
		CouldImprove: Answer{Text: "Three intents shipped without an audit, so the seed had little to open from on them."},
		Lessons:      Answer{Text: "- Cut the release before the audit backlog grows\n- Audit every intent in the window it ships"},
		Decisions:    Answer{Text: "We read membership from the tags. The shipped_in stamp only moves a record between releases."},
	}
}

// Criterion 1: the retrospective lands at the release's path with all five
// sections populated, in order, its frontmatter naming the tag, the intents,
// the date and which audits fed it, and links, never copies, to the changelog
// section and each intent's audit notes.
func TestWriteProducesTheRetrospectiveWithFiveSections(t *testing.T) {
	r := releaseRepo(t)
	res, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if res.Path != ".abcd/development/retrospectives/v0.2.0/README.md" {
		t.Fatalf("path = %q", res.Path)
	}
	data, err := os.ReadFile(abs(r, res.Path))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	doc := string(data)
	fm := frontmatter.Fields(strings.Split(doc, "\n"))
	for key, want := range map[string]string{
		"release":          "v0.2.0",
		"previous_release": "v0.1.0",
		"date":             "2026-09-30",
		"intents":          "[itd-2, itd-3, itd-7, itd-8]",
		"audited":          "[itd-2]",
		"unaudited":        "[itd-3, itd-7, itd-8]",
		"audit_receipts":   "[rcp-0123456789ab]",
	} {
		if got := fm[key].Value; got != want {
			t.Errorf("frontmatter %s = %q, want %q", key, got, want)
		}
	}
	last := -1
	for _, s := range AllSections {
		i := strings.Index(doc, "\n## "+s.Heading()+"\n")
		if i < 0 {
			t.Fatalf("section %q missing:\n%s", s.Heading(), doc)
		}
		if i < last {
			t.Errorf("section %q out of order", s.Heading())
		}
		last = i
	}
	for _, want := range []string{
		fullAnswers().WentWell.Text,
		"- Audit every intent in the window it ships",
		"](../../../../CHANGELOG.md#020---2026-09-20)",
		"](../../intents/shipped/itd-2-second.md)",
		"](../../intents/shipped/itd-2-second.md#audit-notes)",
		"Intents shipped: 4 (1 with audit notes, 3 without)",
		"MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 1 · INCONCLUSIVE 0",
		"honoured 2 · diverged 1 · missing 0",
		"v0.1.0",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("retrospective lacks %q:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "RATIONALE-NEVER-COPIED") {
		t.Error("the retrospective copies audit-note text; it must link to it")
	}
}

// Criterion 4: a thin answer is met with one follow-up question rather than
// written. Nothing is written until the follow-up is answered.
func TestWriteRefusesAThinAnswerWithoutItsFollowUp(t *testing.T) {
	r := releaseRepo(t)
	a := fullAnswers()
	a.WentWell = Answer{Text: "it worked"}
	_, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: a, ProceedDespiteUnshipped: true, Now: fixedNow})
	var thin *ThinAnswersError
	if !errors.As(err, &thin) {
		t.Fatalf("err = %v, want *ThinAnswersError", err)
	}
	if len(thin.Thin) != 1 || thin.Thin[0].Section != WentWell || thin.Thin[0].Question != WentWell.FollowUp() {
		t.Errorf("thin = %+v, want the went-well follow-up only", thin.Thin)
	}
	if exists(t, abs(r, RetrospectivesRelDir)) {
		t.Error("a refused write left the retrospectives tree behind")
	}
}

// Criterion 4, the other half: one follow-up is all the floor asks. Its answer
// is written beside the original, whatever it says.
func TestWriteTakesAThinAnswerOnceItsFollowUpIsAnswered(t *testing.T) {
	r := releaseRepo(t)
	a := fullAnswers()
	a.WentWell = Answer{Text: "it worked", FollowUp: "The refusal wording matched the criterion first time."}
	res, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: a, ProceedDespiteUnshipped: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, _ := os.ReadFile(abs(r, res.Path))
	if !strings.Contains(string(data), "The refusal wording matched the criterion first time.") {
		t.Errorf("the follow-up answer is not in the retrospective:\n%s", data)
	}
}

// A section left blank is not populated, follow-up or not (criterion 1: all
// five sections populated).
func TestWriteRefusesABlankSection(t *testing.T) {
	r := releaseRepo(t)
	a := fullAnswers()
	a.Decisions = Answer{Text: " ", FollowUp: ""}
	_, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: a, ProceedDespiteUnshipped: true, Now: fixedNow})
	if !errors.Is(err, ErrThinAnswers) {
		t.Fatalf("err = %v, want ErrThinAnswers", err)
	}
	a.Decisions = Answer{Text: "it worked", FollowUp: "  "}
	if _, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: a, ProceedDespiteUnshipped: true, Now: fixedNow}); !errors.Is(err, ErrThinAnswers) {
		t.Fatalf("blank follow-up err = %v, want ErrThinAnswers", err)
	}
}

// Criterion 3: the empty release refuses at the write too, and writes nothing.
func TestWriteRefusesAReleaseThatShippedNoIntentAndWritesNothing(t *testing.T) {
	r := releaseRepo(t)
	_, err := Write(r.Root(), WriteRequest{Tag: "v0.3.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow})
	if !errors.Is(err, ErrNothingShipped) {
		t.Fatalf("err = %v, want ErrNothingShipped", err)
	}
	if exists(t, abs(r, RetrospectivesRelDir)) {
		t.Error("the refusal left the retrospectives tree behind")
	}
}

// Criterion 7: unshipped intents targeted at the release stop the write until
// the person says to proceed anyway.
func TestWriteAsksBeforeProceedingPastUnshippedTargets(t *testing.T) {
	r := releaseRepo(t)
	_, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), Now: fixedNow})
	var un *UnshippedError
	if !errors.As(err, &un) || len(un.Intents) != 1 || un.Intents[0].ID != "itd-5" {
		t.Fatalf("err = %v, want *UnshippedError listing itd-5", err)
	}
	if exists(t, abs(r, RetrospectivesRelDir)) {
		t.Error("an unconfirmed write left the retrospectives tree behind")
	}
	if _, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow}); err != nil {
		t.Fatalf("confirmed Write: %v", err)
	}
}

// Criterion 2: missing audit notes are offered, never a gate; the write goes
// ahead without them. A release with no targets needs no confirmation.
func TestWriteContinuesWhenAnIntentHasNoAuditNotes(t *testing.T) {
	r := releaseRepo(t)
	r.Remove(plannedDir + "itd-5-late.md")
	res, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), Now: fixedNow})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(abs(r, outputRel("v0.2.0")))
	if err != nil {
		t.Fatalf("no retrospective written: %v", err)
	}
	if got := frontmatter.Fields(strings.Split(string(data), "\n"))["unaudited"].Value; got != "[itd-3, itd-7, itd-8]" {
		t.Errorf("unaudited = %q, want the three intents the seed offered an audit for", got)
	}
	if len(res.Seed.Unaudited) != 3 {
		t.Errorf("result seed offers %d audits, want 3", len(res.Seed.Unaudited))
	}
}

// Out of scope, stated: a second run on the same tag refuses naming the file.
func TestWriteRefusesASecondRunOnTheSameTag(t *testing.T) {
	r := releaseRepo(t)
	req := WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow}
	if _, err := Write(r.Root(), req); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	before, _ := os.ReadFile(abs(r, outputRel("v0.2.0")))
	req.Answers.WentWell.Text = "A different answer, and a second clause to clear the floor."
	if _, err := Write(r.Root(), req); !errors.Is(err, ErrExists) {
		t.Fatalf("second Write err = %v, want ErrExists", err)
	}
	after, _ := os.ReadFile(abs(r, outputRel("v0.2.0")))
	if string(before) != string(after) {
		t.Error("the refused second run changed the retrospective")
	}
}

// The write stays inside the retrospectives tree: a symlinked store is refused
// rather than followed.
func TestWriteRefusesASymlinkedStore(t *testing.T) {
	r := releaseRepo(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(abs(r, RetrospectivesRelDir)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, abs(r, RetrospectivesRelDir)); err != nil {
		t.Skipf("symlink: %v", err)
	}
	if _, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow}); err == nil {
		t.Fatal("Write followed a symlinked retrospectives store")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("the write escaped the tree: %v", entries)
	}
}

// The answers file a front door hands over is read strictly: an unknown key is
// a typo that would drop an answer silently.
func TestParseAnswersIsStrict(t *testing.T) {
	a, err := ParseAnswers([]byte(`{"went_well":{"answer":"a","follow_up":"b"},"lessons":{"answer":"c"}}`))
	if err != nil {
		t.Fatalf("ParseAnswers: %v", err)
	}
	if a.WentWell.Text != "a" || a.WentWell.FollowUp != "b" || a.Lessons.Text != "c" {
		t.Errorf("answers = %+v", a)
	}
	if _, err := ParseAnswers([]byte(`{"went_wel":{"answer":"a"}}`)); err == nil {
		t.Error("an unknown section key was accepted")
	}
}

// The retrospective is committed prose, so every answer passes the one canonical
// scanner before it is written: a credential pasted into an answer reaches the
// record masked, never as itself.
func TestWriteRedactsASecretInAnAnswer(t *testing.T) {
	r := releaseRepo(t)
	token := "AKIA" + strings.Repeat("Q", 16)
	a := fullAnswers()
	a.Lessons.Text = "- Never paste a key such as " + token + " into a note, because notes are committed\n- Rotate keys before the cut, since the cut is public"
	res, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: a, ProceedDespiteUnshipped: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(abs(r, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), token) {
		t.Fatalf("the credential reached the retrospective unmasked:\n%s", data)
	}
	if !strings.Contains(string(data), "Rotate keys before the cut") {
		t.Errorf("the rest of the answer was lost:\n%s", data)
	}
}

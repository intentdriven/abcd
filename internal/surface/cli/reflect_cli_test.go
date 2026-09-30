package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// reflect_cli_test.go holds `abcd reflect` to itd-24's criteria at the front
// door: the seed a retrospective opens from, the refusals, the write, and the
// help placement. The core's own tests (internal/core/reflect) hold the seed's
// arithmetic; these hold what the person and the host see.

const (
	reflectShipped = ".abcd/development/intents/shipped/"
	reflectPlanned = ".abcd/development/intents/planned/"
	reflectAudited = "## Audit Notes\n\n<!-- abcd-review: INGESTED receipt=rcp-0123456789ab -->\n" +
		"Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 1 · INCONCLUSIVE 0\n"
	reflectEmpty = "## Audit Notes\n\n_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._\n"
)

func reflectIntent(id, title, extraFM, notes string) string {
	return "---\nid: " + id + "\nimpact: additive\n" + extraFM + "---\n\n# " + title + "\n\n## Press Release\n\n> A line.\n\n" + notes
}

// reflectRepo is a history of three releases: v0.1.0 ships itd-1; v0.2.0 ships
// itd-2 (audited) and itd-3 (no audit notes), while itd-5, targeted at v0.2.0,
// is still planned; v0.3.0 ships nothing.
func reflectRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(reflectShipped+"itd-1-first.md", reflectIntent("itd-1", "The first promise", "", reflectAudited))
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-09-01\n\n- The first promise (itd-1).\n")
	r.Commit("first release")
	r.Git("tag", "v0.1.0")
	r.Write(reflectShipped+"itd-2-second.md", reflectIntent("itd-2", "The second promise", "", reflectAudited))
	r.Write(reflectShipped+"itd-3-third.md", reflectIntent("itd-3", "The third promise", "", reflectEmpty))
	r.Write(reflectPlanned+"itd-5-late.md", "---\nid: itd-5\ntarget_release: v0.2.0\n---\n\n# A promise that missed its release\n")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n## [0.2.0] - 2026-09-20\n\n- The second promise (itd-2).\n\n## [0.1.0] - 2026-09-01\n\n- The first promise (itd-1).\n")
	r.Commit("second release")
	r.Git("tag", "v0.2.0")
	r.Commit("a release that shipped nothing")
	r.Git("tag", "v0.3.0")
	return r
}

func reflectIn(t *testing.T, r *gittest.Repo, args ...string) ([]byte, error) {
	t.Helper()
	t.Chdir(r.Root())
	return runCLIErr(t, append([]string{"reflect"}, args...)...)
}

// fullAnswers answers the four asked sections above the floor.
const fullAnswers = `{
  "went_well": {"answer": "The seed builder read the release from the tag tree, and the refusals named the tag every time."},
  "could_improve": {"answer": "Two intents shipped without audit notes, so the verdict counts undercount the release."},
  "lessons": {"answer": "- Run the audit before the cut, because the retrospective reads its verdicts.\n- Keep the release notes short, since the page is read once."},
  "decisions": {"answer": "The release is the unit of reflection, and the seed reads the intents that reached shipped between two tags."}
}`

func writeAnswers(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func retroPath(r *gittest.Repo, tag string) string {
	return filepath.Join(r.Root(), ".abcd", "development", "retrospectives", tag, "README.md")
}

// TestReflectSeedRendersTheRelease is criteria 1, 2 and 7 at the front door: the
// seed names what the release shipped, offers the audit for the intent that has
// no audit notes, and warns about the targeted intent still unshipped.
func TestReflectSeedRendersTheRelease(t *testing.T) {
	r := reflectRepo(t)
	out, err := reflectIn(t, r, "v0.2.0")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	for _, want := range []string{
		"itd-2", "The second promise", "itd-3",
		"abcd intent audit itd-3",
		"itd-5",
		".abcd/development/retrospectives/v0.2.0/README.md",
		"previous release v0.1.0",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the seed does not mention %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "itd-1 ") {
		t.Errorf("the seed names itd-1, which v0.1.0 shipped:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(r.Root(), ".abcd", "development", "retrospectives")); err == nil {
		t.Error("rendering the seed created the retrospectives store")
	}
}

// TestReflectSeedJSON is the host's view: the seed as one JSON object.
func TestReflectSeedJSON(t *testing.T) {
	r := reflectRepo(t)
	out, err := reflectIn(t, r, "v0.2.0", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	var seed struct {
		Tag     string `json:"tag"`
		Intents []struct {
			ID string `json:"id"`
		} `json:"intents"`
		Unaudited []struct {
			Command string `json:"command"`
		} `json:"unaudited"`
		Unshipped []struct {
			ID string `json:"id"`
		} `json:"unshipped_targets"`
		Questions []struct {
			Section  string `json:"section"`
			Question string `json:"question"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(out, &seed); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if seed.Tag != "v0.2.0" || len(seed.Intents) != 2 || seed.Intents[0].ID != "itd-2" || seed.Intents[1].ID != "itd-3" {
		t.Errorf("seed = %+v", seed)
	}
	if len(seed.Unaudited) != 1 || seed.Unaudited[0].Command != "abcd intent audit itd-3" {
		t.Errorf("unaudited = %+v", seed.Unaudited)
	}
	if len(seed.Unshipped) != 1 || seed.Unshipped[0].ID != "itd-5" {
		t.Errorf("unshipped = %+v", seed.Unshipped)
	}
	if len(seed.Questions) != 4 || seed.Questions[0].Section != "went_well" || seed.Questions[0].Question == "" {
		t.Errorf("the seed does not carry the four asked questions in order: %+v", seed.Questions)
	}
}

// TestReflectRefusesAReleaseThatShippedNothing is criterion 3: the criterion's
// own wording, a non-zero exit, and nothing written.
func TestReflectRefusesAReleaseThatShippedNothing(t *testing.T) {
	r := reflectRepo(t)
	for _, args := range [][]string{
		{"v0.3.0"},
		{"write", "v0.3.0", "--answers", writeAnswers(t, fullAnswers)},
	} {
		out, err := reflectIn(t, r, args...)
		if code := exitCodeOf(err); code != 1 {
			t.Fatalf("reflect %v: exit = %d, want 1\n%s", args, code, out)
		}
		if msg := errText(err); !strings.Contains(msg, "no intent shipped in `v0.3.0` — nothing shipped to reflect on") {
			t.Errorf("reflect %v: refusal = %q", args, msg)
		}
	}
	if _, err := os.Lstat(filepath.Join(r.Root(), ".abcd", "development", "retrospectives")); err == nil {
		t.Error("a refused release created the retrospectives store")
	}
}

// TestReflectRefusesAnUnknownOrMalformedTag: a tag the repository does not hold,
// and a value that is not a release tag at all, are usage refusals, each saying
// which it is: an exit 2 alone is what an unknown verb also returns.
func TestReflectRefusesAnUnknownOrMalformedTag(t *testing.T) {
	r := reflectRepo(t)
	for tag, want := range map[string]string{
		"v9.9.9":    "no release tag v9.9.9 in this repository",
		"0.2.0":     `"0.2.0" is not a release tag (want vMAJOR.MINOR.PATCH`,
		"v0.2":      `"v0.2" is not a release tag (want vMAJOR.MINOR.PATCH`,
		"../v0.2.0": `"../v0.2.0" is not a release tag (want vMAJOR.MINOR.PATCH`,
	} {
		out, err := reflectIn(t, r, tag)
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("reflect %s: exit = %d, want 2\n%s", tag, code, out)
		}
		if msg := errText(err); !strings.Contains(msg, want) {
			t.Errorf("reflect %s: refusal = %q, want it to say %q", tag, msg, want)
		}
	}
}

// TestReflectRefusesAnIntentID: the release is the only grain; an intent id is
// refused with the reason.
func TestReflectRefusesAnIntentID(t *testing.T) {
	r := reflectRepo(t)
	out, err := reflectIn(t, r, "itd-2")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s", code, out)
	}
	if msg := errText(err); !strings.Contains(msg, "release tag") || !strings.Contains(msg, "intent audit") {
		t.Errorf("refusal = %q, want it to say reflect takes a release tag and name the intent audit", msg)
	}
}

// TestReflectBarePrintsHelpAndWritesNothing: bare `abcd reflect` is help.
func TestReflectBarePrintsHelpAndWritesNothing(t *testing.T) {
	r := reflectRepo(t)
	before := cliTreeDigest(t, r.Root())
	out, err := reflectIn(t, r)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(string(out), "Usage:") || !strings.Contains(string(out), "write") {
		t.Errorf("bare reflect did not print its help:\n%s", out)
	}
	if after := cliTreeDigest(t, r.Root()); after != before {
		t.Error("bare reflect changed the tree")
	}
}

// TestReflectWriteProducesTheRetrospective is criterion 1's file: the five
// sections, written once, with --proceed confirming the unshipped target.
func TestReflectWriteProducesTheRetrospective(t *testing.T) {
	r := reflectRepo(t)
	out, err := reflectIn(t, r, "write", "v0.2.0", "--answers", writeAnswers(t, fullAnswers), "--proceed")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if !strings.Contains(string(out), ".abcd/development/retrospectives/v0.2.0/README.md") {
		t.Errorf("the write does not name its file:\n%s", out)
	}
	data, err := os.ReadFile(retroPath(r, "v0.2.0"))
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"## What went well", "## What could improve", "## Lessons learned", "## Decisions made", "## Metrics"} {
		if !strings.Contains(string(data), h) {
			t.Errorf("the retrospective has no %q section:\n%s", h, data)
		}
	}

	// A second run on the same tag refuses and names the file.
	out, err = reflectIn(t, r, "write", "v0.2.0", "--answers", writeAnswers(t, fullAnswers), "--proceed")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("second write: exit = %d, want 1\n%s", code, out)
	}
	if msg := errText(err); !strings.Contains(msg, "already exists") {
		t.Errorf("second write refusal = %q", msg)
	}
}

// TestReflectWriteAsksBeforeUnshippedTargets is criterion 7: without --proceed
// the write refuses, lists the targeted intents, and writes nothing; the JSON
// form carries them for the host to ask about.
func TestReflectWriteAsksBeforeUnshippedTargets(t *testing.T) {
	r := reflectRepo(t)
	out, err := reflectIn(t, r, "write", "v0.2.0", "--answers", writeAnswers(t, fullAnswers), "--json")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	var ref struct {
		Refused   string `json:"refused"`
		Unshipped []struct {
			ID string `json:"id"`
		} `json:"unshipped_targets"`
	}
	if err := json.Unmarshal(out, &ref); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if ref.Refused != "unshipped_targets" || len(ref.Unshipped) != 1 || ref.Unshipped[0].ID != "itd-5" {
		t.Errorf("refusal = %+v", ref)
	}
	if _, err := os.Lstat(retroPath(r, "v0.2.0")); err == nil {
		t.Error("a refused write wrote the retrospective")
	}
}

// TestReflectWriteMeetsAThinAnswerWithAQuestion is criterion 4: a thin answer is
// not committed; the refusal carries the follow-up question to ask.
func TestReflectWriteMeetsAThinAnswerWithAQuestion(t *testing.T) {
	r := reflectRepo(t)
	thin := strings.Replace(fullAnswers,
		`"The seed builder read the release from the tag tree, and the refusals named the tag every time."`,
		`"it worked"`, 1)
	out, err := reflectIn(t, r, "write", "v0.2.0", "--answers", writeAnswers(t, thin), "--proceed", "--json")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	var ref struct {
		Refused string `json:"refused"`
		Thin    []struct {
			Section  string `json:"section"`
			Question string `json:"question"`
		} `json:"thin"`
	}
	if err := json.Unmarshal(out, &ref); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if ref.Refused != "thin_answers" || len(ref.Thin) != 1 || ref.Thin[0].Section != "went_well" || !strings.Contains(ref.Thin[0].Question, "?") {
		t.Errorf("refusal = %+v", ref)
	}
	if _, err := os.Lstat(retroPath(r, "v0.2.0")); err == nil {
		t.Error("a thin answer was committed")
	}

	// The text form names the question too.
	_, err = reflectIn(t, r, "write", "v0.2.0", "--answers", writeAnswers(t, thin), "--proceed")
	if msg := errText(err); !strings.Contains(msg, "Which specific piece of work went well") {
		t.Errorf("text refusal does not ask the follow-up: %q", msg)
	}
}

// TestReflectWriteRefusesAMistypedAnswersKey: a strict parse, so a section's
// answer cannot be dropped by a typo.
func TestReflectWriteRefusesAMistypedAnswersKey(t *testing.T) {
	r := reflectRepo(t)
	bad := strings.Replace(fullAnswers, `"lessons"`, `"lesson"`, 1)
	out, err := reflectIn(t, r, "write", "v0.2.0", "--answers", writeAnswers(t, bad), "--proceed")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s", code, out)
	}
	// The refusal names the answers and the mistyped key; the key's wording
	// around it is the standard library's, so it is not pinned.
	if msg := errText(err); !strings.Contains(msg, "abcd reflect write: reflect: answers:") || !strings.Contains(msg, "lesson") || !strings.Contains(msg, "(nothing written)") {
		t.Errorf("refusal = %q, want it to name the answers, the mistyped key and that nothing was written", msg)
	}
	if _, err := os.Lstat(filepath.Join(r.Root(), ".abcd", "development", "retrospectives")); err == nil {
		t.Error("a refused answers file created the retrospectives store")
	}
}

// TestReflectIsListedUnderRelease is ruling H13: reflect is a person's verb,
// listed in the Release group.
func TestReflectIsListedUnderRelease(t *testing.T) {
	help, _ := executedHelp(t, "--help")
	_, entries := helpSections(help)
	found := false
	for _, n := range entries["Release:"] {
		found = found || n == "reflect"
	}
	if !found {
		t.Fatalf("reflect is not listed under Release:\n%s", help)
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestLaunchShipSaysOnceThatARetrospectiveIsOwed is criterion 8: a written cut
// ends with the one line naming the command, said exactly once; the JSON form
// carries it as a field; a cut that writes nothing says nothing about it.
func TestLaunchShipSaysOnceThatARetrospectiveIsOwed(t *testing.T) {
	r := shipReadyRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload)
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	line := "A retrospective for v0.4.1 is owed: run /abcd:reflect v0.4.1 when you are ready."
	if n := strings.Count(string(out), "retrospective"); n != 1 {
		t.Errorf("the cut mentions the retrospective %d times, want once:\n%s", n, out)
	}
	if !strings.HasSuffix(strings.TrimRight(string(out), "\n"), line) {
		t.Errorf("the cut does not end with the nudge %q:\n%s", line, out)
	}

	// The deterministic emit writes no cut, so it owes nothing yet.
	r2 := shipReadyRepo(t)
	out, _ = shipIn(t, r2, "launch", "ship")
	if strings.Contains(string(out), "retrospective") {
		t.Errorf("a cut that wrote nothing mentions a retrospective:\n%s", out)
	}
}

func TestLaunchShipJSONCarriesTheNudge(t *testing.T) {
	r := shipReadyRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload, "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	var res struct {
		Owed string `json:"retrospective_owed"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if res.Owed != "A retrospective for v0.4.1 is owed: run /abcd:reflect v0.4.1 when you are ready." {
		t.Errorf("retrospective_owed = %q", res.Owed)
	}
}

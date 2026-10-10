package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

// questionCall builds the host's PreToolUse payload for a call to its question
// tool: abcd's well-built question (a "Product Q1" chip, within every field
// limit), so the mode gate is what these tests exercise. The field checks are
// guard_question_limits_test.go's.
func questionCall(t *testing.T, cwd string) string {
	t.Helper()
	q := wellBuilt()
	q.Header = "Product Q1"
	return askPayload(t, cwd, q)
}

func questionMarker(root string) string {
	return filepath.Join(root, filepath.FromSlash(mode.QuestionOpenRelPath))
}

// chipCall is questionCall with the chip header given.
func chipCall(t *testing.T, cwd, header string) string {
	t.Helper()
	q := wellBuilt()
	q.Header = header
	return askPayload(t, cwd, q)
}

// chipStates are the chips the gate reads and the mode each one sets
// (iss-2610100626211810): Product names the product thinker, and Tech and
// Setup the technical facilitator.
var chipStates = []struct {
	header string
	want   mode.State
}{
	{"Product Q1", mode.ProductThinker},
	{"Tech Q3", mode.Facilitator},
	{"Setup Q1/4", mode.Facilitator},
}

// TestGuardChipSetsTheModeWhileManaged (iss-2610100626211810): a question
// carrying abcd's chip already names whom it is for, so with the mode reading
// managed it is never refused on the mode. It runs silently, the mode is set
// from the chip's role, and the question is marked open for the reset.
func TestGuardChipSetsTheModeWhileManaged(t *testing.T) {
	for _, tc := range chipStates {
		t.Run(tc.header, func(t *testing.T) {
			root := managedCheckout(t)
			stdout, stderr, code := runGuard(chipCall(t, root, tc.header), "guard", "hook")
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("a chipped question must be admitted silently while managed; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if got, err := mode.ReadAt(root); err != nil || got != tc.want {
				t.Errorf("mode after the question = %q (%v), want %q", got, err, tc.want)
			}
			data, err := os.ReadFile(questionMarker(root))
			if err != nil {
				t.Fatalf("the admitted question was not marked open: %v", err)
			}
			if strings.TrimSpace(string(data)) != string(tc.want) {
				t.Errorf("marker = %q, want %q", data, tc.want)
			}
		})
	}
}

// TestGuardChipReSetsADifferentMode (iss-2610100626211810): the status line
// names whom the question on screen is for, so a chip whose role differs from
// the mode a person or an earlier stop set re-sets it to the chip's role.
func TestGuardChipReSetsADifferentMode(t *testing.T) {
	for _, tc := range []struct {
		from   mode.State
		header string
		want   mode.State
	}{
		{mode.Facilitator, "Product Q2", mode.ProductThinker},
		{mode.ProductThinker, "Tech Q1", mode.Facilitator},
		{mode.ProductThinker, "Setup Q2", mode.Facilitator},
	} {
		t.Run(string(tc.from)+"/"+tc.header, func(t *testing.T) {
			root := managedCheckout(t)
			setMode(t, root, tc.from)
			stdout, stderr, code := runGuard(chipCall(t, root, tc.header), "guard", "hook")
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("the question must be admitted silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if got, _ := mode.ReadAt(root); got != tc.want {
				t.Errorf("mode = %q, want the chip's %q", got, tc.want)
			}
			if !markedOpen(t, root) {
				t.Error("the admitted question was not marked open")
			}
		})
	}
}

// TestGuardAdmitsAQuestionOnceAddressed: with the mode already naming the
// person the chip names, the question runs silently, the mode stays, and the
// question is marked open for the reset.
func TestGuardAdmitsAQuestionOnceAddressed(t *testing.T) {
	for st, header := range map[mode.State]string{mode.ProductThinker: "Product Q1", mode.Facilitator: "Tech Q1"} {
		t.Run(string(st), func(t *testing.T) {
			root := managedCheckout(t)
			if err := mode.SetAt(root, st); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := runGuard(chipCall(t, root, header), "guard", "hook")
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("an addressed question must be admitted silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			data, err := os.ReadFile(questionMarker(root))
			if err != nil {
				t.Fatalf("the admitted question was not marked open: %v", err)
			}
			if strings.TrimSpace(string(data)) != string(st) {
				t.Errorf("marker = %q, want %q", data, st)
			}
			if got, _ := mode.ReadAt(root); got != st {
				t.Errorf("mode = %q, want %q", got, st)
			}
		})
	}
}

// TestGuardLeavesQuestionsAloneWhereUnmanaged: a repository abcd does not
// manage has no badge and nowhere to set the mode, so its questions are never
// gated, and the gate creates nothing there.
func TestGuardLeavesQuestionsAloneWhereUnmanaged(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	t.Chdir(repo)

	stdout, stderr, code := runGuard(questionCall(t, repo), "guard", "hook")
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("an unmanaged question must pass silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the gate created .abcd/ in an unmanaged repository: %v", err)
	}
}

// TestGuardQuestionGateFailsOpenLoud: a mode store the gate cannot read is not
// a decision, so the question runs and the gate says so on the loud,
// non-blocking status — the guard's own fail-open-loud contract.
func TestGuardQuestionGateFailsOpenLoud(t *testing.T) {
	root := managedCheckout(t)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(mode.FileRelPath)), []byte("sideways\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runGuard(questionCall(t, root), "guard", "hook")
	if code != 1 {
		t.Fatalf("an unreadable mode store must fail open loud (exit 1); got %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stderr, "NOT CHECKED") {
		t.Errorf("the fail-open must say so; stderr = %q", stderr)
	}
}

// TestGuardQuestionGateFailsOpenWhereTheModeCannotBeSet
// (iss-2609260100382261, iss-2610100626211810): with the tier present but not
// writable, the chip's mode cannot be set and the question cannot be marked
// open. That is not a decision: the question runs on the loud, non-blocking
// status, the gate says why, the mode stays where it was, and nothing is left
// behind.
func TestGuardQuestionGateFailsOpenWhereTheModeCannotBeSet(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a read-only directory does not refuse root")
	}
	root := managedCheckout(t)
	tier := filepath.Join(root, filepath.FromSlash(mode.TierRelPath))
	if err := os.Chmod(tier, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tier, 0o700) })

	if _, _, err := runSplit(t, "", "mode", "facilitator"); err == nil {
		t.Fatal("precondition: `abcd mode facilitator` succeeded on a read-only tier")
	}
	stdout, stderr, code := runGuard(questionCall(t, root), "guard", "hook")
	if code != 1 || stdout != "" {
		t.Fatalf("a question whose mode cannot be set must fail open loud (exit 1); got %d (stdout %q stderr %q)", code, stdout, stderr)
	}
	for _, want := range []string{"NOT CHECKED", "UNGATED", "could not be"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the fail-open must say %q; stderr = %q", want, stderr)
		}
	}
	if got, _ := mode.ReadAt(root); got != mode.Managed {
		t.Errorf("mode = %q, want it left managed", got)
	}
	ents, err := os.ReadDir(tier)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 0 {
		t.Errorf("the gate left residue in the tier: %v", ents)
	}
}

// TestPostToolUseHookResetsTheModeOnTheAnswer (iss-2610100626211810): the
// question tool's PostToolUse hook is the answer coming back, so it runs the
// prompt hook's reset there and then: the mode goes back to managed, the
// marker is cleared, and one stderr line says so, with nothing on stdout,
// which the host would hand the agent. With no question open, or for a tool
// that is not a question tool, it changes nothing and says nothing.
func TestPostToolUseHookResetsTheModeOnTheAnswer(t *testing.T) {
	root := managedCheckout(t)
	if _, _, code := runGuard(chipCall(t, root, "Tech Q3"), "guard", "hook"); code != 0 {
		t.Fatalf("precondition: the chipped question was not admitted (exit %d)", code)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Fatalf("precondition: mode = %q, want facilitator", got)
	}
	answered := func(tool string) string {
		return `{"session_id":"s-post","hook_event_name":"PostToolUse","tool_name":"` + tool +
			`","tool_input":{"questions":[]},"tool_response":{"answers":{}},"cwd":"` + root + `"}`
	}

	// Another tool's PostToolUse is none of the reset's business.
	stdout, stderr, err := runSplit(t, answered("Bash"), "hook", "question-answered")
	if err != nil || stdout != "" {
		t.Fatalf("question-answered on another tool: err=%v stdout=%q stderr=%q", err, stdout, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Errorf("another tool's PostToolUse moved the mode to %q", got)
	}

	stdout, stderr, err = runSplit(t, answered(questionTools[0]), "hook", "question-answered")
	if err != nil || stdout != "" {
		t.Fatalf("question-answered: err=%v stdout=%q stderr=%q", err, stdout, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.Managed {
		t.Errorf("mode after the answer = %q, want managed", got)
	}
	if markedOpen(t, root) {
		t.Error("the marker survived the answer")
	}
	if n := modeResetLines(stderr); n != 1 {
		t.Errorf("want exactly one stderr line saying the mode was reset, got %d:\n%s", n, stderr)
	}

	// A hand-set mode with no question open is left alone.
	setMode(t, root, mode.ProductThinker)
	_, stderr, err = runSplit(t, answered(questionTools[0]), "hook", "question-answered")
	if err != nil {
		t.Fatalf("question-answered: %v\n%s", err, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.ProductThinker {
		t.Errorf("with no question open the mode moved to %q", got)
	}
	if n := modeResetLines(stderr); n != 0 {
		t.Errorf("no question was open, yet the hook reported a reset:\n%s", stderr)
	}
}

// TestQuestionAnswerHookIsInstalled holds the PostToolUse wiring: one entry,
// scoped to exactly the question tools the gate reads, whose script runs
// `hook question-answered`.
func TestQuestionAnswerHookIsInstalled(t *testing.T) {
	doc := decodedHooksManifest(t)
	entries := doc.Hooks["PostToolUse"]
	if len(entries) != 1 || len(entries[0].Hooks) != 1 {
		t.Fatalf("want one PostToolUse entry with one command; got %+v", entries)
	}
	if want := strings.Join(questionTools, "|"); entries[0].Matcher != want {
		t.Errorf("the PostToolUse entry must be scoped to the question tools; matcher = %q, want %q", entries[0].Matcher, want)
	}
	if body := resolveHookCommand(t, entries[0].Hooks[0].Command); !strings.Contains(body, "hook question-answered") {
		t.Errorf("the PostToolUse script must run `hook question-answered`; it runs:\n%s", body)
	}
}

// TestPromptHookResetsTheModeAfterAQuestion is criterion 4: the next human
// message after an admitted question resets the mode to managed, clears the
// marker and says so in one stderr line; a message with no question open
// changes nothing and says nothing about the mode.
func TestPromptHookResetsTheModeAfterAQuestion(t *testing.T) {
	root := managedCheckout(t)
	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runGuard(questionCall(t, root), "guard", "hook"); code != 0 {
		t.Fatalf("precondition: the addressed question was not admitted (exit %d)", code)
	}

	prompt := `{"session_id":"s-reset","hook_event_name":"UserPromptSubmit","prompt":"yes, ship it","cwd":"` + root + `"}`
	_, stderr, err := runSplit(t, prompt, "hook", "prompt-router")
	if err != nil {
		t.Fatalf("prompt-router: %v\n%s", err, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.Managed {
		t.Errorf("mode after the answer = %q, want managed", got)
	}
	if _, err := os.Lstat(questionMarker(root)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the marker survived the answer: %v", err)
	}
	if n := modeResetLines(stderr); n != 1 {
		t.Errorf("want exactly one stderr line saying the mode was reset, got %d:\n%s", n, stderr)
	}

	// A human who sets the hat by hand keeps it across their next message.
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	_, stderr, err = runSplit(t, prompt, "hook", "prompt-router")
	if err != nil {
		t.Fatalf("prompt-router: %v\n%s", err, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.Facilitator {
		t.Errorf("with no question open the mode moved to %q", got)
	}
	if n := modeResetLines(stderr); n != 0 {
		t.Errorf("no question was open, yet the hook reported a reset:\n%s", stderr)
	}
}

// TestPromptHookLeavesAHandSetModeOverAnUnremovableMarker
// (iss-2609260100393814): with a directory planted at the marker's path, every
// message would otherwise reset the hand-set mode and repeat the same error.
// Each message says so in one line, and the mode stays where the human put it.
func TestPromptHookLeavesAHandSetModeOverAnUnremovableMarker(t *testing.T) {
	root := managedCheckout(t)
	if err := os.MkdirAll(filepath.Join(questionMarker(root), "planted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := mode.SetAt(root, mode.Facilitator); err != nil {
		t.Fatal(err)
	}
	prompt := `{"session_id":"s-planted","hook_event_name":"UserPromptSubmit","prompt":"carry on","cwd":"` + root + `"}`
	for i := range 2 {
		_, stderr, err := runSplit(t, prompt, "hook", "prompt-router")
		if err != nil {
			t.Fatalf("message %d: prompt-router: %v\n%s", i+1, err, stderr)
		}
		if got, _ := mode.ReadAt(root); got != mode.Facilitator {
			t.Fatalf("message %d: the hand-set mode moved to %q", i+1, got)
		}
		if n := modeResetLines(stderr); n != 1 {
			t.Errorf("message %d: want one loud line about the marker, got %d:\n%s", i+1, n, stderr)
		}
		if strings.Contains(stderr, "reset to managed") {
			t.Errorf("message %d: the hook claims a reset it must not make:\n%s", i+1, stderr)
		}
	}
}

func modeResetLines(stderr string) int {
	n := 0
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, "abcd mode:") {
			n++
		}
	}
	return n
}

// TestModeVerbSetsTheLine is criterion 3: the `mode` verb is the setter, as it
// was, and the line shows the state it set on its next render.
func TestModeVerbSetsTheLine(t *testing.T) {
	root := managedCheckout(t)
	for _, tc := range []struct{ word, label string }{
		{"product-thinker", "waiting on the product thinker"},
		{"facilitator", "waiting on the technical facilitator"},
		{"managed", "abcd-managed"},
	} {
		if _, stderr, err := runSplit(t, "", "mode", tc.word); err != nil {
			t.Fatalf("mode %s: %v\n%s", tc.word, err, stderr)
		}
		stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
		if err != nil {
			t.Fatalf("statusline: %v\n%s", err, stderr)
		}
		if !strings.Contains(stdout, " "+tc.label+" ") {
			t.Errorf("after `mode %s` the line reads %q, want the badge %q", tc.word, stdout, tc.label)
		}
	}
}

// TestBadgeAndNoticeNameTheSamePerson (iss-2609260100396332): the badge and the
// one-line notice `abcd mode` prints where no status surface exists are two
// renderings of one fact, so they name the owed person in the same words. The
// notice's "waiting on the …" must appear, whole, as the badge the line renders
// for the same state.
func TestBadgeAndNoticeNameTheSamePerson(t *testing.T) {
	root := managedCheckout(t)
	for _, st := range []mode.State{mode.Facilitator, mode.ProductThinker} {
		notice := answerOwedNotice(st)
		words := strings.TrimSuffix(strings.TrimPrefix(notice, "abcd: "), " — an answer is owed")
		if words == notice || !strings.HasPrefix(words, "waiting on the ") {
			t.Fatalf("%s: the notice %q does not have the expected shape", st, notice)
		}
		if err := mode.SetAt(root, st); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, err := runSplit(t, payloadFor(root), "statusline")
		if err != nil {
			t.Fatalf("statusline: %v\n%s", err, stderr)
		}
		if !strings.Contains(stdout, " "+words+" ") {
			t.Errorf("%s: the notice says %q but the badge reads %q", st, words, stdout)
		}
	}
}

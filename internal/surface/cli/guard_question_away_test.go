package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/mode"
)

// The question reroute (spc-2610180914469489, step 6): while the away record
// holds, the guard hook refuses every call to the host's question tool with
// one reason whatever the chip, and the reason routes the question by the act
// it asks about.

// setAwayHere sets the away mode from the checkout the process stands in,
// through the verb, as a person or a session on their word would.
func setAwayHere(t *testing.T) {
	t.Helper()
	if _, stderr, err := runSplit(t, "", "mode", "autonomous"); err != nil {
		t.Fatalf("mode autonomous: %v\n%s", err, stderr)
	}
}

// modeFile is the badge's store file in the checkout at root.
func modeFile(root string) string {
	return filepath.Join(root, filepath.FromSlash(mode.FileRelPath))
}

// TestAwayQuestionDeniedOneReason: while the person is away, a Product-chip,
// a Tech-chip, a Setup-chip and an unchipped question, another tool's (one
// that would break abcd's rules among them), a questions field the check
// cannot read, and a question asked from a sibling worktree of the same
// repository are each refused with the host's deny, and every refusal gives
// the same reason, which carries the person-only act list.
func TestAwayQuestionDeniedOneReason(t *testing.T) {
	foreign := hostQuestion{
		Header:   "Pick a theme",
		Question: "Which theme?",
		Options:  []hostOption{{Label: "Dark (Recommended)"}, {Label: "Light"}},
	}
	unchipped := wellBuilt()
	unchipped.Header = "Label"
	cases := []struct {
		name    string
		before  mode.State // the badge before the mode was set
		sibling bool       // asked from a sibling worktree
		payload func(t *testing.T, cwd string) string
	}{
		{name: "product chip", payload: func(t *testing.T, cwd string) string { return chipCall(t, cwd, "Product Q1") }},
		{name: "tech chip", payload: func(t *testing.T, cwd string) string { return chipCall(t, cwd, "Tech Q3") }},
		{name: "setup chip", payload: func(t *testing.T, cwd string) string { return chipCall(t, cwd, "Setup Q1/4") }},
		{name: "unchipped, the badge naming the facilitator", before: mode.Facilitator,
			payload: func(t *testing.T, cwd string) string { return askPayload(t, cwd, unchipped) }},
		{name: "another tool's", payload: func(t *testing.T, cwd string) string { return askPayload(t, cwd, foreign) }},
		{name: "unreadable questions field", payload: func(t *testing.T, cwd string) string {
			return askToolPayload(t, cwd, map[string]any{"questions": 5})
		}},
		{name: "from a sibling worktree", sibling: true,
			payload: func(t *testing.T, cwd string) string { return chipCall(t, cwd, "Product Q1") }},
	}
	var first string
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := awayCheckout(t)
			if tc.before != "" {
				setMode(t, root, tc.before)
			}
			setAwayHere(t)
			cwd := root
			if tc.sibling {
				cwd = filepath.Join(t.TempDir(), "sibling")
				gitAt(t, root, "worktree", "add", "-q", "-b", "feat/sibling", cwd)
				cwd = realPath(t, cwd)
				t.Chdir(cwd)
			}
			reason := hookDenyOf(t, tc.payload(t, cwd))
			if first == "" {
				first = reason
			} else if reason != first {
				t.Errorf("the reason differs from the first case's:\n got %q\nwant %q", reason, first)
			}
			if !strings.Contains(reason, "the person is away") {
				t.Errorf("the reason does not say the person is away:\n%s", reason)
			}
			// The route carries every person-only act (autonomy's own test
			// holds the route to the list), so the reason carrying it whole
			// carries each act.
			if !strings.Contains(reason, autonomy.QuestionRoute()) {
				t.Errorf("the reason leaves out the question route and its person-only acts:\n%s", reason)
			}
			if strings.Contains(reason, "asking rules") {
				t.Errorf("an away refusal names the asking rules, so the agent would redraft instead of rerouting:\n%s", reason)
			}
		})
	}
}

// runGuardTriple is runGuard on the guard hook.
func runGuardTriple(payload string) (string, string, int) {
	return runGuard(payload, "guard", "hook")
}

// hookDenyOf runs the guard hook on payload, asserts it denied the call, and
// returns the reason.
func hookDenyOf(t *testing.T, payload string) string {
	t.Helper()
	stdout, stderr, code := runGuardTriple(payload)
	return mustDeny(t, stdout, stderr, code)
}

// TestAwayQuestionSetsNoMode: a question refused while away sets no mode and
// writes no marker. The badge stays as it was (absent, or the person an
// earlier stop named), the question is not marked open, and the away record
// is untouched, so the chip-set path is never reached while away.
func TestAwayQuestionSetsNoMode(t *testing.T) {
	for _, before := range []mode.State{"", mode.Facilitator} {
		t.Run("badge "+string(before), func(t *testing.T) {
			root := awayCheckout(t)
			if before != "" {
				setMode(t, root, before)
			}
			setAwayHere(t)
			modeBefore, modeErrBefore := os.ReadFile(modeFile(root))
			recBefore, err := os.ReadFile(awayStore(t, root).RecordPath())
			if err != nil {
				t.Fatal(err)
			}
			hookDenyOf(t, chipCall(t, root, "Product Q1"))

			modeAfter, modeErrAfter := os.ReadFile(modeFile(root))
			if string(modeAfter) != string(modeBefore) || (modeErrBefore == nil) != (modeErrAfter == nil) {
				t.Errorf("the badge's store changed: before %q (%v), after %q (%v)", modeBefore, modeErrBefore, modeAfter, modeErrAfter)
			}
			if markedOpen(t, root) {
				t.Error("a question refused while away was marked open")
			}
			recAfter, err := os.ReadFile(awayStore(t, root).RecordPath())
			if err != nil {
				t.Fatal(err)
			}
			if string(recAfter) != string(recBefore) {
				t.Errorf("the away record changed on a refused question:\n before %s\n after %s", recBefore, recAfter)
			}
		})
	}
}

// TestPresentQuestionGateUnchanged: while the person is present (no record,
// a record ended with `abcd mode back`, or one whose time has passed) the
// gate is what it was: a chipped question is admitted silently, sets the
// badge from its chip and is marked open; another tool's question runs
// unchecked; and a question that breaks abcd's rules is refused on them.
func TestPresentQuestionGateUnchanged(t *testing.T) {
	broken := wellBuilt()
	broken.Options[0].Label = "Role and number (Recommended)"
	foreign := hostQuestion{
		Header:   "Pick a theme",
		Question: "Which theme?",
		Options:  []hostOption{{Label: "Dark (Recommended)"}, {Label: "Light"}},
	}
	for _, present := range []struct {
		name string
		make func(t *testing.T, root string)
	}{
		{"no record", func(*testing.T, string) {}},
		{"ended with mode back", func(t *testing.T, root string) {
			setAwayHere(t)
			if _, stderr, err := runSplit(t, "", "mode", "back"); err != nil {
				t.Fatalf("mode back: %v\n%s", err, stderr)
			}
		}},
		{"lapsed", func(t *testing.T, root string) {
			past := time.Now().Add(-8 * 24 * time.Hour)
			if _, err := autonomy.SetAway(root, "", false, autonomy.Options{Now: func() time.Time { return past }}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(present.name, func(t *testing.T) {
			root := awayCheckout(t)
			present.make(t, root)
			if _, holds, err := autonomy.AwayAt(root, autonomy.Options{}); err != nil || holds {
				t.Fatalf("the fixture is not present: holds=%v err=%v", holds, err)
			}

			stdout, stderr, code := runGuardTriple(chipCall(t, root, "Product Q1"))
			if code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("a chipped question must be admitted silently while present; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if got, _ := mode.ReadAt(root); got != mode.ProductThinker {
				t.Errorf("mode = %q, want the chip's product-thinker", got)
			}
			if !markedOpen(t, root) {
				t.Error("the admitted question was not marked open")
			}

			// The badge names the product thinker now, so another tool's
			// question would be held to abcd's rules; the answer's reset
			// puts it back to managed first.
			setMode(t, root, mode.Managed)
			if stdout, stderr, code := runGuardTriple(askPayload(t, root, foreign)); code != 0 || stdout != "" || stderr != "" {
				t.Fatalf("another tool's question must run unchecked while present; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}

			reason := hookDenyOf(t, askPayload(t, root, broken))
			if !strings.Contains(reason, "break abcd's asking rules") || strings.Contains(reason, "the person is away") {
				t.Errorf("a broken question must be refused on the asking rules alone while present:\n%s", reason)
			}
		})
	}
}

// TestAwayRecordUnreadableFailsOpenLoud: an away record the store refuses (here
// one another account could write) reads as not away. The question is checked
// as if the person were present, the badge set from its chip, and the hook
// says on exit 1, which the host shows, that the away record was not read.
func TestAwayRecordUnreadableFailsOpenLoud(t *testing.T) {
	root := awayCheckout(t)
	setAwayHere(t)
	if err := os.Chmod(awayStore(t, root).RecordPath(), 0o666); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runGuardTriple(chipCall(t, root, "Product Q1"))
	if isDeny(stdout) {
		t.Fatalf("an unreadable away record must not refuse the question; stdout=%q", stdout)
	}
	if code != 1 {
		t.Errorf("exit = %d, want 1 so the host shows the warning", code)
	}
	if !strings.Contains(stderr, "away record") || !strings.Contains(stderr, "writable by another account") {
		t.Errorf("stderr does not name the unread away record and why:\n%s", stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.ProductThinker {
		t.Errorf("mode = %q, want the chip's product-thinker", got)
	}
}

package ahoy

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// armHooksCommand is the one command that points a clone's git at the committed
// hooks. Run under a global or system hooks dispatcher, it sets a LOCAL hooks path
// that shadows the dispatcher in that clone, which is why ahoy never advises it for
// a foreign hooks path.
const armHooksCommand = "git config core.hooksPath .githooks"

// hooksPathStates is every state ahoy reports in banlist.hooks_path. A state added
// to HooksPathState belongs here too, so the page is held to say what to do in it.
var hooksPathStates = []HooksPathState{HooksPathStateArmed, HooksPathStateUnarmed, HooksPathStateForeign}

// stateBullet matches a bullet that opens one hooks_path state's branch of the
// commit-gates step, "- `unarmed` — …", capturing the state.
var stateBullet = regexp.MustCompile("^\\s*- `([a-z]+)`")

// numberedStep matches the start of a numbered step, which closes any state branch
// above it.
var numberedStep = regexp.MustCompile(`^\s*\d+\.\s`)

// hooksPathAdviceFaults returns what the ahoy page gets wrong about
// arming the committed hooks, one sentence each; none means the page sets a local
// core.hooksPath only in the unarmed branch and leaves a foreign one alone.
//
// The arming command is judged by the branch it sits in: the nearest state bullet
// above it, unless a numbered step comes first. A command outside every branch is
// the unconditional advice iss-2610080546210831 found.
func hooksPathAdviceFaults(page string) []string {
	var faults []string
	lines := strings.Split(page, "\n")
	branchOf := func(i int) string {
		for j := i; j >= 0; j-- {
			if m := stateBullet.FindStringSubmatch(lines[j]); m != nil {
				return m[1]
			}
			if numberedStep.MatchString(lines[j]) {
				return ""
			}
		}
		return ""
	}
	branches := map[string]string{} // state -> its bullet's text, to the next bullet or step
	for i, line := range lines {
		if strings.Contains(line, armHooksCommand) {
			if b := branchOf(i); b != string(HooksPathStateUnarmed) {
				where := "outside every hooks_path branch"
				if b != "" {
					where = "in the `" + b + "` branch"
				}
				faults = append(faults, "line "+strconv.Itoa(i+1)+" advises `"+armHooksCommand+"` "+where+
					"; only an unarmed clone may be pointed at the committed hooks")
			}
		}
		if b := branchOf(i); b != "" {
			branches[b] += line + "\n"
		}
	}
	for _, s := range hooksPathStates {
		if _, ok := branches[string(s)]; !ok {
			faults = append(faults, "the page gives no branch for hooks_path `"+string(s)+"`")
		}
	}
	if foreign, ok := branches[string(HooksPathStateForeign)]; ok {
		if !strings.Contains(foreign, "dispatcher") || !strings.Contains(foreign, ".githooks/") {
			faults = append(faults, "the `foreign` branch does not tell the person to have the dispatcher call the hook in `.githooks/`")
		}
	}
	if !strings.Contains(page, "banlist.hooks_path") {
		faults = append(faults, "the page does not name `banlist.hooks_path`, the field the branches key on")
	}
	return faults
}

// TestAhoyPageArmsHooksOnlyWhenUnarmed is iss-2610080546210831's detector. The
// preparation workflow's commit-gates step, on the prepare-this-repo page until
// it folded into the ahoy page's install section (spc-2610100613109045, step 4), told every prepared repository to run
// `git config core.hooksPath .githooks` with no condition, which shadows a global
// hooks dispatcher: the exact advice ahoy refuses to give when hooks_path reads
// foreign. The page must key the command on the state ahoy reports.
func TestAhoyPageArmsHooksOnlyWhenUnarmed(t *testing.T) {
	for _, f := range hooksPathAdviceFaults(readAhoyPage(t)) {
		t.Errorf("commands/ahoy.md: %s", f)
	}
}

// TestPrepareHooksPathFieldIsWhatAhoyEmits pins the JSON path the page tells the
// agent to read to the tags the detection result actually carries, so renaming
// either field cannot leave the page keying on a field that no longer exists.
func TestPrepareHooksPathFieldIsWhatAhoyEmits(t *testing.T) {
	tag := func(v any, field string) string {
		f, ok := reflect.TypeOf(v).FieldByName(field)
		if !ok {
			t.Fatalf("%T has no field %s", v, field)
		}
		return strings.Split(f.Tag.Get("json"), ",")[0]
	}
	if got := tag(DetectionResult{}, "Banlist") + "." + tag(BanlistHealth{}, "HooksPath"); got != "banlist.hooks_path" {
		t.Errorf("ahoy --json carries the hooks path state at %q; the ahoy page reads banlist.hooks_path", got)
	}
}

// TestHooksPathAdviceFaultsCatchesTheUnconditionalStep is the detector's must-fail
// half: the step as iss-2610080546210831 found it, and a foreign branch that
// overrides the dispatcher, must both be named.
func TestHooksPathAdviceFaultsCatchesTheUnconditionalStep(t *testing.T) {
	for _, tc := range []struct{ name, page string }{
		{"unconditional step", "5. **Commit gates.**\n\n   ```bash\n   abcd ahoy install\n   " + armHooksCommand + "\n   ```\n"},
		{"foreign branch overrides", "5. **Commit gates.** read `banlist.hooks_path`:\n\n" +
			"   - `unarmed` — run it.\n   - `armed` — nothing to do.\n" +
			"   - `foreign` — a dispatcher is in force; override it:\n\n     ```bash\n     " + armHooksCommand + "\n     ```\n"},
		{"foreign branch missing", "5. **Commit gates.** read `banlist.hooks_path`:\n\n" +
			"   - `unarmed` — run:\n\n     ```bash\n     " + armHooksCommand + "\n     ```\n   - `armed` — nothing to do.\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if faults := hooksPathAdviceFaults(tc.page); len(faults) == 0 {
				t.Errorf("the detector passed a page that arms the hooks outside the unarmed branch:\n%s", tc.page)
			}
		})
	}
}

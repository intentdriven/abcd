package ahoy

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/identity"
)

// TestIdentityPinActionNamesWhatCorrectsAWrongPin is iss-2609260057117878's
// first sentence: the summary told a person whose recorded name or email is
// wrong to run abcd ahoy install again, but the pin step writes only while no
// pin exists, so a re-run leaves a wrong pin exactly as it is. The behaviour is
// proved here first, and then the item is held to naming the file that does
// correct it.
func TestIdentityPinActionNamesWhatCorrectsAWrongPin(t *testing.T) {
	dir := idGitRepo(t, "Right Person", "right@example.com")
	idWritePin(t, dir, `{"name":"Wrong Person","email":"wrong@example.com"}`)
	gapPresent := map[string]bool{}
	for _, g := range detectGitIdentity(dir) {
		gapPresent[g.ID] = true
	}
	a := &applyCtx{cwd: dir, approved: map[GapCategory]bool{ConfigChange: true}, gapPresent: gapPresent}
	a.stepIdentityPin()
	pin, ok, err := identity.LoadPin(dir)
	if err != nil || !ok || pin.Name != "Wrong Person" {
		t.Fatalf("precondition: a re-run leaves a recorded pin as it is; got %+v ok=%v err=%v", pin, ok, err)
	}

	action := writeKindHelp[writeIdentityPin].Action
	if !strings.Contains(action, identity.PinRelPath) {
		t.Errorf("the identity-pin item must name the file that corrects a wrong pin (%s), since a re-run does not: %q", identity.PinRelPath, action)
	}
}

// TestSkippedPinActionNamesWhatThePersonChecks is the second sentence: the
// item told the person to answer y "if the name and email shown are yours",
// but the install shows no name or email; the pin rides on the question about
// the category it belongs to. The item must say where the values come from and
// name the question that is actually asked.
func TestSkippedPinActionNamesWhatThePersonChecks(t *testing.T) {
	action := optionalSkippedHelp[OptionalPinGapID].Action
	for _, want := range []string{"user.name", "user.email", string(ConfigChange)} {
		if !strings.Contains(action, want) {
			t.Errorf("the skipped-pin item does not name %q: %q", want, action)
		}
	}
	if strings.Contains(action, "shown") {
		t.Errorf("the skipped-pin item promises values the install never shows: %q", action)
	}
}

// TestVisibilityMeaningNamesEveryFencedPath is the third: each visibility's
// meaning must name every path its .gitignore block keeps out of git, so the
// public meaning cannot again omit the memory/ fence the block writes.
func TestVisibilityMeaningNamesEveryFencedPath(t *testing.T) {
	for _, c := range promptHelp["visibility"].Choices {
		entries, ok := visibilityEntries[c.Value]
		if !ok {
			t.Fatalf("visibility choice %q has no entry set", c.Value)
		}
		for _, e := range entries {
			bare := strings.Trim(e, "/")
			if !strings.Contains(c.Meaning, bare) {
				t.Errorf("the %s meaning does not name %s, which its .gitignore block keeps out of git: %q", c.Value, e, c.Meaning)
			}
		}
	}
}

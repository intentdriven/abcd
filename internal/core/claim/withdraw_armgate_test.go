package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// noReceipt is a merge gate that arms nothing, naming why.
func noReceipt(_, _, _, head string) (bool, string, error) {
	return false, "no review receipt names " + head, nil
}

// A withdrawn merge goes back on the forge's merge path only at a head the
// merge gate arms: nothing is sent to the forge otherwise, it stays withdrawn
// for a later act, and the gate's refusal is not counted as the forge's.
func TestAReArmAsksTheMergeGateForItsHead(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	var asked []string
	releaseForge.armGate = func(root, repo, pr, head string) (bool, string, error) {
		asked = append(asked, repo+" #"+pr+" "+head)
		return noReceipt(root, repo, pr, head)
	}
	rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range f.mutations() {
		if strings.HasPrefix(m, "enablePullRequestAutoMerge") || strings.HasPrefix(m, "enqueuePullRequest") {
			t.Fatalf("mutations = %v; want nothing armed at a head the gate does not arm", f.mutations())
		}
	}
	if !slices.Contains(asked, "acme/widgets #11 "+f.pr(11).head) {
		t.Fatalf("the gate was asked %v; want #11 at its head", asked)
	}
	_, ws := w.tipWithdrawals("rel-v0.16.0")
	if stateOf(ws, 11) != WithdrawnPending {
		t.Fatalf("#11 recorded %q; want it still withdrawn", stateOf(ws, 11))
	}
	said := ""
	for _, s := range rel.Settles {
		for _, ff := range s.Failures {
			if ff.Number == 11 {
				said = ff.Reason
			}
		}
	}
	if !strings.Contains(said, "no review receipt names") || strings.Contains(said, "did not answer") {
		t.Fatalf("the failure for #11 says %q; want the gate's reason", said)
	}
	// Asked past the bound on the forge's refusals, it is still withdrawn.
	for range maxRearmRefusals + 1 {
		w.set(w.clock().Add(time.Minute))
		if _, err := p.reserve("iss-3", "a.go"); err != nil {
			t.Fatal(err)
		}
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnPending {
		t.Fatalf("#11 recorded %q; want the gate's refusal not counted as the forge's", stateOf(ws, 11))
	}
	// A receipt given since arms it on the next act.
	releaseForge.armGate = func(_, _, _, _ string) (bool, string, error) { return true, "", nil }
	if _, err := p.reserve("iss-4", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnRearmed {
		t.Fatalf("#11 recorded %q; want it re-armed once the gate arms its head", stateOf(ws, 11))
	}
}

// A gate that cannot answer, or a binary with none wired, arms nothing.
func TestAReArmWithNoAnswerFromTheGateArmsNothing(t *testing.T) {
	for _, c := range []struct {
		name string
		gate func(root, repo, pr, head string) (bool, string, error)
		says string
	}{
		{"it fails", func(_, _, _, _ string) (bool, string, error) { return false, "", errors.New("the mode cannot be read") }, "the mode cannot be read"},
		{"none is wired", nil, "merge gate"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			f := armedForge(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
				t.Fatal(err)
			}
			f.reset()
			releaseForge.armGate = c.gate
			rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range f.mutations() {
				if strings.HasPrefix(m, "enablePullRequestAutoMerge") || strings.HasPrefix(m, "enqueuePullRequest") {
					t.Fatalf("mutations = %v", f.mutations())
				}
			}
			said := ""
			for _, s := range rel.Settles {
				for _, ff := range s.Failures {
					said += ff.Reason + "\n"
				}
			}
			if !strings.Contains(said, c.says) {
				t.Fatalf("the failures say %q; want %q", said, c.says)
			}
		})
	}
}

func TestAnAdmissionAsksTheMergeGateForItsHead(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.reset()
	releaseForge.armGate = noReceipt
	_, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "no review receipt names") || strings.Contains(err.Error(), "the forge refused") {
		t.Fatalf("admit = %v; want refused with the gate's reason", err)
	}
	if got := f.mutations(); slices.Contains(got, "enablePullRequestAutoMerge #11") {
		t.Fatalf("mutations = %v; want nothing armed", got)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnPending {
		t.Fatalf("#11 recorded %q; want it withdrawn", stateOf(ws, 11))
	}
}

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

// An admitted pull request found armed counts as done only at a head the gate
// arms. Past a head it refuses it is left for its owner, nothing sent.
func TestAnAdmittedMergeFoundArmedIsAskedOfTheGateAtItsHead(t *testing.T) {
	admitted := func(t *testing.T) (*world, *fakeForge, *participant) {
		w := newWorld(t)
		f := armedForge(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
			t.Fatal(err)
		}
		f.reset()
		return w, f, p
	}
	refusing := func(head string) ArmGate {
		return func(_, _, _, h string) (bool, string, error) {
			if h == head {
				return false, "no review receipt names " + h, nil
			}
			return true, "", nil
		}
	}
	t.Run("moved to a head the gate refuses", func(t *testing.T) {
		w, f, p := admitted(t)
		f.prs[11].head = headC
		releaseForge.armGate = refusing(headC)
		if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnMoved {
			t.Fatalf("#11 recorded %q; want it left for its owner", stateOf(ws, 11))
		}
	})
	t.Run("moved to a head the gate arms", func(t *testing.T) {
		w, f, p := admitted(t)
		f.prs[11].head = headC
		releaseForge.armGate = refusing("")
		if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnFoundArmed {
			t.Fatalf("#11 recorded %q; want it done", stateOf(ws, 11))
		}
	})
	t.Run("at its head, the gate refuses", func(t *testing.T) {
		w, f, p := admitted(t)
		releaseForge.armGate = refusing(f.prs[11].head)
		rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) == WithdrawnFoundArmed {
			t.Fatalf("#11 recorded as done at a head the gate refuses: %v", ws)
		}
		said := ""
		for _, s := range rel.Settles {
			for _, ff := range s.Failures {
				if ff.Number == 11 {
					said = ff.Reason
				}
			}
		}
		if !strings.Contains(said, "no review receipt names") {
			t.Fatalf("the failure for #11 says %q", said)
		}
	})
	t.Run("the gate cannot answer", func(t *testing.T) {
		w, f, p := admitted(t)
		f.prs[11].head = headC
		releaseForge.armGate = func(_, _, _, _ string) (bool, string, error) { return false, "", errors.New("a read failed") }
		if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) == WithdrawnFoundArmed || stateOf(ws, 11) == WithdrawnMoved {
			t.Fatalf("#11 settled %q on no answer", stateOf(ws, 11))
		}
	})
}

// A re-arm the gate holds tells the pull request once: it stays withdrawn, why,
// and what re-arms it. An answer the gate could not read posts nothing.
func TestAHeldReArmIsSaidOnThePullRequestOnce(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	releaseForge.comments = f
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.comments = nil
	releaseForge.armGate = noReceipt
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		w.set(w.clock().Add(time.Minute))
		if _, err := p.reserve("iss-3", "a.go"); err != nil {
			t.Fatal(err)
		}
	}
	held := 0
	for _, c := range f.posted() {
		if strings.HasPrefix(c, "#11 ") {
			held++
			for _, want := range []string{"stays withdrawn", "no review or approval", "abcd land"} {
				if !strings.Contains(c, want) {
					t.Fatalf("comment %q does not say %q", c, want)
				}
			}
			if strings.Contains(c, "http") || strings.Contains(c, "@") {
				t.Fatalf("comment %q carries a URL or a mention", c)
			}
		}
	}
	if held != 1 {
		t.Fatalf("#11 was told %d times: %v", held, f.posted())
	}
}

func TestAReArmTheGateCouldNotAnswerPostsNothing(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	releaseForge.comments = f
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.comments = nil
	releaseForge.armGate = func(_, _, _, _ string) (bool, string, error) { return false, "", errors.New("a read failed") }
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.posted() {
		if strings.Contains(c, "stays withdrawn") {
			t.Fatalf("a comment went out on no answer: %q", c)
		}
	}
}

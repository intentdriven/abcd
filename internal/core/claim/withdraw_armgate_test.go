package claim

import (
	"errors"
	"fmt"
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
	t.Run("moved to a head the gate refuses, its own arming", func(t *testing.T) {
		w, f, p := admitted(t)
		releaseForge.comments = f
		f.prs[11].head = headC
		releaseForge.armGate = refusing(headC)
		if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnMoved {
			t.Fatalf("#11 recorded %q; want it left for its owner", stateOf(ws, 11))
		}
		if f.pr(11).method != "" || !slices.Contains(f.mutations(), "disablePullRequestAutoMerge #11") {
			t.Fatalf("#11 = %+v, mutations %v; want its arming withdrawn, so it is not re-armed as the comment says", f.pr(11), f.mutations())
		}
		said := strings.Join(f.posted(), "\n")
		if !strings.Contains(said, "#11 abcd release hold: the merge of #11 is not re-armed") {
			t.Fatalf("comments %v", f.posted())
		}
	})
	t.Run("moved to a head the gate refuses, its own arming stays on the forge", func(t *testing.T) {
		w, f, p := admitted(t)
		releaseForge.comments = f
		f.prs[11].head = headC
		f.prs[11].stuck = true
		releaseForge.armGate = refusing(headC)
		rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) == WithdrawnMoved {
			t.Fatalf("#11 recorded moved while the forge still reports it armed")
		}
		said := ""
		for _, s := range rel.Settles {
			for _, ff := range s.Failures {
				if ff.Number == 11 {
					said += ff.Reason
				}
			}
		}
		if !strings.Contains(said, "still reports") || strings.Contains(strings.Join(f.posted(), "\n"), "#11 is not re-armed") {
			t.Fatalf("failures %q, comments %v", said, f.posted())
		}
		wantStandsArmed(t, rel, 11)
	})
	t.Run("moved to a head the gate refuses, the forge refuses the withdrawal", func(t *testing.T) {
		w, f, p := admitted(t)
		f.prs[11].head = headC
		f.refuse["disablePullRequestAutoMerge #11"] = "GraphQL: not permitted"
		releaseForge.armGate = refusing(headC)
		rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) == WithdrawnMoved {
			t.Fatalf("#11 recorded moved while it stands armed")
		}
		wantStandsArmed(t, rel, 11)
	})
	t.Run("moved to a head the gate refuses, the read-back fails", func(t *testing.T) {
		w, f, p := admitted(t)
		f.prs[11].head = headC
		f.prs[11].noReadBack = true
		releaseForge.armGate = refusing(headC)
		rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) == WithdrawnMoved {
			t.Fatalf("#11 recorded moved with no read-back")
		}
		line := failureLine(rel, 11)
		if !strings.Contains(line, "could not be read back") || strings.Contains(line, "stays withdrawn") {
			t.Fatalf("the line for #11 is %q", line)
		}
	})
	t.Run("moved to a head the gate refuses, another account's arming", func(t *testing.T) {
		w, f, p := admitted(t)
		releaseForge.comments = f
		f.prs[11].head = headC
		f.prs[11].by = "quinn"
		releaseForge.armGate = refusing(headC)
		rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnStanding {
			t.Fatalf("#11 recorded %q; want it standing", stateOf(ws, 11))
		}
		if f.pr(11).method == "" || slices.Contains(f.mutations(), "disablePullRequestAutoMerge #11") {
			t.Fatalf("#11 = %+v, mutations %v; another account's arming is left", f.pr(11), f.mutations())
		}
		said := strings.Join(f.posted(), "\n")
		for _, want := range []string{"#11 abcd release hold: the merge of #11 stays armed at " + headC[:12], "does not arm", "gh pr merge 11 --disable-auto"} {
			if !strings.Contains(said, want) {
				t.Fatalf("comments do not say %q: %v", want, f.posted())
			}
		}
		if strings.Contains(said, "#11 is not re-armed") || strings.Contains(said, "needs to re-arm") {
			t.Fatalf("comments say it needs re-arming: %v", f.posted())
		}
		lines := ""
		for _, s := range rel.Settles {
			for _, wd := range s.Settled {
				lines += SettledLine(s.Subject, wd) + "\n"
			}
		}
		if !strings.Contains(lines, "#11 stays armed") || !strings.Contains(lines, "gh pr merge 11 --disable-auto") || strings.Contains(lines, "#11 was not re-armed") {
			t.Fatalf("the person's line says %q", lines)
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

// A pull request armed at its recorded head is not withdrawn, so the gate's
// refusal there posts no sentence saying it is; the person is told.
func TestAnArmedPullRequestTheGateRefusesPostsNoStaysWithdrawn(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	releaseForge.comments = f
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.comments = nil
	f.prs[11].method = "SQUASH"
	releaseForge.armGate = noReceipt
	rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range f.posted() {
		if strings.HasPrefix(c, "#11 ") {
			t.Fatalf("a comment went to #11, which is armed: %q", c)
		}
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
	wantStandsArmed(t, rel, 11)
}

// wantStandsArmed holds the failure for pull request n to what is true of it: it
// stands armed and may merge, with the command that withdraws it.
func wantStandsArmed(t *testing.T, res Result, n int) {
	t.Helper()
	line := failureLine(res, n)
	for _, want := range []string{"stands armed", "may merge", fmt.Sprintf("gh pr merge %d --disable-auto", n)} {
		if !strings.Contains(line, want) {
			t.Fatalf("the line for #%d is %q; want %q", n, line, want)
		}
	}
	if strings.Contains(line, "stays withdrawn") || strings.Contains(line, "was not re-armed") {
		t.Fatalf("the line for #%d says it is withdrawn: %q", n, line)
	}
}

// failureLine is the surface's line for pull request n's failure.
func failureLine(res Result, n int) string {
	line := ""
	for _, s := range res.Settles {
		for _, ff := range s.Failures {
			if ff.Number == n {
				line = ff.Line()
			}
		}
	}
	return line
}

// A comment the forge took counts as told, whatever its read-back said.
func TestAHeldCommentPostedButNotAsSentIsNotPostedAgain(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	releaseForge.comments = f
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.comments = nil
	f.commentEcho = func(body string) string { return body + "\nA footer" }
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
		}
	}
	if held != 1 {
		t.Fatalf("#11 was told %d times: %v", held, f.posted())
	}
}

// A withdrawn pull request armed by hand at a moved head is asked of the gate like
// an admitted one, and settles by what is true of it.
func TestAWithdrawnMergeArmedAtAMovedHeadIsAskedOfTheGate(t *testing.T) {
	withdrawn := func(t *testing.T) (*world, *fakeForge, *participant) {
		w := newWorld(t)
		f := armedForge(t)
		releaseForge.comments = f
		p := w.clone("1234+octo@users.noreply.github.com", "")
		if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatal(err)
		}
		f.comments = nil
		f.prs[11].head, f.prs[11].method = headC, "SQUASH"
		f.reset()
		return w, f, p
	}
	refuseC := func(_, _, _, h string) (bool, string, error) {
		if h == headC {
			return false, "no review receipt names " + h, nil
		}
		return true, "", nil
	}
	for _, c := range []struct {
		name  string
		gate  ArmGate
		by    string
		state string
		says  string
	}{
		{"the gate arms it", func(_, _, _, _ string) (bool, string, error) { return true, "", nil }, "", WithdrawnFoundArmed, "is armed"},
		{"the gate refuses, its own arming", refuseC, "", WithdrawnMoved, "needs to re-arm"},
		{"the gate refuses, another account's", refuseC, "quinn", WithdrawnStanding, "stays armed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, f, p := withdrawn(t)
			f.prs[11].by = c.by
			releaseForge.armGate = c.gate
			if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
				t.Fatal(err)
			}
			if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != c.state {
				t.Fatalf("#11 recorded %q; want %q", stateOf(ws, 11), c.state)
			}
			told := ""
			for _, m := range f.posted() {
				if strings.HasPrefix(m, "#11 ") {
					told = m
				}
			}
			if !strings.Contains(told, c.says) {
				t.Fatalf("#11 was told %q; want %q", told, c.says)
			}
			if c.state == WithdrawnMoved && f.pr(11).method != "" {
				t.Fatalf("#11 = %+v; moved while it stands armed", f.pr(11))
			}
		})
	}
}

// An admission names the commits withdrawn, so one armed at a moved head is refused
// even at a head the gate arms.
func TestAnAdmissionOfAMergeArmedAtAMovedHeadIsRefused(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.prs[11].head, f.prs[11].method = headC, "SQUASH"
	if _, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "its head moved") {
		t.Fatalf("admit = %v; want refused", err)
	}
}

// A standing line the act just read is present tense, with the command. One read
// back from the tombstone is history: nothing retires it, so it says when it was
// true and to check before acting.
func TestAStandingLineFromTheTombstoneIsInThePastTense(t *testing.T) {
	now := SettledLine("rel-v0.16.0", Withdrawal{Number: 11, Head: headA, State: WithdrawnStanding, ArmedAt: headC})
	if !strings.Contains(now, "stays armed at "+headC[:12]) || !strings.Contains(now, "gh pr merge 11 --disable-auto") {
		t.Fatalf("the line from the act is %q", now)
	}
	then := SettledLine("rel-v0.16.0", Withdrawal{Number: 11, Head: headA, State: WithdrawnStanding})
	if !strings.Contains(then, "was left armed when") || !strings.Contains(then, "gh pr view 11") {
		t.Fatalf("the line from the tombstone is %q", then)
	}
	if strings.Contains(then, "--disable-auto") || strings.Contains(then, "stays armed") {
		t.Fatalf("the line from the tombstone reads as present: %q", then)
	}
}

// The listing reads no forge, so it does not promise a re-arm the gate may hold.
func TestTheListingOfDueWithdrawalsPromisesOnlyWhatTheGateAllows(t *testing.T) {
	r := ReleaseWithdrawals{Subject: "rel-v0.16.0", Ended: true, Due: []Withdrawal{{Number: 11, Head: headA, State: WithdrawnPending}}}
	line := strings.Join(r.Lines(), "\n")
	if !strings.Contains(line, "where the merge gate arms") || strings.Contains(line, "re-arms them\n") || strings.HasSuffix(line, "re-arms them") {
		t.Fatalf("the listing says %q", line)
	}
}

// --admit refuses in words true of the pull request: armed on the forge, it stands
// armed and may merge, with the command that withdraws it, whatever the refusal.
func TestARefusedAdmissionSaysWhatIsTrueOfItsArming(t *testing.T) {
	refuses := func(_, _, _, h string) (bool, string, error) { return false, "no review receipt names " + h, nil }
	unreadable := func(_, _, _, _ string) (bool, string, error) { return false, "", errors.New("a read failed") }
	for _, c := range []struct {
		name   string
		head   string
		method string
		gate   ArmGate
		armed  bool
	}{
		{"armed at its recorded head, the gate refusing", "", "SQUASH", refuses, true},
		{"armed at a moved head, the gate refusing", headC, "SQUASH", refuses, true},
		{"armed at a moved head the gate arms", headC, "SQUASH", nil, true},
		{"armed, the gate unreadable", "", "SQUASH", unreadable, true},
		{"armed at a moved head, the gate unreadable", headC, "SQUASH", unreadable, true},
		{"not armed, at a moved head", headC, "", refuses, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			f := armedForge(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
				t.Fatal(err)
			}
			if c.head != "" {
				f.prs[11].head = c.head
			}
			f.prs[11].method = c.method
			if c.gate != nil {
				releaseForge.armGate = c.gate
			}
			_, err := p.do(Request{Record: "rel-v0.16.0", Admit: 11})
			if !errors.Is(err, ErrRefused) {
				t.Fatalf("admit = %v; want refused", err)
			}
			says := err.Error()
			if c.armed {
				for _, want := range []string{"stands armed", "may merge", "gh pr merge 11 --disable-auto"} {
					if !strings.Contains(says, want) {
						t.Fatalf("the refusal does not say %q: %s", want, says)
					}
				}
				if strings.Contains(says, "stays withdrawn") {
					t.Fatalf("the refusal says it is withdrawn: %s", says)
				}
			} else if !strings.Contains(says, "stays withdrawn") || strings.Contains(says, "stands armed") {
				t.Fatalf("the refusal of an unarmed one says: %s", says)
			}
		})
	}
}

// A first read that fails while settling names what it was reading, under an act
// that says so.
func TestAFailedReadWhileSettlingSaysWhatItWasReading(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.prs[11].head = headC
	f.prs[11].views, f.prs[11].failViewAt = 0, 2
	rel, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	var got ForgeFailure
	for _, s := range rel.Settles {
		for _, ff := range s.Failures {
			if ff.Number == 11 {
				got = ff
			}
		}
	}
	if got.Act != "read" || !strings.Contains(got.Line(), "could not read whether it is armed") || !strings.Contains(got.Line(), "HTTP 502") {
		t.Fatalf("the failure is %+v, its line %q", got, got.Line())
	}
}

package guard

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// recordingProver answers every target with one proof and records what it
// was asked.
type recordingProver struct {
	proof DeleteProof
	asked []DeleteTarget
}

func (p *recordingProver) ProveDelete(t DeleteTarget) DeleteProof {
	p.asked = append(p.asked, t)
	return p.proof
}

// deleteRegistry is the registry loaded for dir and reading from it, with p
// proving its deletes (none when p is nil).
func deleteRegistry(t *testing.T, dir string, p DeleteProver) Registry {
	t.Helper()
	r, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	r = r.ReadingFrom(NewFiles(dir))
	if p != nil {
		r = r.ProvingDeletes(p)
	}
	return r
}

// TestGuardLeavesTheBundledDefaultsTiers pins the precedent the proof
// follows: the bundled defaults on their own, with no repository behind them,
// keep each delete entry's own tier.
func TestGuardLeavesTheBundledDefaultsTiers(t *testing.T) {
	d, err := Defaults().Check("git branch -D feat/x")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictWarn || d.EntryID != "git-branch-force-delete" {
		t.Errorf("verdict %q under %q, want the entry's own warn", d.Verdict, d.EntryID)
	}
}

// TestGuardBlocksADeleteNoProverProves pins the fail-closed default: a
// registry no front door attached a prover to refuses every branch delete and
// worktree removal under the delete id, with the person's one-command way
// past it named.
func TestGuardBlocksADeleteNoProverProves(t *testing.T) {
	r := deleteRegistry(t, t.TempDir(), nil)
	for _, line := range []string{
		"git push origin --delete feat/x",
		"git push origin :feat/x",
		"git branch -D feat/x",
		"git branch -d -f feat/x",
		"git worktree remove ../wt",
		"git worktree remove --force ../wt",
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != deleteEntryID {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, deleteEntryID)
		}
		if !strings.Contains(d.Message, "abcd guard allow --once") {
			t.Errorf("%q: the refusal does not name the person's allow: %s", line, d.Message)
		}
	}
}

// TestGuardAsksTheProverForEveryTargetTheLineFixes pins the reading: each
// spelling of a delete reaches the prover as the target it names, with git
// -C's directory, and a delete the prover proves safe is not refused for it.
func TestGuardAsksTheProverForEveryTargetTheLineFixes(t *testing.T) {
	dir := t.TempDir()
	rb := func(b string) DeleteTarget {
		return DeleteTarget{Kind: DeleteRemoteBranch, Dir: dir, Remote: "origin", Branch: b}
	}
	lb := func(b string) DeleteTarget { return DeleteTarget{Kind: DeleteLocalBranch, Dir: dir, Branch: b} }
	cases := map[string][]DeleteTarget{
		"git push origin --delete a b":          {rb("a"), rb("b")},
		"git push --delete origin refs/heads/a": {rb("a")},
		"git push -d origin a":                  {rb("a")},
		"git push --del origin a":               {rb("a")},
		"git push -qd origin a":                 {rb("a")},
		"git push origin :a :b":                 {rb("a"), rb("b")},
		"git push -o ci.skip origin --delete a": {rb("a")},
		"git push --repo=origin origin :a":      {rb("a")},
		"git branch -D a b":                     {lb("a"), lb("b")},
		"git branch -d -f a":                    {lb("a")},
		"git branch --delete --force a":         {lb("a")},
		"git branch --del --forc a":             {lb("a")},
		"git -C sub branch -D a": {
			{Kind: DeleteLocalBranch, Dir: filepath.Join(dir, "sub"), Branch: "a"},
		},
		"git worktree remove ../wt":               {{Kind: DeleteWorktree, Dir: dir, Path: filepath.Join(filepath.Dir(dir), "wt")}},
		"git worktree remove -f /srv/wt":          {{Kind: DeleteWorktree, Dir: dir, Path: "/srv/wt"}},
		"git push origin main && git branch -D a": {lb("a")},
	}
	for line, want := range cases {
		p := &recordingProver{proof: DeleteProof{Safe: true}}
		d, err := deleteRegistry(t, dir, p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if !reflect.DeepEqual(p.asked, want) {
			t.Errorf("%q: asked %+v, want %+v", line, p.asked, want)
		}
		if d.Verdict == VerdictBlock {
			t.Errorf("%q: a proven delete is blocked under %q", line, d.EntryID)
		}
	}
}

// TestGuardRefusesADeleteItsProverFindsUnsafe pins the refusal's text: it
// names the target, the check that failed and what settles it.
func TestGuardRefusesADeleteItsProverFindsUnsafe(t *testing.T) {
	p := &recordingProver{proof: DeleteProof{Failed: "pull request #12 is open on it", Settle: "Close or merge #12 first."}}
	d, err := deleteRegistry(t, t.TempDir(), p).Check("git push origin --delete feat/x")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != deleteEntryID {
		t.Fatalf("verdict %q under %q, want a block under %q", d.Verdict, d.EntryID, deleteEntryID)
	}
	for _, want := range []string{"the branch feat/x on origin", "pull request #12 is open on it", "Close or merge #12 first."} {
		if !strings.Contains(d.Message, want) {
			t.Errorf("the refusal does not say %q: %s", want, d.Message)
		}
	}
}

// TestGuardRefusesADeleteWhoseTargetsItCannotRead pins every doubt the
// reading fails closed on: the prover, which would prove anything safe, is
// never asked, and the line is refused under the delete id.
func TestGuardRefusesADeleteWhoseTargetsItCannotRead(t *testing.T) {
	for _, line := range []string{
		`git branch -D "$b"`,
		`git push origin --delete "$(git branch --show-current)"`,
		`git -C "$d" branch -D a`,
		`cd ../other && git branch -D a`,
		`GIT_DIR=/x/.git git branch -D a`,
		`git push origin --delete 'feat/*'`,
		`git branch -D refs/heads/a`,
		`git push --frobnicate origin :a`,
		`git push origin --delete`,
		`git branch -D --frob a`,
		`git worktree remove a b`,
		`git worktree remove --sideways a`,
		`git --exec-path=/x branch -D a`,
		`git branch -D @{-1}`,
		`git push origin --delete @{-1}`,
		`git branch -D @`,
		`git for-each-ref --format="%(refname:short)" refs/heads/old/ | xargs -I{} git branch -D {}`,
		`printf "a\nb\n" | xargs git branch -D keep`,
		`printf "a\n" | xargs -I{} git push origin --delete {}`,
		`git branch -D {}`,
	} {
		p := &recordingProver{proof: DeleteProof{Safe: true}}
		d, err := deleteRegistry(t, t.TempDir(), p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock {
			t.Errorf("%q: verdict %q, want a block", line, d.Verdict)
		}
		if len(p.asked) != 0 {
			t.Errorf("%q: the prover was asked %+v of a line the reading could not fix", line, p.asked)
		}
	}
}

// TestGuardLeavesWhatDeletesNothingItJudges pins the other side: a line no
// delete entry fires on is never read for targets, and a delete of
// remote-tracking refs, which removes only this copy's record of a remote
// branch, has nothing to prove.
func TestGuardLeavesWhatDeletesNothingItJudges(t *testing.T) {
	for _, line := range []string{
		"git push origin feat/x",
		"git branch -d feat/x",
		"git branch -Dr origin/feat/x",
		"git push origin --delete refs/tags/v1",
		"git push origin :",
		"git push -n origin --delete feat/x",
		"git push --dry-run origin :feat/x",
		"git push origin :refs/tags/v1",
		"git worktree list",
		"git status",
	} {
		p := &recordingProver{proof: DeleteProof{Failed: "never safe"}}
		d, err := deleteRegistry(t, t.TempDir(), p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict == VerdictBlock || len(p.asked) != 0 {
			t.Errorf("%q: verdict %q under %q, prover asked %+v", line, d.Verdict, d.EntryID, p.asked)
		}
	}
}

// TestDeleteLessonIsTheRefusal: the teaching plane's rule for the delete proof
// is the refusal's own words, so an agent taught the delete entries' warns is
// also taught that, in a repository, an unproven delete is refused, and the
// person's one-shot way past it.
func TestDeleteLessonIsTheRefusal(t *testing.T) {
	got := Defaults().DeleteLesson()
	sig := deleteRefusal(nil, nil)
	if want := "Refused by the guard (" + deleteEntryID + "), in a repository: "; !strings.HasPrefix(got, want) {
		t.Errorf("the delete lesson does not open with %q:\n%s", want, got)
	}
	for _, want := range []string{"`git push --delete`", "`git branch -D`", "`git worktree remove`", "no open pull request",
		"a deleted branch or worktree takes with it whatever only it held", "abcd guard allow --once", "an agent cannot"} {
		if !strings.Contains(got, want) {
			t.Errorf("the delete lesson does not carry %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, strings.TrimPrefix(sig.successor, " ")) {
		t.Errorf("the delete lesson's instead is not the refusal's:\n%s", got)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("the delete lesson spans lines; a rule is one bullet:\n%s", got)
	}
	off := Defaults()
	off.Disabled = true
	if l := off.DeleteLesson(); !strings.HasPrefix(l, guardOffLead+" ("+deleteEntryID+")") {
		t.Errorf("under a disabled registry the delete lesson does not say the guard is off:\n%s", l)
	}
}

// TestGuardRefusesADeleteInARepositoryTheLineMayChange pins the repository a
// proof runs in: a directory change or a GIT_ word anywhere on the line, in
// any spelling, refuses the delete, since git may then delete in a repository
// other than the one proved.
func TestGuardRefusesADeleteInARepositoryTheLineMayChange(t *testing.T) {
	for _, line := range []string{
		`command cd /other && git branch -D feat`,
		`builtin cd /other; git branch -D feat`,
		`X=1 cd /other && git branch -D feat`,
		`export GIT_DIR=/other/.git; git branch -D feat`,
		`git branch -D feat; pushd /other`,
	} {
		p := &recordingProver{proof: DeleteProof{Safe: true}}
		d, err := deleteRegistry(t, t.TempDir(), p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || len(p.asked) != 0 {
			t.Errorf("%q: verdict %q, prover asked %+v; want a block before any proof", line, d.Verdict, p.asked)
		}
	}
}

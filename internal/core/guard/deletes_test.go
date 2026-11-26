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
		"git push --delete origin refs/heads/a": {{Kind: DeleteRemoteBranch, Dir: dir, Remote: "origin", Branch: "a", Ref: "refs/heads/a"}},
		"git push origin --delete heads/a":      {rb("heads/a")},
		"git push origin :tags/v1":              {rb("tags/v1")},
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

// TestTheDeleteRefusalNamesTheSweep: once `abcd sweep` is wired
// (spc-2610230730269295, step 5), every refusal of a delete the guard cannot
// prove safe, and the lesson made from it, names the sweep as what settles a
// spent branch or worktree, the dry run to see it and --apply to clear it.
func TestTheDeleteRefusalNamesTheSweep(t *testing.T) {
	p := &recordingProver{proof: DeleteProof{Failed: "pull request #12 is open on it", Settle: "Close or merge #12 first."}}
	d, err := deleteRegistry(t, t.TempDir(), p).Check("git push origin --delete feat/x")
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"the refusal": d.Message, "the lesson": Defaults().DeleteLesson()} {
		for _, want := range []string{"`abcd sweep`", "`abcd sweep --apply`", "spent branch or worktree"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not name %q: %s", name, want, text)
			}
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

// TestGuardRefusesAPushThatPrunesOrMirrors pins the pushes that delete what
// the line never names: --prune and --mirror, in every spelling git accepts,
// on the line, in a payload, through an alias or a setting the line carries,
// delete each branch of the remote's this copy lacks. The prover, which would
// prove anything safe, is never asked, and the line is refused under the
// delete id with the safe form named.
func TestGuardRefusesAPushThatPrunesOrMirrors(t *testing.T) {
	for _, line := range []string{
		`git push --prune origin 'refs/heads/*:refs/heads/*'`,
		`git push origin --prune 'refs/heads/*:refs/heads/*'`,
		`git push --prune origin main`,
		`git push --prune`,
		`git push --pru origin 'refs/heads/*:refs/heads/*'`,
		`git push --pr origin 'refs/heads/*:refs/heads/*'`,
		`git push --all --prune origin`,
		`git push --no-prune --prune origin 'refs/heads/*:refs/heads/*'`,
		`git push --dry-run --no-dry-run --prune origin 'refs/heads/*:refs/heads/*'`,
		`git push --prune origin --delete feat/x`,
		`git push --prune origin :feat/x`,
		`git -C sub push --prune origin 'refs/heads/*:refs/heads/*'`,
		`git push --mirror origin`,
		`git push --mirr origin`,
		`git push --mirror`,
		`git -c remote.origin.mirror=true push origin`,
		`git -c alias.p='push --prune' p origin 'refs/heads/*:refs/heads/*'`,
		`sh -c "git push --prune origin 'refs/heads/*:refs/heads/*'"`,
		execPath + `git-push --prune origin 'refs/heads/*:refs/heads/*'`,
	} {
		p := &recordingProver{proof: DeleteProof{Safe: true}}
		d, err := deleteRegistry(t, t.TempDir(), p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || !containsString(d.Matches, deleteEntryID) {
			t.Errorf("%q: verdict %q under %q (matches %v), want a block under %q", line, d.Verdict, d.EntryID, d.Matches, deleteEntryID)
		}
		if len(p.asked) != 0 {
			t.Errorf("%q: the prover was asked %+v of a push whose deletes the line does not name", line, p.asked)
		}
		// A mirror is also the git-push-mirror blocker's, whose refusal speaks
		// for the line and which no allow passes.
		if d.EntryID == deleteEntryID && (!strings.Contains(d.Message, "every target spelled out") || !strings.Contains(d.Message, "`abcd sweep`")) {
			t.Errorf("%q: the refusal does not name the safe form: %s", line, d.Message)
		}
		if mirror := strings.Contains(line, "mir"); mirror == d.AllowableOnce || mirror != (d.EntryID == "git-push-mirror") {
			t.Errorf("%q: refused under %q, allowable once %v", line, d.EntryID, d.AllowableOnce)
		}
	}
}

// TestGuardLeavesAPruneThatIsNoPush pins the other side: a fetch or a
// `git remote prune` with --prune removes only this copy's record of the
// remote's branches, a push that turns --prune or --mirror back off or is a
// dry run deletes nothing, and a plain push reads as it did.
func TestGuardLeavesAPruneThatIsNoPush(t *testing.T) {
	for _, line := range []string{
		"git fetch --prune origin",
		"git fetch -p origin",
		"git fetch --prune origin 'refs/heads/*:refs/remotes/origin/*'",
		"git remote prune origin",
		"git pull --prune origin main",
		"git push --prune --no-prune origin main",
		"git push --mirror --no-mirror origin main",
		"git push --dry-run --prune origin 'refs/heads/*:refs/heads/*'",
		"git push -n --mirror origin",
		"git push origin main",
		"git push origin 'refs/heads/*:refs/heads/*'",
		"git push --all origin",
		"git log --grep push --prune",
	} {
		p := &recordingProver{proof: DeleteProof{Failed: "never safe"}}
		d, err := deleteRegistry(t, t.TempDir(), p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if containsString(d.Matches, deleteEntryID) || len(p.asked) != 0 {
			t.Errorf("%q: verdict %q under %q (matches %v), prover asked %+v", line, d.Verdict, d.EntryID, d.Matches, p.asked)
		}
	}
}

// TestGuardReadsADryRunTurnedBackOff pins --no-dry-run: a push whose dry run
// a later --no-dry-run turns off deletes what it names, so each target
// reaches the prover.
func TestGuardReadsADryRunTurnedBackOff(t *testing.T) {
	dir := t.TempDir()
	for _, line := range []string{
		"git push --dry-run --no-dry-run origin --delete a",
		"git push -n --no-dry-run origin :a",
	} {
		p := &recordingProver{proof: DeleteProof{Failed: "never safe"}}
		d, err := deleteRegistry(t, dir, p).Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		want := []DeleteTarget{{Kind: DeleteRemoteBranch, Dir: dir, Remote: "origin", Branch: "a"}}
		if !reflect.DeepEqual(p.asked, want) || d.Verdict != VerdictBlock {
			t.Errorf("%q: verdict %q, asked %+v, want %+v refused", line, d.Verdict, p.asked, want)
		}
	}
}

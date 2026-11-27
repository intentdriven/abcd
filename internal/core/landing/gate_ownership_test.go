package landing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The two rulings of 2026-11-26 the merge gate holds (itd-2611252049031692,
// adr-2611262044127943): a person's own global auto merges only the pull
// requests their own account opened ("only your own"), and abcd never arms a
// change to the repository's committed facilitator.mode line, in any mode.

// globalAuto makes the fixture's home the account's own and sets the person's
// global setting there to config.
func globalAuto(t *testing.T, f *fixture, config string) {
	t.Helper()
	t.Setenv("HOME", f.home)
	dir := abcdhome.Path(f.home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestAPersonsGlobalAutoArmsOnlyTheirOwnPullRequest: where the mode is auto by
// the person's global setting, a reviewed pull request their own account
// opened is armed; a colleague's, with no approval of its head, waits as for
// human and says whose it is, and one an approval counts for is armed as
// human arms it. An account the forge cannot name reads as not the author.
// Where the repository enforces auto, a colleague's goes through as shipped.
func TestAPersonsGlobalAutoArmsOnlyTheirOwnPullRequest(t *testing.T) {
	t.Run("your own", func(t *testing.T) {
		f := newFixture(t, queueRuleset)
		globalAuto(t, f, autoConfig)
		f.forge.reviews = "[]"
		f.reviewed(t)
		wantArmed(t, f, landGate(t, f))
	})
	t.Run("a colleague's, unapproved", func(t *testing.T) {
		f := newFixture(t, queueRuleset)
		globalAuto(t, f, autoConfig)
		f.forge.reviews, f.forge.viewer = "[]", "colleague-z"
		f.reviewed(t)
		s := landGate(t, f)
		wantLeft(t, f, s, "opened by author-a, not by colleague-z")
		if !strings.Contains(s.Note, "only a pull request your own account opened") {
			t.Errorf("the note %q does not say why", s.Note)
		}
	})
	t.Run("a colleague's, approved", func(t *testing.T) {
		f := newFixture(t, queueRuleset)
		globalAuto(t, f, autoConfig)
		f.forge.viewer = "colleague-z"
		wantArmed(t, f, landGate(t, f))
	})
	t.Run("your own, its login in another case", func(t *testing.T) {
		f := newFixture(t, queueRuleset)
		globalAuto(t, f, autoConfig)
		f.forge.reviews, f.forge.viewer = "[]", "Author-A"
		f.reviewed(t)
		wantArmed(t, f, landGate(t, f))
	})
	t.Run("an account answer that does not parse", func(t *testing.T) {
		f := newFixture(t, queueRuleset)
		globalAuto(t, f, autoConfig)
		f.forge.reviews, f.forge.viewerAnswer = "[]", `{"data":{"viewer":null}}`
		f.reviewed(t)
		wantLeft(t, f, landGate(t, f), "naming which account abcd runs as does not parse")
	})
	t.Run("an account the forge cannot name", func(t *testing.T) {
		f := newFixture(t, queueRuleset)
		globalAuto(t, f, autoConfig)
		f.forge.reviews, f.forge.viewerErr = "[]", errors.New("gh: HTTP 502")
		f.reviewed(t)
		wantLeft(t, f, landGate(t, f), "which account abcd runs as")
	})
	t.Run("the repository enforces auto", func(t *testing.T) {
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		f.forge.reviews, f.forge.viewer = "[]", "colleague-z"
		f.reviewed(t)
		wantArmed(t, f, landGate(t, f))
		if f.forge.called("api", "graphql", "-f", "query="+ViewerQuery) != 0 {
			t.Error("the gate asked who it runs as where the repository enforces auto")
		}
	})
}

// modeLineFixture is a fixture whose base commits .abcd/config.json as base
// ("" for no file) and whose branch then writes it as branch ("-" removes it).
func modeLineFixture(t *testing.T, base, branch string) *fixture {
	t.Helper()
	files := map[string]string{}
	if base != "" {
		files[".abcd/config.json"] = base
	}
	f := newFixtureFiles(t, queueRuleset, files)
	if branch == "-" {
		f.repo.Git("rm", "-q", ".abcd/config.json")
	} else {
		f.repo.Write(".abcd/config.json", branch)
	}
	f.repo.Commit("touch the config")
	return f
}

// TestAChangeToTheCommittedModeLineIsNeverArmed: a pull request whose diff
// adds, removes or alters facilitator.mode in .abcd/config.json, or leaves it
// unreadable, is armed in no mode, whatever approvals and receipts it has, and
// the note says the person merges it on the forge's page. A change to another
// key of the file, and a branch whose base the default branch has moved past
// on the line, are armed as before: the pull request's diff is read from the
// branch's merge base with the default branch, not from the tip. A re-arm is
// held to the same refusal.
func TestAChangeToTheCommittedModeLineIsNeverArmed(t *testing.T) {
	const other = `{"facilitator":{"mode":"auto"},"land":{"push":true}}` + "\n"
	for _, c := range []struct {
		name, base, branch string
		armed              bool
	}{
		{"enforced auto made human", autoConfig, humanConfig, false},
		{"prevented auto made auto", humanConfig, autoConfig, false},
		{"auto added", "", autoConfig, false},
		{"the line removed", autoConfig, `{}` + "\n", false},
		{"an explicit human removed", humanConfig, `{}` + "\n", false},
		{"the file removed", autoConfig, "-", false},
		{"left unreadable", autoConfig, `{"facilitator":{"mode":"robot"}}` + "\n", false},
		{"another key changed", autoConfig, other, true},
	} {
		for _, approved := range []bool{true, false} {
			name := c.name + ", unapproved"
			if approved {
				name = c.name + ", approved"
			}
			t.Run(name, func(t *testing.T) {
				f := modeLineFixture(t, c.base, c.branch)
				if !approved {
					f.forge.reviews = "[]"
				}
				f.reviewed(t)
				s := landGate(t, f)
				switch {
				case c.armed && (approved || c.base == autoConfig):
					wantArmed(t, f, s)
				case c.armed:
					wantLeft(t, f, s, "waits for the technical facilitator's approval")
				default:
					wantLeft(t, f, s, "on the forge's page")
					if !strings.Contains(s.Note, "facilitator.mode") {
						t.Errorf("the note %q does not name the line", s.Note)
					}
				}
			})
		}
	}
	// What lands is the merge of the head into the tip, so a head that reads
	// as its merge base does is still refused where the merge moves the line
	// the tip holds.
	for _, c := range []struct {
		name, says string
		build      func(t *testing.T) *fixture
	}{
		{"a second merge base that moved the line", "at the merge base", secondBase},
		{"a rename from before the line", "once merged", renamedFromBefore},
		{"the line wrapped under another key", "once merged", wrappedLine},
		{"a merge that conflicts in the file", "conflicts in .abcd/config.json", conflictedFile},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := c.build(t)
			f.reviewed(t)
			s := landGate(t, f)
			wantLeft(t, f, s, "on the forge's page")
			if !strings.Contains(s.Note, "facilitator.mode") || !strings.Contains(s.Note, c.says) {
				t.Errorf("the note %q does not name the line and %q", s.Note, c.says)
			}
		})
	}
	// A rebase lands the head's commits replayed onto the tip, and abcd does
	// not judge that replay: where the queue rebases, or the method is not
	// known (a re-arm), a head with any commit that touches the file is left
	// to the person, whatever the reads at the head and in the merge say.
	rebaseRuleset := strings.Replace(queueRuleset, `"MERGE"`, `"REBASE"`, 1)
	const touches = "touches .abcd/config.json"
	for _, c := range []struct {
		name  string
		build func(*testing.T, string) *fixture
	}{
		{"a rebase of a head that sets the file aside and back", replayRestores},
		{"a rebase of a head whose last commit is already upstream", alreadyUpstream},
		{"a rebase of a head that holds a merge commit", mergedSides},
		{"a rebase of a head that changes another key of the file", otherKeyChanged},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := c.build(t, rebaseRuleset)
			f.reviewed(t)
			s := landGate(t, f)
			wantLeft(t, f, s, "on the forge's page")
			if !strings.Contains(s.Note, touches) {
				t.Errorf("the note %q does not say a commit %s", s.Note, touches)
			}
		})
	}
	t.Run("a rebase of a head that leaves the file alone", func(t *testing.T) {
		f := newFixtureFiles(t, rebaseRuleset, map[string]string{".abcd/config.json": autoConfig})
		f.reviewed(t)
		wantArmed(t, f, landGate(t, f))
	})
	t.Run("the same commits merged", func(t *testing.T) {
		f := replayRestores(t, queueRuleset)
		f.reviewed(t)
		wantArmed(t, f, landGate(t, f))
	})
	t.Run("the re-arm of a head that touches the file", func(t *testing.T) {
		f := replayRestores(t, queueRuleset)
		f.forge.head = f.head()
		f.reviewed(t)
		ask := func(args ...string) (string, error) { return f.forge.run(f.repo.Root(), args...) }
		arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head())
		if arm || err == nil || !strings.Contains(err.Error(), touches) {
			t.Fatalf("arm %v, why %q, err %v: want no re-arm, whose method is not known, of a head with a commit that %s", arm, why, err, touches)
		}
	})
	t.Run("the re-arm of a head that leaves the file alone", func(t *testing.T) {
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		f.forge.head = f.head()
		f.reviewed(t)
		ask := func(args ...string) (string, error) { return f.forge.run(f.repo.Root(), args...) }
		if arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head()); !arm || err != nil {
			t.Fatalf("arm %v, why %q, err %v: want the re-arm", arm, why, err)
		}
	})
	// The merge is computed with git 2.38's merge-tree --write-tree: on an
	// older git nothing is armed, and the answer names the floor.
	t.Run("a git too old to compute the merge", func(t *testing.T) {
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		oldGit(t, "2.38")
		tip := f.repo.Git("rev-parse", "origin/main")
		d, err := DecideMerge(MergeQuery{RepoRoot: f.repo.Root(), Roots: f.options().Roots, Tip: tip, TipName: "origin/main", Default: "main",
			Rule: MergeRule{Method: "MERGE"}, RuleRead: true,
			Repo: "o/r", PR: "7", Head: f.head(), Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }})
		if says := "needs git 2.38 or later, and this git is 2.37"; err != nil || d.Arm || !d.Unread || !strings.Contains(d.Why, says) {
			t.Fatalf("decision %+v, %v: want nothing armed, unread, saying %q", d, err, says)
		}
	})
	t.Run("the re-arm of a second merge base", func(t *testing.T) {
		f := secondBase(t)
		f.forge.head = f.head()
		f.reviewed(t)
		ask := func(args ...string) (string, error) { return f.forge.run(f.repo.Root(), args...) }
		arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head())
		if err != nil || arm || !strings.Contains(why, "on the forge's page") {
			t.Fatalf("arm %v, why %q, err %v: want no re-arm of a merge that moves the mode line", arm, why, err)
		}
	})
	t.Run("a head the checkout does not hold", func(t *testing.T) {
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		tip := f.repo.Git("rev-parse", "origin/main")
		absent := strings.Repeat("ab", 20)
		d, err := DecideMerge(MergeQuery{RepoRoot: f.repo.Root(), Roots: f.options().Roots, Tip: tip, TipName: "origin/main", Default: "main",
			Repo: "o/r", PR: "7", Head: absent, Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }})
		if err != nil || d.Arm || !d.Unread || !strings.Contains(d.Why, "the merge bases of "+absent[:12]+" and origin/main cannot be read (") {
			t.Fatalf("decision %+v, %v: want nothing armed, unread, naming the merge bases and why", d, err)
		}
	})
	t.Run("the re-arm", func(t *testing.T) {
		f := modeLineFixture(t, autoConfig, humanConfig)
		f.forge.head = f.head()
		f.reviewed(t)
		ask := func(args ...string) (string, error) { return f.forge.run(f.repo.Root(), args...) }
		arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head())
		if err != nil || arm || !strings.Contains(why, "on the forge's page") {
			t.Fatalf("arm %v, why %q, err %v: want no re-arm of a change to the mode line", arm, why, err)
		}
	})
	t.Run("a branch behind a tip that moved the line", func(t *testing.T) {
		f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
		f.repo.Git("checkout", "-q", "main")
		f.repo.Write(".abcd/config.json", humanConfig)
		f.repo.Commit("main changes the mode line")
		f.repo.Git("push", "-q", "origin", "main")
		f.repo.Git("fetch", "-q", "origin")
		f.repo.Git("checkout", "-q", "feat/x")
		f.reviewed(t)
		wantArmed(t, f, landGate(t, f))
	})
}

// oldGit makes the gate's version check answer as a git whose version is
// one minor release below floor ("2.38" answers 2.37 for 2.38 and above,
// and at the floor for anything older), for the rest of the test.
func oldGit(t *testing.T, floor string) {
	t.Helper()
	var major, minor int
	if _, err := fmt.Sscanf(floor, "%d.%d", &major, &minor); err != nil {
		t.Fatal(err)
	}
	was := gitAtLeast
	gitAtLeast = func(_ context.Context, ma, mi int) (bool, string, error) {
		return ma < major || (ma == major && mi < minor), fmt.Sprintf("%d.%d", major, minor-1), nil
	}
	t.Cleanup(func() { gitAtLeast = was })
}

// pushMain pushes the fixture's main to origin and fetches it back.
func pushMain(f *fixture) {
	f.repo.Git("push", "-q", "origin", "main")
	f.repo.Git("fetch", "-q", "origin")
}

// secondBase: the project commits auto; a branch x removes the line and is
// merged by hand, with one commit landing on main between x's fork and its
// merge. The head, cut from that commit, merges x keeping its own side, so
// its file reads auto as one of its merge bases does, while the merge of the
// head into the tip, which holds no line, restores auto.
func secondBase(t *testing.T) *fixture {
	t.Helper()
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.repo.Git("checkout", "-q", "main")
	f.repo.Git("checkout", "-q", "-b", "x")
	f.repo.Write(".abcd/config.json", `{}`+"\n")
	f.repo.Commit("x removes the line")
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write("between.txt", "a commit between\n")
	f.repo.Commit("between")
	f.repo.Git("merge", "-q", "--no-ff", "-m", "merge x", "x")
	pushMain(f)
	f.repo.Git("checkout", "-q", "-B", "feat/x", "main~1")
	f.repo.Git("merge", "-q", "-s", "ours", "-m", "take x, keep ours", "x")
	f.repo.Write("feature.txt", "the change\n")
	f.repo.Commit("the change")
	return f
}

// replayRestores: the project commits auto and main makes it human; the head,
// cut from before, renames the file aside and then adds it back with the
// base's text. The head reads as its base does, and its merge into the tip
// keeps the tip's human, while its commits replayed onto the tip carry the
// tip's file aside and add the base's auto back.
func replayRestores(t *testing.T, ruleset string) *fixture {
	t.Helper()
	const block = "{\n  \"land\": {\n    \"push\": true\n  },\n  \"facilitator\": {\n    \"mode\": \"%s\"\n  }\n}\n"
	f := newFixtureFiles(t, ruleset, map[string]string{".abcd/config.json": fmt.Sprintf(block, "auto")})
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(".abcd/config.json", fmt.Sprintf(block, "human"))
	f.repo.Commit("main makes it human")
	pushMain(f)
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Git("mv", ".abcd/config.json", ".abcd/config.old.json")
	f.repo.Commit("set the config aside")
	f.repo.Write(".abcd/config.json", fmt.Sprintf(block, "auto"))
	f.repo.Commit("add the config back")
	return f
}

// modeBlock is a multi-line .abcd/config.json committing mode.
func modeBlock(mode string) string {
	return "{\n  \"land\": {\n    \"push\": true\n  },\n  \"facilitator\": {\n    \"mode\": \"" + mode + "\"\n  }\n}\n"
}

// alreadyUpstream: the project commits human; main makes it auto and then
// human again. The head, cut from before, sets the file aside, adds it back
// reading auto, and then makes it human with main's own last patch. Base,
// head and merge all read human; a rebase that skips the commit already
// upstream lands auto.
func alreadyUpstream(t *testing.T, ruleset string) *fixture {
	t.Helper()
	f := newFixtureFiles(t, ruleset, map[string]string{".abcd/config.json": modeBlock("human")})
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(".abcd/config.json", modeBlock("auto"))
	f.repo.Commit("main makes it auto")
	f.repo.Write(".abcd/config.json", modeBlock("human"))
	f.repo.Commit("main makes it human again")
	pushMain(f)
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Git("mv", ".abcd/config.json", ".abcd/config.old.json")
	f.repo.Commit("set the config aside")
	f.repo.Write(".abcd/config.json", modeBlock("auto"))
	f.repo.Commit("add it back reading auto")
	f.repo.Write(".abcd/config.json", modeBlock("human"))
	f.repo.Commit("make it human")
	return f
}

// mergedSides: the project commits human, and the default branch stays
// there. A side branch makes the file auto and human again; the head makes
// it auto, merges the side branch keeping the side's tree, and adds another
// file. Base, head and merge all read human; a replay that orders the
// commits otherwise than git does lands auto.
func mergedSides(t *testing.T, ruleset string) *fixture {
	t.Helper()
	f := newFixtureFiles(t, ruleset, map[string]string{".abcd/config.json": modeBlock("human")})
	f.repo.Git("checkout", "-q", "-b", "y", "main")
	f.repo.Write(".abcd/config.json", modeBlock("auto"))
	f.repo.Commit("y makes it auto")
	f.repo.Write(".abcd/config.json", modeBlock("human"))
	f.repo.Commit("y makes it human")
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Write(".abcd/config.json", modeBlock("auto"))
	f.repo.Commit("x makes it auto")
	f.repo.Git("merge", "-q", "-s", "ours", "--no-commit", "y")
	f.repo.Git("read-tree", "-u", "--reset", "y")
	f.repo.Commit("merge y, keeping its tree")
	f.repo.Write("other.txt", "another file\n")
	f.repo.Commit("add another file")
	return f
}

// otherKeyChanged: the head changes another key of the file and leaves the
// mode line as it is.
func otherKeyChanged(t *testing.T, ruleset string) *fixture {
	t.Helper()
	f := newFixtureFiles(t, ruleset, map[string]string{".abcd/config.json": autoConfig})
	f.repo.Write(".abcd/config.json", `{"facilitator":{"mode":"auto"},"land":{"push":true}}`+"\n")
	f.repo.Commit("turn the push on")
	return f
}

// renamedFromBefore: the file exists without the line, main then adds the
// line, and the head, cut from before it, renames the file. Base and head
// both read no line, while the merge carries main's change to the new path
// and leaves the tip's line gone.
func renamedFromBefore(t *testing.T) *fixture {
	t.Helper()
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": "{\n  \"land\": {\n    \"push\": true\n  }\n}\n"})
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(".abcd/config.json", "{\n  \"land\": {\n    \"push\": true\n  },\n  \"facilitator\": {\n    \"mode\": \"auto\"\n  }\n}\n")
	f.repo.Commit("main adds the line")
	pushMain(f)
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Git("mv", ".abcd/config.json", ".abcd/config.old.json")
	f.repo.Commit("rename the config")
	return f
}

// wrappedLine: the project commits auto and main changes it in place to
// human; the head, cut from before, adds a new top-level auto and leaves the
// old block where it was under another key. Base and head both read auto,
// while the merge keeps the head's auto over the tip's human.
func wrappedLine(t *testing.T) *fixture {
	t.Helper()
	const block = "  \"facilitator\": {\n    \"mode\": \"%s\"\n  }\n"
	base := "{\n  \"land\": {\n    \"push\": true\n  },\n" + fmt.Sprintf(block, "auto") + "}\n"
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": base})
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(".abcd/config.json", "{\n  \"land\": {\n    \"push\": true\n  },\n"+fmt.Sprintf(block, "human")+"}\n")
	f.repo.Commit("main makes it human")
	pushMain(f)
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Write(".abcd/config.json", "{\n  \"facilitator\": {\n    \"mode\": \"auto\"\n  },\n  \"wrapped\": {\n\n  \"land\": {\n    \"push\": true\n  },\n"+fmt.Sprintf(block, "auto")+"\n  }\n}\n")
	f.repo.Commit("wrap the old block")
	return f
}

// conflictedFile: the file is one line, and main and the head each change it
// elsewhere than the mode, so the merge conflicts in the file while base and
// head read the same mode.
func conflictedFile(t *testing.T) *fixture {
	t.Helper()
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": `{"facilitator":{"mode":"auto"},"land":{"push":true}}` + "\n"})
	f.repo.Git("checkout", "-q", "main")
	f.repo.Write(".abcd/config.json", `{"facilitator":{"mode":"auto"},"land":{"push":false}}`+"\n")
	f.repo.Commit("main turns the push off")
	pushMain(f)
	f.repo.Git("checkout", "-q", "feat/x")
	f.repo.Write(".abcd/config.json", `{"facilitator":{"mode":"auto"},"land":{"push":true,"enqueue":true}}`+"\n")
	f.repo.Commit("the head turns the queue on")
	return f
}

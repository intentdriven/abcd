package gitutil

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestAReadOnlyMergeRunsNoFilterUnderRenormalize: merge.renormalize makes a
// three-way content merge run each side through the content filters the
// attributes select (git-config(1), merge.renormalize; gitattributes(5)), and
// merge-tree reads it, so a read-only merge abcd computes under
// MergeDriverOverrides alone started the repository's clean filter with no
// check asked. MergeDriverOverrides pins it off, and the command-line judge
// takes a merge-tree as starting nothing only when it carries that pin too.
func TestAReadOnlyMergeRunsNoFilterUnderRenormalize(t *testing.T) {
	repo := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	// The isolated runner itself: ExecPins and no global or system settings.
	git := func(args ...string) {
		t.Helper()
		if _, err := Run(repo, append([]string{"-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, "x.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, ".gitattributes"), []byte("*.txt filter=f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	write("a\n1\n2\n3\n4\nb\n")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("switch", "-q", "-c", "side")
	write("a\n1\n2\n3\n4\nB\n")
	git("commit", "-q", "-a", "-m", "side")
	git("switch", "-q", "main")
	write("A\n1\n2\n3\n4\nb\n")
	git("commit", "-q", "-a", "-m", "main")
	git("config", "filter.f.clean", "touch '"+marker+"'; cat")
	git("config", "filter.f.smudge", "cat")
	git("config", "merge.renormalize", "true")

	overrides, err := MergeDriverOverrides(repo)
	if err != nil {
		t.Fatal(err)
	}
	args := append(slices.Clip(overrides), "merge-tree", "--write-tree", "main", "side")
	if _, err := Run(repo, args...); err != nil {
		t.Fatalf("merge-tree: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("the read-only merge ran the repository's clean filter under merge.renormalize (overrides %q)", overrides)
	}
	if LeavesContentProgramsOn(args) {
		t.Errorf("a merge-tree carrying MergeDriverOverrides must start nothing: %q", args)
	}
	if !LeavesContentProgramsOn([]string{"-c", "merge.default=text", "merge-tree", "--write-tree", "main", "side"}) {
		t.Error("a merge-tree whose renormalize is not pinned off can start a content filter, so it must ask")
	}
}

// TestTheTreesOwnFormattedReadsDoNotAsk: every log, show, reflog,
// rev-list, shortlog and diff-tree form the tree runs through an isolated
// runner, as written at its call site, is judged by the format git reads from
// it (formatAsks: the last format word, the options before it stepped over by
// the subcommand's table), so a call that fixes its format does not ask.
// dates.go's history asks for its merge diff (patchy), not its format.
func TestTheTreesOwnFormattedReadsDoNotAsk(t *testing.T) {
	const sha, ref, rng = "0123456789abcdef0123456789abcdef01234567", "refs/heads/x", "a..b"
	for _, tc := range []struct {
		at   string
		args []string
		want bool
	}{
		{"ahoy/worktree/quiet.go:150", []string{"log", "-1", "--format=%ct", "HEAD", "--"}, false},
		{"ahoy/worktree/quiet.go:281", []string{"log", "--format=%s", rng, "--"}, false},
		{"ahoy/worktree/merged.go:209", []string{"log", "-g", "--format=%H%x00%gs", ref, "--"}, false},
		{"ahoy/worktree/merged.go:295", []string{"log", "-p", "--no-ext-diff", "--no-textconv", "--no-color", "--no-decorate", "--format=commit %H", rng, "--"}, false},
		{"capture/reframe.go:419", []string{"log", "--format=%H", "-n", "64", "--first-parent", "HEAD", "--", "commands"}, false},
		{"capture/reframe.go:752", []string{"log", "--diff-filter=A", "--format=%H", "-n", "1", "HEAD", "--", "a.md"}, false},
		{"capture/mentions.go:339", []string{"log", ref, "--root", "--no-merges", "--name-status", "--format=%x1e%H%x1f%cI%x1f%B%x1f", "--"}, false},
		{"landing/prbody.go:62", []string{"log", "--reverse", "--no-merges", "--max-count=51", "--format=%x1e%s%x1f%B", rng, "--"}, false},
		{"landing/announce.go:36", []string{"log", "--reverse", "--format=%(trailers:valueonly,key=Refs)", rng, "--"}, false},
		{"embargo/check.go:469", []string{"log", "-1", "--no-show-signature", "--format=%B", sha, "--"}, false},
		{"embargo/check.go:473", []string{"log", "-1", "--no-show-signature", "--format=%P", sha, "--"}, false},
		{"embargo/check.go:487", []string{"log", "-1", "--no-show-signature", "--format=", "-p", "--root", sha, "--no-ext-diff", "--no-textconv",
			"--text", "--no-color", "--no-renames", "--no-relative", "--unified=0", "--src-prefix=a/", "--dst-prefix=b/", "--"}, false},
		{"landings/landings.go:452", []string{"log", "-z", "--no-show-signature", "--format=%H%x1f%P%x1f%ct%x1f%cn%x1f%B", rng, "--"}, false},
		{"landings/landings.go:613", []string{"log", "--first-parent", "--no-show-signature", "--format=%H", rng, "--", "a.md"}, false},
		{"landings/revert.go:952", []string{"log", "-z", "--first-parent", "--no-show-signature", "--max-count=8", "--format=%H%x1f%B", sha, "--"}, false},
		{"lifeboat/plan.go:586", []string{"log", "--format=%an"}, false},
		{"lifeboat/graveyard_archaeology.go:54", []string{"log", "--format=%H%x00%s"}, false},
		{"lifeboat/graveyard_archaeology.go:152", []string{"log", "-1", "--format=%ct", sha}, false},
		{"lifeboat/graveyard_archaeology.go:229", []string{"log", "--name-only", "--format="}, false},
		{"lifeboat/graveyard_archaeology.go:245", []string{"log", "--format=%h", "--", "m"}, false},
		{"lifeboat/graveyard_archaeology.go:254", []string{"log", "--format=%H", "--reverse", "--", "m"}, false},
		{"lifeboat/graveyard_archaeology.go:258", []string{"show", sha + ":m"}, false},
		{"lifeboat/graveyard_archaeology.go:390", []string{"log", "--no-merges", "--format=%H%x00%P%x00%s", "--shortstat"}, false},
		{"lifeboat/sources_git.go:37", []string{"log", "--format=%s"}, false},
		{"lifeboat/sources_git.go:49", []string{"log", "--diff-filter=D", "--name-only", "--format="}, false},
		{"lifeboat/sources_git.go:157", []string{"log", "--format=%h", "--", "m"}, false},
		{"autonomy/owedread.go:662", []string{"reflog", "show", "-n", "5000", "--format=%gs", "HEAD", "--"}, false},
		{"autonomy/owedread.go:716", []string{"log", "-n", "200", "--format=%ae%x09%at%x09%ct", sha, "--not", sha, "--"}, false},
		{"autonomy/owedread.go:868", []string{"reflog", "show", "-n", "1", "--format=%ct", "main", "--"}, false},
		{"autonomy/owedread.go:913", []string{"reflog", "show", "-n", "1000", "--format=%H%x09%gs", ref, "--"}, false},
		{"implement/loop/pickcommit.go:355", []string{"log", "-1", "--format=%s", sha, "--"}, false},
		{"implement/loop/pickcommit.go:359", []string{"diff-tree", "--no-commit-id", "--name-only", "-z", "-r", "--no-renames", sha, "--"}, false},
		{"implement/loop/validate.go:376", []string{"diff-tree", "-r", "--raw", "--no-renames", "--no-abbrev", "--ignore-submodules=none", "-z", sha, sha, "--"}, false},
		{"implement/loop/land.go:432", []string{"log", "-1", "--format=%P", sha, "--"}, false},
		{"implement/loop/land.go:433", []string{"log", "-1", "--format=%s", sha, "--"}, false},
		{"implement/loop/land.go:439", []string{"diff-tree", "-r", "--raw", "--no-renames", "--no-abbrev", "--ignore-submodules=none", "-z", sha, sha, "--"}, false},
		{"implement/loop/land.go:1630", []string{"log", "-1", "--format=%B", sha, "--"}, false},
		{"landing/delta.go:551", []string{"diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none", "--raw", "-z", sha, sha}, false},
		{"landing/delta.go:624", []string{"diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none", "--numstat", "-z", sha, sha}, false},
		{"landing/delta.go:636", []string{"diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none", "-p", "-U0", sha, sha}, false},
		{"landing/recordsshape.go:29", []string{"diff-tree", "-r", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-abbrev", "--ignore-submodules=none", "--raw", "-z", sha, sha}, false},
		{"site/contributors.go:215", []string{"log", "--pretty=format:%x00%p%x1e%(trailers:key=Assisted-by,valueonly,separator=%x1f)"}, false},
		{"site/contributors.go:295", []string{"shortlog", "-sne", "HEAD"}, false},
		{"site/dates.go:80", []string{"log", "--reverse", "--name-status", "-M", "--diff-merges=first-parent", "--date=short", "--pretty=format:%x00%ad"}, true},
		{"claim/write.go:197", []string{"show", "-s", "--format=%ct", ref}, false},
	} {
		if got := LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%s: LeavesContentProgramsOn(%q) = %v, want %v", tc.at, tc.args, got, tc.want)
		}
	}
}

// TestAFormatThatCanShowASignatureCheckAsks: format.pretty is the default
// format of log and show, and pretty.<name> an alias a --pretty=<name> or
// --format=<name> takes (git-config(1)); either can name a format holding
// %G, whose signature check starts gpg.program. git reads the last format word
// on the command line (--format=, --pretty=, --pretty, --oneline), so a
// command whose last one names an alias, holds a signature placeholder, or is
// absent where the default is format.pretty asks, and one whose last is a %
// format, an empty one or a built-in name does not, for log, show, reflog,
// rev-list, shortlog and diff-tree alike. A deprecated command asks whatever it
// carries.
func TestAFormatThatCanShowASignatureCheckAsks(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"log", "-1", "HEAD"}, true},
		{[]string{"show", "-s", "HEAD"}, true},
		{[]string{"show", "--no-textconv", "--no-ext-diff", "HEAD"}, true},
		{[]string{"log", "--format=changelog", "-1"}, true},
		{[]string{"log", "--pretty=changelog", "-1"}, true},
		{[]string{"log", "--format=%H", "--pretty=changelog", "-1"}, true},
		{[]string{"log", "--oneline", "--format=changelog"}, true},
		{[]string{"log", "--format=%H", "--format=%G?"}, true},
		{[]string{"log", "--format=%H", "--pretty=tformat:%GS"}, true},
		{[]string{"log", "--frobnicate", "--format=%H"}, true}, // the word before is not known to take no value
		{[]string{"log", "--author", "--format=%H"}, true},     // the format word is --author's value
		{[]string{"log", "--pretty=changelog", "--format=%H", "-1"}, false},
		{[]string{"log", "--format=changelog", "--oneline"}, false},
		{[]string{"log", "--format=changelog", "--pretty"}, false},
		{[]string{"log", "--format=%H", "-1"}, false},
		{[]string{"log", "--format=", "-1"}, false},
		{[]string{"log", "--pretty=format:%x00%ad"}, false},
		{[]string{"log", "--pretty=tformat:x"}, false},
		{[]string{"log", "--pretty=fuller"}, false},
		{[]string{"log", "-p", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"log", "-p", "--no-textconv", "--no-ext-diff", "--format=commit %H"}, false},
		{[]string{"show", "-s", "--format=%ct", "HEAD"}, false},
		{[]string{"show", "-s", "--format=%ct", "--pretty=changelog", "HEAD"}, true},
		{[]string{"show", "HEAD:a.txt"}, false},
		{[]string{"reflog", "show", "-n", "1", "main", "--"}, false},
		{[]string{"reflog", "show", "-n", "1", "--format=changelog", "main", "--"}, true},
		{[]string{"reflog", "show", "-n", "1", "--format=%ct", "main", "--"}, false},
		{[]string{"rev-list", "--max-count=1", "HEAD"}, false},
		{[]string{"rev-list", "--format=changelog", "HEAD"}, true},
		{[]string{"rev-list", "--pretty=changelog", "HEAD"}, true},
		{[]string{"rev-list", "--format=%H %s", "HEAD"}, false},
		{[]string{"shortlog", "-sne", "HEAD"}, false},
		{[]string{"shortlog", "--format=changelog", "HEAD"}, true},
		{[]string{"diff-tree", "-r", "--no-commit-id", "HEAD"}, false},
		{[]string{"diff-tree", "--pretty=changelog", "HEAD"}, true},
		{[]string{"diff-tree", "--format=%G?", "HEAD"}, true},
		{[]string{"diff-tree", "--show-signature", "HEAD"}, true},
		{[]string{"whatchanged", "--format=%H", "-1"}, true},
		{[]string{"pack-redundant", "--all"}, true},
	} {
		if got := LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("LeavesContentProgramsOn(%q) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

// deprecatedCommands are the commands git 2.52 documents as deprecated
// (git-whatchanged(1), git-pack-redundant(1)). git-config(1) says of alias.*:
// "To avoid confusion and troubles with script usage, aliases that hide
// existing Git commands are ignored except for deprecated commands." A
// command in git's exec path that is not built in needs no entry: git tries
// it before any alias, so an alias does not stand for it either.
var deprecatedCommands = []string{"whatchanged", "pack-redundant"}

// TestNoAliasStandsForASubcommandAbcdJudges: an alias stands for a git
// command only when the command is deprecated (deprecatedCommands), so no
// subcommand the table of what an isolated command can start judges, or the
// networked runners run, is one: the hold leaves alias.* out for that reason
// (driveUnreadSettings in internal/core/implement/loop). A deprecated
// subcommand is on neither list and asks as one the table does not know.
func TestNoAliasStandsForASubcommandAbcdJudges(t *testing.T) {
	var verbs []string
	for v := range startsNothing {
		verbs = append(verbs, v)
	}
	for v := range contentJudges {
		verbs = append(verbs, v)
	}
	for v := range valuedNext {
		verbs = append(verbs, v)
	}
	// The networked runners' subcommands, which neither list holds.
	verbs = append(verbs, "fetch", "push", "ls-remote")
	slices.Sort(verbs)
	for _, v := range slices.Compact(verbs) {
		if slices.Contains(deprecatedCommands, v) {
			t.Errorf("%q is a deprecated git command, for which git lets an alias stand, so the table must not judge it", v)
		}
	}
}

// TestOnlyAFetchCarriesTheRecursionPins: a fetch a networked runner builds
// carries submodule recursion off before its subcommand; a push, an ls-remote
// and a read carry nothing, so a push reads push.recurseSubmodules as the
// person set it.
func TestOnlyAFetchCarriesTheRecursionPins(t *testing.T) {
	pins := []string{"-c", "fetch.recurseSubmodules=false", "-c", "submodule.recurse=false"}
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"fetch", "--no-tags", "origin"}, pins},
		{[]string{"-c", "core.hooksPath=/dev/null", "fetch", "--quiet", "origin"}, pins},
		{[]string{"push", "--porcelain", "origin", "HEAD:refs/heads/x"}, nil},
		{[]string{"ls-remote", "--", "origin", "HEAD"}, nil},
		{[]string{"config", "--null", "--list"}, nil},
	} {
		if got := fetchRecursionOff(tc.args); !slices.Equal(got, tc.want) {
			t.Errorf("fetchRecursionOff(%q) = %q, want %q", tc.args, got, tc.want)
		}
		cmd := hookedCommand(t.Context(), t.TempDir(), true, tc.args)
		if has := slices.Contains(cmd.Args, "submodule.recurse=false"); has != (tc.want != nil) {
			t.Errorf("the networked command for %q carries the recursion pins: %v, want %v (%q)", tc.args, has, tc.want != nil, cmd.Args)
		}
	}
}

package gitutil_test

// filter_ask_test.go holds ruling 1 of fix round 5 of iss-2611131308266006:
// an isolated git that can start a program the repository names which
// ExecPins leaves on (a content filter, a textconv or external diff program,
// a merge driver) asks the check SetBeforeHookedGit set, as the hooked
// runners do; one that blanks them, or reads no content through them, does
// not, and with no check set nothing changes.

import (
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// countingCheck sets, for the rest of the test, a check that records each
// command it is asked about and refuses with refused (nil lets git run).
func countingCheck(t *testing.T, refused error) *[][]string {
	t.Helper()
	var asked [][]string
	unset, err := gitutil.SetBeforeHookedGit(func(args []string) error {
		asked = append(asked, append([]string(nil), args...))
		return refused
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unset)
	return &asked
}

// TestAnIsolatedReadThatCanRunAFilterAsksTheCheck: in a repository whose
// settings name a clean filter for its file, an isolated hash-object of that
// file and a status with the filters switched back on (filter-roots) each
// ask the check before git runs; under a refusing check each fails with a
// NotRunError naming its subcommand, and the filter does not run. A
// hash-object with --no-filters and a status with the filters off ask
// nothing. With no check set the hash-object runs the filter, as it always
// has. Watched failing before isolatedGit asked: both reads ran the filter
// under the refusing check.
func TestAnIsolatedReadThatCanRunAFilterAsksTheCheck(t *testing.T) {
	refused := errors.New("held setting changed: filter.evil.clean")
	t.Run("hash-object", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		asked := countingCheck(t, refused)
		_, err := gitutil.Run(r.Root(), "hash-object", "--", "a.txt")
		var nr *gitutil.NotRunError
		if !errors.As(err, &nr) || nr.Verb != "hash-object" || !errors.Is(err, refused) || len(*asked) != 1 {
			t.Fatalf("the read asks the check and is not run: %v, asked %q", err, *asked)
		}
		if ran(mark) {
			t.Fatal("the clean filter ran under a refusing check")
		}
		if _, err := gitutil.Run(r.Root(), "hash-object", "--no-filters", "--", "a.txt"); err != nil || len(*asked) != 1 {
			t.Fatalf("a read with the filters off asks nothing: %v, asked %q", err, *asked)
		}
	})
	t.Run("status with the filters on", func(t *testing.T) {
		r, home, mark := filterFixture(t)
		declare(t, home, r.Root()+"\n", 0o600)
		asked := countingCheck(t, refused)
		_, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
		var nr *gitutil.NotRunError
		if !errors.As(err, &nr) || nr.Verb != "status" || len(*asked) != 1 {
			t.Fatalf("the status asks the check and is not run: %v, asked %q", err, *asked)
		}
		if ran(mark) {
			t.Fatal("the clean filter ran under a refusing check")
		}
	})
	t.Run("status with the filters off", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		asked := countingCheck(t, refused)
		if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil || len(*asked) != 0 {
			t.Fatalf("a status that starts no filter asks nothing: %v, asked %q", err, *asked)
		}
		if ran(mark) {
			t.Fatal("the filter is off")
		}
	})
	t.Run("no check set", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		if _, err := gitutil.Run(r.Root(), "hash-object", "--", "a.txt"); err != nil {
			t.Fatal(err)
		}
		if !ran(mark) {
			t.Fatal("with no check set the read runs as it always has, filter and all")
		}
	})
}

// TestWhatAnIsolatedCommandCanStartIsReadFromItsCommandLine: the commands a
// drive's isolated reads run, and their neighbours, are judged by what they
// can start.
func TestWhatAnIsolatedCommandCanStartIsReadFromItsCommandLine(t *testing.T) {
	blank := []string{"-c", "filter.lfs.clean=", "-c", "filter.lfs.smudge=", "-c", "filter.lfs.process="}
	drivers := []string{"-c", "merge.default=git merge-file %A %O %B"}
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"hash-object", "--", "a"}, true},
		{[]string{"hash-object", "--no-filters", "--", "a"}, false},
		{[]string{"hash-object", "-w", "-t", "tree", "--stdin"}, false},
		{[]string{"hash-object", "-w", "--stdin"}, false},
		{[]string{"hash-object", "--stdin", "--path", "a"}, true},
		{append(blank, "hash-object", "--", "a"), false},
		{[]string{"--no-optional-locks", "status", "--porcelain=v1"}, true},
		{append(blank, "--no-optional-locks", "status", "--porcelain=v1"), false},
		{[]string{"diff", "--quiet"}, true},
		{[]string{"diff", "--cached", "--raw", "--no-ext-diff", "HEAD", "--"}, false},
		{[]string{"diff", "--cached", "--binary", "--full-index", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-color", "HEAD", "--"}, false},
		{[]string{"diff", "--cached", "--binary", "HEAD", "--"}, true},
		{[]string{"diff", "--name-only", "a", "b", "--"}, false},
		{[]string{"diff", "--text", "--ignore-submodules=none", "--no-ext-diff", "--no-textconv", "--no-color", "a..b", "--"}, false},
		{[]string{"diff", "a..b"}, true},
		{[]string{"diff-tree", "-r", "--raw", "a", "b"}, false},
		{[]string{"diff-tree", "-p", "--textconv", "a", "b"}, true},
		{[]string{"diff-files", "-p"}, true},
		{[]string{"diff-index", "--cached", "-p", "HEAD"}, false},
		{[]string{"show", "-s", "--format=%H", "HEAD"}, false},
		{[]string{"show", "HEAD:a.txt"}, false},
		{[]string{"show", "--textconv", "HEAD:a.txt"}, true},
		{[]string{"show", "HEAD"}, true},
		{[]string{"show", "--no-textconv", "--no-ext-diff", "HEAD"}, false},
		{[]string{"log", "--format=%H", "-n", "1"}, false},
		{[]string{"log", "-p", "--no-ext-diff", "--no-textconv"}, false},
		{[]string{"log", "-p"}, true},
		{[]string{"log", "-Gneedle"}, true},
		{[]string{"cat-file", "--batch"}, false},
		{[]string{"cat-file", "--filters", "HEAD:a"}, true},
		{[]string{"cat-file", "--textconv", "HEAD:a"}, true},
		{[]string{"grep", "-n", "x"}, false},
		{[]string{"blame", "--no-textconv", "HEAD", "--", "a"}, false},
		{[]string{"blame", "--", "a"}, true},
		{[]string{"archive", "HEAD"}, true},
		{[]string{"apply", "--cached"}, false},
		{[]string{"apply"}, true},
		{[]string{"merge-tree", "--write-tree", "a", "b"}, true},
		{append(drivers, "merge-tree", "--write-tree", "a", "b"), false},
		{[]string{"worktree", "add", "x"}, true},
		{[]string{"worktree", "list", "--porcelain"}, false},
		{[]string{"reset", "--soft", "HEAD"}, false},
		{[]string{"reset", "-q", "--", "a"}, true},
		{[]string{"ls-files", "-z"}, false},
		{[]string{"ls-files", "-m"}, true},
		{[]string{"rev-parse", "HEAD"}, false},
		{[]string{"-c", "core.splitIndex=false", "config", "--get", "x"}, false},
		{[]string{"var", "GIT_ATTR_GLOBAL"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
	if gitutil.Subcommand([]string{"-c", "a=b", "--no-optional-locks", "-C", "x", "status"}) != "status" {
		t.Error("the subcommand is read past git's global options")
	}
}

// TestASubcommandTheTableDoesNotKnowAsks: a subcommand neither judged nor
// on the starts-nothing list asks, whatever it carries; one on the list
// does not; none is on both. Watched failing before fix round 6: every row
// below that wants true got false, since an unknown subcommand asked nothing.
func TestASubcommandTheTableDoesNotKnowAsks(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"merge-file", "a", "b", "c"}, true},
		{[]string{"verify-commit", "HEAD"}, true},
		{[]string{"interpret-trailers"}, true},
		{[]string{"notes", "merge", "x"}, true},
		{[]string{"sparse-checkout", "reapply"}, true},
		{[]string{"--no-optional-locks"}, true},
		{[]string{"describe", "--tags", "--match", "v*", "--abbrev=0", "HEAD"}, false},
		{[]string{"tag", "--list", "v*"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
	for _, v := range []string{"rev-parse", "config", "var", "tag", "describe"} {
		if !gitutil.StartsNothing(v) || gitutil.JudgesContentPrograms(v) {
			t.Errorf("%s is on the starts-nothing list alone", v)
		}
	}
	for _, v := range []string{"cat-file", "ls-files", "hash-object", "read-tree", "worktree", "status"} {
		if gitutil.StartsNothing(v) {
			t.Errorf("%s is judged, so it is not on the starts-nothing list", v)
		}
	}
}

// TestASubcommandOnTheStartsNothingListAsksInAFormThatCanStartAProgram: a
// subcommand on the starts-nothing list is judged by the form it runs in, and
// asks in one that can start a program (a signing program, an editor, a hook
// a template copies in) or reach the network; in the form abcd runs it asks
// nothing. ls-remote is on no list, so through an isolated runner it asks.
// A diff of a revision and a path with no "--" between them asks, since git
// may read the path as a second revision or as a path. Watched failing
// before: every row wanting true got false.
func TestASubcommandOnTheStartsNothingListAsksInAFormThatCanStartAProgram(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"tag", "-v", "v1"}, true},
		{[]string{"tag", "--verify", "v1"}, true},
		{[]string{"tag", "-s", "-m", "x", "v1"}, true},
		{[]string{"tag", "-u", "key", "v1"}, true},
		{[]string{"tag", "--local-user=key", "v1"}, true},
		{[]string{"tag", "v1"}, true},
		{[]string{"tag", "-a", "v1"}, true},
		{[]string{"tag", "--list", "v*", "--sort=-v:refname"}, false},
		{[]string{"tag", "-l", "v*"}, false},
		{[]string{"tag", "--contains", "abc", "--list", "v*"}, false},
		{[]string{"tag", "--list", "--merged", "abc", "v*"}, false},
		{[]string{"tag"}, false},
		{[]string{"tag", "--list", "-v", "v*"}, true},
		{[]string{"describe", "--dirty"}, true},
		{[]string{"describe", "--dirty=-x", "HEAD"}, true},
		{[]string{"describe", "--broken"}, true},
		{[]string{"describe", "--tags", "--match", "v*", "--abbrev=0", "HEAD"}, false},
		{[]string{"branch", "--edit-description", "x"}, true},
		{[]string{"branch", "--edit-desc"}, true},
		{[]string{"branch", "--format=%(refname:short)", "--no-merged", "abc"}, false},
		{[]string{"config", "--edit"}, true},
		{[]string{"config", "-e"}, true},
		{[]string{"config", "--local", "--edit"}, true},
		{[]string{"config", "--get", "x"}, false},
		{[]string{"config", "--null", "--list", "--show-scope", "--show-origin"}, false},
		{[]string{"config", "-z", "--type=path", "--get-all", "core.hooksPath"}, false},
		{[]string{"commit-tree", "-S", "t", "-m", "x"}, true},
		{[]string{"commit-tree", "-Skey", "t", "-m", "x"}, true},
		{[]string{"commit-tree", "--gpg-sign", "t", "-m", "x"}, true},
		{[]string{"commit-tree", "--gpg-sign=key", "t", "-m", "x"}, true},
		{[]string{"commit-tree", "t", "-p", "p", "-F", "-"}, false},
		{[]string{"commit-tree", "--no-gpg-sign", "t", "-m", "x"}, false},
		{[]string{"init", "-q", "--template=/some/dir"}, true},
		{[]string{"init", "--template", "/some/dir", "x"}, true},
		{[]string{"init", "-q", "--templ=/some/dir"}, true},
		{[]string{"init", "-q", "--template="}, false},
		{[]string{"init", "-q", "--template=", "--object-format=sha1"}, false},
		{[]string{"remote", "update"}, true},
		{[]string{"remote", "show", "origin"}, true},
		{[]string{"remote", "prune", "origin"}, true},
		{[]string{"remote", "add", "-f", "x", "u"}, true},
		{[]string{"remote", "set-head", "-a", "origin"}, true},
		{[]string{"remote"}, false},
		{[]string{"remote", "-v"}, false},
		{[]string{"remote", "get-url", "--all", "origin"}, false},
		{[]string{"remote", "remove", "origin"}, false},
		{[]string{"remote", "add", "x", "u"}, false},
		{[]string{"ls-remote", "origin"}, true},
		{[]string{"diff", "--name-only", "HEAD", "a.txt"}, true},
		{[]string{"diff", "--name-only", "HEAD", "a.txt", "--"}, false},
		{[]string{"diff", "--name-only", "HEAD~1..HEAD"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
	if gitutil.StartsNothing("ls-remote") {
		t.Error("ls-remote reaches the network, so it is on no list")
	}
}

// TestAListingIsRecognisedOnlyByExactWords: a tag does not ask only for a
// listing given in exact words with no word that makes, signs, verifies or
// deletes a tag, bundled or glued to a value included; config's edit
// subcommand asks as --edit does; a format that shows a signature check, or
// --show-signature, asks on every subcommand that takes one; and init asks
// unless the empty template is given. Watched failing on 5dd7deff: every row
// wanting true got false.
func TestAListingIsRecognisedOnlyByExactWords(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"tag", "-m", "-l", "v1"}, true},
		{[]string{"tag", "-mfinal", "v1"}, true},
		{[]string{"tag", "-f", "-n", "v1"}, true},
		{[]string{"tag", "-d", "-l", "v1"}, true},
		{[]string{"tag", "-e", "--list", "v1"}, true},
		{[]string{"tag", "-Fmsg", "--contains", "abc", "v1"}, true},
		{[]string{"tag", "--message=x", "--list", "v1"}, true},
		{[]string{"tag", "--anno", "--list", "v1"}, true},
		{[]string{"tag", "--trailer=k:v", "--list", "v1"}, true},
		{[]string{"tag", "--lis", "v*"}, true},
		{[]string{"tag", "-ln", "v*"}, true},
		{[]string{"tag", "--list", "--format=%(signature)", "v*"}, true},
		{[]string{"tag", "--list", "--format", "%(signature:grade)", "v*"}, true},
		{[]string{"tag", "-n5", "-l", "v*"}, false},
		{[]string{"tag", "-n", "v*"}, false},
		{[]string{"tag", "--points-at=HEAD"}, false},
		{[]string{"tag", "--list", "--format=%(refname:short)", "v*"}, false},
		{[]string{"config", "edit"}, true},
		{[]string{"config", "--local", "edit"}, true},
		{[]string{"config", "--get", "core.editor"}, false},
		{[]string{"log", "--format=%G?", "-1"}, true},
		{[]string{"log", "--pretty", "format:%GS", "HEAD"}, true},
		{[]string{"log", "--pre=%GK", "HEAD"}, true},
		{[]string{"log", "--show-signature", "-1"}, true},
		{[]string{"log", "--format=%H", "-1"}, false},
		{[]string{"show", "-s", "--format=%G?", "HEAD"}, true},
		{[]string{"show", "--show-sig", "-s", "HEAD"}, true},
		{[]string{"show", "-s", "--format=%H", "HEAD"}, false},
		{[]string{"rev-list", "--format=%G?", "HEAD"}, true},
		{[]string{"rev-list", "--form", "%GG", "HEAD"}, true},
		{[]string{"rev-list", "--format=%H", "HEAD"}, false},
		{[]string{"shortlog", "--format", "%GS", "HEAD"}, true},
		{[]string{"shortlog", "-sne", "HEAD"}, false},
		{[]string{"for-each-ref", "--format=%(signature:grade)"}, true},
		{[]string{"for-each-ref", "--format=%(refname)"}, false},
		{[]string{"branch", "--format=%(signature)", "--list"}, true},
		{[]string{"branch", "--format=%(refname:short)", "--no-merged", "abc"}, false},
		{[]string{"init"}, true},
		{[]string{"init", "-q", "x"}, true},
		{[]string{"init", "-q", "--template=", "x"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
}

// TestAWordThatIsAnOptionsValueNeverMakesACommandNotAsk: a word that follows
// an option git takes its value from as the next word is that value, so it
// is never read as a word that makes the command not ask: a tag's listing
// word, init's empty template, --cached, --soft, --no-filters, --stat, the
// two words that switch the diff programs off, or a revision that keeps the
// working tree or a commit's patch out. Watched failing on 73c7e73a: every
// row wanting true got false.
func TestAWordThatIsAnOptionsValueNeverMakesACommandNotAsk(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"tag", "--format", "-l", "v1"}, true},
		{[]string{"tag", "--format", "--list", "v1"}, true},
		{[]string{"tag", "--format", "-n", "v1"}, true},
		{[]string{"tag", "--form", "-l", "v1"}, true},
		{[]string{"tag", "--sort", "-l", "v1"}, true},
		{[]string{"tag", "--cleanup", "--list", "v1"}, true},
		{[]string{"init", "--separate-git-dir", "--template=", "x"}, true},
		{[]string{"init", "-b", "--template=", "x"}, true},
		{[]string{"init", "--initial-branch", "--template=", "x"}, true},
		{[]string{"init", "--object-format", "--template=", "x"}, true},
		{[]string{"init", "--ref-format", "--template=", "x"}, true},
		{[]string{"diff-index", "--output", "--cached", "-p", "HEAD"}, true},
		{[]string{"diff", "--name-only", "--output", "a", "b", "--"}, true},
		{[]string{"show", "-O", "HEAD:a.txt"}, true},
		{[]string{"log", "-p", "--output", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"blame", "--no-textconv", "-L", "1,2", "--", "a"}, true},
		{[]string{"hash-object", "--path", "--no-filters", "a"}, true},
		{[]string{"hash-object", "--path", "-t", "tree", "a"}, true},
		{[]string{"rm", "--pathspec-from-file", "--cached"}, true},
		{[]string{"reset", "--pathspec-from-file", "--soft", "HEAD"}, true},
		{[]string{"apply", "-p", "--cached"}, true},
		{[]string{"apply", "--exclude", "--stat", "x.patch"}, true},
		{[]string{"init", "--object-format", "sha1", "-q", "--template=", "x"}, false},
		{[]string{"tag", "--sort", "v:refname", "--list", "v*"}, false},
		{[]string{"blame", "--no-textconv", "-L", "1,2", "HEAD", "--", "a"}, false},
		{[]string{"diff", "--name-only", "--output", "o", "a", "b", "--"}, false},
		{[]string{"rm", "--pathspec-from-file", "f", "--cached"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
}

// TestAWordAfterAnOptionNotKnownToTakeNoValueNeverMakesACommandNotAsk: a
// word that makes a command not ask counts only where the word before it is
// known not to take it as a value: an option on no list (--default,
// --encoding, --glob, --exclude-hidden, a made-up --zzz) may take the next
// word, so the command asks. update-index --cacheinfo with a plain path
// reads that path, and init with --no-template copies git's own template, so
// each asks. The rows that do not ask hold each place a word counts: first
// after the subcommand, after a word that is not an option, after an option
// holding "=", after another counted word that makes it not ask, and after
// an option on the list of those that take no value. Watched failing on
// 6f65dbb7: every row wanting true got false. The two diff rows with
// --binary and --text wanted false there; that test now holds the forms
// abcd runs.
func TestAWordAfterAnOptionNotKnownToTakeNoValueNeverMakesACommandNotAsk(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"log", "-p", "--default", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"log", "-p", "--encoding", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"log", "-p", "--glob", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"log", "-p", "--exclude-hidden", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"log", "-p", "--zzz", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"diff", "--name-only", "--zzz", "a..b", "--"}, true},
		{[]string{"rm", "--zzz", "--cached", "a"}, true},
		{[]string{"diff", "--cached", "--binary", "--no-ext-diff", "--no-textconv", "HEAD", "--"}, true},
		{[]string{"diff", "--text", "--no-ext-diff", "--no-textconv", "a..b", "--"}, true},
		{[]string{"update-index", "--cacheinfo", "100644,e69de29bb2d1d6434b8b29ae775ad8c2e48c5391,a.txt", "b.txt"}, true},
		{[]string{"init", "-q", "--template=", "--no-template"}, true},
		{[]string{"log", "--no-textconv", "--no-ext-diff", "-p"}, false},
		{[]string{"diff", "--name-only", "a", "b", "--"}, false},
		{[]string{"diff", "-p", "--ignore-submodules=none", "--no-ext-diff", "--no-textconv", "a..b", "--"}, false},
		{[]string{"diff", "--no-renames", "--ignore-submodules=none", "--name-only", "-z", "a", "b", "--"}, false},
		{[]string{"rm", "-r", "-q", "--cached", "--ignore-unmatch", "--", "a"}, false},
		{[]string{"update-index", "--cacheinfo", "100644,e69de29bb2d1d6434b8b29ae775ad8c2e48c5391,a.txt"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
}

// TestTheOptionsThatStartAProgramAreReadAsGitTakesThem: the options that
// switch a program on are read in any prefix git takes for them and inside
// a bundle of short options, --attr-source takes its value as the next
// argument, an apply with --3way asks, a merge's diff and --binary are a
// patch, and :/text and rev^{/text} name commits, not blobs. Watched failing
// before fix round 6: each row wanting true got false (the --attr-source row
// read "rev-parse" as the subcommand).
func TestTheOptionsThatStartAProgramAreReadAsGitTakesThem(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"cat-file", "--filt", "HEAD:a"}, true},
		{[]string{"cat-file", "--textc", "HEAD:a"}, true},
		{[]string{"grep", "--textc", "-n", "x"}, true},
		{[]string{"grep", "--text", "-n", "x"}, false},
		{[]string{"diff-tree", "-p", "--ext-d", "a", "b"}, true},
		{[]string{"ls-files", "--mod"}, true},
		{[]string{"ls-files", "-tm"}, true},
		{[]string{"ls-files", "-z", "--stage", "-t", "-v"}, false},
		{[]string{"read-tree", "-mu", "HEAD"}, true},
		{[]string{"--attr-source", "rev-parse", "status"}, true},
		{[]string{"apply", "--cached", "--3way"}, true},
		{[]string{"apply", "--cached", "-3"}, true},
		{[]string{"apply", "--stat", "--3w"}, true},
		{[]string{"log", "-c"}, true},
		{[]string{"log", "--remerge-diff"}, true},
		{[]string{"log", "--binary"}, true},
		{[]string{"log", "--diff-merges=first-parent"}, true},
		{[]string{"show", "-s", "-c", "HEAD"}, true},
		{[]string{"show", "-s", "--remerge-diff", "HEAD"}, true},
		{[]string{"reflog", "show", "-p", "HEAD"}, true},
		{[]string{"reflog", "show", "-n", "1", "--format=%gs", "HEAD", "--"}, false},
		{[]string{"show", ":/fix"}, true},
		{[]string{"show", "HEAD^{/a:b}"}, true},
		{[]string{"show", "HEAD:a.txt"}, false},
	} {
		if got := gitutil.LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.args, got, tc.want)
		}
	}
	if v := gitutil.Subcommand([]string{"--attr-source", "HEAD", "status"}); v != "status" {
		t.Errorf("--attr-source takes the next argument as its value: read %q as the subcommand", v)
	}
}

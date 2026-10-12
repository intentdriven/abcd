package gitutil

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ExecPins is the one list of `git -c` overrides that keep a repository's own
// config from making a git command abcd composes start a program the command
// would not otherwise run. A repository's .git/config is fully trusted by git
// and no environment variable switches it off, so each knob is forced on the
// command line, where it outranks every config file:
//
//   - core.hooksPath=/dev/null: no hook fires.
//   - core.fsmonitor=false: no fsmonitor daemon is spawned to refresh the index.
//   - log.showSignature=false: `log` and `show` do not verify a signed commit's
//     signature, which starts gpg.program (iss-2610090821531394).
//   - commit.gpgsign=false: a commit or merge commit abcd composes is not
//     signed, which would start gpg.program, gpg.ssh.program or
//     gpg.ssh.defaultKeyCommand (iss-2610090821520843).
//   - merge.verifySignatures=false: a merge does not verify the merged tip's
//     signature, which starts gpg.program or gpg.ssh.program
//     (iss-2610090821520843).
//   - diff.submodule=short: a patch diff shows a moved submodule as its
//     pointer change alone, never by starting a second git diff inside the
//     submodule, which reads the submodule's own config and is passed none of
//     the parent's --no-ext-diff/--no-textconv, so its diff.external or
//     textconv would run and write into the text a check parses
//     (iss-2610091935325886).
//
// Every isolated command (Run and its siblings) carries them, and a caller that
// must build its own git command (one that keeps global config, say) prepends
// them rather than copying the list. The pins come before -C and the
// subcommand: after the subcommand, `-c` is that subcommand's option. A commit
// abcd makes in a working tree it made for the work leaves core.hooksPath,
// commit.gpgsign and merge.verifySignatures to the repository
// (WorkingTreeCommitCommand): those are how the repository commits and merges.
//
// They do not blank content filters or diff and merge drivers, because those
// are keyed on a name the repository chooses; FilterOverrides covers filters
// for a read that must not run one, MergeDriverOverrides makes every merge
// driver git's built-in merge for a read-only merge (the conflict probe's
// merge-tree), and a diff passes --no-ext-diff and --no-textconv. A merge abcd
// makes in a lane runs the repository's merge drivers, as the person's own
// merge does.
func ExecPins() []string {
	return []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "log.showSignature=false",
		"-c", "commit.gpgsign=false",
		"-c", "merge.verifySignatures=false",
		"-c", "diff.submodule=short",
	}
}

// FilterOverrides returns the `git -c` overrides that blank every content
// filter the repository at root configures — `filter.<name>.clean`, `.smudge`
// and `.process` for each name — so a command that re-hashes the working tree
// (a `diff` against HEAD over a copied checkout whose index stat no longer
// matches) compares bytes instead of running the repository's program
// (iss-2610090821548169). The overrides go before the subcommand.
//
// Config is read the way the isolated command reads it (repository config and
// its includes; global and system neutralised). No filter configured is an
// empty list, not an error. A filter git still insists on (`required = true`)
// with its commands blanked makes the command fail, which a caller reports as
// unreadable rather than clean. A name `-c` cannot carry intact (one holding
// `=` or a line break) is refused: blanking a different key would leave the
// filter live.
func FilterOverrides(root string) ([]string, error) {
	names, err := configNames(root, "filter")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range names {
		for _, v := range []string{"clean", "smudge", "process"} {
			out = append(out, "-c", "filter."+name+"."+v+"=")
		}
	}
	return out, nil
}

// git 2.44 taught merge-file --diff-algorithm and gave a merge driver its
// conflict labels as %S, %X and %Y (RelNotes 2.44.0). An older git rejects the
// option, so a driver passing it fails every merge it is asked for, and leaves
// the placeholders unexpanded (iss-2610111549593667).
const (
	mergeDriverFloorMajor = 2
	mergeDriverFloorMinor = 44
)

// BuiltinMergeDriver returns the merge.<name>.driver command line that makes
// a driver git's built-in text merge: `git merge-file` over the three
// versions, with the conflict-marker size git passes a driver, exiting
// non-zero on a conflict, as a driver must.
//
// From git 2.44 it also passes the conflict labels and the histogram diff
// git's merge itself uses, so the result, clean or conflicted, is byte for
// byte the built-in driver's. Below 2.44 merge-file can take neither (it reads
// no diff algorithm from config, and a driver is not given the labels), so the
// driver runs merge-file's default diff with fixed labels: the repository's
// driver still never starts, but a conflict's markers read ours, base and
// theirs, and where the two diffs align a changed region differently the
// result can differ from git's own merge of the same commits, in where a
// conflict's hunks fall or in whether a region conflicts at all. A version the
// probe cannot read gets that form, which every git accepts.
func BuiltinMergeDriver() string {
	if ok, _ := gitVersion.atLeast(mergeDriverFloorMajor, mergeDriverFloorMinor); ok {
		return "git merge-file --marker-size=%L --diff-algorithm=histogram -L %X -L %S -L %Y %A %O %B"
	}
	return "git merge-file --marker-size=%L -L ours -L base -L theirs %A %O %B"
}

// MergeDriverOverrides returns the `git -c` overrides that make every merge
// driver the repository at root configures git's built-in text merge —
// `merge.<name>.driver` set to BuiltinMergeDriver for each name — and pin
// merge.default to the built-in text driver, so a read-only merge abcd
// computes (the conflict probe's merge-tree, a prediction of the forge's
// merge) never starts a program the repository names, whichever driver an
// attribute or merge.default selects (iss-2610090821510097). A merge made in a
// lane's worktree does not take them: it runs the repository's own drivers
// (WorkingTreeCommitCommand, iss-2610111729399920). A driver cannot be blanked as
// a filter is: git fails to start an empty driver command and reports every
// path both sides changed as a conflict. The overrides go before the
// subcommand.
//
// Config is read as FilterOverrides reads it, and a name `-c` cannot carry
// intact is refused the same way.
func MergeDriverOverrides(root string) ([]string, error) {
	names, err := configNames(root, "merge")
	if err != nil {
		return nil, err
	}
	out := []string{"-c", "merge.default=text"}
	driver := BuiltinMergeDriver()
	for _, name := range names {
		out = append(out, "-c", "merge."+name+".driver="+driver)
	}
	return out, nil
}

// configNames lists, once each and in config order, the subsection names of
// section that the repository at root configures (`<section>.<name>.<var>`),
// reading config the way the isolated command reads it: repository config and
// its includes, global and system neutralised. A key with no subsection
// (`merge.ff`) names nothing. A name `-c` cannot carry intact (one holding `=`
// or a line break) is refused: overriding a different key would leave the
// configured program live.
func configNames(root, section string) ([]string, error) {
	cmd := isolatedGit(root, "config", "--null", "--name-only", "--get-regexp", `^`+section+`\.`)
	e := &capWriter{remaining: 4096}
	w := &capWriter{remaining: 1 << 20}
	cmd.Stdout, cmd.Stderr = w, e
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 && len(w.buf) == 0 {
			return nil, nil // no key in the section at all
		}
		return nil, fmt.Errorf("reading the repository's %s config: %w (stderr: %q)", section, err, strings.TrimSpace(string(e.buf)))
	}
	if w.overflowed {
		return nil, fmt.Errorf("reading the repository's %s config: the key list exceeded its cap", section)
	}
	seen := map[string]bool{}
	var out []string
	for _, key := range strings.Split(string(w.buf), "\x00") {
		if key == "" {
			continue
		}
		// <section>.<name>.<variable>: the name is everything between the
		// first and the last dot, and may itself hold dots.
		first, last := strings.IndexByte(key, '.'), strings.LastIndexByte(key, '.')
		if last <= first {
			continue // <section>.<variable>: no name
		}
		name := key[first+1 : last]
		if seen[name] {
			continue
		}
		if strings.ContainsAny(name, "=\n\r") {
			return nil, fmt.Errorf("the repository configures a %s whose name %q cannot be passed to git -c, so it cannot be switched off", section, name)
		}
		seen[name] = true
		out = append(out, name)
	}
	return out, nil
}

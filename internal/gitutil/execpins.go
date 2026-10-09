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
// subcommand: after the subcommand, `-c` is that subcommand's option.
//
// They do not blank content filters or diff and merge drivers, because those
// are keyed on a name the repository chooses; FilterOverrides covers filters
// for a command that must not run one, MergeDriverOverrides makes every merge
// driver git's built-in merge, and a diff passes --no-ext-diff and
// --no-textconv.
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

// BuiltinMergeDriver is the merge.<name>.driver command line that makes a
// driver git's built-in text merge: `git merge-file` over the three versions,
// with the conflict-marker size and the three conflict labels git passes a
// driver, and the histogram diff git's merge itself uses. Its result, clean
// or conflicted, is byte for byte the built-in driver's, and it exits non-zero
// on a conflict, as a driver must.
const BuiltinMergeDriver = "git merge-file --marker-size=%L --diff-algorithm=histogram -L %X -L %S -L %Y %A %O %B"

// MergeDriverOverrides returns the `git -c` overrides that make every merge
// driver the repository at root configures git's built-in text merge —
// `merge.<name>.driver` set to BuiltinMergeDriver for each name — and pin
// merge.default to the built-in text driver, so a merge abcd composes never
// starts a program the repository names, whichever driver an attribute or
// merge.default selects (iss-2610090821510097). A driver cannot be blanked as
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
	for _, name := range names {
		out = append(out, "-c", "merge."+name+".driver="+BuiltinMergeDriver)
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

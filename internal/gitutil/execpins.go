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
//
// Every isolated command (Run and its siblings) carries them, and a caller that
// must build its own git command (one that keeps global config, say) prepends
// them rather than copying the list. The pins come before -C and the
// subcommand: after the subcommand, `-c` is that subcommand's option.
//
// They do not blank content filters or diff and merge drivers, because those
// are keyed on a name the repository chooses; FilterOverrides covers filters
// for a command that must not run one, and a diff passes --no-ext-diff and
// --no-textconv.
func ExecPins() []string {
	return []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "log.showSignature=false",
		"-c", "commit.gpgsign=false",
		"-c", "merge.verifySignatures=false",
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
	cmd := isolatedGit(root, "config", "--null", "--name-only", "--get-regexp", `^filter\.`)
	e := &capWriter{remaining: 4096}
	w := &capWriter{remaining: 1 << 20}
	cmd.Stdout, cmd.Stderr = w, e
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 && len(w.buf) == 0 {
			return nil, nil // no filter key at all
		}
		return nil, fmt.Errorf("reading the repository's filter config: %w (stderr: %q)", err, strings.TrimSpace(string(e.buf)))
	}
	if w.overflowed {
		return nil, errors.New("reading the repository's filter config: the key list exceeded its cap")
	}
	seen := map[string]bool{}
	var out []string
	for _, key := range strings.Split(string(w.buf), "\x00") {
		if key == "" {
			continue
		}
		// filter.<name>.<variable>: the name is everything between the first
		// and the last dot, and may itself hold dots.
		first, last := strings.IndexByte(key, '.'), strings.LastIndexByte(key, '.')
		if last <= first {
			continue // filter.<variable>: no name, no driver
		}
		name := key[first+1 : last]
		if seen[name] {
			continue
		}
		if strings.ContainsAny(name, "=\n\r") {
			return nil, fmt.Errorf("the repository configures a filter whose name %q cannot be passed to git -c, so it cannot be switched off", name)
		}
		seen[name] = true
		for _, v := range []string{"clean", "smudge", "process"} {
			out = append(out, "-c", "filter."+name+"."+v+"=")
		}
	}
	return out, nil
}

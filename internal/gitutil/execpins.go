package gitutil

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
//
// Every isolated command (Run and its siblings) carries them, and a caller that
// must build its own git command (one that keeps global config, say) prepends
// them rather than copying the list. The pins come before -C and the
// subcommand: after the subcommand, `-c` is that subcommand's option.
//
// They do not blank content filters or diff and merge drivers, because those
// are keyed on a name the repository chooses; a diff that must not run one
// passes --no-ext-diff and --no-textconv.
func ExecPins() []string {
	return []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.fsmonitor=false",
		"-c", "log.showSignature=false",
	}
}

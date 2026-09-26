package guard

import "strings"

// git's option parser accepts any unambiguous prefix of a long option
// (parse-options.c, parse_long_opt), so `git push --force-w` is
// `--force-with-lease` and `git commit --no-veri` is `--no-verify`. The matcher
// compares flags exactly, and every such spelling allowed a blocker
// (iss-2609251640354925).
//
// gitLongOptions is the long-option table of each git subcommand a bundled
// blocker constrains by flag, taken from `git <subcommand>
// --git-completion-helper-all` on git 2.52: the options, their `--no-`
// negations, and the negations git accepts for a `no-` option (`--verify`).
// A `=` suffix is dropped. The table is what makes a prefix decidable: a prefix
// is an abbreviation of a blocked option when no option OUTSIDE the entry's
// flag group shares it. Two consequences of reading it that way, both on the
// fail-closed side:
//
//   - A prefix shared only among blocked alternatives (`--forc`) is one git
//     refuses as ambiguous; it is read as blocked, because a git older than one
//     of those alternatives resolves it to the other.
//   - An option a newer git adds and this table lacks can only make git call
//     more prefixes ambiguous, never resolve a new one into a blocked option,
//     so a stale table over-blocks rather than misses.
//
// git's own global options (`git --no-pager`, `-C`) are parsed by hand and take
// no abbreviation, so only the subcommand's options are modelled.
var gitLongOptions = map[string][]string{
	"push": {
		"--verbose", "--quiet", "--repo", "--all", "--branches", "--mirror", "--delete", "--tags",
		"--dry-run", "--porcelain", "--force", "--force-with-lease", "--force-if-includes",
		"--recurse-submodules", "--thin", "--receive-pack", "--exec", "--set-upstream", "--progress",
		"--prune", "--no-verify", "--follow-tags", "--signed", "--atomic", "--push-option", "--ipv4",
		"--ipv6", "--verify",
		"--no-verbose", "--no-quiet", "--no-repo", "--no-all", "--no-branches", "--no-mirror",
		"--no-delete", "--no-tags", "--no-dry-run", "--no-porcelain", "--no-force",
		"--no-force-with-lease", "--no-force-if-includes", "--no-recurse-submodules", "--no-thin",
		"--no-receive-pack", "--no-exec", "--no-set-upstream", "--no-progress", "--no-prune",
		"--no-follow-tags", "--no-signed", "--no-atomic", "--no-push-option",
	},
	"commit": {
		"--quiet", "--verbose", "--file", "--author", "--date", "--message", "--reedit-message",
		"--reuse-message", "--fixup", "--squash", "--reset-author", "--trailer", "--signoff",
		"--template", "--edit", "--cleanup", "--status", "--gpg-sign", "--all", "--include",
		"--interactive", "--patch", "--unified", "--inter-hunk-context", "--only", "--no-verify",
		"--dry-run", "--short", "--branch", "--ahead-behind", "--porcelain", "--long", "--null",
		"--amend", "--no-post-rewrite", "--untracked-files", "--pathspec-from-file",
		"--pathspec-file-nul", "--allow-empty", "--allow-empty-message", "--verify", "--post-rewrite",
		"--no-quiet", "--no-verbose", "--no-file", "--no-author", "--no-date", "--no-message",
		"--no-reedit-message", "--no-reuse-message", "--no-fixup", "--no-squash", "--no-reset-author",
		"--no-signoff", "--no-template", "--no-edit", "--no-cleanup", "--no-status", "--no-gpg-sign",
		"--no-all", "--no-include", "--no-interactive", "--no-patch", "--no-only", "--no-dry-run",
		"--no-short", "--no-branch", "--no-ahead-behind", "--no-porcelain", "--no-long", "--no-null",
		"--no-amend", "--no-untracked-files", "--no-pathspec-from-file", "--no-pathspec-file-nul",
		"--no-allow-empty", "--no-allow-empty-message",
	},
}

// gitOptionTable returns the long-option table an entry's flag constraints are
// read against, or nil when the entry is not a git subcommand the table models
// — and then flags compare exactly, as before.
func gitOptionTable(p Pattern) []string {
	if !strings.EqualFold(p.Command, "git") {
		return nil
	}
	return gitLongOptions[p.Subcommand]
}

// abbreviatesAlternative reports whether arg is a long option git would read as
// an abbreviation of alt: a strict prefix of it, past the `--`, with or without
// a `=value`, that no option in opts outside the group's own alternatives
// shares. The exact spelling is flagMatches' business, not this one's.
func abbreviatesAlternative(arg, alt string, group, opts []string) bool {
	if !strings.HasPrefix(alt, "--") || !strings.HasPrefix(arg, "--") {
		return false
	}
	name, _, _ := strings.Cut(arg, "=")
	if len(name) <= 2 || len(name) >= len(alt) || !strings.HasPrefix(alt, name) {
		return false
	}
	for _, o := range opts {
		if strings.HasPrefix(o, name) && !containsString(group, o) {
			return false
		}
	}
	return true
}

package guard

import "strings"

// git's option parser accepts any unambiguous prefix of a long option
// (parse-options.c, parse_long_opt), so `git push --force-w` is
// `--force-with-lease` and `git commit --no-veri` is `--no-verify`. The matcher
// compares flags exactly, and every such spelling allowed a blocker
// (iss-2609251640354925).
//
// gitLongOptions is the long-option table of each git subcommand a blocking or
// warning entry constrains by flag, bundled or in a repository's own guard file
// (merge, rebase, am and pull serve abcd's .abcd/guard.json), taken from `git <subcommand>
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
	"branch": {
		"--verbose", "--quiet", "--track", "--set-upstream", "--set-upstream-to",
		"--unset-upstream", "--color", "--remotes", "--contains", "--with", "--without",
		"--abbrev", "--all", "--delete", "--create-reflog", "--edit-description", "--force",
		"--move", "--copy", "--column", "--sort", "--merged", "--points-at", "--format", "--list",
		"--show-current", "--ignore-case", "--omit-empty", "--recurse-submodules",
		"--no-verbose", "--no-quiet", "--no-track", "--no-set-upstream", "--no-set-upstream-to",
		"--no-unset-upstream", "--no-color", "--no-contains", "--no-abbrev", "--no-delete",
		"--no-create-reflog", "--no-edit-description", "--no-force", "--no-move", "--no-copy",
		"--no-column", "--no-sort", "--no-merged", "--no-points-at", "--no-format", "--no-list",
		"--no-show-current", "--no-ignore-case", "--no-omit-empty", "--no-recurse-submodules",
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
	"reset": {
		"--quiet", "--no-refresh", "--mixed", "--soft", "--hard", "--merge", "--keep",
		"--recurse-submodules", "--patch", "--unified", "--inter-hunk-context", "--intent-to-add",
		"--pathspec-from-file", "--pathspec-file-nul", "--refresh",
		"--no-quiet", "--no-recurse-submodules", "--no-patch", "--no-intent-to-add",
		"--no-pathspec-from-file", "--no-pathspec-file-nul",
	},
	"merge": {
		"--stat", "--summary", "--compact-summary", "--log", "--squash", "--commit", "--edit",
		"--cleanup", "--ff", "--ff-only", "--rerere-autoupdate", "--verify-signatures", "--strategy",
		"--strategy-option", "--message", "--file", "--into-name", "--verbose", "--quiet", "--abort",
		"--quit", "--continue", "--allow-unrelated-histories", "--progress", "--gpg-sign", "--autostash",
		"--overwrite-ignore", "--signoff", "--no-verify", "--verify", "--no-stat", "--no-summary",
		"--no-compact-summary", "--no-log", "--no-squash", "--no-commit", "--no-edit", "--no-cleanup",
		"--no-ff", "--no-rerere-autoupdate", "--no-verify-signatures", "--no-strategy",
		"--no-strategy-option", "--no-message", "--no-into-name", "--no-verbose", "--no-quiet",
		"--no-abort", "--no-quit", "--no-continue", "--no-allow-unrelated-histories", "--no-progress",
		"--no-gpg-sign", "--no-autostash", "--no-overwrite-ignore", "--no-signoff",
	},
	"rebase": {
		"--onto", "--keep-base", "--no-verify", "--quiet", "--verbose", "--no-stat", "--signoff",
		"--committer-date-is-author-date", "--reset-author-date", "--ignore-date", "--ignore-whitespace",
		"--whitespace", "--force-rebase", "--no-ff", "--continue", "--skip", "--abort", "--quit",
		"--edit-todo", "--show-current-patch", "--apply", "--merge", "--interactive",
		"--preserve-merges", "--rerere-autoupdate", "--empty", "--keep-empty", "--autosquash",
		"--update-refs", "--gpg-sign", "--autostash", "--exec", "--allow-empty-message",
		"--rebase-merges", "--fork-point", "--strategy", "--strategy-option", "--root",
		"--reschedule-failed-exec", "--reapply-cherry-picks", "--verify", "--stat", "--ff", "--no-onto",
		"--no-keep-base", "--no-quiet", "--no-verbose", "--no-signoff",
		"--no-committer-date-is-author-date", "--no-reset-author-date", "--no-ignore-date",
		"--no-ignore-whitespace", "--no-whitespace", "--no-force-rebase", "--no-preserve-merges",
		"--no-rerere-autoupdate", "--no-keep-empty", "--no-autosquash", "--no-update-refs",
		"--no-gpg-sign", "--no-autostash", "--no-exec", "--no-allow-empty-message", "--no-rebase-merges",
		"--no-fork-point", "--no-strategy", "--no-strategy-option", "--no-root",
		"--no-reschedule-failed-exec", "--no-reapply-cherry-picks",
	},
	"am": {
		"--interactive", "--no-verify", "--binary", "--3way", "--quiet", "--signoff", "--utf8", "--keep",
		"--keep-non-patch", "--message-id", "--keep-cr", "--scissors", "--quoted-cr", "--whitespace",
		"--ignore-space-change", "--ignore-whitespace", "--directory", "--exclude", "--include",
		"--patch-format", "--reject", "--resolvemsg", "--continue", "--resolved", "--skip", "--abort",
		"--quit", "--show-current-patch", "--retry", "--allow-empty", "--committer-date-is-author-date",
		"--ignore-date", "--rerere-autoupdate", "--gpg-sign", "--empty", "--rebasing", "--verify",
		"--no-interactive", "--no-binary", "--no-3way", "--no-quiet", "--no-signoff", "--no-utf8",
		"--no-keep", "--no-keep-non-patch", "--no-message-id", "--no-keep-cr", "--no-scissors",
		"--no-whitespace", "--no-ignore-space-change", "--no-ignore-whitespace", "--no-directory",
		"--no-exclude", "--no-include", "--no-patch-format", "--no-reject", "--no-resolvemsg",
		"--no-committer-date-is-author-date", "--no-ignore-date", "--no-rerere-autoupdate",
		"--no-gpg-sign", "--no-rebasing",
	},
	"pull": {
		"--verbose", "--quiet", "--progress", "--recurse-submodules", "--rebase", "--stat", "--summary",
		"--compact-summary", "--log", "--signoff", "--squash", "--commit", "--edit", "--cleanup", "--ff",
		"--ff-only", "--verify", "--verify-signatures", "--autostash", "--strategy", "--strategy-option",
		"--gpg-sign", "--allow-unrelated-histories", "--all", "--append", "--upload-pack", "--force",
		"--tags", "--prune", "--jobs", "--dry-run", "--keep", "--depth", "--shallow-since",
		"--shallow-exclude", "--deepen", "--unshallow", "--update-shallow", "--refmap",
		"--server-option", "--ipv4", "--ipv6", "--negotiation-tip", "--show-forced-updates",
		"--set-upstream", "--no-verbose", "--no-quiet", "--no-progress", "--no-recurse-submodules",
		"--no-rebase", "--no-stat", "--no-summary", "--no-compact-summary", "--no-log", "--no-signoff",
		"--no-squash", "--no-commit", "--no-edit", "--no-cleanup", "--no-ff", "--no-verify",
		"--no-verify-signatures", "--no-autostash", "--no-strategy", "--no-strategy-option",
		"--no-gpg-sign", "--no-allow-unrelated-histories", "--no-all", "--no-append", "--no-upload-pack",
		"--no-force", "--no-tags", "--no-prune", "--no-jobs", "--no-dry-run", "--no-keep", "--no-depth",
		"--no-shallow-since", "--no-shallow-exclude", "--no-deepen", "--no-update-shallow",
		"--no-server-option", "--no-ipv4", "--no-ipv6", "--no-negotiation-tip",
		"--no-show-forced-updates", "--no-set-upstream",
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

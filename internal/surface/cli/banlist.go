package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newBanlistCommand builds the `banlist` verb — the front door onto
// internal/core/banlist (itd-74, spc-20). Bare `abcd banlist` renders both layers
// read-only (abcd's bare-command-as-status convention); `add` and `remove` carry
// the mutations and take the layer explicitly, because writing a private pattern
// into committed config is the exact accident this feature exists to prevent.
func newBanlistCommand(asJSON *bool) *cobra.Command {
	banlistCmd := &cobra.Command{
		Use: "banlist",
		// NOT cobra.NoArgs: on an unknown token that validator's message quotes the
		// token verbatim (`unknown command "<token>"`), which on this verb may be a
		// would-be private value landing in stderr, scrollback, and logs. This one
		// withholds the token and names the real subcommands instead.
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf(`unknown subcommand (its text is withheld — it may be a private value); use "list", "add", or "remove"`)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := banlistRoot(cmd.ErrOrStderr())
			if err != nil {
				return usageError("abcd banlist", err)
			}
			rep, err := banlist.List(root)
			if err != nil {
				// The SAME refusal `banlist list` renders, so the same exit code: the
				// bare command and `list` are one read, and a caller scripting either
				// must not have to learn two contracts for one error.
				return usageError("abcd banlist", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				renderPrivateLayer(w, rep.Private)
				renderInheritedPrivateLayer(w, rep.Inherited)
				fmt.Fprintln(w)
				renderPublicLayer(w, rep.Public)
			})
		},
	}

	banlistCmd.AddCommand(newBanlistListCommand(asJSON))
	banlistCmd.AddCommand(newBanlistAddCommand(asJSON))
	banlistCmd.AddCommand(newBanlistRemoveCommand(asJSON))
	return banlistCmd
}

// newBanlistListCommand is the explicit read verb. Unscoped it renders both layers
// (the same result the bare command renders); a layer flag scopes it to one.
func newBanlistListCommand(asJSON *bool) *cobra.Command {
	var private, public bool
	listCmd := &cobra.Command{
		Use:  "list [--private | --public]",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := banlistRoot(cmd.ErrOrStderr())
			if err != nil {
				return usageError("abcd banlist list", err)
			}
			if private && public {
				return &exitError{Code: 2, Msg: "abcd banlist list: --private and --public are alternatives; omit both to render every layer"}
			}
			switch {
			case private:
				rep, err := banlist.ListPrivate(root)
				if err != nil {
					return usageError("abcd banlist list", err)
				}
				inherited, err := banlist.InheritedPrivate(root)
				if err != nil {
					return usageError("abcd banlist list", err)
				}
				view := privateView{PrivateReport: rep, Inherited: inherited}
				return render(cmd.OutOrStdout(), *asJSON, view, func(w io.Writer) {
					renderPrivateLayer(w, rep)
					renderInheritedPrivateLayer(w, inherited)
				})
			case public:
				rep, err := banlist.ListPublic(root)
				if err != nil {
					return usageError("abcd banlist list", err)
				}
				return render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) { renderPublicLayer(w, rep) })
			}
			rep, err := banlist.List(root)
			if err != nil {
				return usageError("abcd banlist list", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				renderPrivateLayer(w, rep.Private)
				renderInheritedPrivateLayer(w, rep.Inherited)
				fmt.Fprintln(w)
				renderPublicLayer(w, rep.Public)
			})
		},
	}
	addLayerFlags(listCmd, &private, &public)
	return listCmd
}

// newBanlistAddCommand adds one entry to the named layer.
func newBanlistAddCommand(asJSON *bool) *cobra.Command {
	var private, public bool
	var severity, successor string
	addCmd := &cobra.Command{
		Use:  "add --private|--public <key> <pattern|->",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := banlistRoot(cmd.ErrOrStderr())
			if err != nil {
				return usageError("abcd banlist add", err)
			}
			layer, err := chooseLayer("abcd banlist add", private, public)
			if err != nil {
				return err
			}
			key, pattern := args[0], args[1]
			if pattern == stdinPattern {
				pattern, err = readPatternFromStdin(cmd)
				if err != nil {
					return usageError("abcd banlist add", err)
				}
			}
			if layer == layerPrivate {
				res, err := banlist.AddPrivate(banlist.AddPrivateRequest{RepoRoot: root, Key: key, Pattern: pattern})
				if err != nil {
					return usageError("abcd banlist add", err)
				}
				return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
					// The key, never the pattern: the value the user just typed stays
					// out of the terminal, the scrollback, and any captured log.
					fmt.Fprintf(w, "private banlist — added %s (%d entr%s, %s)\n",
						termsafe.Sanitize(res.Key), res.Entries, plural(res.Entries), res.Path)
					fmt.Fprintln(w, "  the pattern is withheld from output by design; it lives only on this machine")
				})
			}
			res, err := banlist.AddPublic(banlist.AddPublicRequest{
				RepoRoot: root, Key: key, Pattern: pattern, Severity: severity, Successor: successor,
			})
			if err != nil {
				return usageError("abcd banlist add", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "public banlist — added %s (%s) in %s\n",
					termsafe.Sanitize(res.Entry.ID), res.Entry.Severity, res.Path)
				fmt.Fprintf(w, "  pattern: %s\n", termsafe.Sanitize(res.Entry.Pattern))
				fmt.Fprintln(w, "  commit the config change: the public layer is enforced in CI, for everyone")
			})
		},
	}
	addLayerFlags(addCmd, &private, &public)
	addCmd.Flags().StringVar(&severity, "severity", "", "public entry severity: blocker (default) | warn")
	addCmd.Flags().StringVar(&successor, "successor", "", "public entry's replacement, cited in the finding (default \"a generic term\")")
	return addCmd
}

// newBanlistRemoveCommand removes one entry from the named layer.
func newBanlistRemoveCommand(asJSON *bool) *cobra.Command {
	var private, public bool
	removeCmd := &cobra.Command{
		Use:  "remove --private|--public <key>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := banlistRoot(cmd.ErrOrStderr())
			if err != nil {
				return usageError("abcd banlist remove", err)
			}
			layer, err := chooseLayer("abcd banlist remove", private, public)
			if err != nil {
				return err
			}
			if layer == layerPrivate {
				res, err := banlist.RemovePrivate(root, args[0])
				if err != nil {
					return usageError("abcd banlist remove", err)
				}
				return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
					fmt.Fprintf(w, "private banlist — removed %s (%d entr%s remaining, %s)\n",
						termsafe.Sanitize(res.Key), res.Entries, plural(res.Entries), res.Path)
				})
			}
			res, err := banlist.RemovePublic(root, args[0])
			if err != nil {
				return usageError("abcd banlist remove", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "public banlist — removed %s from %s\n", termsafe.Sanitize(res.Entry.ID), res.Path)
			})
		},
	}
	addLayerFlags(removeCmd, &private, &public)
	return removeCmd
}

// layer names the two stores.
type layer int

const (
	layerPrivate layer = iota
	layerPublic
)

// addLayerFlags declares the layer selectors on a sub-verb. They are alternatives,
// never defaults: the destination of a banned pattern is the one decision this verb
// must never make on the user's behalf.
func addLayerFlags(cmd *cobra.Command, private, public *bool) {
	cmd.Flags().BoolVar(private, "private", false, "the gitignored per-machine layer ("+banlist.PrivateRelPath+")")
	cmd.Flags().BoolVar(public, "public", false, "the committed, CI-enforced layer ("+banlist.PublicConfigRelPath+")")
}

// chooseLayer resolves the mutually exclusive layer flags. Neither flag and both
// flags are usage errors: a mutation must name its layer, because a private pattern
// written into the committed config is precisely the leak this feature prevents.
// The requirement is enforced here rather than by a cobra required-flag annotation,
// keeping the tree's no-required-flags invariant intact.
func chooseLayer(verb string, private, public bool) (layer, error) {
	switch {
	case private && public:
		return 0, &exitError{Code: 2, Msg: verb + ": --private and --public are alternatives; name exactly one layer"}
	case private:
		return layerPrivate, nil
	case public:
		return layerPublic, nil
	default:
		return 0, &exitError{Code: 2, Msg: verb + ": name the layer — --private (per-machine, untracked) or --public (committed, CI-enforced)"}
	}
}

// usageError renders a core refusal as an exit-2 usage failure: every refusal this
// verb can raise (an unknown key, a duplicate, a hand-curated entry, an invalid
// pattern, an absent store) is the caller's input to fix, not a fault.
func usageError(verb string, err error) error {
	// scrubPaths, not err.Error(): a refusal from the core can carry an *os.PathError
	// whose text is an absolute path under the developer's home, and every sibling
	// verb routes its error text through the same scrub.
	return &exitError{Code: 2, Msg: verb + ": " + scrubPaths(err)}
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// renderPrivateLayer prints the private layer by key. It states the layer's reach
// plainly: the guard is local by construction, so a reader must never infer CI
// coverage from a populated list.
func renderPrivateLayer(w io.Writer, rep banlist.PrivateReport) {
	fmt.Fprintf(w, "private layer — %s\n", rep.Path)
	if !rep.Present {
		fmt.Fprintln(w, "  INACTIVE on this machine: the store is absent, so nothing is checked against it.")
		fmt.Fprintln(w, "  opt in with: abcd banlist add --private <key> <pattern>")
	} else {
		for _, e := range rep.Entries {
			fmt.Fprintf(w, "  %s (line %d)\n", termsafe.Sanitize(e.Key), e.Line)
		}
		if len(rep.Entries) == 0 {
			fmt.Fprintln(w, "  no entries")
		}
		fmt.Fprintln(w, "  keys only: the pattern values never reach any output, by design")
		if rep.NotIgnored {
			fmt.Fprintf(w, "  WARNING: git does NOT ignore %s — it is one `git add -A` from committing\n", rep.Path)
			fmt.Fprintf(w, "  the patterns it holds; add `%s/` to .gitignore before you rely on this layer\n", banlist.PrivateDirRelPath)
		}
		if rep.IgnoreUnanswerable {
			fmt.Fprintf(w, "  WARNING: git cannot answer whether it ignores %s (git absent from PATH,\n", rep.Path)
			fmt.Fprintln(w, "  an unreadable repository, or an ownership refusal) — the untracked-store proof this")
			fmt.Fprintln(w, "  layer rests on is missing; restore git's answer before you rely on it")
		}
		if !rep.Keyed && len(rep.Entries) > 0 {
			fmt.Fprintln(w, "  format: LEGACY (no `# abcd-banlist: keyed` first line) — every line is one whole-line")
			fmt.Fprintln(w, "  pattern and `add`/`remove` refuse until you add that line and give each entry a key")
		}
		// The two failure directions are reported apart because they need opposite
		// responses: an unusable line stops every commit until it is fixed, an inert
		// one stops nothing and leaves the name unguarded while looking healthy.
		for _, line := range rep.Malformed {
			fmt.Fprintf(w, "  line %d is not usable by the guard's engine — the pre-commit guard refuses every commit until it is fixed\n", line)
		}
		for _, line := range rep.Inert {
			fmt.Fprintf(w, "  line %d uses a Perl-style escape or a `(?…)` group; the guard's POSIX grep reads it as an extended regular expression, which may match differently than written (grep implementations diverge) — the guard does NOT refuse it, so verify it against the guard's grep\n", line)
		}
	}
	fmt.Fprintln(w, "  reach: "+banlist.PrivateReachNote)
}

// privateView is the JSON envelope `abcd banlist list --private` renders: the
// private layer of the checkout the verb ran in, plus — in a LINKED git worktree —
// the layer it inherits from its primary checkout.
//
// PrivateReport is EMBEDDED, so its fields stay at the top level and a standalone
// checkout's envelope is byte-for-byte what it always was; the inherited half is
// omitted entirely where there is nothing to inherit.
type privateView struct {
	banlist.PrivateReport
	Inherited *banlist.InheritedReport `json:"inherited,omitempty"`
}

// renderInheritedPrivateLayer prints the layer a LINKED git worktree inherits from
// its primary checkout, and prints NOTHING where there is nothing to inherit.
//
// The local tier is per-worktree and gitignored, so a `git worktree add` checkout
// has no store of its own — and the committed guard resolves and enforces the
// primary checkout's there (itd-150). Omitting the layer here would leave this
// render saying "INACTIVE on this machine" about a worktree whose every commit is
// being checked, which is the one disagreement between a board and its guard that
// cannot be allowed to stand.
func renderInheritedPrivateLayer(w io.Writer, rep *banlist.InheritedReport) {
	if rep == nil {
		return
	}
	// The store is named RELATIVE to the other checkout, and the other checkout is
	// not named at all — the same restraint the committed shell guard applies, for
	// the same reason. A checkout's directory name is very often a private name its
	// own store bans (a project codename is the commonest entry there is), and this
	// layer's contract is that no pattern value reaches output. RedactHome is not
	// enough here: it rewrites a `$HOME` prefix and leaves the directory name — the
	// banned string — printed in full, and this line renders on the success path of
	// an ordinary status read, into scrollback, transcripts and CI logs.
	fmt.Fprintf(w, "  inherited from the primary checkout — its %s\n",
		termsafe.Sanitize(rep.Private.Path))
	fmt.Fprintln(w, "  this is a linked git worktree; the guard reads the primary checkout's store as a")
	fmt.Fprintln(w, "  fallback, so these entries are enforced here too. The fallback is READ-SIDE only:")
	fmt.Fprintln(w, "  `add --private` here writes to THIS worktree's store, which the primary checkout's")
	fmt.Fprintln(w, "  own guard does not read — declare a name in the checkout that must enforce it")
	for _, e := range rep.Private.Entries {
		fmt.Fprintf(w, "    %s (line %d)\n", termsafe.Sanitize(e.Key), e.Line)
	}
	if len(rep.Private.Entries) == 0 {
		fmt.Fprintln(w, "    no entries")
	}
	// The same two failure directions the local layer reports, for the same reason:
	// an unusable line in the INHERITED store stops every commit in this worktree,
	// and the remedy is in the other checkout.
	for _, line := range rep.Private.Malformed {
		fmt.Fprintf(w, "    line %d is not usable by the guard's engine — the pre-commit guard refuses every commit until it is fixed in the primary checkout\n", line)
	}
	for _, line := range rep.Private.Inert {
		fmt.Fprintf(w, "    line %d uses a Perl-style escape or a `(?…)` group; the guard's POSIX grep may match it differently than written — verify it against the guard's grep\n", line)
	}
}

// banlistHealthLines renders ahoy's name-guard verdict: what occupies each hook
// path, whether the public family can actually be enforced, and what shape the
// private layer is in on this machine. It never reads an entry — the store's
// content is the secret, and a status board is exactly the surface that must not
// hold it — but it does report counts, because a store whose lines do not parse
// stops every commit while looking populated.
//
// The reach caveat is printed beside it by the caller, unconditionally: "hook
// committed" beside a present store otherwise reads as coverage that neither a pull
// request nor a rebase actually has.
func banlistHealthLines(h ahoy.BanlistHealth) []string {
	// The merge half's phrasing depends on the half it delegates to: install writes it
	// only beside abcd's own guard, so beside a foreign one "install writes it" would
	// send a reader to run a command that declines.
	hooks := hookPhrase("pre-commit", h.Hook) + ", " + mergeHookPhrase(h)
	if (h.Hook == ahoy.HookInstalled || h.MergeHook == ahoy.HookInstalled) && !h.HooksPathArmed {
		// A committed hook is not a running hook. This clone's LOCAL config does not
		// point at the hooks directory — which a user-level dispatcher may still do, so
		// this is an instruction, never a verdict that the guard is off.
		hooks += " (arm this clone: git config core.hooksPath .githooks)"
	}
	return []string{hooks, publicFamilyPhrase(h.PublicFamily) + "; " + privateStorePhrase(h)}
}

// hookPhrase words one hook path's state. "Committed" and "installed" are different
// claims and only the first is abcd's to make: git runs the hook the clone's hooks
// path selects, which abcd neither sets nor fully observes.
func hookPhrase(name string, state ahoy.HookState) string {
	switch state {
	case ahoy.HookInstalled:
		return name + " hook committed"
	case ahoy.HookForeign:
		return "a foreign " + name + " hook occupies .githooks/" + name + " — the banlist is NOT checked there"
	case ahoy.HookUnreadable:
		return ".githooks/" + name + " cannot be read as a hook"
	default:
		return name + " hook MISSING (`abcd ahoy install` writes it)"
	}
}

// mergeHookPhrase words the merge half, whose remedy is not its own. An absent shim
// beside abcd's guard is install's to write; an absent shim beside a hook abcd does
// not own is not, because install refuses to delegate to a foreign hook — so the
// line gives the remedy the merge_hook_inert gap gives, and never one install
// declines to perform.
func mergeHookPhrase(h ahoy.BanlistHealth) string {
	if h.MergeHook == ahoy.HookAbsent && h.Hook != ahoy.HookInstalled {
		return "merge commits NOT checked (move the foreign pre-commit hook aside and re-run `abcd ahoy install`)"
	}
	return hookPhrase("pre-merge-commit", h.MergeHook)
}

// publicFamilyPhrase words the committed layer's state. Four faults, four remedies:
// a file to write, an array to add, a file to restore, and a placement to settle.
// One shared wording would send a reader to do the wrong one.
func publicFamilyPhrase(state ahoy.PublicFamilyState) string {
	switch state {
	case ahoy.PublicFamilyPresent:
		return "public family present"
	case ahoy.PublicFamilyUnusable:
		return "public family unusable (no banned_tokens array)"
	case ahoy.PublicFamilyUnreadable:
		return "public family unreadable (" + banlist.PublicConfigRelPath + " cannot be opened)"
	case ahoy.PublicFamilyIgnored:
		// Present, readable, and enforced by nobody: git ignores the file, so CI never
		// sees it. Under `visibility: public` the abcd fence ignores the whole .abcd/
		// namespace, which is exactly where the public family lives.
		return "public family NOT ENFORCEABLE (git ignores " + banlist.PublicConfigRelPath + ", so CI never sees it)"
	default:
		return "public family MISSING"
	}
}

// privateStorePhrase words this machine's opt-in state and the store's shape.
func privateStorePhrase(h ahoy.BanlistHealth) string {
	if !h.PrivateStore {
		// Absent is not "clean": nothing on this machine is checked against a private
		// name, and the guard itself says so at commit time in these words.
		return "private layer INACTIVE on this machine"
	}
	if h.PrivateUnreadable {
		return "private store UNREADABLE — the guard refuses every commit until it is fixed"
	}
	format := "legacy"
	if h.PrivateKeyed {
		format = "keyed"
	}
	phrase := fmt.Sprintf("private store present (%s, %d entr%s", format, h.PrivateEntries, plural(h.PrivateEntries))
	if h.PrivateUnparsed > 0 {
		// An unusable line stops EVERY commit, so it belongs beside the count rather
		// than one verb away.
		phrase += fmt.Sprintf(", %d unusable line%s", h.PrivateUnparsed, pluralS(h.PrivateUnparsed))
	}
	phrase += ")"
	if !h.PrivateStoreIgnored {
		phrase += " — WARNING: NOT gitignored, one `git add -A` from committing its patterns"
	}
	return phrase
}

// pluralS is the plain -s plural, for nouns `plural` (entry/entries) does not fit.
func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderPublicLayer prints the public family in full: committed config is
// reviewable config, so its patterns belong on screen.
func renderPublicLayer(w io.Writer, rep banlist.PublicReport) {
	fmt.Fprintf(w, "public layer — %s\n", rep.Path)
	if !rep.Present {
		fmt.Fprintln(w, "  absent: this repo has no docs-lint config, so no public token is gated")
		return
	}
	for _, e := range rep.Entries {
		owner := "hand-curated"
		if e.Managed {
			owner = "verb-managed"
		}
		fmt.Fprintf(w, "  %-32s %-8s %-13s %s\n", termsafe.Sanitize(e.ID), termsafe.Sanitize(e.Severity), owner, termsafe.Sanitize(e.Pattern))
	}
	if len(rep.Entries) == 0 {
		fmt.Fprintln(w, "  no entries")
	}
	fmt.Fprintln(w, "  reach: enforced deterministically by `abcd docs lint` in CI, with the per-line escape")
}

// stdinPattern is the pattern argument that means "read the pattern from stdin".
const stdinPattern = "-"

// maxPatternBytes caps the pattern read from stdin (trust boundary), generously
// above any real expression.
const maxPatternBytes = 8 << 10

// banlistRoot resolves the repo whose banlist is being read or written. It resolves
// the GIT WORKING-TREE TOPLEVEL, because that is the exact root the committed
// pre-commit guard enforces at (`git rev-parse --show-toplevel`), and the store must
// live where the guard reads it. rulesRoot's "nearest .abcd inside the working
// tree" would disagree when a subdirectory carries its own .abcd/: it would write
// the store there — where the guard never reads it and the root-anchored gitignore
// does not match it — leaving the layer inactive while `add` reports a repo-relative
// path. Falls back to rulesRoot, which is cwd outside a git working tree (it never
// walks past one, GHSA-vvqc-3mv2-5p49), only when git cannot answer, so a non-git
// use still resolves.
//
// In a LINKED git worktree it stays the WORKTREE's own root, deliberately. This is
// the write root, and itd-150 is read-side resolution only: an `add` run in a
// worktree writes the entry into the store of the checkout the user is standing in,
// never reaching across into another working tree they did not name. The inherited
// layer the guard also enforces there is composed on the read paths above, from
// banlist.InheritedPrivate — the same primary-checkout resolution the committed
// pre-commit guard makes, kept in lockstep so the board and the guard cannot
// disagree about which entries are in force.
func banlistRoot(w io.Writer) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if top, err := gitutil.Run(cwd, "rev-parse", "--show-toplevel"); err == nil && top != "" {
		return top, nil
	}
	return rulesRoot(cwd, w), nil
}

// readPatternFromStdin reads the pattern as EXACTLY one line. It is the recommended
// way to enter a REAL private pattern: an argument is world-readable in
// /proc/<pid>/cmdline for the life of the process, is captured verbatim by process
// auditing, and lands in the shell's history file.
//
// It reads ALL of stdin (not just the first line) and refuses trailing data: reading
// one line and discarding the rest would report success while storing only the first
// of a multi-line pattern — the argv path refuses the same input, so the two must
// reach the same verdict. Multi-line or trailing-data stdin is refused with the same
// "carriage return or newline" verdict AddPrivate gives a multi-line pattern argument.
func readPatternFromStdin(cmd *cobra.Command) (string, error) {
	// One byte past the cap distinguishes "exactly at the cap" from "overflowed".
	data, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxPatternBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxPatternBytes {
		return "", fmt.Errorf("the stdin pattern is too long (over %d bytes); a banlist pattern is a short expression", maxPatternBytes)
	}
	// A single trailing newline is the terminator of the one piped line; strip it, then
	// refuse anything that still spans lines.
	pattern := strings.TrimRight(string(data), "\r\n")
	if pattern == "" {
		return "", fmt.Errorf("no pattern on stdin (pass the pattern as `-` and pipe one line, e.g. `printf %%s 'PATTERN' | abcd banlist add --private KEY -`)")
	}
	if strings.ContainsAny(pattern, "\r\n") {
		return "", fmt.Errorf("the stdin pattern spans more than one line: it contains a carriage return or newline, which a line-oriented store cannot hold; pipe exactly one line")
	}
	return pattern, nil
}

// applyBanlistFlagErrors installs banlistFlagError on the banlist verb and every
// subcommand under it. It runs after the tree-wide usage tagging, which sets a
// FlagErrorFunc on every command and would otherwise replace this one — and the
// generic one quotes the offending token, which on this verb may be the pattern.
func applyBanlistFlagErrors(root *cobra.Command) {
	for _, cmd := range root.Commands() {
		if cmd.Name() != "banlist" {
			continue
		}
		cmd.SetFlagErrorFunc(banlistFlagError)
		for _, sub := range cmd.Commands() {
			sub.SetFlagErrorFunc(banlistFlagError)
		}
	}
}

// banlistFlagError renders a flag-parse failure WITHOUT the offending token. A
// private pattern beginning with a dash is read by the parser as flags, and cobra's
// own message quotes it verbatim — which for this verb is the one value that must
// never reach a terminal, a scrollback, or a log. The remedy is named instead.
func banlistFlagError(cmd *cobra.Command, err error) error {
	_ = err // deliberately discarded: it quotes the token that must not be echoed
	return &exitError{Code: 2, Msg: cmd.CommandPath() + ": an argument was read as a flag (its text is withheld — " +
		"it may be a pattern). A pattern beginning with `-` cannot be an argument: pipe it instead, " +
		"`printf %s 'PATTERN' | abcd banlist add --private KEY -`, which also keeps it out of argv and shell history"}
}

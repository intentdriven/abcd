// Package cli is abcd's default front door: a Cobra command tree that marshals
// internal/core results to the terminal (human text or, with --json, machine
// output). It holds no business logic — every command delegates to core and
// only formats the result, so an MCP or other front door can expose the same
// core verbs without duplicating behaviour.
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"unicode"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/memory"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/record"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// exitError carries a specific process exit code out of a command. The root
// command sets SilenceErrors, so main inspects this to choose the exit code and
// (when Msg is non-empty) print a single diagnostic line. An empty Msg means the
// command already rendered its output and only the exit code should propagate.
type exitError struct {
	Code int
	Msg  string
}

func (e *exitError) Error() string { return e.Msg }
func (e *exitError) ExitCode() int { return e.Code }

// helpRunE is the RunE every sub-verb parent carries. A cobra parent with no RunE
// is not Runnable, so cobra prints help and exits 0 WITHOUT running the command's
// Args validator: an unknown sub-verb — a typo, or a retired spelling like the
// pre-spc-30 `disembark oracle` — then reads as SUCCESS to a script (iss-266).
// Giving the parent a RunE keeps bare invocation showing help while letting its
// declared Args validator run and refuse a stray token at the usual exit 2. A
// parent must therefore have BOTH: with Args nil, cobra's legacyArgs falls
// through to ArbitraryArgs and a Runnable parent still exits 0. Most parents
// declare cobra.NoArgs; `banlist` declares a custom validator, and `capture` and
// `intent` are deliberately cobra.ArbitraryArgs because their positional is free
// text (they guard typos themselves). TestEveryParentRefusesAnUnknownSubverb
// holds both halves tree-wide, so a parent added later cannot reintroduce the
// hole by missing either one.
func helpRunE(cmd *cobra.Command, _ []string) error { return cmd.Help() }

// failOpenNoArgs is cobra.NoArgs for the two parents a HOST HOOK reaches: `guard`
// (PreToolUse runs `guard hook` before every shell command) and `hook`
// (UserPromptSubmit runs `hook prompt-router` before every prompt).
//
// On the hook plane an exit status is not a diagnostic, it is an INSTRUCTION: the
// host reads 2 as "block this action". Cobra's usage error exits 2, which is
// right in a terminal and wrong here — it makes abcd answer a question it did not
// evaluate. `guard hook`'s contract (spc-16, itd-103 AC 1) is fail-open-loud:
// exit 2 means "the guard decided to block", and every path that is NOT a
// decision exits 1 so the command still runs and the warning is still seen. An
// unknown sub-verb is not a decision.
//
// This is reachable because the manifest and the binary can skew — hooks/hooks.json
// ships with the plugin git clone while hooks/bootstrap.sh fetches the binary from
// the latest release — so a renamed hook sub-verb would otherwise block every
// shell command in the session, and the PreToolUse wrapper cannot rescue it: that
// wrapper treats 2 as a recognised code, so its "FAILED TO RUN … UNGUARDED" net
// never fires (iss-267).
//
// The refusal stays loud and non-zero, so iss-266's guarantee is intact: a
// mistyped sub-verb never reads as success. Only the code moves, from the host's
// blocking status to its non-blocking one.
//
// SCOPE, stated plainly because the gap matters more than the fix: this covers
// the unknown-SUB-VERB path under these two parents, and nothing else. Three
// neighbouring paths still exit 2, deliberately — an unknown TOP-LEVEL token hits
// the root's validator (root's exit-2 contract is pinned by three other tests and
// inverting it is a larger change than this), a stray positional on a LEAF such
// as `guard hook` exits 2, and any unknown FLAG exits 2 through FlagErrorFunc.
// None is reachable from today's manifest. What keeps them unreachable is not
// this function but the doctrine the manifest test enforces: hooks.json's
// spellings are frozen, and a rename is absorbed by an alias in the binary. That
// is iss-269.
// failOpenFlagError is FlagErrorFunc for the hook plane, for the same reason as
// failOpenNoArgs: an unknown flag is a usage error abcd cannot answer, and on this
// plane cobra's exit 2 is the host's instruction to BLOCK. This is the half
// iss-267 left behind, and it is the one a future manifest change makes
// reachable — add a flag to a hooks.json invocation and it skews against every
// older binary that has never heard of it (iss-269).
func failOpenFlagError(_ *cobra.Command, err error) error {
	return &exitError{Code: 1, Msg: err.Error() + hookPlaneSkewNote}
}

// hookPlaneSkewNote is the second line both hook-plane refusals carry: what the
// exit code means here, and the one thing that actually fixes it.
const hookPlaneSkewNote = "\nabcd: refusing at exit 1, not the host's blocking status — a usage error abcd" +
	" cannot answer is not a decision to block. If a hook invoked this, the plugin manifest and the" +
	" binary have skewed; re-run hooks/bootstrap.sh or reinstall the plugin."

// applyHookPlaneFailOpen installs the fail-open usage handling on every command a
// host hook can reach — the paths named in hooks/hooks.json, plus the parents on
// the way to them. It runs AFTER markUsageErrorsExitTwo, which sets a
// FlagErrorFunc on every command and would otherwise replace this one; the same
// ordering applyBanlistFlagErrors needs, and for the same reason.
//
// The set is spelled out rather than "everything under guard and hook" because
// `guard check` sits under the same parent and its contract is the OPPOSITE: it
// is the human/scriptable verb, where a fault exits 2 so a caller never reads
// silence as clearance (spc-16). Sweeping the subtree would quietly invert it.
// TestHookPlaneFailsOpenOnEveryUsageError derives the same set from the manifest
// and would fail if this list drifted from it.
func applyHookPlaneFailOpen(root *cobra.Command) {
	for _, path := range [][]string{
		{"guard"}, {"guard", "hook"},
		{"hook"}, {"hook", "prompt-router"}, {"hook", "prompt-router-reset"},
		{"hook", "session-start"}, {"hook", "session-end"},
	} {
		if cmd := findByPath(root, path); cmd != nil {
			cmd.SetFlagErrorFunc(failOpenFlagError)
			cmd.Args = failOpenNoArgs
		}
	}
}

// findByPath walks the tree by name. Deliberately NOT cobra's Find: that calls
// stripFlags, which calls mergePersistentFlags, which makes HasAvailableFlags()
// true — so merely LOOKING UP a command at construction time appends " [flags]"
// to its UseLine and silently rewrites the generated CLI reference. The drift
// gate caught it; this walk has no side effect at all.
func findByPath(root *cobra.Command, path []string) *cobra.Command {
	cur := root
	for _, name := range path {
		var next *cobra.Command
		for _, sub := range cur.Commands() {
			if sub.Name() == name {
				next = sub
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

func failOpenNoArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return &exitError{Code: 1, Msg: err.Error() + hookPlaneSkewNote}
	}
	return nil
}

// NewRootCommand builds the abcd command tree. Bare `abcd` renders a read-only
// status board (abcd's convention: bare invocation never mutates); subcommands
// carry the actions.
func NewRootCommand() *cobra.Command {
	var asJSON bool
	var noColor bool

	root := &cobra.Command{
		Use:   "abcd [<record-id>]",
		Short: "Agent-based configuration for development",
		Long: "Agent-based configuration for development.\n\n" +
			"Bare `abcd` renders the read-only status board — what can I do. A single\n" +
			"positional matching a record id (`iss-N`, `itd-N`, `spc-N`, `adr-N`) instead\n" +
			"reports what that record is, where it lives, and the next move for its\n" +
			"lifecycle state — what is this. Both forms are strictly read-only; any other\n" +
			"positional is refused as an unknown command.",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Bare answers "what can I do"; `abcd <id>` answers "what is this, and
		// what is my next move" (spc-26). The positional is accepted iff it
		// matches the record-id shape — any other positional reproduces
		// cobra.NoArgs' unknown-command error byte-for-byte, so the id gate
		// never widens the root's surface.
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && record.IDRe.MatchString(args[0]) {
				return nil
			}
			return cobra.NoArgs(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// Record-id dispatch: read-only describe, never a write.
			if len(args) == 1 {
				d, err := record.Describe(cwd, args[0])
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), asJSON, d, func(w io.Writer) {
					// Title and link values come from record files a hostile
					// clone can shape — sanitise before the terminal.
					fmt.Fprintf(w, "%s (%s, %s) — %s\n", d.ID, d.Family, d.Status, termsafe.Sanitize(d.Title))
					fmt.Fprintf(w, "  path: %s\n", termsafe.Sanitize(d.Path))
					for _, k := range slices.Sorted(maps.Keys(d.Links)) {
						fmt.Fprintf(w, "  %s: %s\n", k, termsafe.Sanitize(d.Links[k]))
					}
					for _, m := range d.NextMoves {
						fmt.Fprintf(w, "  next: %s\n", termsafe.Sanitize(m))
					}
				})
			}
			// The banner: interactive-TTY bare invocation only, per adr-49 —
			// --json, pipes, and hooks never receive a decoration byte, and
			// the status board below renders exactly as it always has.
			if !asJSON && bannerTTY(cmd.OutOrStdout()) {
				writeBanner(cmd.OutOrStdout(), noColor, os.Getenv)
			}
			st, err := core.Status(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), asJSON, st, func(w io.Writer) {
				fmt.Fprintf(w, "abcd — %s\n", st.Dir)
				fmt.Fprintf(w, "  git repo:   %v\n", st.IsGitRepo)
				fmt.Fprintf(w, "  record:     %v\n", st.HasRecord)
				fmt.Fprintf(w, "  work tiers: %v\n", st.WorkTiers)
			})
		},
	}
	root.PersistentFlags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	// Root-local by design: colour exists only on the bare invocation, so a
	// persistent flag would be dead surface on every subcommand (itd-112).
	root.Flags().BoolVar(&noColor, "no-color", false, "render the banner without color")

	root.AddCommand(newVersionCommand(&asJSON))
	root.AddCommand(newUpdateCommand(&asJSON))

	root.AddCommand(newAhoyCommand(&asJSON))
	root.AddCommand(newLintCommand(&asJSON))
	root.AddCommand(newGuardCommand(&asJSON))
	root.AddCommand(newIdentityCommand(&asJSON))

	var launchDryRun bool
	launchCmd := &cobra.Command{
		Use:   "launch",
		Short: "Preview the public launch bundle and release gates (--dry-run required; read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if !launchDryRun {
				return fmt.Errorf("abcd launch: pass --dry-run to preview the bundle (publishing is not wired at this stage)")
			}
			rep, err := launch.DryRun(launch.DryRunRequest{
				RepoRoot: cwd,
				Version:  publishedVersion(cwd),
				// Grading the citation baseline needs the lint engine, which
				// imports launch for its semver — so the measurement is taken
				// HERE, where both are already in scope, and handed in as data.
				Citations: citationPreflight(cwd),
				// Same reason, same shape: measured here, handed in as data.
				Receipts: receiptPreflight(cwd),
			})
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), asJSON, rep, func(w io.Writer) {
				fmt.Fprintf(w, "abcd launch (dry-run) — version %s\n", rep.Version)
				fmt.Fprintf(w, "  files bundled:  %d\n", len(rep.Bundle.Included))
				fmt.Fprintf(w, "  scan hardfails: %d\n", rep.Scan.HardFails)
				for _, g := range rep.Gates {
					if g.Name == "citation-baseline" && g.Status == "ran" {
						fmt.Fprintf(w, "  citations:      %s\n", termsafe.Sanitize(g.Detail))
					}
					// The semantic-receipt gate refuses releases and CI cannot run it,
					// so the plain render must not stay silent about it either: a row
					// only --json shows is invisible to everyone who does not know to
					// ask (iss-2608231226342272).
					if g.Name == "semantic-receipts" {
						fmt.Fprintf(w, "  receipts:       %s\n", termsafe.Sanitize(g.Detail))
					}
				}
				fmt.Fprintf(w, "  would publish:  %v\n", rep.WouldPublish)
				for _, reason := range rep.WouldRefuseOn {
					// Each reason embeds a raw repo filename (a control-char-rejected
					// path carries the offending bytes), so it is untrusted terminal
					// output and passes through the canonical sanitiser, matching the
					// citation line above.
					fmt.Fprintf(w, "  would refuse on: %s\n", termsafe.Sanitize(reason))
				}
			})
		},
	}
	launchCmd.Flags().BoolVar(&launchDryRun, "dry-run", false, "preview the launch bundle and gates without publishing")
	// `ship` is the release-cut verb: the dry-run above previews the launch
	// BUNDLE, this cuts the RELEASE (version + changelog record set). They hang
	// off one command because they gate the same event.
	launchCmd.AddCommand(newLaunchShipCommand(&asJSON))
	// `scaffold` writes the changelog-driven release machinery (release.yml,
	// auto-release.yml, runbook) into a managed repo that lacks it (itd-93). It
	// extends 04-launch because launch already owns how a release is cut and gated.
	launchCmd.AddCommand(newLaunchScaffoldCommand(&asJSON))
	root.AddCommand(launchCmd)

	root.AddCommand(newChangelogCommand(&asJSON))

	root.AddCommand(newCaptureCommand(&asJSON))
	root.AddCommand(newBanlistCommand(&asJSON))
	root.AddCommand(newMemoryCommand(&asJSON))
	root.AddCommand(newRulesCommand(&asJSON))
	root.AddCommand(newHookCommand())
	root.AddCommand(newHistoryCommand(&asJSON))
	root.AddCommand(newDocsCommand(&asJSON))
	root.AddCommand(newIntentCommand(&asJSON))
	root.AddCommand(newIdeateCommand(&asJSON))
	root.AddCommand(newDecideCommand(&asJSON))
	root.AddCommand(newSpecCommand(&asJSON))
	root.AddCommand(newDisembarkCommand(&asJSON))
	root.AddCommand(newEmbarkCommand(&asJSON))
	root.AddCommand(newSiteCommand(&asJSON))
	root.AddCommand(newReadingCommand(&asJSON))

	// A cobra usage error (unknown flag, unknown subcommand, stray positional
	// argument) is a plain error with no ExitCode(), so Run() would map it to
	// exit 1 — but `abcd lint` documents Conftest's tri-state where exit 1 means
	// "warnings only" (lint.go). A mistyped invocation must not masquerade as a
	// clean-ish gate pass: usage errors exit 2, like every usage error abcd raises
	// itself. Flag-parse errors route through FlagErrorFunc; argument errors come
	// from each command's Args validator — wrap both across the whole tree (B13).
	markUsageErrorsExitTwo(root)
	// AFTER the generic tagging, which sets a FlagErrorFunc on every command: the
	// banlist verbs need one that does NOT quote the offending token, because for
	// them the token may be a private pattern. Applied here rather than in the verb
	// so the ordering is explicit — the generic pass would otherwise overwrite it.
	applyBanlistFlagErrors(root)
	// Also after the generic tagging: the assemble verb's flag refusal names the
	// two operands the design admits, because the operand it most often refuses
	// is one it used to take (adr-2609021016286571).
	applyReadingFlagErrors(root)
	// Also after the generic tagging, and last: on the hook plane exit 2 is the
	// host's instruction to BLOCK, so every usage error a hook can provoke refuses
	// at exit 1 instead (iss-269).
	applyHookPlaneFailOpen(root)

	return root
}

// markUsageErrorsExitTwo walks the command tree and tags every cobra usage error
// with exit code 2, so a parse/usage failure never lands on the ambiguous exit 1
// that the audit tri-state reserves for "warnings only" (B13). Flag-parse errors
// are routed through FlagErrorFunc (inherited by children, but set on each for
// clarity); argument-validation errors (cobra.NoArgs violations, unknown
// subcommands) surface from each command's Args validator, which is wrapped.
//
// A validator that ALREADY chose an exit code keeps it. Exit 2 is the right
// default for a usage error, but it is not universal: on the hook plane 2 is the
// host's blocking status, so the `guard` and `hook` parents refuse at 1 instead
// (failOpenNoArgs, iss-267). Re-stamping every validator error here would silently
// undo that — which it did, until this branch.
func markUsageErrorsExitTwo(c *cobra.Command) {
	c.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &exitError{Code: 2, Msg: err.Error()}
	})
	if validate := c.Args; validate != nil {
		c.Args = func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd, args); err != nil {
				var coded *exitError
				if errors.As(err, &coded) {
					return err
				}
				return &exitError{Code: 2, Msg: err.Error()}
			}
			return nil
		}
	}
	for _, sub := range c.Commands() {
		markUsageErrorsExitTwo(sub)
	}
}

// docsLintResult is the machine-readable envelope for `abcd docs lint`: the
// findings plus the blocker count that decides the exit status.
type docsLintResult struct {
	Findings []lint.Finding `json:"findings"`
	Blockers int            `json:"blockers"`
}

// newDocsCommand builds the `docs` sub-tree. Its `lint` verb is the docs-currency
// drift gate: it loads .abcd/docs-lint.json (or --config), runs the shared
// internal/core/lint engine over the repo, renders the findings (text or --json),
// and exits non-zero when any blocker survives — the same engine record-lint uses.
func newDocsCommand(asJSON *bool) *cobra.Command {
	docsCmd := &cobra.Command{
		Use:   "docs",
		Short: "Documentation-currency checks for this repo",
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
	}

	var configPath string
	var rootDir string
	var releaseGate bool
	lintCmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint docs for change-narration, broken links, and stray root markdown",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := rootDir
			if root == "" {
				cwd, err := os.Getwd()
				if err != nil {
					// A fault, not a rendered refusal: exit 2 (could not be
					// evaluated), consistent with the engine-fault path below.
					return &exitError{Code: 2, Msg: "docs lint: " + scrubPaths(err)}
				}
				root = cwd
			}
			root, err := filepath.Abs(root)
			if err != nil {
				return &exitError{Code: 2, Msg: "docs lint: " + scrubPaths(err)}
			}
			cfgPath := configPath
			if cfgPath == "" {
				cfgPath = filepath.Join(root, ".abcd", "docs-lint.json")
			}
			cfg, err := lint.LoadConfig(cfgPath)
			if err != nil {
				// Surface config-load failures as clean, repo-relative
				// diagnostics — never a raw os.Open/os.ReadFile error, whose
				// *PathError embeds the absolute path (iss-29: no absolute path
				// in machine output). Reference what the user typed when they
				// passed --config, else the relative default.
				ref := filepath.Join(".abcd", "docs-lint.json")
				if configPath != "" {
					ref = configPath
				}
				if os.IsNotExist(err) {
					return &exitError{Code: 2, Msg: fmt.Sprintf(
						"docs lint: config not found at %s — run in a prepared repo or pass --config", ref)}
				}
				// Strip the path-bearing wrapper: a *PathError's inner Err is the
				// bare cause ("is a directory", "permission denied"), no path.
				detail := err.Error()
				var pe *os.PathError
				if errors.As(err, &pe) {
					detail = pe.Err.Error()
				}
				return &exitError{Code: 2, Msg: fmt.Sprintf("docs lint: cannot read config %s: %s", ref, detail)}
			}
			// --release-gate promotes the citation staleness finding from the
			// commit gate's warn to a blocker (spc-17: commits are never
			// calendar-blocked; a release is). The FLAG is the trust root, the
			// way `record-lint --release-gate` arms the receipt gate: a repo
			// must not be able to defang its own release by editing the
			// committed config.
			if releaseGate {
				cfg = lint.ArmCitationOverdue(cfg)
			}
			findings, err := lint.Lint(cfg, root)
			if err != nil {
				// A lint the engine could not run is "could not be evaluated",
				// exit 2 — the same tri-state code `abcd lint` and record-lint
				// return, never the code-1 a blocker finding reserves, so a CI
				// gate keying on >=2 does not read a lint that never happened as
				// an ordinary findings-pass. scrubPaths keeps an absolute path
				// out of the message.
				return &exitError{Code: 2, Msg: "docs lint: " + scrubPaths(err)}
			}
			blockers := 0
			for _, f := range findings {
				if f.Severity == "blocker" {
					blockers++
				}
			}
			res := docsLintResult{Findings: findings, Blockers: blockers}
			if err := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				for _, f := range findings {
					// Every non-numeric field embeds untrusted repo content: File and
					// Message carry paths and link targets, and Severity/RuleID come
					// verbatim from the committed config (LoadConfig validates rule
					// severities, but a banned token's id is free text), so all four
					// are sanitised.
					fmt.Fprintf(w, "%s:%d: [%s %s] %s\n",
						termsafe.Sanitize(f.File), f.Line, termsafe.Sanitize(strings.ToUpper(f.Severity)), termsafe.Sanitize(f.RuleID), termsafe.Sanitize(f.Message))
				}
				fmt.Fprintf(w, "abcd docs lint — %d finding(s), %d blocker(s)\n", len(findings), blockers)
			}); err != nil {
				return err
			}
			if blockers > 0 {
				return fmt.Errorf("docs lint: %d blocker finding(s)", blockers)
			}
			return nil
		},
	}
	lintCmd.Flags().StringVar(&configPath, "config", "", "path to docs-lint.json (default: <root>/.abcd/docs-lint.json)")
	lintCmd.Flags().StringVar(&rootDir, "root", "", "repo root to lint (default: current working directory)")
	lintCmd.Flags().BoolVar(&releaseGate, "release-gate", false,
		"run as the release gate: a citation past its staleness threshold blocks instead of warning (release-time only)")
	docsCmd.AddCommand(lintCmd)
	// `cite` maintains the baseline `lint` enforces: the refresh does the live
	// fetching the gate refuses to do, and confirm closes the manual queue.
	docsCmd.AddCommand(newCiteCommand(asJSON))

	return docsCmd
}

// ignoredScope turns the --include-ignored flag into the probe options it means.
// The narrow scan passes NO option, so the default lives in one place — the core
// — and the front door cannot drift into asserting its own.
func ignoredScope(include bool) []lifeboat.ProbeOption {
	if !include {
		return nil
	}
	return []lifeboat.ProbeOption{lifeboat.IncludeIgnored()}
}

// newDisembarkCommand builds the operator `disembark` sub-tree:
//
//   - `probe <repo>` walks a repository read-only and reports, per brief
//     section, whether a lifeboat could ground it, at what tier and confidence,
//     citing the evidence — and, for a blank, what was searched and the question
//     a human must answer.
//   - `coverage <report.json>...` reduces several probe reports to the cross-repo
//     table (section × repo) that answers whether the brief structure is sound.
//   - `plan <repo>` shows the full file set a pack would write, without writing.
//   - `pack <repo> <dest>` writes that file set to <dest> — never to the source.
//
// `pack` is the packer M3b ships, backed by the `/abcd:disembark` command
// surface (`commands/disembark.md`), so the surface-registry row is
// `shipped`. probe/coverage/plan are read-only; pack writes only to <dest>,
// behind a destination safety gate, and never mutates the source repository.
func newDisembarkCommand(asJSON *bool) *cobra.Command {
	disembarkCmd := &cobra.Command{
		Use:   "disembark",
		Short: "Lifeboat tooling: coverage probe, pack dry-run, and out-of-tree pack",
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
	}

	// --include-ignored widens the walk to files git ignores. Off by default:
	// disembark reads a working tree and a packed lifeboat cites evidence by
	// path:line, so a file the user told git to ignore stays out of scope until
	// they say otherwise (iss-2608241828356533). It is offered on all three verbs
	// because the salvage case — a dead or archived repository whose uncommitted
	// residue is the only thing left — is a PACK, not just a report.
	var probeIgnored, planIgnored, packIgnored bool

	probeCmd := &cobra.Command{
		Use:   "probe [repo]",
		Short: "Report which brief sections a lifeboat could ground from a repository (read-only)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			abs, err := filepath.Abs(target)
			if err != nil {
				return err
			}
			if info, err := os.Stat(abs); err != nil || !info.IsDir() {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark probe: %s is not a directory", target)}
			}
			cov, err := lifeboat.Probe(abs, ignoredScope(probeIgnored)...)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark probe: %s", scrubPaths(err))}
			}
			return render(cmd.OutOrStdout(), *asJSON, cov, func(w io.Writer) {
				fmt.Fprint(w, cov.Render())
			})
		},
	}

	coverageCmd := &cobra.Command{
		Use:   "coverage <report.json>...",
		Short: "Aggregate probe reports into the cross-repo section×repo coverage table",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			covs := make([]lifeboat.Coverage, 0, len(args))
			for _, path := range args {
				// A probe report is a cross-repo artifact (produced by `probe` on
				// other repos), so its content is untrusted: read it behind the same
				// guards as every other operand (O_NOFOLLOW, regular-file, size cap),
				// never a raw os.ReadFile that follows a symlink or reads unbounded.
				data, err := fsutil.ReadGuarded(path, maxOperandJSONBytes)
				if err != nil {
					// Reference the path the user typed, not an absolute PathError.
					detail := err.Error()
					switch {
					case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
						detail = "not a readable regular file (a symlink or non-regular operand is refused)"
					case errors.Is(err, fsutil.ErrTooBig):
						detail = fmt.Sprintf("exceeds the %d-byte cap", maxOperandJSONBytes)
					default:
						var pe *os.PathError
						if errors.As(err, &pe) {
							detail = pe.Err.Error()
						}
					}
					return &exitError{Code: 2, Msg: fmt.Sprintf("disembark coverage: cannot read %s: %s", path, detail)}
				}
				var cov lifeboat.Coverage
				if err := json.Unmarshal(data, &cov); err != nil {
					return &exitError{Code: 2, Msg: fmt.Sprintf("disembark coverage: %s is not a coverage report: %s", path, err)}
				}
				// A probe report always stamps schema_version >= 1; json.Unmarshal
				// of any other JSON object succeeds with the zero value (schema
				// version 0), which would otherwise sail past the guard as an
				// all-blank phantom repo (B38). Reject it as not a coverage report,
				// mirroring the type-mismatch message above.
				if cov.SchemaVersion < 1 {
					return &exitError{Code: 2, Msg: fmt.Sprintf(
						"disembark coverage: %s is not a coverage report: missing schema_version", path)}
				}
				if cov.SchemaVersion > lifeboat.SchemaVersion {
					return &exitError{Code: 2, Msg: "disembark coverage: " +
						update.TooNew(path, cov.SchemaVersion, lifeboat.SchemaVersion).Error()}
				}
				covs = append(covs, cov)
			}
			agg := lifeboat.Aggregate(covs)
			return render(cmd.OutOrStdout(), *asJSON, agg, func(w io.Writer) {
				fmt.Fprint(w, agg.Render())
			})
		},
	}

	planCmd := &cobra.Command{
		Use:   "plan [repo]",
		Short: "Show the full lifeboat file set a pack would write, without writing anything (dry run)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}
			abs, err := filepath.Abs(target)
			if err != nil {
				return err
			}
			if info, err := os.Stat(abs); err != nil || !info.IsDir() {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark plan: %s is not a directory", target)}
			}
			lb, err := lifeboat.Plan(abs, ignoredScope(planIgnored)...)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark plan: %s", scrubPaths(err))}
			}
			return render(cmd.OutOrStdout(), *asJSON, lb.Manifest(), func(w io.Writer) {
				fmt.Fprint(w, lb.RenderManifest())
			})
		},
	}

	packCmd := &cobra.Command{
		Use:   "pack <repo> <dest>",
		Short: "Pack a lifeboat from a repository into a destination directory (writes <dest>, never the source)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoAbs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if info, err := os.Stat(repoAbs); err != nil || !info.IsDir() {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark pack: %s is not a directory", args[0])}
			}
			// Build the source repo's secret scanner and fail closed if its config
			// is degraded — a pack must not ship secrets under a weakened ruleset.
			sc, err := scanner.New(repoAbs)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark pack: %s", scrubPaths(err))}
			}
			if bad, reason := sc.Unavailable(); bad {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark pack: secret scanner unavailable, refusing: %s", reason)}
			}
			scan := func(files []lifeboat.PlannedFile) error {
				hard, first := 0, ""
				for _, f := range files {
					for _, fnd := range sc.ScanText(string(f.Content), f.Path) {
						if fnd.Severity == scanner.SeverityHardFail {
							hard++
							if first == "" {
								first = fmt.Sprintf("%s (%s)", f.Path, fnd.Kind)
							}
						}
					}
				}
				if hard > 0 {
					return fmt.Errorf("%d hard-fail secret(s) in planned content (first: %s); fix at source, not in the lifeboat", hard, first)
				}
				return nil
			}
			res, err := lifeboat.Pack(repoAbs, args[1], scan, ignoredScope(packIgnored)...)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("disembark pack: %s", scrubPaths(err))}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, res.Render())
			})
		},
	}

	var lessonsJSON string
	graveyardCmd := &cobra.Command{
		Use:   "graveyard <lifeboat-dir> --lessons-json <file|->",
		Short: "Validate host-produced lesson JSON against a packed lifeboat and write the survivors (cite-or-be-dropped)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if lessonsJSON == "" {
				return &exitError{Code: 2, Msg: "disembark graveyard: --lessons-json <file|-> is required"}
			}
			dirAbs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			raw, err := readLessonsPayload(cmd, lessonsJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark graveyard: " + scrubPaths(err)}
			}
			// Exit 0 even when every entry was dropped or nothing was written:
			// a drop is reported honestly, not a failure. Only structural faults
			// (not a lifeboat, unreadable graveyard, unparseable/oversize payload)
			// return an error and exit 2.
			res, err := lifeboat.IngestLessons(dirAbs, raw)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark graveyard: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, res.Render())
			})
		},
	}
	graveyardCmd.Flags().StringVar(&lessonsJSON, "lessons-json", "", "path to the host-produced lesson JSON (or - for stdin)")

	// The three M6 synthesis verbs (itd-88) share one dual-mode shape: WITHOUT the
	// --*-json flag they run the core's deterministic evidence-only fallback (raw ==
	// nil); WITH the flag they validate an untrusted host-delegated payload. Every
	// core error — including the press-release whole-document refusal
	// (ErrPressReleaseUncited) — is a scrubbed exit 2; a per-entry drop is reported
	// honestly and stays exit 0.
	var principlesJSON string
	principlesCmd := &cobra.Command{
		Use:   "principles <lifeboat-dir> [--principles-json <file|->]",
		Short: "Distil principles from a packed lifeboat (deterministic from the ADRs, or validate host-produced principle JSON)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirAbs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			raw, err := readSynthesisPayload(cmd, principlesJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark principles: " + scrubPaths(err)}
			}
			res, err := lifeboat.SynthesizePrinciples(dirAbs, raw)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark principles: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, res.Render())
			})
		},
	}
	principlesCmd.Flags().StringVar(&principlesJSON, "principles-json", "", "path to host-produced principle JSON (or - for stdin); absent runs deterministic mode")

	var pressReleaseJSON string
	pressReleaseCmd := &cobra.Command{
		Use:   "press-release <lifeboat-dir> [--press-release-json <file|->]",
		Short: "Compose the lifeboat's press release (deterministic from the brief/spine, or validate host-produced press-release JSON)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirAbs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			raw, err := readSynthesisPayload(cmd, pressReleaseJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark press-release: " + scrubPaths(err)}
			}
			// A delegated press release citing nothing resolvable is a whole-document
			// refusal (ErrPressReleaseUncited): exit 2, the derived file left untouched
			// (design §5 exception). It flows through the generic exit-2 wrapping like
			// every other structural fault.
			res, err := lifeboat.ComposePressRelease(dirAbs, raw)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark press-release: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, res.Render())
			})
		},
	}
	pressReleaseCmd.Flags().StringVar(&pressReleaseJSON, "press-release-json", "", "path to host-produced press-release JSON (or - for stdin); absent runs deterministic mode")

	var reviewJSON string
	reviewCmd := &cobra.Command{
		Use:   "review <lifeboat-dir> <source-repo> [--review-json <file|->]",
		Short: "Review a packed lifeboat against its source repo — a registered verdict and cited findings (deterministic, or validate a host-produced verdict JSON)",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return &exitError{Code: 2, Msg: "disembark review: <lifeboat-dir> <source-repo> are both required"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dirAbs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			raw, err := readSynthesisPayload(cmd, reviewJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark review: " + scrubPaths(err)}
			}
			// The source repo's content is never read (the core gates it as a real dir
			// only); a manifest failure is a MAJOR_RETHINK verdict input, exit 0.
			res, err := lifeboat.ReviewLifeboat(dirAbs, args[1], raw)
			if err != nil {
				return &exitError{Code: 2, Msg: "disembark review: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, res.Render())
			})
		},
	}
	reviewCmd.Flags().StringVar(&reviewJSON, "review-json", "", "path to the host-produced review verdict JSON (or - for stdin); absent runs deterministic mode")

	disembarkCmd.AddCommand(probeCmd)
	disembarkCmd.AddCommand(coverageCmd)
	const includeIgnoredUsage = "also read files git ignores (widens the scan; the report says so)"
	probeCmd.Flags().BoolVar(&probeIgnored, "include-ignored", false, includeIgnoredUsage)
	planCmd.Flags().BoolVar(&planIgnored, "include-ignored", false, includeIgnoredUsage)
	packCmd.Flags().BoolVar(&packIgnored, "include-ignored", false, includeIgnoredUsage)

	disembarkCmd.AddCommand(planCmd)
	disembarkCmd.AddCommand(packCmd)
	disembarkCmd.AddCommand(graveyardCmd)
	disembarkCmd.AddCommand(principlesCmd)
	disembarkCmd.AddCommand(pressReleaseCmd)
	disembarkCmd.AddCommand(reviewCmd)
	return disembarkCmd
}

// newEmbarkCommand builds the operator `embark` sub-tree — the write half of the
// M5 record round-trip (itd-88, adr-35), the inverse of `disembark`:
//
//   - `probe <lifeboat-dir> [target-dir]` inspects a packed lifeboat against a
//     target read-only and reports what would land where, what conflicts would
//     block a write, which files are not embarked, the marker action, and the
//     coverage handoff. A plan WITH conflicts is a success (a report, exit 0).
//   - `from <lifeboat-dir> [target-dir]` writes the record families back into the
//     target through two-layer containment; on ANY conflict it refuses and writes
//     nothing (exit 1, one bulk report), re-injecting the current marker block
//     (never foreign prose) into the target CLAUDE.md.
//
// The target defaults to the working directory. Structural faults (not a lifeboat,
// schema too new, failed manifest verification, bad target) exit 2 with a scrubbed
// diagnostic; the conflict refusal exits 1 after rendering the report. `embark`
// backs the `/abcd:embark` command surface (commands/embark.md), so its
// surface-registry row is shipped.
func newEmbarkCommand(asJSON *bool) *cobra.Command {
	embarkCmd := &cobra.Command{
		Use:   "embark",
		Short: "Unpack a lifeboat's record families back into a target repo (probe read-only; from writes)",
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
	}

	// resolveDirs turns the args into absolute lifeboat + target dirs; the target
	// defaults to the working directory when omitted.
	resolveDirs := func(args []string) (lbAbs, tgtAbs string, err error) {
		lbAbs, err = filepath.Abs(args[0])
		if err != nil {
			return "", "", err
		}
		target := "."
		if len(args) == 2 {
			target = args[1]
		}
		tgtAbs, err = filepath.Abs(target)
		return lbAbs, tgtAbs, err
	}

	probeCmd := &cobra.Command{
		Use:   "probe <lifeboat-dir> [target-dir]",
		Short: "Report what a lifeboat would write into a target, read-only (coverage blanks first)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			lbAbs, tgtAbs, err := resolveDirs(args)
			if err != nil {
				return err
			}
			plan, err := lifeboat.EmbarkProbe(lbAbs, tgtAbs)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("embark probe: %s", scrubPaths(err))}
			}
			return render(cmd.OutOrStdout(), *asJSON, plan, func(w io.Writer) {
				fmt.Fprint(w, plan.Render())
			})
		},
	}

	fromCmd := &cobra.Command{
		Use:   "from <lifeboat-dir> [target-dir]",
		Short: "Write a lifeboat's record families into a target repo; refuses on any conflict",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			lbAbs, tgtAbs, err := resolveDirs(args)
			if err != nil {
				return err
			}
			res, err := lifeboat.EmbarkFrom(lbAbs, tgtAbs)
			if err != nil {
				// A conflict refusal is an EXPECTED outcome, not a fault: render the
				// bulk report, then propagate exit 1 with an empty message (the report
				// is the output, the exit code the only extra signal). Every other
				// error is a structural fault: exit 2, scrubbed to one line.
				if errors.Is(err, lifeboat.ErrEmbarkConflicts) {
					if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
						fmt.Fprint(w, res.Render())
					}); rerr != nil {
						return rerr
					}
					return &exitError{Code: 1}
				}
				return &exitError{Code: 2, Msg: fmt.Sprintf("embark from: %s", scrubPaths(err))}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprint(w, res.Render())
			})
		},
	}

	embarkCmd.AddCommand(probeCmd)
	embarkCmd.AddCommand(fromCmd)
	return embarkCmd
}

// readLessonsPayload reads the untrusted lesson JSON behind the trust guards,
// mirroring the intent verdict reader: a file must be a regular, non-symlink,
// size-capped file; "-" reads stdin bounded to the same cap. The cap is the
// exported lifeboat.MaxLessonsBytes. The file read uses fsutil.ReadGuarded so
// the symlink refusal, regular-file check, and size cap all happen on the open
// fd in one call — no lstat→ReadFile swap window.
func readLessonsPayload(cmd *cobra.Command, spec string) ([]byte, error) {
	if spec == "-" {
		return readCappedStdin(cmd, lifeboat.MaxLessonsBytes)
	}
	return readGuardedOperand(spec, lifeboat.MaxLessonsBytes)
}

// readSynthesisPayload reads an untrusted synthesis payload (principles,
// press-release, or review verdict JSON) behind the same trust guards as
// readLessonsPayload, capped at the exported lifeboat.MaxSynthesisBytes. An EMPTY
// spec (the flag absent) returns a nil slice — the sentinel the dual-mode cores
// read as "run deterministic mode", never a delegated payload. A "-" spec reads
// stdin bounded to the cap; a file must be regular, non-symlink, and under the cap.
func readSynthesisPayload(cmd *cobra.Command, spec string) ([]byte, error) {
	if spec == "" {
		return nil, nil // flag absent → deterministic mode (nil raw)
	}
	if spec == "-" {
		return readCappedStdin(cmd, lifeboat.MaxSynthesisBytes)
	}
	return readGuardedOperand(spec, lifeboat.MaxSynthesisBytes)
}

// readGuardedOperand reads an untrusted operand file behind fsutil.ReadGuarded
// (O_NOFOLLOW + regular-file on the open fd + size cap, one call, no lstat→read
// TOCTOU) and maps the guard's sentinels to clean, path-scrubbed messages —
// the shared body behind readLessonsPayload/readSynthesisPayload's file branch.
func readGuardedOperand(spec string, cap int64) ([]byte, error) {
	data, err := fsutil.ReadGuarded(spec, cap)
	if err != nil {
		switch {
		case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
			return nil, fmt.Errorf("%s is not a readable regular file (a symlink or non-regular operand is refused)", spec)
		case errors.Is(err, fsutil.ErrTooBig):
			return nil, fmt.Errorf("%s exceeds the %d-byte cap", spec, cap)
		default:
			return nil, err
		}
	}
	return data, nil
}

// maxHookStdinBytes caps the hook payload read from stdin (trust boundary).
const maxHookStdinBytes = 1 << 20 // 1 MiB

// hookInput is the subset of the Claude Code hook stdin payload the hook
// entrypoints read. Unknown fields are ignored.
type hookInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	Prompt    string `json:"prompt"`
	Source    string `json:"source"`
	Event     string `json:"hook_event_name"`
	// TranscriptPath is supplied by the Stop hook; it names the session
	// transcript on disk. Read by `hook session-end` only.
	TranscriptPath string `json:"transcript_path"`
}

// readCappedStdin reads a "-" operand one byte past the cap so an over-cap
// payload is refused whole rather than truncated into a severed prefix — the
// same refuse-whole guarantee readGuardedOperand gives the file transport
// (iss-201's class; spc-4's refuse-whole invariant on the transcript path). A
// bare LimitReader(cap) reads exactly cap bytes, so an over-cap payload is
// silently cut and its length-cap refusal never fires; the cap+1 probe closes
// that.
func readCappedStdin(cmd *cobra.Command, cap int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), cap+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > cap {
		return nil, fmt.Errorf("stdin payload exceeds the %d-byte cap", cap)
	}
	return raw, nil
}

// readHookInput reads and size-caps the hook stdin payload. It reads one byte
// past the cap so an over-cap payload is reported as such rather than
// truncated into a severed prefix that json.Unmarshal misblames as malformed
// host JSON (iss-201's class; guardCandidate is the pattern).
func readHookInput(cmd *cobra.Command) (hookInput, error) {
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxHookStdinBytes+1))
	if err != nil {
		return hookInput{}, err
	}
	if len(raw) > maxHookStdinBytes {
		return hookInput{}, fmt.Errorf("payload is over the %d-byte cap; it was discarded unparsed", maxHookStdinBytes)
	}
	var in hookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return hookInput{}, err
	}
	return in, nil
}

// hookSession returns a stable session key, defaulting when the harness omits
// the id (the hash in the state layer neutralises any hostile value). The
// harness supplies session_id in practice; the "default" fallback means two
// concurrent id-less sessions would share one dedup ledger — an accepted
// edge-case degradation, never a correctness or safety issue.
func hookSession(in hookInput) string {
	if in.SessionID == "" {
		return "default"
	}
	return in.SessionID
}

// newHookCommand builds the operator-internal `hook` sub-tree: the Claude Code
// prompt-router entrypoints (itd-3). These are NOT a user surface — they are the
// injection transport, one front door onto internal/core/rules alongside the
// `abcd rules` verb. Every path is fail-closed and NON-blocking: a malformed
// payload, an unreadable rules.json, or a state error injects nothing, logs a
// diagnostic to stderr (out-of-band, per D3), and exits 0 so it can never wedge
// a session.
func newHookCommand() *cobra.Command {
	hookCmd := &cobra.Command{
		Use:    "hook",
		Short:  "Claude Code hook entrypoints (operator-internal)",
		Hidden: true,
		Args:   failOpenNoArgs,
		RunE:   helpRunE,
	}

	// prompt-router — UserPromptSubmit: recall-match, dedup, inject.
	hookCmd.AddCommand(&cobra.Command{
		Use:   "prompt-router",
		Short: "UserPromptSubmit: inject the rules matching the prompt",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in, err := readHookInput(cmd)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd rules: unreadable hook payload (%v); injecting nothing\n", err)
				return nil
			}
			cwd := in.Cwd
			if cwd == "" {
				if wd, err := os.Getwd(); err == nil {
					cwd = wd
				}
			}
			root := rulesRoot(cwd, cmd.ErrOrStderr())
			rs, err := rules.Load(root)
			if err != nil {
				// rules.Load errors already carry their own "rules:" prefix, so
				// wrap with a bare "abcd" to avoid "abcd rules: rules: …"
				// (iss-2608261550491547).
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd %v; injecting nothing\n", err)
				return nil
			}
			// A domain Load dropped (no rules of its own) is skipped, not
			// fatal — but silently missing is the shape the drop exists to
			// prevent, so each one is named here, out of band.
			for _, note := range rs.Notes() {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", note)
			}
			session := hookSession(in)
			// The fixed-N backstop comes from the repo's config (default 15 when
			// unset); event-driven reset is the primary refresh (D1).
			res := rules.Inject(rs, in.Prompt, rules.LoadState(session), rules.LoadBackstop(root))
			if err := rules.SaveState(session, res.State); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd rules: state save failed (%v)\n", err)
			}
			// The names carry their layer ("PII (repo override)"), the same
			// label the injected heading bears, so the out-of-band log says
			// whose words went into the context (GHSA-22f8-qf5r-gjgq).
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd rules: turn %d, injected %d domain(s) %v, %d bytes\n",
				res.State.Count, len(res.Injected), res.Labels(), len(res.Text))
			if res.Text != "" {
				fmt.Fprint(cmd.OutOrStdout(), res.Text)
			}
			return nil
		},
	})

	// prompt-router-reset — SessionStart / PreCompact: clear the dedup ledger so
	// the next prompt re-injects (the event-driven refresh, D1/B2).
	hookCmd.AddCommand(&cobra.Command{
		Use:   "prompt-router-reset",
		Short: "SessionStart/PreCompact: clear the dedup ledger",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in, err := readHookInput(cmd)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd rules: unreadable reset payload (%v)\n", err)
				return nil
			}
			session := hookSession(in)
			if err := rules.ResetState(session); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd rules: reset failed (%v)\n", err)
				return nil
			}
			// SessionStart is a natural sweep point for stale ledgers.
			rules.PruneState(rules.StateTTL)
			// %q quotes the untrusted hook_event_name so an embedded newline or
			// ANSI escape cannot spoof the operator's diagnostic stream.
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd rules: reset session (%q)\n", in.Event)
			return nil
		},
	})

	// session-end — SessionEnd: redact and store the session transcript (adr-29).
	//
	// Wired to SessionEnd, NOT Stop. The plan said Stop, but Stop fires once per
	// assistant *turn*: a 40-turn session would store 40 growing supersets of one
	// transcript, since Capture's sha256 dedup only collapses byte-identical
	// re-captures and a live transcript grows between turns. SessionEnd fires once
	// when the session terminates, which is the session-granular record the gate
	// asks for. SessionEnd also ignores exit code and stdout by contract, which
	// matches this verb's fail-closed, non-blocking shape exactly.
	//
	// This is a new verb because `history capture` cannot be wired to a hook: from
	// stdin it *requires* --session <id>, and the hook delivers its session id
	// inside a JSON payload, not as a flag.
	//
	// It is the only irreversible thing abcd does. A session that ends without
	// being captured is gone: no later code can reconstruct a transcript that was
	// never stored. That asymmetry — a missed capture is permanent, a failed
	// capture is merely a lost session — is why every path here degrades to "log
	// and exit 0" rather than surfacing an error to the host.
	hookCmd.AddCommand(&cobra.Command{
		Use:   "session-end",
		Short: "SessionEnd: stage the raw transcript for the next session to redact and store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Diagnostics go to stderr, out of band; stdout stays empty, since a
			// Stop hook's stdout is not a place to speak to the model.
			warn := func(format string, a ...any) error {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: "+format+"\n", a...)
				return nil // never non-zero: a Stop hook must not wedge the session
			}

			in, err := readHookInput(cmd)
			if err != nil {
				return warn("unreadable Stop payload (%v); capturing nothing", err)
			}
			if in.TranscriptPath == "" {
				return warn("Stop payload carries no transcript_path; capturing nothing")
			}
			cwd := in.Cwd
			if cwd == "" {
				if wd, err := os.Getwd(); err == nil {
					cwd = wd
				}
			}
			det, err := ahoy.Detect(cwd)
			if err != nil || det.RootSHA == "" {
				return warn("cannot resolve the repo's root-commit SHA from %q; capturing nothing", cwd)
			}
			raw, err := readTranscript(in.TranscriptPath)
			if err != nil {
				return warn("%v; capturing nothing", err)
			}
			// Stage, do NOT capture. Redaction costs ~0.7s/MB and the host cancels
			// a shutdown hook rather than wait, so capturing here dropped every
			// transcript past a couple of MB — silently, and precisely the long,
			// dense sessions most worth keeping (iss-2608230817034768). Staging is
			// one write, so this hook's cost no longer scales with the transcript;
			// the next SessionStart drains it through the same fail-closed Capture.
			// Resolve first, so the store's own diagnostics — a corpus migrated
			// off the legacy location, an opt-in declaration that was not
			// honoured — reach stderr rather than being discarded inside Stage.
			if st, err := history.Resolve(captureRoot(cwd), det.RootSHA); err == nil {
				for _, n := range st.Notes {
					fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", termsafe.Sanitize(n))
				}
			}
			res, err := history.Stage(captureRoot(cwd), det.RootSHA, in.SessionID, raw)
			if err != nil {
				return warn("staging failed (%v); this session was not captured", err)
			}
			if !res.Wrote {
				return warn("session %s already staged with identical bytes (no-op)", res.Staged.SessionID)
			}
			if res.Replaced {
				// Different bytes for an already-staged session: the later
				// snapshot replaced the earlier one, and this says so rather
				// than reporting a no-op that would hide a replaced transcript.
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: re-staged %s (%d bytes), replacing %d stale bytes; the next session redacts and stores it\n",
					res.Staged.SessionID, res.Staged.Bytes, res.ReplacedBytes)
				return nil
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd history: staged %s (%d bytes); the next session redacts and stores it\n",
				res.Staged.SessionID, res.Staged.Bytes)
			return nil
		},
	})

	// session-start — SessionStart: warn, visibly, when the transcript store is
	// not bootstrapped for this repo (iss-95).
	//
	// The session-end hook cannot say this at session end — SessionEnd ignores a
	// hook's exit code and stdout, so a not-installed session captures nothing and
	// no one is told. SessionStart is the one session hook with a user-visible
	// channel: it renders a hook's stderr as a notice ONLY on a non-zero exit, and
	// never blocks the session on it. So the "loud" path here deliberately exits 2
	// — the only way the warning reaches the user — while every other path stays
	// silent at exit 0. `ahoy install` and `ahoy doctor` already handle the
	// installed and health-check cases; this covers the plugin-enabled-but-never-
	// installed gap that was otherwise silent.
	hookCmd.AddCommand(&cobra.Command{
		Use:   "session-start",
		Short: "SessionStart: drain staged transcripts into the store, and warn about an unbootstrapped store or a stale binary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			in, err := readHookInput(cmd)
			if err != nil {
				return nil // malformed payload: not a store problem, stay silent
			}
			cwd := in.Cwd
			if cwd == "" {
				if wd, err := os.Getwd(); err == nil {
					cwd = wd
				}
			}
			var notices []string
			// Drain first: SessionEnd only stages the raw transcript, because
			// redaction at exit loses the race with the host's shutdown
			// cancellation (iss-2608230817034768). This is where the previous
			// session's transcript actually becomes a record, and it is the one
			// hook with a time budget to do it in.
			//
			// The budget bounds an interactive stall: a repo with a backlog would
			// otherwise redact all of it before the user's first prompt. Whatever
			// it leaves is said out loud rather than dropped, so a partial pass
			// never reads as a complete one.
			if det, err := ahoy.Detect(cwd); err == nil && det.RootSHA != "" {
				// Resolving bootstraps the store for this repo — the reason a
				// machine that never ran `ahoy install` still captures (iss-95)
				// — and carries out any migration off the legacy location. Its
				// notes are user-visible facts about where their transcripts
				// now are, so they are notices, not stderr chatter.
				if st, err := history.Resolve(captureRoot(cwd), det.RootSHA); err == nil {
					for _, n := range st.Notes {
						notices = append(notices, "abcd "+termsafe.Sanitize(n))
					}
				} else {
					notices = append(notices, fmt.Sprintf(
						"abcd: the transcript store could not be opened (%s); this session will not be captured.",
						termsafe.Sanitize(fsutil.RedactHome(err.Error()))))
				}
				if dr, err := history.Drain(captureRoot(cwd), det.RootSHA, sessionStartDrainBudget); err == nil {
					for _, f := range dr.Failed {
						notices = append(notices, fmt.Sprintf(
							"abcd: session %s ended but could not be stored (%s). Its raw transcript is kept at %s — capture it by hand or delete it; it is unredacted.",
							termsafe.Sanitize(f.SessionID), termsafe.Sanitize(fsutil.RedactHome(f.Err)), termsafe.Sanitize(fsutil.RedactHome(f.Path))))
					}
					if dr.Remaining > 0 {
						notices = append(notices, fmt.Sprintf(
							"abcd: %d earlier session(s) are still awaiting capture — run `abcd history drain` to finish, or start another session.",
							dr.Remaining))
					}
				} else {
					notices = append(notices, fmt.Sprintf(
						"abcd: could not drain staged transcripts (%s); previously ended sessions are not yet stored.",
						termsafe.Sanitize(err.Error())))
				}
			}
			// There is no "the store is not set up for this repo" notice any
			// more, and there must not be one: the store creates itself on the
			// line above, so a notice telling the user to run `ahoy install`
			// before transcripts are captured would now be false (iss-95).
			// The skew notice is a plugin-root fact, not a repo one, so it stands
			// whatever the repo detection above could answer (itd-105).
			if n := binarySkewNotice(); n != "" {
				notices = append(notices, n)
			}
			// itd-111: a dogfood binary behind (or dirty against) its own source
			// checkout tip. os.Executable names the binary; the comparison is
			// git-only and never touches the network (adr-38 tier 1).
			if exe, err := os.Executable(); err == nil {
				if n := stalenessNotice(cwd, exe); n != "" {
					notices = append(notices, n)
				}
			}
			// itd-111 (AC6): a version transition performed since this repo was
			// last set up — the running binary differs from the recorded
			// setup_version. Report only; the fetch that changed it is
			// provisioning's job. Both values come from disk (config + build info).
			if from, to, changed := ahoy.VersionTransition(cwd); changed {
				notices = append(notices, fmt.Sprintf(
					"abcd: the running binary is version %s, but this repo was last set up with %s — run `/abcd:ahoy install` (or `abcd ahoy install`) to reconcile the recorded version.",
					termsafe.Sanitize(to), termsafe.Sanitize(from)))
			}
			if len(notices) == 0 {
				return nil
			}
			// Exit ZERO, and put the notices where a SessionStart hook's output is
			// actually read (iss-2608241115201044).
			//
			// This returned exit 2 on the documented assumption that a non-zero
			// SessionStart surfaces stderr without blocking. It does not: the
			// harness renders the non-zero exit as an opaque "SessionStart:startup
			// hook error" banner followed by a truncated echo of the hooks.json
			// command string, and DROPS the stderr text. So every notice this hook
			// exists to deliver — transcript-capture gaps, staged-drain failures,
			// the backlog count, binary skew — reached the user as an error with no
			// content, which is worse than silence: it reports a fault in abcd
			// rather than the condition it was trying to report.
			//
			// The notice TEXT goes to stderr, sanitised. Only a CONSTANT goes to
			// stdout.
			//
			// SessionStart's stdout is injected into the session's context, and the
			// notices interpolate repo-derived strings — `meta.setup_version` from
			// .abcd/config.json is a TRACKED file, so a pull request or a fork can
			// set it, and an adversarial review demonstrated a directive payload
			// reaching context through exactly this line. termsafe.Sanitize defends a
			// terminal, not a context window: it masks control bytes and leaves prose
			// untouched, which is the whole of an injection. Nothing repo-derived is
			// worth putting on that channel for a notice.
			//
			// So stdout carries a fixed sentence and a count — enough for the session
			// to know notices exist and say where to look — and the content stays on
			// stderr where it has always been. The exit code stays 0 because a notice
			// is not a hook failure, which is the delivery bug this fixes
			// (iss-2608241115201044): a non-zero exit renders as an opaque error
			// banner with the text dropped.
			for _, n := range notices {
				fmt.Fprintln(cmd.ErrOrStderr(), n)
			}
			// Name the verbs that actually hold the detail. An earlier draft sent
			// the reader to `abcd ahoy` alone, which renders install state and a
			// gap COUNT and says nothing about a staged transcript or a failed
			// drain — so the one notice naming a privacy artefact was the one it
			// lost. `abcd history staged` is where that lives.
			fmt.Fprintf(cmd.OutOrStdout(),
				"abcd: %d session-start notice(s) on this hook's stderr; "+
					"run `abcd history staged` for transcript backlog or `abcd ahoy` for install state.\n",
				len(notices))
			return nil
		},
	})

	return hookCmd
}

// maxTranscriptBytes caps the transcript read from disk. Generous for a JSONL
// session log, and bounded so a pathological file cannot stall the Stop hook
// while the scanner walks it.
const maxTranscriptBytes = 64 << 20 // 64 MiB

// sessionStartDrainBudget bounds how many staged transcripts one SessionStart
// redacts before handing control back to the user. Redaction runs at roughly
// 0.7s per MB, so an unbounded drain of a backlog would stall the first prompt
// by however long the backlog happens to be. Four is a compromise: enough that a
// normal one-session-behind case always clears in a single start, small enough
// that the worst case stays a few seconds. Anything left is reported, never
// dropped — `abcd history drain` finishes it without waiting for a new session.
const sessionStartDrainBudget = 4

// readTranscript reads the file named by the Stop payload's transcript_path.
//
// The path is external input, so the read goes through fsutil.ReadGuarded —
// O_NOFOLLOW so a planted symlink is refused rather than followed, O_NONBLOCK
// so a FIFO or device node cannot hang the hook (a hung Stop hook wedges the
// user's session), a regular-file check on the opened descriptor, and a cap+1
// probe so a file that grows past the cap between stat and read is refused
// whole instead of stored silently truncated (iss-347).
func readTranscript(path string) ([]byte, error) {
	raw, err := fsutil.ReadGuarded(path, maxTranscriptBytes)
	if err != nil {
		switch {
		case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
			return nil, fmt.Errorf("transcript %q is not a readable regular file (a symlink or non-regular transcript is refused)", path)
		case errors.Is(err, fsutil.ErrTooBig):
			return nil, fmt.Errorf("transcript %q is over the %d-byte cap", path, maxTranscriptBytes)
		default:
			return nil, fmt.Errorf("cannot read transcript %q (%v)", path, err)
		}
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("transcript %q is empty", path)
	}
	return raw, nil
}

// rulesView is the machine-readable envelope for bare `abcd rules`: the kill
// switch plus the active domains.
type rulesView struct {
	Disabled bool                   `json:"disabled"`
	Domains  []rules.ResolvedDomain `json:"domains"`
}

// newRulesCommand builds the `rules` verb — the vendor-neutral front door onto
// internal/core/rules (itd-3). Bare `abcd rules` renders the active rule set;
// a positional DOMAIN scopes to one domain (case-insensitive). Read-only,
// diagnostic — it never mutates and there is no `show` sub-verb (the positional
// argument is the scope, per the bare-command-as-render discipline).
func newRulesCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "rules [domain]",
		Short: "Render the active rule set; a positional DOMAIN scopes to one (read-only)",
		Long: `Render the rule set the modular-rules loader injects: the bundled default
domains merged with this repo's .abcd/rules.json. Bare, it renders every active
domain; a positional DOMAIN (case-insensitive) renders that one domain regardless
of its state or the kill switch, so a dormant domain is still inspectable.

Every domain says which layer it came from. A domain the repo override names —
its rules replaced, its state changed, or a custom domain declared — renders as
"## NAME (repo override)" here, in the injected block and in the hook's
diagnostic, and carries "source": "repo" in --json; an untouched bundled domain
renders bare and carries "source": "bundled". Read-only.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rs, err := rules.Load(rulesRoot(cwd, cmd.ErrOrStderr()))
			if err != nil {
				return err
			}
			// Stderr, never stdout: --json renders one document, and a
			// diagnostic mixed into it would break every parser reading it.
			for _, note := range rs.Notes() {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", note)
			}
			// Scoped: inspect one domain's configured content regardless of its
			// state OR the kill switch — this diagnostic shows what a domain holds,
			// not what would inject right now (bare `abcd rules` reports disabled).
			if len(args) == 1 {
				name := strings.ToUpper(args[0])
				d, ok := rs.Lookup(name)
				if !ok {
					return &exitError{Code: 2, Msg: fmt.Sprintf("abcd rules: unknown domain %q", name)}
				}
				return render(cmd.OutOrStdout(), *asJSON, d, func(w io.Writer) {
					fmt.Fprint(w, rules.Render([]rules.ResolvedDomain{d}))
				})
			}
			// Bare: render the full active set.
			active := rs.Active()
			return render(cmd.OutOrStdout(), *asJSON, rulesView{Disabled: rs.Disabled, Domains: active}, func(w io.Writer) {
				if rs.Disabled {
					fmt.Fprintln(w, "abcd rules — disabled (kill switch set in .abcd/rules.json)")
					return
				}
				if out := rules.Render(active); out != "" {
					fmt.Fprint(w, out)
					return
				}
				fmt.Fprintln(w, "abcd rules — no active domains")
			})
		},
	}
}

// newIntentCommand builds the `intent` verb — the front door onto
// internal/core/intent (itd-80). Bare `abcd intent` renders the read-only
// lifecycle status board (never mutates); the `plan` and `link` sub-verbs carry
// the mutations. Usage/lookup failures exit 2.
func newIntentCommand(asJSON *bool) *cobra.Command {
	var intentImpact, intentProductionMode string
	intentCmd := &cobra.Command{
		Use:   "intent [text]",
		Short: "Intent lifecycle; bare invocation is read-only status, quoted text files a draft",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// Quoted text (no sub-verb) files a new draft — the symmetric create path
			// (itd-46). Bare invocation stays read-only status + help (never mutates).
			if len(args) > 0 {
				// Guard: a mistyped subcommand (e.g. `intent lnk itd-5`) must not be
				// swallowed as draft text and filed. Mirrors the capture guard
				// (unrecognized-input-never-writes, iss-29); genuine prose still files.
				if sug, ok := suspectedTypoedSubcommand(cmd, args); ok {
					return &exitError{Code: 2, Msg: fmt.Sprintf(
						"unknown intent subcommand %q; did you mean %q? (nothing created — reword the text if you meant to file a draft)",
						args[0], sug)}
				}
				// Guard: a lone bare word near NO sub-verb fell through the
				// did-you-mean above and was filed as a draft title
				// (iss-2608221328552172). A one-word positional is a sub-verb by
				// shape; only prose is a draft title.
				if loneBareToken(args) {
					if args[0] == "" {
						return &exitError{Code: 2, Msg: "abcd intent: the draft title is empty (nothing created)"}
					}
					return &exitError{Code: 2, Msg: fmt.Sprintf(
						"unknown intent subcommand %q (nothing created — a lone word is read as a sub-verb, never as a draft title; a draft title must contain a space, so write the whole sentence)",
						args[0])}
				}
				return createIntentFromText(cmd, cwd, strings.Join(args, " "), intentImpact, intentProductionMode, *asJSON)
			}
			v, err := intent.Status(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, v, func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent — drafts %d · planned %d · shipped %d · disciplines %d · superseded %d\n",
					v.Buckets[intent.BucketDrafts], v.Buckets[intent.BucketPlanned], v.Buckets[intent.BucketShipped],
					v.Buckets[intent.BucketDisciplines], v.Buckets[intent.BucketSuperseded])
				fmt.Fprintf(w, "  specs: open %d · closed %d\n", v.SpecsOpen, v.SpecsClosed)
				for _, p := range v.Linked {
					// p.Spec is the intent file's spec_id frontmatter, not charset-validated.
					fmt.Fprintf(w, "  link: %s -> %s\n", termsafe.Sanitize(p.Intent), termsafe.Sanitize(p.Spec))
				}
				fmt.Fprint(w, ledgerDecisionRule)
				fmt.Fprint(w, ideateRoutingRule)
			})
		},
	}
	// --impact stamps an optional product judgement onto the seeded draft. It is
	// optional (a draft is "not judged yet"), but when set it is validated and
	// travels unchanged to shipped/, where intent_impact_valid requires it — so the
	// tool's own create->plan->ship path can produce a record that clears the gate.
	intentCmd.Flags().StringVar(&intentImpact, "impact", "", "stamp the draft's product impact: additive|breaking|fix (optional)")
	// --production-mode is a CLOSED CHOICE, refused outright outside the
	// vocabulary — the same shape as --impact and --severity, both of which
	// already stamp machine-read enums. There is no flag for `origin`: it is
	// derived from which command ran (itd-178).
	intentCmd.Flags().StringVar(&intentProductionMode, "production-mode", "", productionModeFlagHelp)

	// new "<text>" — backwards-compatible alias for the sub-verb-free create path
	// (itd-46, lean a): routes to the same create engine and warns on stderr that
	// the `new` sub-verb is deprecated in favour of `abcd intent "<text>"`. The
	// stdout artefact is identical to the quoted-text form.
	intentCmd.AddCommand(&cobra.Command{
		Use:   "new <text>",
		Short: "Deprecated alias for `abcd intent \"<text>\"` (files a draft from the text)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return &exitError{Code: 2, Msg: "abcd intent new: text is required — use `abcd intent \"<text>\"`"}
			}
			fmt.Fprintln(cmd.ErrOrStderr(),
				"WARNING: `abcd intent new` is deprecated; use `abcd intent \"<text>\"` (quoted text is the create signal).")
			return createIntentFromText(cmd, cwd, strings.Join(args, " "), "", "", *asJSON)
		},
	})

	// plan <itd-N> — mint the spec, write both link sides, move drafts -> planned.
	var planProductionMode string
	planCmd := &cobra.Command{
		Use:   "plan <itd-N>",
		Short: "Plan a draft intent (mint its spec, link both sides, move drafts -> planned); on an already-planned intent, stamp its unmarked scope conditions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// The mode belongs to the SPEC this mints; the intent's own stamp was
			// written when its draft was created and is never rewritten.
			mode, err := resolveProductionMode(cwd, planProductionMode)
			if err != nil {
				return err
			}
			res, err := intent.Plan(cwd, args[0], mode)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent plan: " + err.Error()}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				if res.StampOnly {
					// The identity step alone, over a record already planned: say what
					// was done and nothing more, so the line cannot read as a move.
					fmt.Fprintf(w, "abcd intent plan — %s already planned; stamped its unmarked scope conditions\n", res.Intent.ID)
				} else {
					fmt.Fprintf(w, "abcd intent plan — %s drafts -> planned, linked %s\n", res.Intent.ID, res.Spec.ID)
				}
				fmt.Fprintf(w, "  intent: %s\n", termsafe.Sanitize(res.Intent.Path))
				if res.Spec.Path != "" {
					fmt.Fprintf(w, "  spec:   %s\n", termsafe.Sanitize(res.Spec.Path))
				}
				if res.ConditionsStamped > 0 {
					fmt.Fprintf(w, "  scope-condition identities stamped: %d\n", res.ConditionsStamped)
				}
			})
		},
	}
	planCmd.Flags().StringVar(&planProductionMode, "production-mode", "", productionModeFlagHelp)
	intentCmd.AddCommand(planCmd)

	// ready <itd-N> — the read-only implement-readiness gate. Exit codes are the
	// machine seam an autonomous run gates on: 0 ready, 1 not ready (the rendered
	// report is the output, empty message — the embark-conflicts precedent), 2
	// structural fault.
	//
	// --grounds is wired here as TWO calls rather than as a parameter of the gate:
	// intent.Ready is documented and tested as a reporter that never mutates the
	// store, and that contract is worth more than flag placement (spc-57). So the
	// flag records first through intent.RecordGrounds and then reports through the
	// unchanged Ready. A failed write is a STRUCTURAL fault (exit 2), never the
	// gate's own exit 1: a caller that maps 1 to SKIP must not read a lost write
	// as a skipped item.
	var readyGrounds string
	readyCmd := &cobra.Command{
		Use:   "ready <itd-N> [--grounds \"" + grounds.UsageSpelling() + ": <conjecture>\"]",
		Short: "Report whether an intent is ready to implement (planned + AC + claims + written spec + recorded grounds); exit 1 when not",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			var recorded *intent.GroundsResult
			if strings.TrimSpace(readyGrounds) != "" {
				g, perr := grounds.Parse(readyGrounds)
				if perr != nil {
					return &exitError{Code: 2, Msg: "abcd intent ready: " + perr.Error() + " (nothing recorded)"}
				}
				rec, rerr := intent.RecordGrounds(cwd, args[0], g)
				if rerr != nil {
					return &exitError{Code: 2, Msg: "abcd intent ready: " + rerr.Error()}
				}
				recorded = &rec
				// The receipt is emitted HERE, before the gate is consulted. A
				// structural fault in the gate exits 2 carrying no result at all, and
				// a caller who never learns the write happened retries and appends a
				// second entry to a record that is append-only by design
				// (iss-2608300930057882). In --json the receipt goes to stderr, so
				// stdout stays a single machine payload and the news still survives a
				// fault that emits no envelope.
				emitGroundsReceipt(cmd, *asJSON, rec)
			}
			res, err := intent.Ready(cwd, args[0])
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent ready: " + err.Error()}
			}
			// With the flag, the payload is an envelope carrying BOTH halves: the
			// write the flag performed and the report the verb makes. Without it the
			// payload is the readiness result unchanged, so no existing consumer's
			// shape moves under it.
			var payload any = res
			if recorded != nil {
				payload = readyGroundsEnvelope{Grounds: recorded, Ready: res}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, payload, func(w io.Writer) {
				verdict := "READY"
				if !res.Ready {
					verdict = "NOT READY"
				}
				fmt.Fprintf(w, "abcd intent ready — %s %s (%s)\n", termsafe.Sanitize(res.IntentID), verdict, termsafe.Sanitize(res.Bucket))
				for _, c := range res.Checks {
					mark := "[ ok ]"
					if !c.OK {
						mark = "[fail]"
					}
					// Detail/remedy interpolate frontmatter values, not charset-validated.
					fmt.Fprintf(w, "  %s %s: %s\n", mark, c.Name, termsafe.Sanitize(c.Detail))
					if c.Remedy != "" {
						fmt.Fprintf(w, "         remedy: %s\n", termsafe.Sanitize(c.Remedy))
					}
				}
				// The scope conditions with their stamped identities: what a later
				// fidelity disposition attaches to, shown so a human reading the
				// report sees the same claims the machine seam carries.
				for _, cond := range res.Conditions {
					id := cond.ID
					if id == "" {
						id = "unstamped"
					}
					// The condition's prose is a human's, not a validated charset.
					fmt.Fprintf(w, "  cond %d [%s] %s\n", cond.Ordinal, termsafe.Sanitize(id), termsafe.Sanitize(cond.Text))
				}
			}); rerr != nil {
				return rerr
			}
			if !res.Ready {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	readyCmd.Flags().StringVar(&readyGrounds, "grounds", "",
		"record the conjecture behind this gate decision: \""+grounds.UsageSpelling()+": <what is expected, and what would show it wrong>\"")
	intentCmd.AddCommand(readyCmd)

	// link <itd-N> <spc-N> — retroactively set spec_id on a planned intent.
	intentCmd.AddCommand(&cobra.Command{
		Use:   "link <itd-N> <spc-N>",
		Short: "Link a planned intent to an existing spec (writes the intent's spec_id)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := intent.Link(cwd, args[0], args[1])
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent link: " + err.Error()}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent link — %s -> %s\n  intent: %s\n", res.Intent.ID, res.Spec.ID, termsafe.Sanitize(res.Intent.Path))
			})
		},
	})

	intentCmd.AddCommand(newIntentAuditCommand(asJSON))
	return intentCmd
}

// ledgerDecisionRule is the one-line capture-vs-intent decision rule shown in
// both ledgers' bare-form help (itd-46 AC5), so a user knows which ledger to reach
// for. It stays host-agnostic (binary command forms, no plugin/tool names).
const ledgerDecisionRule = "  which ledger? half-formed observation, question, or nitpick -> `abcd capture \"…\"`; a user-facing change you want to ship -> `abcd intent \"…\"`\n"

// ideateRoutingRule sits beside the ledger rule and names the optional third
// route: a big, unproven idea can go through the admission gauntlet first
// (itd-104 AC1).
//
// It is a POINTER, and the wording is load-bearing. Ideate is never a
// precondition for capture or intent, so this line offers a route and never
// implies one is missed — no "should", no "first", no warning when it is skipped.
// The one line of capture friction the ledgers promise stays one line.
const ideateRoutingRule = "  a big, unproven idea? `abcd ideate` runs the optional admission gauntlet and records the verdict either way\n"

// createIntentFromText is the shared quoted-text create path behind both
// `abcd intent "<text>"` and the deprecated `abcd intent new "<text>"` alias: it
// files a new draft via intent.CreateFromText and renders the created record. The
// engine refuses empty/whitespace text and mints the id under the store lock, so
// resolveProductionMode turns the --production-mode flag into the value a MINT
// path stamps: the operator's declared choice, or the repo's own declared
// default from the identity pin (itd-91's seam), which is hand-written when the
// pin declares none.
//
// The value is validated HERE as well as in the core, because a surface refusal
// costs nothing and names the closed set before any id is minted — and because
// "the operator supplies a closed choice, never free text" is the property
// itd-178 rests on, which belongs at the door the operator types at.
//
// It is deliberately NOT used by the ledger transitions: a resolve or wontfix
// that declares no mode must leave the record's existing stamp alone, and
// defaulting there would silently overwrite it.
func resolveProductionMode(cwd, flag string) (string, error) {
	if flag != "" {
		m, err := provenance.ParseMode(flag)
		if err != nil {
			return "", &exitError{Code: 2, Msg: "abcd: " + err.Error() + " (nothing written)"}
		}
		return string(m), nil
	}
	m, err := identity.DeclaredProductionMode(cwd)
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd: " + err.Error() + " (nothing written)"}
	}
	return string(m), nil
}

// productionModeFlagHelp is the one help string every verb carrying the flag
// shows, composed from the vocabulary so it cannot drift from the enum.
var productionModeFlagHelp = "how this record's text was produced: " + provenance.ModeList() +
	" (default: the repo's declared mode, else " + string(provenance.DefaultMode) + ")"

// this surface stays a thin marshaller.
func createIntentFromText(cmd *cobra.Command, cwd, text, impact, productionMode string, asJSON bool) error {
	mode, err := resolveProductionMode(cwd, productionMode)
	if err != nil {
		return err
	}
	it, err := intent.CreateFromText(cwd, text, impact, mode)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd intent: " + err.Error()}
	}
	return render(cmd.OutOrStdout(), asJSON, it, func(w io.Writer) {
		fmt.Fprintf(w, "created %s (%s) — %s\n", it.ID, it.Bucket, termsafe.Sanitize(it.Path))
	})
}

// newIntentAuditCommand builds `abcd intent audit`: `ingest --verdict-json`
// applies a host-produced intent-audit verdict to the shipped intent's Audit
// Notes (fail-closed: ingested | dead_letter | noop); bare `audit <itd-N>`
// re-emits the OWED stub + ephemeral request for a shipped intent.
func newIntentAuditCommand(asJSON *bool) *cobra.Command {
	auditCmd := &cobra.Command{
		Use:   "audit [<itd-N>]",
		Short: "Intent audit (promise vs delivered): re-emit a shipped intent's request, or ingest a verdict",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := intent.ReEmitAudit(cwd, args[0])
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent audit: " + err.Error()}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent audit — %s %s (receipt %s)\n  request: %s\n",
					res.IntentID, res.Status, res.ReceiptID, res.RequestPath)
			})
		},
	}

	var verdictJSON string
	ingestCmd := &cobra.Command{
		Use:   "ingest --verdict-json <path>",
		Short: "Ingest an intent-audit verdict JSON into the shipped intent's Audit Notes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if verdictJSON == "" {
				return &exitError{Code: 2, Msg: "abcd intent audit ingest: --verdict-json <path> is required"}
			}
			res, err := intent.IngestVerdict(cwd, verdictJSON)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd intent audit ingest: " + err.Error()}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd intent audit ingest — %s (receipt %s, intent %s)\n", res.Status, res.ReceiptID, res.IntentID)
				switch res.Status {
				case "ingested":
					fmt.Fprintf(w, "  criteria %d: MET %d · MET_WITH_CONCERNS %d · NOT_MET %d · INCONCLUSIVE %d\n",
						res.Criteria, res.Met, res.MetWithConcern, res.NotMet, res.Inconclusive)
					// Only an intent that records scope conditions has a disposition
					// split to report; a conditionless one has no surface here.
					if res.Conditions > 0 {
						fmt.Fprintf(w, "  scope conditions %d: survived %d · narrowed %d · falsified %d · untested %d\n",
							res.Conditions, res.Survived, res.Narrowed, res.Falsified, res.Untested)
					}
				case "dead_letter":
					fmt.Fprintf(w, "  DEAD_LETTER: %s\n  raw payload: %s\n", res.Reason, res.DeadLetterPath)
				}
			})
		},
	}
	ingestCmd.Flags().StringVar(&verdictJSON, "verdict-json", "", "path to the intent-audit verdict JSON")
	auditCmd.AddCommand(ingestCmd)
	return auditCmd
}

// specStatusView is the machine-readable envelope for bare `abcd spec`: the
// open/closed counts and every discovered spec record.
type specStatusView struct {
	Open   int         `json:"open"`
	Closed int         `json:"closed"`
	Specs  []spec.Spec `json:"specs"`
}

// newSpecCommand builds the `spec` verb — the front door onto internal/core/spec
// (itd-80). Bare `abcd spec` renders the read-only spec-store status; the `close`
// sub-verb closes a spec AND reconciles its linked intent (planned -> shipped)
// via intent.Reconcile, so one command completes the lifecycle transition.
func newSpecCommand(asJSON *bool) *cobra.Command {
	specCmd := &cobra.Command{
		Use:   "spec",
		Short: "Native spec store; bare invocation is read-only status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			store, err := spec.Load(cwd)
			if err != nil {
				return err
			}
			view := specStatusView{Specs: store.Specs}
			for _, sp := range store.Specs {
				if sp.Status == spec.StatusClosed {
					view.Closed++
				} else {
					view.Open++
				}
			}
			return render(cmd.OutOrStdout(), *asJSON, view, func(w io.Writer) {
				fmt.Fprintf(w, "abcd spec — open %d · closed %d\n", view.Open, view.Closed)
				for _, sp := range store.Specs {
					fmt.Fprintf(w, "  %s  %s  %s  (%s)\n", termsafe.Sanitize(sp.ID), sp.Status, termsafe.Sanitize(sp.Slug), termsafe.Sanitize(sp.Intent))
				}
			})
		},
	}

	// close <spc-N> — closes the spec AND reconciles the linked intent
	// (planned -> shipped). Fail-closed and idempotent (see intent.Reconcile).
	//
	// --impact is the judgement the shipped intent carries. It is optional
	// because a record that already declares one needs nothing here, and it
	// exists because shipped/ is the one bucket intent_impact_valid requires an
	// impact in: without it the ship verb could only either move an impactless
	// record into the bucket that refuses it or refuse forever, with no way for
	// the tool to supply the missing judgement (iss-126).
	var closeImpact string
	closeCmd := &cobra.Command{
		Use:   "close <spc-N>",
		Short: "Close a spec (open/ -> closed/) and ship its linked intent (planned/ -> shipped/)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := intent.Reconcile(cwd, args[0], closeImpact)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd spec close: " + err.Error()}
			}
			// The fidelity-review emit is report-only: a failure does NOT fail the
			// close (the intent already shipped), but it is surfaced loudly on stderr.
			if res.AuditEmitError != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: abcd spec close — fidelity-review emit failed for %s (intent shipped anyway): %s\n", res.Intent.ID, res.AuditEmitError)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd spec close — %s open -> closed\n  %s\n", res.Spec.ID, termsafe.Sanitize(res.Spec.Path))
				if res.IntentMoved {
					fmt.Fprintf(w, "  reconciled intent %s: %s -> %s\n", res.Intent.ID, res.From, res.To)
				} else {
					fmt.Fprintf(w, "  intent %s already %s (no move)\n", res.Intent.ID, res.To)
				}
				if res.ReceiptID != "" {
					fmt.Fprintf(w, "  fidelity review OWED: receipt %s\n", res.ReceiptID)
				}
			})
		},
	}
	closeCmd.Flags().StringVar(&closeImpact, "impact", "", "product impact to stamp on an intent that declares none: additive|breaking|fix (an intent may not be internal)")
	specCmd.AddCommand(closeCmd)

	return specCmd
}

// newAhoyCommand builds the `ahoy` sub-tree. Bare `ahoy` runs the read-only
// detection pass (abcd's convention: bare invocation never mutates); the
// install/uninstall/doctor/dry-run sub-verbs are thin consumers of the same
// core engine (detect -> contract -> apply), matching 04-surfaces/01-ahoy.md.
func newAhoyCommand(asJSON *bool) *cobra.Command {
	ahoyCmd := &cobra.Command{
		Use:   "ahoy",
		Short: "Install/update abcd in this repo; bare invocation is read-only status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := ahoy.DryRun(cwd)
			if err != nil {
				return err
			}
			// Vintage + staleness from the shared comparator (itd-111): the same
			// source `abcd version` and the session-start notice read. Computed
			// once and carried in both the JSON and the text render.
			vin := ahoy.Vintage(cwd)
			out := ahoyOutput{DetectionResult: res, Vintage: vin.DisplayVintage(), Staleness: vin.Staleness()}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "abcd ahoy — %s\n", res.FolderKind)
				fmt.Fprintf(w, "  plugin root: %s\n", res.PluginRootStatus)
				fmt.Fprintf(w, "  root sha:    %s\n", res.RootSHA)
				if mode, _ := res.Signals["install_mode"].(string); mode != "" {
					fmt.Fprintf(w, "  install:     %s\n", mode)
				}
				fmt.Fprintf(w, "  vintage:     %s\n", out.Vintage)
				fmt.Fprintf(w, "  staleness:   %s\n", out.Staleness)
				// The citation baseline's coverage and age, present only in a repo
				// that has armed the citation gate. The line embeds counts and a
				// date derived from repo content, so it is sanitised.
				if citations, _ := res.Signals["citations"].(string); citations != "" {
					fmt.Fprintf(w, "  citations:   %s\n", termsafe.Sanitize(citations))
				}
				fmt.Fprintf(w, "  gaps:        %d\n", len(res.Gaps))
				if res.FolderKind != ahoy.UnmanagedFolder {
					fmt.Fprintf(w, "  guard:       %s\n", guardHealthLine(*res.Guard))
					for i, line := range banlistHealthLines(*res.Banlist) {
						label := "  banlist:     "
						if i > 0 {
							label = "               "
						}
						fmt.Fprintf(w, "%s%s\n", label, line)
					}
					fmt.Fprintf(w, "               reach: %s\n", res.Banlist.Reach)
				}
				// Classification is read-only; the human report names the
				// next step per folder kind (itd-40 AC2/AC3).
				switch res.FolderKind {
				case ahoy.UnmanagedRepo:
					fmt.Fprintf(w, "  unmanaged git repo — run `/abcd:ahoy install` to adopt it.\n")
				case ahoy.UnmanagedFolder:
					fmt.Fprintf(w, "  not a git repository — nothing to act on.\n")
				}
			})
		},
	}

	// install
	var (
		yes           bool
		adopt         bool
		refuseAdopt   bool
		dev           bool
		attribution   bool
		allowStale    bool
		binDir        string
		visibility    string
		docsTarget    string
		oracleBackend string
		scanDeep      string
	)
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install or update abcd in this repo (idempotent)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			opts, err := installOptionsFromFlags(cmd, yes, adopt, refuseAdopt, dev, attribution, allowStale, binDir, visibility, docsTarget, oracleBackend, scanDeep)
			if err != nil {
				return err
			}
			res, err := ahoy.Install(cwd, opts, newPrompter(cmd))
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd ahoy install — %s\n", res.Status)
				for _, c := range res.Changes {
					fmt.Fprintf(w, "  changed: %s\n", c)
				}
				for _, p := range res.Writes {
					fmt.Fprintf(w, "  wrote: %s\n", p)
				}
				// A refusal is louder than an unexplained missing write: it says
				// what abcd did not do and why (a dangling PATH entry it declined
				// to create, a directory it could not write).
				for _, n := range res.Notes {
					fmt.Fprintf(w, "  note: %s\n", n)
				}
				if len(res.DeclinedCategories) > 0 {
					fmt.Fprintf(w, "  declined: %s\n", strings.Join(res.DeclinedCategories, ", "))
				}
				if len(res.Remaining) > 0 {
					fmt.Fprintf(w, "  remaining gaps: %s\n", strings.Join(res.Remaining, ", "))
				}
				// --yes approves every category but never writes the identity
				// pin, so say which optional work it left and how to apply it.
				if len(res.OptionalSkipped) > 0 {
					fmt.Fprintf(w, "  optional, not covered by --yes: %s\n", strings.Join(res.OptionalSkipped, ", "))
					fmt.Fprint(w, "    the pin records the current git identity, so it is only written against an answered prompt:\n")
					fmt.Fprint(w, "    run `abcd ahoy install` (no --yes) and answer y at each prompt — non-interactively, `yes | abcd ahoy install`\n")
				}
			})
		},
	}
	// No backquotes in a flag's usage string: cobra reads the first backquoted
	// word as the flag's argument placeholder, so a quoted answer would render
	// this boolean as "--yes y" in the help and the generated reference.
	installCmd.Flags().BoolVar(&yes, "yes", false, "approve every resolvable change category without prompting; excludes the optional git-identity pin, which needs an answered prompt (run without --yes, or answer every prompt with: yes | abcd ahoy install)")
	installCmd.Flags().BoolVar(&adopt, "adopt", false, "adopt an unmanaged repo without prompting")
	installCmd.Flags().BoolVar(&refuseAdopt, "refuse-adopt", false, "decline to adopt an unmanaged repo")
	installCmd.Flags().BoolVar(&dev, "dev", false, "track-latest dogfood mode: the PATH entry rebuilds from the source tip on every call instead of pinning the built binary")
	installCmd.Flags().BoolVar(&attribution, "attribution", false, "opt this repo into the committed prepare-commit-msg prompt asking every commit to declare whether a tool assisted it; the choice is recorded, so a later install without the flag keeps the hook")
	installCmd.Flags().BoolVar(&allowStale, "allow-stale-binary", false, "proceed even when the running binary is stale against its source tip or its vintage cannot be determined; the default is to refuse before any write and name the rebuild fix")
	installCmd.Flags().StringVar(&binDir, "bin-dir", "", "directory for the PATH entry (default ~/.local/bin, or an existing abcd install adopted in place); fails when it is not writable — abcd never escalates privileges")
	installCmd.Flags().StringVar(&visibility, "visibility", "", "repo visibility: private | public")
	installCmd.Flags().StringVar(&docsTarget, "docs-target", "", "marker target: claude_md | agents_md | both | skip")
	installCmd.Flags().StringVar(&oracleBackend, "oracle-backend", "", "oracle backend: host-delegated | native | cli | api | mcp")
	installCmd.Flags().StringVar(&scanDeep, "scan-deep", "", "enable deep scan: true | false")
	ahoyCmd.AddCommand(installCmd)

	// uninstall
	var uninstallBinDir string
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the marker block, abcd's owned PATH copy, and its provenance record (leaves .abcd/ intact)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			receipt, err := ahoy.Uninstall(cwd, uninstallBinDir)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, receipt, func(w io.Writer) {
				fmt.Fprintf(w, "abcd ahoy uninstall\n")
				fmt.Fprintf(w, "  marker removed: %v\n", receipt.Marker.Removed)
				fmt.Fprintf(w, "  symlink: %s\n", symlinkNote(receipt))
			})
		},
	}
	uninstallCmd.Flags().StringVar(&uninstallBinDir, "bin-dir", "", "directory holding the PATH entry to remove; needed only when it was installed with --bin-dir into a directory that is not on PATH")
	ahoyCmd.AddCommand(uninstallCmd)

	// doctor
	ahoyCmd.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Report every gap read-only, including user-scope state (never mutates)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			report, err := ahoy.Doctor(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, report, func(w io.Writer) {
				fmt.Fprintf(w, "abcd ahoy doctor — %s\n", report.Detection.FolderKind)
				fmt.Fprintf(w, "  detection gaps: %d\n", len(report.Detection.Gaps))
				fmt.Fprintf(w, "  audit gaps:     %d\n", len(report.AuditGaps))
				// A required gap that is NOT resolvable is the one no later
				// `ahoy install` will ever clear — a config.json abcd refuses to
				// touch until a human repairs it is the case that matters. Inside
				// a count it is indistinguishable from work the next install does
				// for you, so name each one and what is wrong with it. The detail
				// quotes a parser's own error over the user's file, so it is
				// sanitised like any other input rendered to a terminal.
				for _, g := range report.Detection.Gaps {
					if g.Required && !g.Resolvable {
						fmt.Fprintf(w, "  diagnostic:     %s — %s\n",
							termsafe.Sanitize(g.ID), termsafe.Sanitize(g.Detail))
					}
				}
			})
		},
	})

	// dry-run
	ahoyCmd.AddCommand(&cobra.Command{
		Use:   "dry-run",
		Short: "Render the detection-result JSON envelope; never mutates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := ahoy.DryRun(cwd)
			if err != nil {
				return err
			}
			// dry-run always emits the canonical JSON envelope (spc-16 T1).
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		},
	})

	ahoyCmd.AddCommand(newAhoyRemoteCommand(asJSON))

	// identity-check — the iss-62 gate's canonical, testable entrypoint. Exits
	// non-zero when the commit identity diverges from the committed pin, so a
	// pre-commit hook (or CI) can fail closed. A match, or an un-pinned repo,
	// exits zero.
	ahoyCmd.AddCommand(&cobra.Command{
		Use:   "identity-check",
		Short: "Exit non-zero if the git commit identity does not match .abcd/config/identity.json",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := identity.Check(cwd)
			if err != nil {
				return err
			}
			if res.Blocks() {
				return fmt.Errorf("%s\n  fix: git config user.name %q && git config user.email %q",
					res.Reason, res.Pin.Name, res.Pin.Email)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "identity ok (%s)\n", res.Status)
			return nil
		},
	})

	return ahoyCmd
}

// newAhoyRemoteCommand builds `ahoy remote` — abcd's remote config surface for a
// managed repo (itd-153). Bare invocation READS: it reports GitHub's native
// secret-scanning toggles and what an apply would change, and contacts nothing
// else. `apply` is the write, and it is the only thing in abcd that mutates state
// outside this machine, so it is a verb a person types rather than a step any
// other command performs.
func newAhoyRemoteCommand(asJSON *bool) *cobra.Command {
	remoteCmd := &cobra.Command{
		Use:   "remote",
		Short: "Report the managed repo's GitHub secret-scanning settings (read-only); apply enables them",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := ahoy.RemoteRead(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderRemoteResult(w, "abcd ahoy remote", res)
			})
		},
	}
	var remoteYes bool
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Enable GitHub secret scanning and push protection on this managed repo, and mirror the desired state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// adr-44 / invariant 10: the remote write is CONFIRMED as well as
			// invoked. An unanswered run declines, so a script that pipes nothing
			// changes nothing; --yes is the explicit way to say yes in advance.
			p := newPrompter(cmd)
			if remoteYes {
				p = alwaysConfirm{}
			}
			res, err := ahoy.RemoteApply(cwd, p)
			if err != nil {
				return err
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderRemoteResult(w, "abcd ahoy remote apply", res)
			}); rerr != nil {
				return rerr
			}
			// A change that did not happen exits non-zero. `aborted` is on this list
			// deliberately: a non-interactive run without --yes reads EOF, declines,
			// and would otherwise print "aborted" and exit 0 — which to a script is
			// indistinguishable from a write that landed. `opted_out` is the one
			// non-change that exits clean, because leaving the repo alone IS what the
			// repo asked for. The reason is on stdout above either way.
			if res.Status == "refused" || res.Status == "aborted" {
				return &exitError{Code: 1, Msg: "abcd ahoy remote apply: " + res.Status +
					" — nothing was changed (the reason is in the result's notes above)"}
			}
			return nil
		},
	}
	applyCmd.Flags().BoolVar(&remoteYes, "yes", false, "confirm the remote change without being asked; without it an unanswered run declines and changes nothing")
	remoteCmd.AddCommand(applyCmd)
	return remoteCmd
}

// alwaysConfirm is the --yes answer to the remote apply's confirmation. It lives
// in the front door rather than beside RefusingPrompter in the core, because a
// core type that says yes to everything is a seam any future caller could reach
// for by accident — and the one thing this confirmation guards is the only abcd
// operation that changes state outside the machine.
type alwaysConfirm struct{}

func (alwaysConfirm) Confirm(string) bool                            { return true }
func (alwaysConfirm) Prompt(_ string, _ []string, def string) string { return def }

// renderRemoteResult prints one remote result. Every path is sanitised: the repo
// name comes from a git remote and the notes carry a subprocess's own stderr, both
// of which are input rather than abcd's own words.
func renderRemoteResult(w io.Writer, verb string, res ahoy.RemoteResult) {
	fmt.Fprintf(w, "%s — %s\n", verb, res.Status)
	if res.Repo != "" {
		fmt.Fprintf(w, "  repo:     %s\n", termsafe.Sanitize(res.Repo))
	}
	fmt.Fprintf(w, "  observed: secret scanning %s, push protection %s\n",
		termsafe.Sanitize(remoteStatusWord(res.Observed.SecretScanning)),
		termsafe.Sanitize(remoteStatusWord(res.Observed.PushProtection)))
	for _, c := range res.Changes {
		fmt.Fprintf(w, "  change:   %s\n", termsafe.Sanitize(c))
	}
	for _, p := range res.Writes {
		fmt.Fprintf(w, "  wrote:    %s\n", termsafe.Sanitize(p))
	}
	// A refusal that showed up only as an unchanged toggle would read as a silent
	// failure, so the reason is printed whatever the status.
	for _, n := range res.Notes {
		fmt.Fprintf(w, "  note:     %s\n", termsafe.Sanitize(n))
	}
}

// remoteStatusWord words a toggle abcd could not read. "unknown" and "disabled"
// need opposite responses, so they are never collapsed into one word.
func remoteStatusWord(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// installOptionsFromFlags validates the install flags and builds InstallOptions.
// Only explicitly-set value flags become overrides; unset values fall through to
// the prompter (interactive) or its default (non-interactive).
func installOptionsFromFlags(cmd *cobra.Command, yes, adopt, refuseAdopt, dev, attribution, allowStale bool, binDir, visibility, docsTarget, oracleBackend, scanDeep string) (ahoy.InstallOptions, error) {
	opts := ahoy.InstallOptions{Yes: yes, Dev: dev, Attribution: attribution, BinDir: binDir, AllowStaleBinary: allowStale}
	if adopt && refuseAdopt {
		return opts, fmt.Errorf("abcd ahoy install: --adopt and --refuse-adopt are mutually exclusive")
	}
	switch {
	case adopt:
		v := true
		opts.Adopt = &v
	case refuseAdopt:
		v := false
		opts.Adopt = &v
	}
	overrides := map[string]string{}
	set := func(key, val string, allowed []string) error {
		if !cmd.Flags().Changed(flagName(key)) {
			return nil
		}
		if len(allowed) > 0 && !contains(allowed, val) {
			return fmt.Errorf("abcd ahoy install: --%s must be one of %s", flagName(key), strings.Join(allowed, " | "))
		}
		overrides[key] = val
		return nil
	}
	if err := set("visibility", visibility, []string{"private", "public"}); err != nil {
		return opts, err
	}
	if err := set("docs_target", docsTarget, []string{"claude_md", "agents_md", "both", "skip"}); err != nil {
		return opts, err
	}
	if err := set("oracle_backend", oracleBackend, []string{"host-delegated", "native", "cli", "api", "mcp"}); err != nil {
		return opts, err
	}
	if err := set("scan_deep", scanDeep, []string{"true", "false"}); err != nil {
		return opts, err
	}
	if len(overrides) > 0 {
		opts.ValueOverrides = overrides
	}
	return opts, nil
}

// flagName maps an override key to its CLI flag name (underscore -> dash).
func flagName(key string) string { return strings.ReplaceAll(key, "_", "-") }

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func symlinkNote(r ahoy.UninstallReceipt) string {
	if r.Symlink.Removed {
		return "removed " + r.Symlink.Target
	}
	return r.Symlink.Note
}

// newPrompter returns the stdin-reading prompter. On a terminal it is the
// interactive path, unchanged. When stdin is NOT a terminal the same prompter
// reads the piped answers (iss-167): `yes | abcd ahoy install` is what a host
// agent reaches for, and a TTY-only prompt turned every such answer into a
// decline, so the interactive path could not be driven at all. One answer
// answers one question, and the questions come in a fixed order
// (ahoy.categoryPromptOrder), so a piped stream lines up with them.
//
// The safe default survives: answers that run out — an empty pipe, a closed
// stdin, /dev/null — read as EOF, and EOF declines every confirm and takes the
// default for every prompt, exactly as the refusing prompter did.
func newPrompter(cmd *cobra.Command) ahoy.Prompter {
	in := cmd.InOrStdin()
	if in == nil {
		return ahoy.RefusingPrompter{}
	}
	p := &stdinPrompter{r: bufio.NewReader(in), w: cmd.ErrOrStderr()}
	if f, ok := in.(*os.File); ok && term.IsTerminal(f) {
		p.tty = true
	}
	return p
}

// stdinPrompter is the Prompter: it reads answers from stdin, whether a human
// types them or a caller pipes them in.
type stdinPrompter struct {
	r *bufio.Reader
	w io.Writer
	// tty records that a human is answering: the terminal echoes their own
	// typing, so the prompter must not echo it a second time. Off a terminal
	// nothing echoes, so the prompter writes the answer it read — a piped run
	// leaves a transcript of what was asked and what it was answered, instead
	// of a column of unanswered-looking questions.
	tty bool
}

// echo reports the answer read off a non-terminal stdin. The bytes come from
// the caller, so they are sanitised before reaching the terminal.
func (p *stdinPrompter) echo(answer string) {
	if p.tty {
		return
	}
	if answer == "" {
		answer = "<no answer>"
	}
	fmt.Fprintf(p.w, "%s\n", termsafe.Sanitize(answer))
}

func (p *stdinPrompter) Confirm(question string) bool {
	fmt.Fprintf(p.w, "%s [y/N] ", question)
	line, _ := p.r.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	p.echo(line)
	return line == "y" || line == "yes"
}

func (p *stdinPrompter) Prompt(key string, choices []string, def string) string {
	fmt.Fprintf(p.w, "%s (%s) [%s]: ", key, strings.Join(choices, "/"), def)
	line, _ := p.r.ReadString('\n')
	line = strings.TrimSpace(line)
	p.echo(line)
	if line == "" {
		return def
	}
	return line
}

// captureLedgerRoot is the shared front-door step for every `capture` verb: the
// checkout whose ledger the verb acts on, resolved from the working directory
// rather than taken to BE it.
//
// Every verb below used to build its request with `RepoRoot: cwd`, which the
// core reads as an explicit root and therefore never resolves. A verb run from a
// subdirectory then addressed a ledger that was not there — silently in both
// directions: a read reported open 0 against a populated checkout, and a write
// minted a second ledger under the subdirectory and reported success with a
// repo-relative path nothing about which looked unusual. Records filed that way
// are invisible to every gate that reads the real ledger, to the release cut,
// and to whoever filed them (iss-2609090951291524). capture.LedgerRoot owns the
// resolution and owns the refusals; this is the door that calls it, the way the
// reading verbs already resolve their toplevel before they call.
//
// The stray-ledger note rides the same step, on stderr, for the reason
// historyStore prints its own diagnostics there: a resolution that silently
// steps over a store the defect already minted would leave those records where
// nothing will ever look again. It REPORTS and moves nothing — no verb here
// migrates a record, and the bare board stays read-only.
func captureLedgerRoot(cmd *cobra.Command) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := capture.LedgerRoot(cwd)
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd capture: " + err.Error() + " (nothing read, nothing written)"}
	}
	for _, note := range strayLedgerNotes(cwd, root) {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd capture: %s\n", termsafe.Sanitize(note))
	}
	return root, nil
}

// strayLedgerNotes names a ledger sitting BELOW the checkout root, between the
// caller and it — the exact deposit iss-2609090951291524 left behind, and the
// one place a front door can name with certainty and at no cost.
//
// The walk is bounded to the chain from cwd up to (not including) the root, so
// it fires for the person who created the stray store, on their next capture
// from the directory that created it. A sweep of the whole checkout would find
// stray stores this walk cannot see; that belongs to a lint rule that reads the
// tree, not to a verb that has one directory to look at.
//
// The note STATES what is there and never accuses: a checkout can hold a ledger
// below its root on purpose (this repository's own cold-reading eval fixtures
// do), and only the person standing in it can tell a fixture from an orphan.
// What the note owes them is the fact that two stores exist and which one the
// verb just used.
func strayLedgerNotes(cwd, root string) []string {
	dir, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		dir = filepath.Clean(cwd)
	}
	top, err := filepath.EvalSymlinks(root)
	if err != nil {
		top = filepath.Clean(root)
	}
	var notes []string
	for dir != top {
		ledger := filepath.Join(dir, filepath.FromSlash(capture.LedgerRelPath))
		if fi, statErr := os.Stat(ledger); statErr == nil && fi.IsDir() {
			rel, relErr := filepath.Rel(top, ledger)
			if relErr != nil {
				rel = ledger
			}
			notes = append(notes, "a ledger also exists below the checkout root, at "+filepath.ToSlash(rel)+
				" — this verb addressed the checkout's ledger and left that one untouched; records filed there reach no gate and no release cut")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return notes
}

// newCaptureCommand builds the `capture` sub-tree — the write side of the issue
// ledger. Bare `capture` renders read-only status; a free-text positional
// appends an issue; list/resolve/wontfix/promote are thin consumers of capture
// core.
func newCaptureCommand(asJSON *bool) *cobra.Command {
	var severity, category, source, slug, foundDuring, foundAt, lapsedAt, blockedBy, captureProductionMode string

	captureCmd := &cobra.Command{
		Use:   "capture [text]",
		Short: "Capture issues to the ledger; bare invocation is read-only status",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			// Bare invocation: read-only status render (never mutates).
			if len(args) == 0 {
				st, err := capture.Status(capture.StatusRequest{RepoRoot: repoRoot})
				if err != nil {
					return err
				}
				// The outstanding-readings report rides the same board. It is the
				// SAME function the lint rule calls, not a second scan: an item
				// nobody has answered has no state to sit in, and two readings of
				// that one question would be two answers to it.
				board, err := captureBoardOf(repoRoot, st)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, board, func(w io.Writer) {
					fmt.Fprintf(w, "abcd capture — open %d · resolved %d · wontfix %d\n",
						st.OpenCount, st.ResolvedCount, st.WontfixCount)
					if len(st.RecentOpen) > 0 {
						fmt.Fprintf(w, "recent open:\n")
						for _, iss := range st.RecentOpen {
							fmt.Fprintf(w, "  %s  %s  %s%s\n", iss.ID, iss.Severity, iss.Slug, blockedNote(iss))
						}
					}
					// The skipped roster, exactly as `capture list` renders it
					// (iss-2608261437041050): a record the reader refuses is counted
					// by none of the three totals above, so a board that printed the
					// totals alone would under-report the ledger and say nothing about
					// the records it dropped. Path and Error echo the malformed file's
					// own name and bytes, so both are sanitised before the terminal.
					for _, sk := range st.Skipped {
						fmt.Fprintf(w, "  skipped %s: %s\n", termsafe.Sanitize(sk.Path), termsafe.Sanitize(sk.Error))
					}
					// Beside the skipped roster, and for the same reason it is
					// there: a record the board does not name is one nobody is
					// counting. An unanswered reading item is reported as
					// outstanding because no state means "already covered", and an
					// open hold renders WITH its exit condition, which is the only
					// thing that distinguishes a hold from a parking space.
					for _, o := range board.Outstanding.Undispositioned {
						fmt.Fprintf(w, "  outstanding %s (run %s) — no disposition\n",
							termsafe.Sanitize(o.Item), termsafe.Sanitize(o.Run))
					}
					// A widening proposal is answered by an admission carrying its
					// grounds or by a decline, so it gets its own line: the
					// disposition-only line above would name the wrong remedy.
					for _, o := range board.Outstanding.Unadmitted {
						fmt.Fprintf(w, "  unadmitted %s (run %s) — a widening proposal with neither an admission nor a decline\n",
							termsafe.Sanitize(o.Item), termsafe.Sanitize(o.Run))
					}
					// More than one standing answer is named in full, never resolved
					// by picking one: which is in force is a judgement the ledger
					// does not contain.
					for _, c := range board.Outstanding.Contested {
						fmt.Fprintf(w, "  contested %s (run %s) — %d standing answers: %s\n",
							termsafe.Sanitize(c.Item), termsafe.Sanitize(c.Run),
							len(c.Standing), termsafe.Sanitize(strings.Join(c.Standing, ", ")))
					}
					// An item answered twice and standing none is a fault, not an
					// unanswered item, and saying "carries no disposition" about it
					// would be a confident wrong statement.
					for _, c := range board.Outstanding.Cyclic {
						fmt.Fprintf(w, "  tangled %s (run %s) — its dispositions supersede one another, so none stands\n",
							termsafe.Sanitize(c.Item), termsafe.Sanitize(c.Run))
					}
					// An answer that exists and cannot be read is not an absent one.
					for _, u := range board.Outstanding.Unreadable {
						fmt.Fprintf(w, "  illegible %s (run %s) — stands on %s, which no reader can read\n",
							termsafe.Sanitize(u.Item), termsafe.Sanitize(u.Run), termsafe.Sanitize(u.Disposition))
					}
					// A tree the walk declined to enter is named, because a tree
					// nobody walked looks exactly like a tree with nothing in it.
					for _, u := range board.Outstanding.Unsafe {
						fmt.Fprintf(w, "  unread %s — %s; what it holds is neither outstanding nor answered\n",
							termsafe.Sanitize(u.Path), termsafe.Sanitize(u.Reason))
					}
					for _, h := range board.Outstanding.OpenHolds {
						fmt.Fprintf(w, "  held %s (%s) — exits when: %s\n",
							termsafe.Sanitize(h.Item), termsafe.Sanitize(h.Disposition),
							termsafe.Sanitize(h.ExitCondition))
					}
					fmt.Fprint(w, ledgerDecisionRule)
					fmt.Fprint(w, ideateRoutingRule)
				})
			}
			// Guard: a mistyped subcommand (e.g. `capture resovle iss-1 …`)
			// must not be swallowed as free text and filed. When args[0] is a
			// near-miss to a real subverb and the shape looks like a subcommand
			// call — a lone token, or a token followed by an issue id — refuse
			// with a did-you-mean and write nothing (unrecognized-input-never-
			// writes, iss-29). Genuine prose still files.
			if sug, ok := suspectedTypoedSubcommand(cmd, args); ok {
				return &exitError{Code: 2, Msg: fmt.Sprintf(
					"unknown capture subcommand %q; did you mean %q? (nothing captured — reword the text if you meant to file it)",
					args[0], sug)}
			}
			// Guard: a lone bare word near NO sub-verb fell through the
			// did-you-mean above and was filed as an issue
			// (iss-2608221328552172). A one-word positional is a sub-verb by
			// shape; only prose is issue text.
			if loneBareToken(args) {
				if args[0] == "" {
					return &exitError{Code: 2, Msg: "abcd capture: the issue text is empty (nothing captured)"}
				}
				return &exitError{Code: 2, Msg: fmt.Sprintf(
					"unknown capture subcommand %q (nothing captured — a lone word is read as a sub-verb, never as issue text; issue text must contain a space, so write the whole sentence)",
					args[0])}
			}
			// Fast path: append a structured issue from the free-form text.
			text := strings.Join(args, " ")
			// The slug is NOT derived here. Deriving it from the raw text before
			// core redacts the inputs kebab-cases a /Users/<name>/… home path into
			// "users-<name>-…", where nothing looks like a path any more, so the
			// ledger redactor leaves the username in the committed filename even as
			// it scrubs the body (gh-485). Core derives from the redacted text when
			// this is empty; an explicit --slug is passed through and redacted there
			// too.
			blocked, err := parseBlockedBy(blockedBy)
			if err != nil {
				return err
			}
			req := capture.CaptureRequest{
				RepoRoot:    repoRoot,
				Text:        text,
				Severity:    capture.Severity(orDefault(severity, "minor")),
				Category:    capture.Category(orDefault(category, "observation")),
				Source:      capture.Source(orDefault(source, "user-observation")),
				Slug:        slug,
				FoundDuring: orDefault(foundDuring, "manual-capture"),
				FoundAt:     foundAt,
				LapsedAt:    lapsedAt,
				BlockedBy:   blocked,
			}
			if req.ProductionMode, err = resolveProductionMode(repoRoot, captureProductionMode); err != nil {
				return err
			}
			// --lapsed-at has NO default, and this is where the caller learns it: a
			// lapse capture that omits the instant is refused in flag terms before
			// anything is reserved or written. Core refuses the same record on its
			// own (validateStrict, in property terms, for every other caller); which
			// category obliges the value is read from the one shared definition, not
			// restated here.
			if issueschema.LapsedAtRequired(string(req.Category)) && strings.TrimSpace(req.LapsedAt) == "" {
				return &exitError{Code: 2, Msg: "abcd capture --category " + issueschema.CategoryLapse +
					" requires --lapsed-at <RFC 3339 instant> (nothing captured — the moment the discipline gave way is never defaulted to the write-up time)"}
			}
			res, err := capture.Capture(req)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "captured %s (%s) — %s\n", res.ID, res.Status, termsafe.Sanitize(res.Path))
				// Redaction alters what the caller filed, so it is never silent: the
				// text on disk differs from the text handed in, and only the caller
				// can judge whether the redacted record still says what they meant.
				if res.Redacted > 0 {
					fmt.Fprintf(w, "  redacted %d span(s) before writing (home paths and identifiers are never committed)\n", res.Redacted)
				}
				if res.Degraded != "" {
					fmt.Fprintf(w, "  WARNING: %s\n", termsafe.Sanitize(res.Degraded))
				}
			})
		},
	}
	captureCmd.Flags().StringVar(&severity, "severity", "", "severity: nitpick | minor | major | critical (default minor)")
	captureCmd.Flags().StringVar(&category, "category", "", "issue category (default observation)")
	captureCmd.Flags().StringVar(&source, "source", "", "surfacing channel (default user-observation)")
	captureCmd.Flags().StringVar(&slug, "slug", "", "override the slug derived from the text")
	captureCmd.Flags().StringVar(&foundDuring, "found-during", "", "session/command context (default manual-capture)")
	captureCmd.Flags().StringVar(&foundAt, "found-at", "", "optional repo-relative path or conceptual location")
	// No default, deliberately: an unsupplied lapse time would default to the wall
	// clock at write-up, which is the one value the lapse log exists to rule out.
	captureCmd.Flags().StringVar(&lapsedAt, "lapsed-at", "", "RFC 3339 instant a discipline gave way (the lapse, not the write-up)")
	captureCmd.Flags().StringVar(&blockedBy, "blocked-by", "", "comma-separated iss-ids this issue is blocked by")
	// A closed choice, refused outright outside the vocabulary — the same shape
	// as --severity. There is no flag for `origin`: a capture is
	// researcher-authored by construction (itd-178).
	captureCmd.Flags().StringVar(&captureProductionMode, "production-mode", "", productionModeFlagHelp)

	// list — the earned SD001 exception: a filter flag is REQUIRED.
	var lsOpen, lsResolved, lsWontfix, lsAll bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List issues by state (one of --open/--resolved/--wontfix/--all required)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			state, err := listState(lsOpen, lsResolved, lsWontfix, lsAll)
			if err != nil {
				return err
			}
			res, err := capture.List(capture.ListRequest{RepoRoot: repoRoot, State: state})
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				for _, iss := range res.Issues {
					fmt.Fprintf(w, "%s  %s  %s  %s%s\n", iss.ID, iss.Status, iss.Severity, iss.Slug, blockedNote(iss))
				}
				for _, sk := range res.Skipped {
					// Path and Error echo a malformed issue file's own name and content
					// (err.Error() carries offending bytes), so sanitise before the terminal.
					fmt.Fprintf(w, "  skipped %s: %s\n", termsafe.Sanitize(sk.Path), termsafe.Sanitize(sk.Error))
				}
			})
		},
	}
	listCmd.Flags().BoolVar(&lsOpen, "open", false, "issues currently in open/")
	listCmd.Flags().BoolVar(&lsResolved, "resolved", false, "issues currently in resolved/")
	listCmd.Flags().BoolVar(&lsWontfix, "wontfix", false, "issues currently in wontfix/")
	listCmd.Flags().BoolVar(&lsAll, "all", false, "issues across all three states")
	captureCmd.AddCommand(listCmd)

	// resolve — open -> resolved with a note, a required product impact, and
	// optional resolved_by provenance (spc-25): the intent, spec, or commit
	// that fixed it.
	var resolveImpact, resolveByIntent, resolveBySpec, resolveByCommit, resolveShippedIn string
	var resolveGrounds, resolveModeRestamp string
	resolveCmd := &cobra.Command{
		Use:   "resolve <iss-N> <note> --impact <additive|breaking|fix|internal> --grounds \"<token>: <text>\" [--intent itd-N] [--spec spc-N] [--commit sha] [--shipped-in vX.Y.Z]",
		Short: "Mark an open issue resolved (open/ -> resolved/), optionally naming what fixed it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			if err := requireGroundsFlag("resolve", resolveGrounds); err != nil {
				return err
			}
			res, err := capture.Resolve(capture.ResolveRequest{
				RepoRoot: repoRoot, ID: args[0], Resolution: args[1], Impact: resolveImpact,
				ByIntent: resolveByIntent, BySpec: resolveBySpec, ByCommit: resolveByCommit,
				ShippedIn: resolveShippedIn, Grounds: resolveGrounds,
				ProductionMode: resolveModeRestamp,
			})
			if err != nil {
				return groundsUsageError("resolve", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "%s  %s -> %s — %s%s\n", res.ID, res.FromStatus, res.ToStatus, termsafe.Sanitize(res.Path), resolvedByNote(res.ResolvedBy))
				emitRedactionNote(w, res.Redacted, res.Degraded)
			})
		},
	}
	// resolved/ is gated by issue_impact_valid: the record carries the product
	// judgement the version derivation reads, and there is no default. The flag is
	// mandatory in effect — the core (capture.Resolve -> changelog.ParseImpact)
	// refuses an empty impact — but it is not marked cobra-required, to keep the
	// tree's no-required-flags invariant (TestLiveTreeMarksNoFlagRequired): the
	// requirement is enforced semantically in the core, not by a usage annotation.
	resolveCmd.Flags().StringVar(&resolveImpact, "impact", "", "product impact: additive|breaking|fix|internal (required)")
	resolveCmd.Flags().StringVar(&resolveGrounds, "grounds", "", groundsFlagUsage)
	resolveCmd.Flags().StringVar(&resolveByIntent, "intent", "", "resolved_by provenance: the itd-N that fixed it (must exist)")
	resolveCmd.Flags().StringVar(&resolveBySpec, "spec", "", "resolved_by provenance: the spc-N that fixed it (must exist)")
	resolveCmd.Flags().StringVar(&resolveByCommit, "commit", "", "resolved_by provenance: the fixing commit sha (7-64 hex chars, shape-checked only)")
	// --shipped-in is a MIGRATION flag, for the ledger-hygiene case only: closing a
	// record whose fix was released long ago. A repo abcd managed from its first
	// commit should never reach for it — RS001 makes resolution ride the fixing
	// commit, so the cut is right without it. The derivation leaves such a record out of the current
	// cut, so the release record cannot announce old work as new. Absent by
	// default, and never inferred — a record that says nothing belongs to this cut.
	// A transition RESTAMPS production_mode, so the flag is passed through
	// unresolved: an undeclared mode must leave the record's existing stamp alone,
	// and taking the repo default here would silently overwrite it (itd-178).
	resolveCmd.Flags().StringVar(&resolveModeRestamp, "production-mode", "",
		"restamp how this record's text was produced: "+provenance.ModeList()+" (default: leave the record's existing stamp alone; refused on a record that predates disclosure)")
	resolveCmd.Flags().StringVar(&resolveShippedIn, "shipped-in", "", "MIGRATION USE: the release that already carried this work (vX.Y.Z), leaving the record out of the current cut; unnecessary in a repo abcd managed from the start")
	captureCmd.AddCommand(resolveCmd)

	// promote — graduate an issue, or a dispositioned reading item, into an
	// intent draft (spc-24, step 2 of the record walk; spc-58 for the reading
	// item). Default mode mints a draft and stamps the source record's
	// promoted_to in one invocation; --intent is the stamp-only repair/link
	// mode. The issue keeps its status folder — promotion is not resolution —
	// and an undispositioned rdi-N is refused before anything is minted.
	var promoteIntent, promoteGrounds, promoteProductionMode string
	promoteCmd := &cobra.Command{
		Use:   "promote <iss-N> --grounds \"<token>: <text>\" | promote <rdi-N>",
		Short: "Graduate an issue or a dispositioned reading item into an intent draft (mints + stamps promoted_to)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			// The grounds argument belongs to the ISSUE route, which has no
			// other place to say why the conjecture is being pursued. A reading
			// item records that in its DISPOSITION, a separate record promote
			// already refuses to act without, so demanding a second conjecture
			// here would collect a value nothing writes.
			if !strings.HasPrefix(args[0], issueschema.ReadingItemFamily+"-") {
				if err := requireGroundsFlag("promote", promoteGrounds); err != nil {
					return err
				}
			}
			// The mode belongs to the DRAFT this mints; stamp-only mode mints
			// nothing, so it carries none.
			mode, err := resolveProductionMode(repoRoot, promoteProductionMode)
			if err != nil {
				return err
			}
			res, err := capture.Promote(capture.PromoteRequest{
				RepoRoot: repoRoot, ID: args[0], LinkIntent: promoteIntent, Grounds: promoteGrounds,
				ProductionMode: mode,
			})
			if err != nil {
				return groundsUsageError("promote", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				verb := "minted"
				if res.Linked {
					verb = "linked"
				}
				// Paths echo ledger/corpus filenames (attacker-shapeable in a
				// hostile clone), so sanitise before the terminal.
				fmt.Fprintf(w, "%s (%s, %s) promoted — %s %s — %s\n",
					res.IssueID, res.IssueStatus, termsafe.Sanitize(res.IssuePath),
					verb, res.IntentID, termsafe.Sanitize(res.IntentPath))
				// The draft's ONE back-edge stayed where it was: an intent
				// occasioned by several reading items is promoted from one, and
				// the item this call linked still points forward.
				if res.BackEdgeKept != "" {
					fmt.Fprintf(w, "back_edge: kept %s\n", termsafe.Sanitize(res.BackEdgeKept))
				}
				emitRedactionNote(w, res.Redacted, res.Degraded)
			})
		},
	}
	promoteCmd.Flags().StringVar(&promoteIntent, "intent", "", "stamp-only mode: link this existing itd-N instead of minting a draft")
	promoteCmd.Flags().StringVar(&promoteGrounds, "grounds", "", groundsFlagUsage)
	promoteCmd.Flags().StringVar(&promoteProductionMode, "production-mode", "", productionModeFlagHelp)
	captureCmd.AddCommand(promoteCmd)

	// disposition — the researcher's answer to ONE reading item, written as a
	// separate record keyed to that item (spc-58). It is a distinct verb rather
	// than a flag on anything else because the reading and the answer are two
	// acts: collapsing them into one write would leave nothing able to show that
	// a finding existed before it was answered.
	var dispState, dispGrounds, dispExit, dispSupersedes, dispRecurs string
	var dispHoldFrame, dispHoldMoscow string
	dispositionCmd := &cobra.Command{
		Use:   "disposition <rdi-N> --state <accepted|rejected|declined|held> [--grounds <text>] [--exit-condition <text>] [--supersedes <dsp-N>] [--recurs <rdi-N,...>]",
		Short: "Answer one reading item (a separate record, keyed to the item)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			recurs, err := parseRecurs(dispRecurs)
			if err != nil {
				return err
			}
			res, err := capture.Disposition(capture.DispositionRequest{
				RepoRoot: repoRoot, Item: args[0], State: dispState,
				Grounds: dispGrounds, ExitCondition: dispExit,
				Supersedes: dispSupersedes, Recurs: recurs,
				HoldFrameLocation: dispHoldFrame, HoldMoscow: dispHoldMoscow,
			})
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "%s  %s %s (%s) — %s\n",
					res.ID, res.Item, res.State, res.Position, termsafe.Sanitize(res.Path))
				if res.Redacted > 0 {
					fmt.Fprintf(w, "  redacted %d span(s) before writing (home paths and identifiers are never committed)\n", res.Redacted)
				}
				if res.Degraded != "" {
					fmt.Fprintf(w, "  WARNING: %s\n", termsafe.Sanitize(res.Degraded))
				}
			})
		},
	}
	// --state carries no default. Which answer a researcher gave is the whole
	// content of the record, and a defaulted judgement is a judgement nobody
	// made; the core refuses an empty state rather than inventing one. It stays
	// unmarked as cobra-required to keep the tree's no-required-flags invariant.
	dispositionCmd.Flags().StringVar(&dispState, "state", "", "the answer: accepted | rejected | declined | held (availability varies by the item's position)")
	dispositionCmd.Flags().StringVar(&dispGrounds, "grounds", "", "disposition_grounds: why this answer (free text; required on every state except held)")
	dispositionCmd.Flags().StringVar(&dispExit, "exit-condition", "", "what would end a held disposition (required on held; a hold exits only through a superseding disposition that cites it)")
	dispositionCmd.Flags().StringVar(&dispSupersedes, "supersedes", "", "the standing dsp-N this answer replaces; required once an item already carries one")
	dispositionCmd.Flags().StringVar(&dispRecurs, "recurs", "", "comma-separated prior rdi-ids this item recurs from — the recorded form of a warm recognition, never a mechanical join")
	// The two-axis hold field is RESERVED and dormant. The flags exist so the
	// reservation is a behaviour a caller meets rather than a comment nobody
	// reads: a populated value is refused, and the refusal states the grammar.
	dispositionCmd.Flags().StringVar(&dispHoldFrame, "hold-frame-location", "", "RESERVED (dormant): the frame element a hold sits at; a populated value is refused until activation is ruled")
	dispositionCmd.Flags().StringVar(&dispHoldMoscow, "hold-moscow", "", "RESERVED (dormant): must | should | could | wont; a populated value is refused until activation is ruled")
	captureCmd.AddCommand(dispositionCmd)

	// wontfix — open -> wontfix with a reason. It needs no required --grounds:
	// the reason is already mandatory, so a wontfix could never be recorded
	// without grounds — what it lacked was the TYPE, which it stamps as
	// `declined: <reason>`. The flag overrides the TEXT for the case where the
	// conjecture is worth stating separately from the user-facing reason.
	var wontfixGrounds, wontfixProductionMode string
	wontfixCmd := &cobra.Command{
		Use:   "wontfix <iss-N> <reason> [--grounds \"declined: <text>\"]",
		Short: "Record an explicit non-action decision (open/ -> wontfix/)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			res, err := capture.Wontfix(capture.WontfixRequest{
				RepoRoot: repoRoot, ID: args[0], Reason: args[1], Grounds: wontfixGrounds,
				ProductionMode: wontfixProductionMode,
			})
			if err != nil {
				return groundsUsageError("wontfix", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "%s  %s -> %s — %s\n", res.ID, res.FromStatus, res.ToStatus, termsafe.Sanitize(res.Path))
				emitRedactionNote(w, res.Redacted, res.Degraded)
			})
		},
	}
	// No backticks: cobra's UnquoteUsage reads the first backquoted word as the
	// flag's value placeholder and strips it from the prose, which printed
	// `--grounds declined` and lost the word (iss-2608301212428844).
	wontfixCmd.Flags().StringVar(&wontfixGrounds, "grounds", "",
		"override the recorded grounds text (the token stays declined — a wontfix IS that non-action)")
	wontfixCmd.Flags().StringVar(&wontfixProductionMode, "production-mode", "",
		"restamp how this record's text was produced: "+provenance.ModeList()+" (default: leave the record's existing stamp alone; refused on a record that predates disclosure)")
	captureCmd.AddCommand(wontfixCmd)

	return captureCmd
}

// readyGroundsEnvelope is `abcd intent ready --grounds --json`'s payload: the
// grounds write beside the readiness report.
//
// It exists because the plugin surface tells the host to report the redaction
// count, and the redaction-is-never-silent promise cannot be kept by a stderr
// line on the plane a machine consumer reads. Without the flag the payload stays
// the bare ReadyResult, so the envelope appears exactly when there is a write to
// describe (iss-2608300930057882).
//
// There is deliberately no `degraded` member. The intent-side redactor is
// FAIL-CLOSED: a scanner that cannot be built, or whose pattern set a per-repo
// override weakened, refuses the write and exits 2 rather than writing under a
// weakened detector. The ledger's redactor degrades and reports, and carries the
// member for it; here a degraded scanner is an error, and a member that could
// only ever be empty would be a second promise nothing keeps.
type readyGroundsEnvelope struct {
	Grounds *intent.GroundsResult `json:"grounds"`
	Ready   intent.ReadyResult    `json:"ready"`
}

// emitGroundsReceipt says that a record was written, on the plane the caller can
// still read after a later fault: stdout in text mode (before the report), stderr
// in --json mode (where stdout must stay a single machine payload).
func emitGroundsReceipt(cmd *cobra.Command, asJSON bool, rec intent.GroundsResult) {
	w := cmd.OutOrStdout()
	if asJSON {
		w = cmd.ErrOrStderr()
	}
	// The path is a corpus filename, attacker-shapeable in a hostile clone.
	fmt.Fprintf(w, "abcd intent ready — recorded grounds on %s (%d entries)\n",
		termsafe.Sanitize(rec.Path), rec.Entries)
	if rec.Redacted > 0 {
		fmt.Fprintf(w, "  redacted %d span(s) before writing (home paths and identifiers are never committed)\n",
			rec.Redacted)
	}
}

// groundsFlagUsage is the one spelling of the argument's help text, so promote
// and resolve cannot describe the same closed vocabulary differently.
var groundsFlagUsage = "REQUIRED — the conjecture being acted on, not the route taken: " +
	"\"" + grounds.UsageSpelling() + ": <what is expected, and what would show it wrong>\""

// groundsUsageError maps a core grounds refusal to exit 2, leaving every other
// failure on its existing path.
//
// A MISSING --grounds exited 2 (the flag check above) while a MALFORMED one
// exited 1, so a caller distinguishing usage errors from real failures learned
// the wrong thing from the same flag (iss-2608300930057882). Both are one thing:
// the argument was not usable and nothing was written. The core carries one
// sentinel for the whole class, so this needs no second copy of the vocabulary,
// the grammar, or the floor.
func groundsUsageError(verb string, err error) error {
	if errors.Is(err, capture.ErrGroundsRefused) {
		return &exitError{Code: 2, Msg: "abcd capture " + verb + ": " + scrubPaths(err)}
	}
	return err
}

// requireGroundsFlag refuses a triage that names no grounds, at the CLI, as a
// USAGE error (exit 2) with nothing written — the same shape `--category lapse`
// without `--lapsed-at` already has. The core refuses the same call on its own
// for every other caller; this is where a person typing the command learns it, in
// flag terms rather than in property terms. Neither flag is marked
// cobra-required, which would break the tree's no-required-flags invariant
// (TestLiveTreeMarksNoFlagRequired): the requirement is semantic, not a usage
// annotation.
func requireGroundsFlag(verb, value string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}
	return &exitError{Code: 2, Msg: "abcd capture " + verb + " requires --grounds \"" + grounds.UsageSpelling() + ": <text>\" " +
		"(nothing written — a triage records the conjecture being acted on, never only the route taken)"}
}

// emitRedactionNote says, on the human surface, that the written text differs
// from the text handed in. Redaction is never silent: only the caller can judge
// whether the rewritten record still says what they meant.
func emitRedactionNote(w io.Writer, redacted int, degraded string) {
	if redacted > 0 {
		fmt.Fprintf(w, "  redacted %d span(s) before writing (home paths and identifiers are never committed)\n", redacted)
	}
	if degraded != "" {
		fmt.Fprintf(w, "  WARNING: %s\n", termsafe.Sanitize(degraded))
	}
}

// resolvedByNote renders the stamped provenance members for the resolve text
// surface ("" when none were written). Members are regex-validated ids/shas,
// so no sanitisation is needed.
func resolvedByNote(rb *capture.ResolvedBy) string {
	if rb == nil {
		return ""
	}
	var parts []string
	if rb.Intent != "" {
		parts = append(parts, "intent="+rb.Intent)
	}
	if rb.Spec != "" {
		parts = append(parts, "spec="+rb.Spec)
	}
	if rb.Commit != "" {
		parts = append(parts, "commit="+rb.Commit)
	}
	return " — resolved_by " + strings.Join(parts, " ")
}

// listState maps the mutually-exclusive filter flags to a capture.State, or
// returns the exit-2 "choose a filter" usage error the brief mandates for the
// unfiltered `abcd capture list` form (04-surfaces/06 § 1).
func listState(open, resolved, wontfix, all bool) (capture.State, error) {
	var chosen capture.State
	n := 0
	if open {
		chosen, n = capture.StateOpen, n+1
	}
	if resolved {
		chosen, n = capture.StateResolved, n+1
	}
	if wontfix {
		chosen, n = capture.StateWontfix, n+1
	}
	if all {
		chosen, n = capture.StateAll, n+1
	}
	if n == 0 {
		return "", &exitError{Code: 2, Msg: "capture list: choose a filter: --open / --resolved / --wontfix / --all"}
	}
	if n > 1 {
		return "", &exitError{Code: 2, Msg: "capture list: the filter flags are mutually exclusive"}
	}
	return chosen, nil
}

// issIDRe validates a --blocked-by token at the CLI boundary (mirrors the core
// ^iss-[0-9]+$ schema constraint).
var issIDRe = regexp.MustCompile(`^iss-[0-9]+$`)

// readingItemIDRe validates a --recurs token at the CLI boundary (mirrors the
// core ^rdi-[0-9]+$ schema constraint).
var readingItemIDRe = regexp.MustCompile(`^rdi-[0-9]+$`)

// recordIDRe matches any abcd record id (issue, intent, or spec). It is used
// only by suspectedTypoedSubcommand's shape check — distinct from issIDRe, which
// validates a real iss-only --blocked-by token — so the typo guard recognises a
// subcommand call in either verb family (capture's iss-N ids, intent's itd/spc
// ids) without loosening iss-id validation elsewhere.
var recordIDRe = regexp.MustCompile(`^(iss|itd|spc)-[0-9]+$`)

// retiredSubverbs maps a parent command to sub-verb spellings that were
// renamed in a clean break (no alias): an invocation shaped like a subcommand
// call using a retired spelling is refused with the successor named, so it can
// never be swallowed as free text and silently filed (the same
// unrecognized-input-never-writes contract as the typo guard, iss-29).
var retiredSubverbs = map[string]map[string]string{
	"intent": {"review": "audit"}, // spc-28 (adr-40)
}

// suspectedTypoedSubcommand reports the nearest real subverb when args[0] is a
// near-miss for one (edit distance 1–2) — or a retired spelling of one — and
// the invocation shape resembles a subcommand call rather than free-text
// prose: a lone token, or a token followed by a record id. It is deliberately
// high-precision so it never refuses a legitimate free-text create whose first
// word merely resembles a verb — those carry no trailing record id and are
// multi-word.
func suspectedTypoedSubcommand(parent *cobra.Command, args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	if successor, ok := retiredSubverbs[parent.Name()][args[0]]; ok {
		// A retired spelling is refused in every shape a subcommand call takes:
		// a lone token, a token before a record id, or the retired verb followed
		// by one of the SUCCESSOR's own registered sub-verbs (`review ingest`
		// must refuse just as `review itd-N` does — the pre-rename two-word
		// invocation may never be swallowed as free text and filed).
		if len(args) == 1 || recordIDRe.MatchString(args[1]) || isSubverbOf(parent, successor, args[1]) {
			return successor, true
		}
	}
	shapedLikeSubcommand := len(args) == 1 || recordIDRe.MatchString(args[1])
	if !shapedLikeSubcommand {
		return "", false
	}
	best, bestDist := "", 3 // accept edit distances 1 and 2
	for _, c := range parent.Commands() {
		name := c.Name()
		if c.Hidden || name == "help" || name == "completion" {
			continue
		}
		if d := levenshtein(args[0], name); d > 0 && d < bestDist {
			best, bestDist = name, d
		}
	}
	return best, best != ""
}

// loneBareToken reports whether the positional is one whitespace-free word — the
// single invocation shape a free-text create verb cannot tell apart from a
// sub-verb call. `abcd capture nosuchthing` and `abcd capture resolve` are the
// same shape; only the second happens to reach cobra's dispatcher first, so the
// first was swallowed as issue text and minted a durable record at exit 0
// (iss-2608221328552172). suspectedTypoedSubcommand catches a lone token NEAR a
// real sub-verb (edit distance 1–2); this catches every other lone token, which
// is the half that had no guard at all.
//
// Prose is unambiguous and still files. The canonical create path passes ONE
// argument carrying whitespace — the shell has already eaten the quotes around
// `abcd intent "widen the public api"` — and an unquoted title arrives as
// several arguments. What is refused is a one-word record title, and a one-word
// title is neither a press release nor an issue report. An empty positional
// (`abcd capture ""` — a script whose variable expanded to nothing) is caught by
// the same rule. Core refused that one too, but two layers down as "slug
// normalises to empty" and at exit 1, so the callers name it here instead: an
// empty text is not an unknown sub-verb, and a usage refusal is exit 2.
//
// unicode.IsSpace rather than an ASCII test, so a non-breaking space between two
// words of a genuine title is read as the whitespace it is.
func loneBareToken(args []string) bool {
	return len(args) == 1 && strings.IndexFunc(args[0], unicode.IsSpace) < 0
}

// isSubverbOf reports whether token names a registered sub-command of parent's
// child command named successor (e.g. is "ingest" a sub-verb of "audit").
func isSubverbOf(parent *cobra.Command, successor, token string) bool {
	for _, c := range parent.Commands() {
		if c.Name() != successor {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() == token {
				return true
			}
		}
	}
	return false
}

// levenshtein is the classic edit distance (insert/delete/substitute each cost
// 1). Inputs are subcommand-name sized, so the simple O(n·m) two-row form is
// more than fast enough.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// parseBlockedBy splits the comma-separated --blocked-by value into iss-ids,
// dropping blanks and rejecting any token that is not ^iss-[0-9]+$. An empty
// input yields a nil slice (the field is omitted).
func parseBlockedBy(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var ids []string
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if !issIDRe.MatchString(tok) {
			return nil, fmt.Errorf("capture: --blocked-by token %q must match iss-N", tok)
		}
		ids = append(ids, tok)
	}
	return ids, nil
}

// captureBoard is the bare status board: the ledger counts and the
// outstanding-readings report, flattened into one envelope so the --json and
// text renders answer the same question. The report is embedded rather than
// re-derived — ReadReadingOutstanding is the one implementation, shared with the
// reading_outstanding lint rule.
type captureBoard struct {
	capture.StatusResult
	Outstanding lint.OutstandingReadings `json:"reading_outstanding"`
}

// captureBoardOf composes the board, normalising the report's collections to
// empty slices: a collection is never null in this surface's envelope.
func captureBoardOf(repoRoot string, st capture.StatusResult) (captureBoard, error) {
	// repoRoot is the SAME value handed to capture.Status, so the two halves of
	// one board can never resolve two different ledgers.
	report, err := lint.ReadReadingOutstanding(repoRoot, capture.LedgerRelPath)
	if err != nil {
		return captureBoard{}, err
	}
	if report.Undispositioned == nil {
		report.Undispositioned = []lint.OutstandingItem{}
	}
	if report.Unadmitted == nil {
		report.Unadmitted = []lint.OutstandingItem{}
	}
	if report.OpenHolds == nil {
		report.OpenHolds = []lint.OpenHold{}
	}
	if report.Unsafe == nil {
		report.Unsafe = []lint.UnsafePath{}
	}
	if report.Cyclic == nil {
		report.Cyclic = []lint.OutstandingItem{}
	}
	if report.Contested == nil {
		report.Contested = []lint.ContestedItem{}
	}
	if report.Unreadable == nil {
		report.Unreadable = []lint.UnreadableAnswer{}
	}
	return captureBoard{StatusResult: st, Outstanding: report}, nil
}

// parseRecurs splits the comma-separated --recurs list into prior reading-item
// ids. Shape is checked here so a typo is a usage error rather than a refusal
// from deep inside the ledger writer; membership (does the item exist) is
// deliberately NOT checked, because a recurrence may cite an item from a run
// this ledger no longer carries.
func parseRecurs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var ids []string
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if !readingItemIDRe.MatchString(tok) {
			return nil, fmt.Errorf("capture: --recurs token %q must match rdi-N", tok)
		}
		ids = append(ids, tok)
	}
	return ids, nil
}

// blockedNote renders the derived-priority annotation for a row: when the issue
// has blocked_by targets still open, " [blocked-by iss-1,iss-2]"; otherwise "".
func blockedNote(iss capture.Issue) string {
	if len(iss.BlockedByOpen) == 0 {
		return ""
	}
	return " [blocked-by " + strings.Join(iss.BlockedByOpen, ",") + "]"
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// newMemoryCommand builds the `memory` sub-tree over internal/core/memory. Bare
// `memory` renders read-only store status; ingest/ask/lint are the mutating and
// diagnostic verbs (04-surfaces/07). The distiller (ingest) and synthesizer
// (ask) are host-delegated seams: the .5 skill emits validated DistilledPage
// JSON, which this surface feeds through --pages-json / --page-json.
func newMemoryCommand(asJSON *bool) *cobra.Command {
	memoryCmd := &cobra.Command{
		Use:   "memory",
		Short: "Curated knowledge substrate; bare invocation is read-only status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			st, err := memory.Bare(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) {
				fmt.Fprintf(w, "abcd memory — %d page(s)", st.Pages)
				if !st.StorePresent {
					fmt.Fprintf(w, " (store not present)")
				}
				fmt.Fprintln(w)
				for _, c := range st.ByClass {
					// Class labels come from page frontmatter; contradiction/headroom lines
					// are read verbatim from repo files — all untrusted terminal output.
					fmt.Fprintf(w, "  %s: %d\n", termsafe.Sanitize(c.Class), c.Count)
				}
				if st.LastIngest != "" {
					fmt.Fprintf(w, "  last ingest: %s\n", termsafe.Sanitize(st.LastIngest))
				}
				for _, line := range st.Contradictions {
					fmt.Fprintf(w, "  contradiction: %s\n", termsafe.Sanitize(line))
				}
				for _, line := range st.Headroom {
					fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(line))
				}
			})
		},
	}

	// ingest <path-or-https-url> [--keep-original] [--pages-json <file|->]
	var pagesJSON string
	var keepOriginalFlag bool
	ingestCmd := &cobra.Command{
		Use:   "ingest <path-or-https-url>",
		Short: "Distil an external source into cited memory pages (https URLs only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := memory.Ingest(memory.IngestRequest{
				RepoRoot:     cwd,
				Source:       args[0],
				KeepOriginal: keepOriginalFlag,
				Distiller:    pagesJSONDistiller(cmd, pagesJSON),
			})
			if err != nil {
				return err
			}
			if err := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd memory ingest — %s\n", res.Status)
				fmt.Fprintf(w, "  content hash: %s\n", res.ContentHash)
				// The licence is free text lifted verbatim from the ingested
				// source's bytes (an SPDX line or an HTTP License: header), not a
				// validated SPDX token, so it can carry raw terminal control/escape/
				// bidi/zero-width runes and must be defanged like the sibling memory
				// render fields before it reaches the TTY (gh-262).
				fmt.Fprintf(w, "  licence:      %s\n", termsafe.Sanitize(res.Licence))
				if len(res.Pages) > 0 {
					fmt.Fprintf(w, "  pages:        %s\n", strings.Join(res.Pages, ", "))
				}
				if res.KeptOriginal != "" {
					fmt.Fprintf(w, "  kept original: %s\n", res.KeptOriginal)
				}
				if res.KeepOriginalError != "" {
					fmt.Fprintf(w, "  warning: --keep-original failed (the source was still ingested): %s\n", res.KeepOriginalError)
				}
			}); err != nil {
				return err
			}
			// The ingest succeeded but an explicitly-requested --keep-original
			// copy did not: signal it with a non-zero exit while leaving the
			// rendered result (which reports what was durably written) intact.
			if res.KeepOriginalError != "" {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	ingestCmd.Flags().BoolVar(&keepOriginalFlag, "keep-original", false, "store the original at .abcd/memory/sources/<sha256>.<ext>")
	ingestCmd.Flags().StringVar(&pagesJSON, "pages-json", "", "DistilledPage JSON array (file path, or - for stdin)")
	memoryCmd.AddCommand(ingestCmd)

	// ask <question> [--top-n N] [--file-back] [--page-json <file|->]
	var topN int
	var fileBack bool
	var pageJSON string
	askCmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Query memory and synthesise a cited answer",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			req := memory.AskRequest{RepoRoot: cwd, Question: strings.Join(args, " "), TopN: topN}
			if fileBack {
				page, err := readPageJSON(cmd, pageJSON)
				if err != nil {
					return err
				}
				req.FileBackPage = page
				req.DecideFileBack = func(memory.DistilledPage) bool { return true }
			}
			res, err := memory.Ask(req)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintln(w, res.Answer)
				if res.FileBack != nil {
					fmt.Fprintf(w, "\nfiled back (%s): %s\n", res.FileBack.Status, strings.Join(res.FileBack.Pages, ", "))
				}
			})
		},
	}
	askCmd.Flags().IntVar(&topN, "top-n", 0, "retrieval depth (0 uses the pinned default)")
	askCmd.Flags().BoolVar(&fileBack, "file-back", false, "file the synthesised answer back as a new memory page")
	askCmd.Flags().StringVar(&pageJSON, "page-json", "", "the answer page dict as JSON (file path, or - for stdin)")
	memoryCmd.AddCommand(askCmd)

	// lint — full-store curator health-check; blockers exit nonzero.
	memoryCmd.AddCommand(&cobra.Command{
		Use:   "lint",
		Short: "Curator health-check over the whole memory store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := memory.Lint(memory.LintRequest{RepoRoot: cwd})
			if err != nil {
				return err
			}
			if err := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd memory lint — %d blocker(s), %d warning(s), %d info(s)\n",
					res.Summary.Blockers, res.Summary.Warnings, res.Summary.Infos)
				for _, f := range res.Findings {
					fmt.Fprintf(w, "  %s [%s] %s:%d %s\n", f.Code, f.Severity, termsafe.Sanitize(f.File), f.Line, termsafe.Sanitize(f.Message))
				}
				fmt.Fprintf(w, "  report: %s\n", res.ReportDir)
			}); err != nil {
				return err
			}
			// Propagate the curator exit contract: blockers -> nonzero.
			if res.ExitCode != 0 {
				return &exitError{Code: res.ExitCode}
			}
			return nil
		},
	})

	return memoryCmd
}

// pagesJSONDistiller is the ingest transport seam: it lazily reads the
// DistilledPage JSON array from --pages-json (a file, or - for stdin) only when
// distillation is actually needed. A registry-only hit never invokes it, so an
// already-known source re-ingests with no payload.
func pagesJSONDistiller(cmd *cobra.Command, pagesJSON string) memory.Distiller {
	return func(_ string, _ map[string]any) ([]map[string]any, error) {
		if pagesJSON == "" {
			return nil, fmt.Errorf("no distiller output supplied: pass --pages-json <file|-> with the DistilledPage JSON array")
		}
		raw, err := readSource(cmd, pagesJSON)
		if err != nil {
			return nil, fmt.Errorf("cannot read --pages-json: %w", err)
		}
		var arr []map[string]any
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("--pages-json must be a JSON array of page dicts: %w", err)
		}
		return arr, nil
	}
}

// readPageJSON reads ONE DistilledPage dict for ask file-back from --page-json.
func readPageJSON(cmd *cobra.Command, pageJSON string) (map[string]any, error) {
	if pageJSON == "" {
		return nil, fmt.Errorf("--file-back requires --page-json <file|-> with the answer page dict")
	}
	raw, err := readSource(cmd, pageJSON)
	if err != nil {
		return nil, fmt.Errorf("cannot read --page-json: %w", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("--page-json must be one JSON object (a DistilledPage dict): %w", err)
	}
	return obj, nil
}

// maxOperandJSONBytes is the house cap (8 MiB, matching the registry/graveyard
// JSON caps) for an untrusted JSON operand read from a file path or stdin.
const maxOperandJSONBytes = 8 << 20

// readSource reads a JSON payload from a file path, or from stdin when spec is
// "-" (the streaming transport the .5 skill uses). The operand is untrusted
// content (host-produced pages, cross-machine artifacts), so both transports are
// bounded and the file path is read behind the trust guards: stdin is refused
// whole when over-cap (a cap+1 probe, never a severed prefix), and a file is
// read with fsutil.ReadGuarded (O_NOFOLLOW so a symlink operand is never
// followed, regular-file on the open fd, and the size cap — all in one call, no
// lstat→read TOCTOU). Refusing the over-cap prefix matters most on the history
// capture path, where a truncated transcript would be stored under a sha256
// idempotency key computed over the prefix (spc-4's refuse-whole invariant).
func readSource(cmd *cobra.Command, spec string) ([]byte, error) {
	return readSourceCapped(cmd, spec, maxOperandJSONBytes)
}

// readSourceCapped is readSource with the cap named by the caller, because the
// right bound depends on what is being read and there is more than one answer.
//
// maxOperandJSONBytes is a JSON-operand cap — the registry/graveyard payload
// size — and applying it to a transcript was a wrong-constant bug
// (iss-2608231029040602). It made `history capture` refuse at 8 MiB while the
// SessionEnd path accepted 64 MiB, so a transcript the hooks would store
// automatically could not be recovered by hand: the recovery verb was bounded
// eight times tighter than the thing it exists to recover from. Real sessions
// in this repo reach 11.8 MB, so that was not a pathology guard but a
// functional limit on ordinary work.
//
// The caps themselves stay. Both transports read whole into memory before the
// scanner walks them, and both refuse an over-cap file WHOLE rather than
// truncating, because a severed prefix would be stored under a sha256
// idempotency key computed over the prefix (spc-4's refuse-whole invariant).
func readSourceCapped(cmd *cobra.Command, spec string, limit int64) ([]byte, error) {
	if spec == "-" {
		return readCappedStdin(cmd, limit)
	}
	return readGuardedOperand(spec, limit)
}

// newHistoryCommand builds the `history` sub-tree over internal/core/history —
// the native session-transcript store (adr-29). `list`/`show` read; `capture`
// is the redacting write path. The per-repo store is keyed on the root-commit
// SHA resolved from cwd.
func newHistoryCommand(asJSON *bool) *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Manage the native session-transcript store",
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
	}

	// capture — the redacting write path: read a raw transcript from a file
	// argument (or stdin with "-"/no arg), sanitise it through the scanner
	// (two-stage, fail-closed), and store the record. This is the ONLY path that
	// writes to the store; list/show never mutate.
	var session, kind string
	captureCmd := &cobra.Command{
		Use:   "capture [<transcript-file>|-]",
		Short: "Redact and store a raw session transcript (reads a file or stdin)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, rootSHA, err := historyStore(cmd)
			if err != nil {
				return err
			}
			src := "-"
			if len(args) == 1 {
				src = args[0]
			}
			// The transcript cap, not the JSON-operand cap: this verb recovers
			// what the hooks store, so it must accept what they accept.
			raw, err := readSourceCapped(cmd, src, maxTranscriptBytes)
			if err != nil {
				return fmt.Errorf("history capture: cannot read transcript: %w", err)
			}
			sess := session
			if sess == "" && src != "-" {
				// Derive a session id from the file basename (sans extension).
				base := filepath.Base(src)
				sess = strings.TrimSuffix(base, filepath.Ext(base))
			}
			if sess == "" {
				return fmt.Errorf("history capture: --session <id> is required when reading from stdin")
			}
			res, err := history.Capture(repoRoot, rootSHA, sess, raw, orDefault(kind, "native"))
			if err != nil {
				return err
			}
			// The stored path is absolute and home-rooted; this is a success
			// envelope the CLI error scrub never sees, so redact the home root to
			// ~ before it is rendered or marshalled. Callers re-derive the file
			// handle from disk, never from this rendered value.
			res.Record.Path = fsutil.RedactHome(res.Record.Path)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				if !res.Wrote {
					fmt.Fprintf(w, "abcd history capture — %s already stored (no-op); redacted secrets=%d home=%d\n",
						res.Record.SessionID, res.Record.Secrets, res.Record.HomePaths)
					return
				}
				fmt.Fprintf(w, "abcd history capture — stored %s (%s)\n", res.Record.SessionID, res.Record.SourceKind)
				fmt.Fprintf(w, "  path:     %s\n", termsafe.Sanitize(res.Record.Path))
				fmt.Fprintf(w, "  redacted: secrets=%d home=%d\n", res.Record.Secrets, res.Record.HomePaths)
			})
		},
	}
	captureCmd.Flags().StringVar(&session, "session", "", "session id for the record (default: transcript filename; required for stdin)")
	captureCmd.Flags().StringVar(&kind, "kind", "", "source kind: native | specstory-import (default native)")
	historyCmd.AddCommand(captureCmd)

	// list — records newest-first for this repo.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List stored transcripts for this repo, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, rootSHA, err := historyStore(cmd)
			if err != nil {
				return err
			}
			records, err := history.List(repoRoot, rootSHA)
			if err != nil {
				return err
			}
			// An empty store is an empty LIST in JSON, not bare `null`: the
			// command doc promises "an empty list means no transcripts", and a
			// consumer that iterates the value should get [], as every other
			// --json verb's collection does.
			if records == nil {
				records = []history.Record{}
			}
			// The path field is absolute and home-rooted; redact the home root to ~
			// in this success envelope (JSON and text) before it is marshalled.
			for k := range records {
				records[k].Path = fsutil.RedactHome(records[k].Path)
			}
			return render(cmd.OutOrStdout(), *asJSON, records, func(w io.Writer) {
				if len(records) == 0 {
					fmt.Fprintln(w, "abcd history — no transcripts stored for this repo")
					return
				}
				for _, r := range records {
					fmt.Fprintf(w, "%s  %s  %s  redacted secrets=%d home=%d\n",
						r.CapturedAt.Format("2006-01-02T15:04:05Z"), termsafe.Sanitize(r.SessionID), termsafe.Sanitize(r.SourceKind), r.Secrets, r.HomePaths)
				}
			})
		},
	})

	// staged — what ended but is not yet stored. This is the outcome axis the
	// store never had: before staging existed, "absent from the store" spanned
	// never-ended, ended-before-the-store-existed and ended-and-lost, and nothing
	// could tell them apart. A staged entry says exactly one thing.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "staged",
		Short: "List transcripts that ended but are not yet redacted into the store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, rootSHA, err := historyStore(cmd)
			if err != nil {
				return err
			}
			staged, err := history.ListStaged(repoRoot, rootSHA)
			if err != nil {
				return err
			}
			if staged == nil {
				staged = []history.Staged{}
			}
			for k := range staged {
				staged[k].Path = fsutil.RedactHome(staged[k].Path)
			}
			return render(cmd.OutOrStdout(), *asJSON, staged, func(w io.Writer) {
				if len(staged) == 0 {
					fmt.Fprintln(w, "abcd history — nothing staged; every ended session is stored")
					return
				}
				for _, s := range staged {
					fmt.Fprintf(w, "%s  %s  %d bytes  awaiting redaction\n",
						s.StagedAt.Format("2006-01-02T15:04:05Z"), termsafe.Sanitize(s.SessionID), s.Bytes)
				}
				fmt.Fprintf(w, "\n%d staged transcript(s) hold UNREDACTED text until drained; run `abcd history drain`.\n", len(staged))
			})
		},
	})

	// drain — finish the capture SessionStart bounded. Unbudgeted by design: the
	// interactive budget exists to protect a session start, and this verb is the
	// explicit ask, so it runs the backlog to completion.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "drain",
		Short: "Redact and store every staged transcript for this repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, rootSHA, err := historyStore(cmd)
			if err != nil {
				return err
			}
			res, err := history.Drain(repoRoot, rootSHA, 0)
			if err != nil {
				return err
			}
			if res.Captured == nil {
				res.Captured = []history.Record{}
			}
			for k := range res.Captured {
				res.Captured[k].Path = fsutil.RedactHome(res.Captured[k].Path)
			}
			for k := range res.Failed {
				res.Failed[k].Path = fsutil.RedactHome(res.Failed[k].Path)
			}
			renderErr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				if len(res.Captured) == 0 && len(res.Failed) == 0 {
					fmt.Fprintln(w, "abcd history — nothing staged; nothing to drain")
					return
				}
				for _, r := range res.Captured {
					fmt.Fprintf(w, "stored  %s  redacted secrets=%d home=%d\n",
						termsafe.Sanitize(r.SessionID), r.Secrets, r.HomePaths)
				}
				for _, f := range res.Failed {
					fmt.Fprintf(w, "FAILED  %s  %s\n  raw transcript kept (unredacted): %s\n",
						termsafe.Sanitize(f.SessionID), termsafe.Sanitize(f.Err), termsafe.Sanitize(f.Path))
				}
			})
			if renderErr != nil {
				return renderErr
			}
			// A drain that could not store something must not exit 0: this verb is
			// the remedy the SessionStart notice points at, and a silent success
			// here would leave the user believing the backlog cleared.
			if len(res.Failed) > 0 {
				return fmt.Errorf("history: %d staged transcript(s) could not be stored", len(res.Failed))
			}
			return nil
		},
	})

	// show <session-id-or-filename> — metadata + redacted body of one record.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "show <session-id-or-filename>",
		Short: "Show one stored transcript's metadata and redacted body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, rootSHA, err := historyStore(cmd)
			if err != nil {
				return err
			}
			rec, body, err := history.Read(repoRoot, rootSHA, args[0])
			if err != nil {
				return err
			}
			// Redact the home root out of the absolute stored path in this success
			// envelope (JSON and the text path line) before it is rendered.
			rec.Path = fsutil.RedactHome(rec.Path)
			out := struct {
				history.Record
				Body string `json:"body"`
			}{Record: rec, Body: string(body)}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				// The stored transcript body is untrusted (it may have ingested
				// hostile fetched pages or target-repo files); capture redacts
				// only secrets/home paths, so neutralise terminal-control bytes
				// here before they reach the terminal. SanitizeBlock keeps the
				// transcript's line structure. The metadata fields are validated
				// at write time but not re-validated on the read path, so pass
				// them through too.
				fmt.Fprintf(w, "session:    %s\n", termsafe.Sanitize(rec.SessionID))
				fmt.Fprintf(w, "captured:   %s\n", rec.CapturedAt.Format("2006-01-02T15:04:05Z"))
				fmt.Fprintf(w, "source:     %s\n", termsafe.Sanitize(rec.SourceKind))
				fmt.Fprintf(w, "path:       %s\n", termsafe.Sanitize(rec.Path))
				fmt.Fprintf(w, "redacted:   secrets=%d home=%d\n", rec.Secrets, rec.HomePaths)
				fmt.Fprintln(w, "---")
				fmt.Fprint(w, termsafe.SanitizeBlock(string(body)))
			})
		},
	})

	return historyCmd
}

// repoRootSHA resolves the current repo's root-commit SHA (the history store
// key) via the ahoy detection pass. An empty SHA means cwd is not a git repo
// with commits, which the history verbs cannot key on.
// captureRoot resolves the git working-tree root containing cwd so history
// capture honours the per-repo redaction override (the scanner resolves it at
// <root>/.abcd/config/pii.json, without walking up). Without this, a capture run
// from a subdirectory hands the subdirectory to scanner.New, which finds no
// override there and silently redacts with defaults only (B12). It falls back to
// cwd when git cannot answer (not a repo, git absent) — the scanner then behaves
// exactly as before, so the fallback never regresses a non-git use.
func captureRoot(cwd string) string {
	if top, err := gitutil.Run(cwd, "rev-parse", "--show-toplevel"); err == nil && top != "" {
		return top
	}
	return cwd
}

// historyStore is the shared front-door step for every `history` verb: resolve
// this repo's root-commit SHA and its transcript store, and print the store's
// own out-of-band diagnostics on stderr.
//
// Resolving here is what makes the store exist for a machine that never ran
// `abcd ahoy install` (iss-95), and it is also where a corpus at the legacy
// location is migrated. The notes are printed rather than swallowed for the
// reason rulesRoot prints its own: a resolution that moved a corpus, or that
// declined to honour an opt-in the caller wrote, must not be silent about it.
// stderr, never stdout — the JSON envelope on stdout stays machine-readable.
func historyStore(cmd *cobra.Command) (string, string, error) {
	rootSHA, err := repoRootSHA()
	if err != nil {
		return "", "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	repoRoot := captureRoot(cwd)
	store, err := history.Resolve(repoRoot, rootSHA)
	if err != nil {
		return "", "", err
	}
	for _, n := range store.Notes {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", termsafe.Sanitize(n))
	}
	return repoRoot, rootSHA, nil
}

// rulesRoot resolves the repo root the modular-rules loader (and the shell
// guard, which shares it) must read: the nearest directory holding a .abcd,
// searched from cwd upward but never past the git working tree, and cwd itself
// outside git. rules.Load/LoadBackstop join ".abcd/rules.json" onto the path
// they are given, so handing them a subdirectory silently ignored the per-repo
// overrides AND the kill switch; an unbounded walk instead let a .abcd planted
// above the working tree govern the session (GHSA-vvqc-3mv2-5p49). The
// resolution lives in core (rules.ResolveRoot) because it is behaviour, not
// formatting; this front door only hands it cwd.
// notes carries the resolver's own diagnostics (the ownership refusal, an
// ignored trust declaration) out of band on w — stderr, never the injected
// context — because a resolution that declined to read a repository's own
// configuration must not be silent about it.
func rulesRoot(cwd string, w io.Writer) string {
	res := rules.Resolve(cwd)
	for _, note := range res.Notes {
		fmt.Fprintf(w, "abcd %s\n", note)
	}
	return res.Root
}

func repoRootSHA() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	det, err := ahoy.Detect(cwd)
	if err != nil {
		return "", err
	}
	if det.RootSHA == "" {
		return "", fmt.Errorf("history: cannot resolve the repo's root-commit SHA (not a git repo with commits)")
	}
	return det.RootSHA, nil
}

// Execute runs the root command; main sets the process exit code on error.
// Run builds the command tree, executes it against args, and renders any error
// as a single diagnostic line — the one place that maps a command error to a
// process exit code, so main stays a thin shell. stdout/stderr are injected so
// the whole front door (including its error surface) is testable.
func Run(args []string, stdout, stderr io.Writer) int {
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.Execute()
	if err == nil {
		return 0
	}
	// A command may request a specific exit code (usage errors, the memory-lint
	// curator contract). An empty message means it already rendered its output
	// and only the exit code should propagate.
	code := 1
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		code = coded.ExitCode()
	}
	if msg := scrubPaths(err); msg != "" {
		// An unknown command or flag on a binary that can prove it is stale
		// names the staleness and the way out, on a second line, from disk
		// alone; otherwise cobra's line stands byte-for-byte
		// (iss-2608230943088357, staleusage.go).
		if note := staleUsageNote(root, args, msg); note != "" {
			msg += "\nabcd: " + note
		}
		// Honour --json for the error surface too: a caller that asked for
		// machine output must get a JSON envelope, never raw Go text (iss-29).
		if asJSON, _ := root.PersistentFlags().GetBool("json"); asJSON {
			enc := json.NewEncoder(stderr)
			enc.SetIndent("", "  ")
			_ = enc.Encode(errorEnvelope{Error: msg})
		} else {
			fmt.Fprintln(stderr, "abcd:", msg)
		}
	}
	return code
}

// errorEnvelope is the --json error shape: a single {"error": "..."} object so
// a machine caller can parse a failure the same way it parses a success.
type errorEnvelope struct {
	Error string `json:"error"`
}

// scrubPaths renders err for machine/stderr output with the DEVELOPER-IDENTITY
// portion of any local path removed. cli.Run routes every command error through
// the --json envelope and the stderr line, and an identity-bearing path reaches
// that surface three ways: an os.PathError/os.LinkError embeds one in Error();
// core fmt-formats one via %s (e.g. capture's ledger-path guards); a custom error
// type renders one (e.g. history's home-rooted StorePathError). All three are
// handled (iss-76 — the identity-scrub generalisation of the one branch iss-29
// fixed):
//
//   - the two roots that carry developer identity — the working directory and the
//     home directory — are redacted to "." and "~" wherever they appear, catching
//     fmt-formatted and custom-error-type paths a typed walk cannot see;
//   - any remaining absolute path embedded by os.PathError/os.LinkError (e.g. a
//     path argument outside both roots) is reduced to its base name.
//
// This is NOT a universal absolute-path scrub: a verb that echoes a user-supplied
// absolute path lying outside both roots (e.g. `memory ingest /tmp/x`) still
// surfaces it — that path carries no developer identity, and sanitising such
// verb-level echoes is tracked separately (iss-81). Scrubbing here rather than by
// regex is deliberate: this error surface also carries URLs (fetch failures) that
// an absolute-path regex would mangle.
//
// The root redaction itself is fsutil.RedactRoot: it moved out of this file when
// the ahoy install receipt had to redact the same two roots (iss-177), so what
// counts as a developer-identity root is stated once and read by both the error
// surface here and the receipt in core.
func scrubPaths(err error) string {
	msg := err.Error()
	if cwd, e := os.Getwd(); e == nil {
		msg = fsutil.RedactRoot(msg, cwd, ".")
	}
	if home, e := os.UserHomeDir(); e == nil {
		msg = fsutil.RedactRoot(msg, home, "~")
	}
	for _, p := range embeddedPaths(err) {
		if filepath.IsAbs(p) {
			msg = strings.ReplaceAll(msg, p, filepath.Base(p))
		}
	}
	return msg
}

// embeddedPaths collects the filesystem paths carried by os.PathError/os.LinkError
// anywhere in err's Unwrap chain, including errors.Join fan-out.
func embeddedPaths(err error) []string {
	var paths []string
	var walk func(error)
	walk = func(e error) {
		for e != nil {
			switch t := e.(type) {
			case *os.PathError:
				paths = append(paths, t.Path)
			case *os.LinkError:
				paths = append(paths, t.Old, t.New)
			}
			switch u := e.(type) {
			case interface{ Unwrap() error }:
				e = u.Unwrap()
			case interface{ Unwrap() []error }:
				for _, sub := range u.Unwrap() {
					walk(sub)
				}
				return
			default:
				return
			}
		}
	}
	walk(err)
	return paths
}

// render writes v as indented JSON when asJSON is set, otherwise delegates to
// the text renderer. Keeping this one helper is what makes every command's
// --json behaviour uniform.
func render(w io.Writer, asJSON bool, v any, text func(io.Writer)) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	text(w)
	return nil
}

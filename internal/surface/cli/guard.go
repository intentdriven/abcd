package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// maxGuardStdinBytes caps the candidate command read from stdin. A command line
// is short; anything larger is not one, and an unbounded read on a hook's stdin
// would let a pathological payload stall the session it is guarding.
const maxGuardStdinBytes = 1 << 20 // 1 MiB

// newGuardCommand builds the `guard` sub-tree: the two front doors onto
// internal/core/guard's one decision. `check` is the human/scriptable verb;
// `hook` is the host adapter that a PreToolUse-style hook runs before a shell
// command executes.
//
// Both are thin: core decides, this file only formats the decision and maps it
// onto an exit status. The two differ in exactly one respect, and it is the whole
// point of the split — `check` treats a guard it cannot evaluate as a FAULT
// (loud, non-zero, so a script never mistakes silence for clearance), while
// `hook` treats the same condition as fail-OPEN (loud, exit 0, so a broken guard
// can never brick a session). spc-16, "Fail-open-loud and health".
func newGuardCommand(asJSON *bool) *cobra.Command {
	guardCmd := &cobra.Command{
		Use:   "guard",
		Short: "Check a shell command against the hazard registry before it runs",
		Args:  failOpenNoArgs,
		RunE:  helpRunE,
	}

	var command string
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Decide whether a candidate shell command is safe to run",
		Long: "Evaluates one candidate command line against the hazard registry — the\n" +
			"bundled defaults merged with this repo's `.abcd/guard.json` — and reports\n" +
			"allow, warn, or block. A blocker exits 1 and names the safe successor; a\n" +
			"warn exits 0 with the warning rendered; an allow exits 0. A guard that\n" +
			"cannot be evaluated at all (an unparsable command line, a malformed\n" +
			"registry) exits 2, so a caller never reads silence as clearance. Unparsable\n" +
			"means an unterminated quote in COMMAND text, which no shell runs either;\n" +
			"an unterminated quote inside a here-document body is document text and is\n" +
			"not one. Grammar a shell does run gets a verdict instead: a trailing\n" +
			"backslash is read as bash reads it, and a here-document whose delimiter\n" +
			"line never comes is a block, because the rest of the input may be commands\n" +
			"the guard did not check.\n\n" +
			"Matching is shell-token-aware and applies in command position only, so a\n" +
			"hazard named inside a quoted argument never fires.\n\n" +
			"The guard is a MISTAKE FILTER, not a security boundary. It catches a hazard\n" +
			"typed by accident or reached through an ordinary wrapper — the cases that\n" +
			"actually cost people work. It does not withstand an author trying to get a\n" +
			"command past it, and it does not claim to: the set of programs that launch\n" +
			"another program is open-ended, so no list inside this binary can enumerate\n" +
			"it, and a repository extends that set with one line in a Makefile. Anything\n" +
			"that needs an enforced boundary needs a control at the execution layer — a\n" +
			"sandbox, a permission system, a restricted shell — with this guard in front\n" +
			"of it to teach, never in place of it.\n\n" +
			"An allow means no registry entry matched — it is never a statement that a\n" +
			"command is safe. A hazard behind a launcher the guard does not recognise is\n" +
			"a WARN naming the entry it matched, rather than an allow, because the guard\n" +
			"cannot tell whether that program runs the rest of the line. What an\n" +
			"allow still does not see is a hazard that never reaches command position at\n" +
			"all: one launched through a known\n" +
			"wrapper carrying a value-taking flag the guard does not name (`sudo -u bob\n" +
			"<hazard>` is seen; the bundled short form `sudo -Hu bob <hazard>` reaches\n" +
			"only the warn, not the entry that names it),\n" +
			"one whose API path an entry names by its ROOT segment but the host serves\n" +
			"under a prefix (a GitHub Enterprise Server install mounts the same endpoints\n" +
			"under `/api/v3/`; the api.github.com URL form IS read), a bare `$VAR` inside\n" +
			"an interpreter payload (an execute-a-string payload IS read — `sh -c`,\n" +
			"`env -S`; one the guard cannot read is warned or, for `env -S`, blocked),\n" +
			"a hazard inside a top-level command substitution (`$(…)` and\n" +
			"backticks are both followed into command position),\n" +
			"a hazard inside a NON-shell interpreter's payload (`python -c`, `perl -e`) —\n" +
			"one opaque token the tokenizer cannot read, today a silent allow (a warn for\n" +
			"it is a recorded design target, not yet raised),\n" +
			"or a dangerous form no entry describes. Coverage is what the registry\n" +
			"names.\n\n" +
			"The candidate comes from --command, or from stdin when the flag is absent.\n" +
			"Prefer stdin for a command line you did not type yourself: the shell expands\n" +
			"a double-quoted --command argument before this verb starts, so a candidate\n" +
			"containing a command substitution would run at check time. A quoted-delimiter\n" +
			"heredoc (`abcd guard check <<'EOF'` ... `EOF`) passes it through untouched.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			candidate, err := guardCandidate(cmd, command)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("guard check: %s", scrubPaths(err))}
			}
			reg, err := loadGuardRegistry(cmd.ErrOrStderr())
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("guard check: %s", scrubPaths(err))}
			}
			// A disabled registry evaluated nothing, so there is no answer to
			// render — and a bare `allow` here is indistinguishable from a real
			// clearance to the CI job or script using this verb as a gate. Same
			// category as an unparsable command line: the question could not be
			// answered, so say that rather than answer it wrongly.
			if reg.Disabled {
				return &exitError{Code: 2, Msg: fmt.Sprintf(
					"guard check: the hazard registry is switched off in %s; nothing was checked", guard.RepoRelPath)}
			}
			dec, err := reg.Check(candidate)
			if err != nil {
				return &exitError{Code: 2, Msg: fmt.Sprintf("guard check: %s", scrubPaths(err))}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, dec, func(w io.Writer) {
				writeGuardDecision(w, dec)
			}); rerr != nil {
				return rerr
			}
			// A refusal is an EXPECTED outcome, not a fault: the render is the
			// output and the exit code the only extra signal (exit 1), matching
			// how `embark from` and `intent ready` report a refusal.
			if dec.Verdict == guard.VerdictBlock {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	checkCmd.Flags().StringVar(&command, "command", "", "the candidate command line (default: read from stdin)")
	guardCmd.AddCommand(checkCmd)

	guardCmd.AddCommand(newGuardHookCommand())
	return guardCmd
}

// newGuardHookCommand builds `guard hook`: the adapter between a host's
// pre-tool-use hook payload and the core decision.
//
// The mapping is the host's own convention for a pre-execution hook: exit 2 is
// the blocking status and the hook's stderr is what the host replays to the
// agent, so the refusal — successor and why — goes to stderr and nowhere else. An
// allow exits 0 and is silent; a warn exits 1 (loud, non-blocking) so its message
// is not discarded — neither exit status blocks, so the command still runs.
//
// Every path that is NOT a decision — an unreadable payload, a tool that is not
// the shell, a command the tokenizer cannot split, a registry that will not
// load — allows the command and says so unmissably. This is the fail-open-loud
// contract: a guard that cannot answer must never be the reason a session stops,
// and must never be silently absent either (itd-103 AC 1).
func newGuardHookCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "hook",
		Short: "Host pre-tool-use adapter: decide a shell command from a hook payload",
		Long: "Reads a host pre-tool-use hook payload on stdin and evaluates its shell\n" +
			"command against the hazard registry. A blocker exits with the host's\n" +
			"blocking status and puts the safe successor and the plain-language why on\n" +
			"stderr, which is the channel the host replays to the agent. A warn and an\n" +
			"allow both let the command run.\n\n" +
			"Anything the adapter cannot turn into a decision — an unreadable payload, a\n" +
			"tool call that is not a shell command, an unparsable command line, a\n" +
			"registry that will not load — allows the command and warns loudly on\n" +
			"stderr. A guard that cannot answer never stops a session, and is never\n" +
			"silently absent. Unparsable means an unterminated quote in COMMAND text,\n" +
			"which no shell runs either — a quote inside a here-document body is\n" +
			"document text and is not one. A trailing backslash and a here-document with\n" +
			"no delimiter line are grammar a shell does run, so each gets a verdict —\n" +
			"the backslash is read as bash reads it, the unterminated document blocks.\n\n" +
			"A host whose shell tool takes a per-call working directory passes it as\n" +
			"tool_input.workdir. It is resolved against the session directory, and a\n" +
			"command whose workdir is an existing directory in another repository is\n" +
			"checked against that repository's registry as well as the session's; the\n" +
			"stricter verdict wins, so the workdir's registry can add a hazard and never\n" +
			"remove one. The workdir is never read as a cd: the one host that has the\n" +
			"field fails the call when the directory is missing, so no failed-cd hazard\n" +
			"exists. A workdir that is not a string, or holds a NUL byte, a control\n" +
			"character or invalid UTF-8, or is over 4096 bytes, is refused with the\n" +
			"blocking status and the reason.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// failOpen is the single exit for every non-decision path, so the
			// contract cannot be violated by forgetting one of them.
			// Exit 1, not 0, is what makes this LOUD. A pre-tool-use hook that
			// exits 0 has its stderr discarded, so the warning would exist and
			// nobody would ever see it; a non-zero, non-blocking status both lets
			// the command run and puts the warning in front of a human. Only the
			// blocking status (2) stops anything.
			failOpen := func(format string, a ...any) error {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"abcd guard: NOT CHECKED — "+format+". This command runs UNGUARDED.\n", a...)
				return &exitError{Code: 1}
			}

			// One byte past the cap, so an over-cap payload names the cap
			// instead of being truncated and misreported as unreadable JSON
			// (iss-201; guardCandidate is the same probe on the check verb).
			raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxHookStdinBytes+1))
			if err != nil {
				return failOpen("the hook payload could not be read (%v)", err)
			}
			if len(raw) > maxHookStdinBytes {
				return failOpen("the hook payload is over the %d-byte cap; it was discarded unparsed", maxHookStdinBytes)
			}
			var in guardHookInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return failOpen("the hook payload is not readable JSON (%v)", err)
			}
			// The manifest scopes this hook to the shell tool, so a different tool
			// name means the wiring is wrong — worth saying rather than ignoring.
			if !strings.EqualFold(in.ToolName, "Bash") {
				return failOpen("the payload is a %q tool call, not a shell command", termsafe.Sanitize(in.ToolName))
			}
			candidate := in.ToolInput.Command
			if strings.TrimSpace(candidate) == "" {
				return failOpen("the payload carries no command to check")
			}

			// The payload names the session's working directory; fall back to the
			// process cwd only when it does not, exactly as the other hooks do.
			cwd := in.Cwd
			if cwd == "" {
				if wd, err := os.Getwd(); err == nil {
					cwd = wd
				}
			}
			sessionRoot := rulesRoot(cwd, cmd.ErrOrStderr())
			reg, err := guard.Load(sessionRoot)
			// A repo-layer error is fail-SAFE, not fail-open: guard.Load returns the
			// bundled defaults alongside the error, so the built-in hazards stay
			// armed even though the repo's own overrides were dropped. We check
			// against that bundled registry rather than running unguarded, and
			// announce the dropped repo layer loudly so a human learns their
			// committed guard config is broken (iss-2608261551087492). Only an
			// EMPTY registry — the bundled layer itself somehow unavailable, which
			// cannot happen with an embedded default — is the remaining fail-open.
			repoDropped := false
			if err != nil {
				if len(reg.Entries) == 0 {
					return failOpen("the hazard registry did not load (%s)", scrubPaths(err))
				}
				repoDropped = true
				fmt.Fprintf(cmd.ErrOrStderr(),
					"abcd guard: the repo %s did not load (%s); its overrides are DROPPED, but the bundled hazards remain armed.\n",
					guard.RepoRelPath, scrubPaths(err))
			}
			// A disabled registry allows everything, which makes it an unguarded
			// session — and it is the CHEAPEST one to reach: the other unguarded
			// states need a broken install, this one needs a single file write.
			// It is not a fault (someone chose it, and the choice sits in a file a
			// reviewer can read), but it must never pass for protection.
			if reg.Disabled {
				return failOpen("the hazard registry is switched off in %s", guard.RepoRelPath)
			}
			// The host's per-call working directory, when it sent one. A value no
			// directory could be named by is REFUSED, not failed open: the
			// registry is armed and the command is readable, so the guard can
			// answer — what it cannot tell is where the command would run, and
			// the model that wrote the value can resend it plain.
			wd, err := hookWorkdir(in.ToolInput.Workdir, cwd)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"Blocked by the abcd guard (tool_input.workdir): the working directory this call names is refused — %s. The guard cannot tell where the command would run. Run instead: the same command with workdir omitted, or set to a plain directory path.\n",
					termsafe.Sanitize(scrubPaths(err)))
				return &exitError{Code: 2}
			}
			dec, err := reg.Check(candidate)
			if err != nil {
				return failOpen("the command line could not be parsed (%s)", scrubPaths(err))
			}
			// The command runs in the workdir, so the registry of the repository
			// it runs in names its hazards too. Only an existing directory has
			// one: the probed host fails a call whose workdir it cannot enter.
			// The session's decision is the floor (guard.Strictest), so a
			// registry the model chose by choosing the workdir can add a hazard
			// and never subtract one.
			if wd.Exists {
				if root := rulesRoot(wd.Path, cmd.ErrOrStderr()); root != sessionRoot {
					wreg, werr := guard.Load(root)
					if werr != nil && len(wreg.Entries) > 0 {
						repoDropped = true
						fmt.Fprintf(cmd.ErrOrStderr(),
							"abcd guard: the working directory's %s did not load (%s); its overrides are DROPPED, but the bundled hazards remain armed.\n",
							guard.RepoRelPath, scrubPaths(werr))
					}
					if !wreg.Disabled && len(wreg.Entries) > 0 {
						if wdec, cerr := wreg.Check(candidate); cerr == nil {
							dec = guard.Strictest(dec, wdec)
						}
					}
				}
			}

			switch dec.Verdict {
			case guard.VerdictBlock:
				// Exit 2 is the host's blocking status; stderr is the message it
				// replays to the agent, so the refusal itself is the lesson.
				fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(dec.Message))
				return &exitError{Code: 2}
			case guard.VerdictWarn:
				// Exit 1, not 0, is what makes a warn LOUD on a pre-tool-use hook.
				// Exit 0 has the hook's stderr DISCARDED — the same reason failOpen
				// uses exit 1 — so a warn that returned nil would write a message
				// nobody ever sees and run as if allowed (iss-231). A non-zero,
				// non-blocking status both lets the command run and surfaces the
				// warning. Only the blocking status (2) stops anything.
				fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(dec.Message))
				return &exitError{Code: 1}
			default:
				// Allow. Normally silent and cheap — but when the repo layer was
				// dropped, exit 0 would DISCARD the stderr drop notice above, so a
				// human would never learn their guard config is broken. Exit 1
				// (loud, non-blocking) is the one channel that lets the command run
				// while keeping the notice in front of a human, exactly as a warn
				// does.
				if repoDropped {
					return &exitError{Code: 1}
				}
				return nil // allow: silent, and cheap
			}
		},
	}
}

// guardHookInput is the subset of the host's pre-tool-use payload the adapter
// reads. Unknown fields are ignored: the host owns this schema and adds to it.
type guardHookInput struct {
	Cwd       string `json:"cwd"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
		// Workdir is the per-call working directory a host's shell tool may
		// carry beside the command (opencode's bash tool names it `workdir`).
		// It is kept raw so a value of the wrong JSON type is refused as a
		// malformed workdir rather than failing the whole payload open, which
		// would run the command unchecked.
		Workdir json.RawMessage `json:"workdir"`
	} `json:"tool_input"`
}

// hookWorkdir decodes and resolves the payload's per-call working directory
// against the session directory. Absent and null are no workdir; any other
// non-string value, and any string guard.ResolveWorkdir refuses, is an error the
// hook turns into a refusal.
func hookWorkdir(raw json.RawMessage, sessionDir string) (guard.Workdir, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return guard.Workdir{}, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return guard.Workdir{}, fmt.Errorf("%w: it is not a string", guard.ErrMalformedWorkdir)
	}
	return guard.ResolveWorkdir(sessionDir, s)
}

// guardCandidate resolves the command to check: the flag when given, otherwise
// stdin. An empty candidate from either channel is a usage error, never an
// implicit allow — "nothing to check" and "checked and clear" must not look the
// same to a caller.
func guardCandidate(cmd *cobra.Command, flag string) (string, error) {
	// Whether the flag was GIVEN decides the channel, not whether its value is
	// non-empty: `--command ""` is an empty question, and falling through to stdin
	// there would leave the verb waiting on a terminal that is never going to
	// answer.
	if cmd.Flags().Changed("command") {
		if strings.TrimSpace(flag) == "" {
			return "", fmt.Errorf("--command is empty; pass a command line to check")
		}
		return flag, nil
	}
	// Read ONE byte past the cap so an overflow is detectable. Truncating to the
	// cap and answering on the prefix is the quietest possible way to hand out a
	// clearance nobody earned: pad a command past the limit and the tail — the
	// part that matters — is never tokenised.
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxGuardStdinBytes+1))
	if err != nil {
		return "", fmt.Errorf("reading the candidate command from stdin failed: %w", err)
	}
	if len(raw) > maxGuardStdinBytes {
		return "", fmt.Errorf("the candidate command is too long (over %d bytes); it was not checked", maxGuardStdinBytes)
	}
	candidate := strings.TrimRight(string(raw), "\r\n")
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("no command to check: pass --command \"<command line>\" or pipe one on stdin")
	}
	return candidate, nil
}

// loadGuardRegistry resolves the repo root the same way the modular-rules loader
// does — the nearest .abcd directory inside the git working tree, never one
// planted above it — so `.abcd/guard.json` is honoured from any nested working
// directory, kill switch included, and only the repo's own file can throw it.
func loadGuardRegistry(w io.Writer) (guard.Registry, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return guard.Registry{}, err
	}
	return guard.Load(rulesRoot(cwd, w))
}

// guardHealthLine renders ahoy's one-line guard-health verdict. A guard that
// cannot check anything fails OPEN inside the session, which is invisible there
// by design — so this line is the place a broken guard becomes visible, and it
// says which of the three parts is missing rather than a bare "unhealthy".
func guardHealthLine(h ahoy.GuardHealth) string {
	if h.Healthy() {
		state := fmt.Sprintf("armed (%d hazards)", h.Entries)
		if h.RepoOverridesDropped {
			// The fail-safe load: the repo's .abcd/guard.json does not load, so
			// its overrides are dropped while the bundled hazards stay armed.
			// Commands are still checked — but the line must not read as a clean
			// bill of health while a committed file is broken.
			state = fmt.Sprintf("armed (%d bundled hazards) — %s does not load, repo overrides dropped", h.Entries, guard.RepoRelPath)
		}
		if h.Disabled {
			// Loadable and wired, but switched off in .abcd/guard.json. Not a
			// fault, and not something to report as protection either. The file is
			// read from the working tree, so this can be true before anyone has
			// reviewed the edit that made it true (iss-147).
			state = "OFF — disabled in " + guard.RepoRelPath
		}
		return state
	}
	// Without a plugin root the manifest was never opened and the binary was never
	// looked for. Saying "hook not installed" here would accuse an install that
	// may be perfectly armed, so the line reports what is unknown instead.
	if !h.PluginRootResolved {
		return "UNKNOWN — " + h.Detail
	}
	var missing []string
	if !h.HookInstalled {
		missing = append(missing, "hook not installed")
	}
	if !h.BinaryReachable {
		missing = append(missing, "binary unreachable")
	}
	if !h.RegistryLoadable {
		// Not the repo file — the fail-safe load keeps bundled hazards armed past
		// that (reported above as dropped overrides). This is no registry at all.
		missing = append(missing, "no hazard registry loaded")
	}
	return "NOT ARMED — " + strings.Join(missing, ", ") + " (shell commands run unchecked)"
}

// writeGuardDecision renders one decision as the human report. Entry text is
// registry-supplied (a per-repo file can add entries), so it is sanitised before
// it reaches a terminal.
func writeGuardDecision(w io.Writer, dec guard.Decision) {
	fmt.Fprintf(w, "abcd guard — %s\n", dec.Verdict)
	if dec.Verdict == guard.VerdictAllow {
		return
	}
	fmt.Fprintf(w, "  entry:       %s (%s)\n", termsafe.Sanitize(dec.EntryID), termsafe.Sanitize(dec.Tier))
	fmt.Fprintf(w, "  why:         %s\n", termsafe.Sanitize(dec.Why))
	fmt.Fprintf(w, "  run instead: %s\n", termsafe.Sanitize(dec.Successor))
	// Matches is ordered blockers, warns, synthetics — NOT winner-first: a
	// synthetic block over registry warns appends the winner last, so the
	// non-winners are selected by id rather than by position (iss-346).
	var also []string
	for _, id := range dec.Matches {
		if id != dec.EntryID {
			also = append(also, id)
		}
	}
	if len(also) > 0 {
		fmt.Fprintf(w, "  also matched: %s\n", termsafe.Sanitize(strings.Join(also, ", ")))
	}
}

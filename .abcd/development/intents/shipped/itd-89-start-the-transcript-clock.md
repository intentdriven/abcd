---
id: itd-89
slug: start-the-transcript-clock
spec_id: spc-4
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-3]
severity: critical
impact: additive
related_adrs: [adr-29]
---

# Capture the Session You Just Had, Because You Cannot Capture It Later

## Press Release

> **abcd now stores your session transcripts as they happen.** A `SessionEnd` hook fires when a session finishes, and abcd redacts the transcript and files it in the local per-repo store — no flag to remember, no command to run. `abcd history list` shows what it has. Everything stays on your machine, secrets and home paths stripped on write, and a session that ends twice is stored once.
>
> "The reason I do this work is that the *why* lives in the conversation, not the commit," said Maya, autonomous-development practitioner. "For a year my rationale evaporated when the window closed. Now it doesn't. The part I did not expect: it captures nothing when it cannot capture safely, and it never once got in my way."

## Why This Matters

**This is the only irreversible thing on the board.** `internal/core/history` is built. `history.Capture` redacts on write, is idempotent, and is fail-closed. It is **called by nothing** — `hooks/hooks.json` wires `UserPromptSubmit`, `SessionStart` and `PreCompact`, and no session-termination hook exists. `~/.abcd/` has never been created. **Zero transcripts.**

Every other gap in abcd can be closed later at the cost of the work. This one cannot: **a session that ends without being captured is gone.** No future feature, however good, can reconstruct a conversation nobody stored. The corpus can only start accruing from the moment the hook is wired — so the cost of not shipping it is not "a delay", it is a permanent hole in the record that grows every day it is not shipped.

The lifeboat's Pass B — mining chat for the rationale nobody wrote down — has no corpus and cannot get one retroactively (adr-35, itd-88). It ships as a declared exemption until this does.

The verb has to be new. **`history capture` cannot be wired to a session hook**: reading from stdin it *requires* `--session <id>`, and a `SessionEnd` hook delivers its session id inside a JSON payload, not as a flag.

**The event is `SessionEnd`, not `Stop`.** `Stop` fires once per assistant *turn*, and Claude Code's transcript file grows through the session, so a `Stop`-wired capture would store a fresh, larger superset of the same session every turn — `history.Capture`'s sha256 dedup only collapses byte-identical re-captures, which never happens on a live transcript. `SessionEnd` fires once when the session terminates, and by the harness contract its exit code and stdout are ignored — a perfect fit for a fail-closed, non-blocking side-effect hook. This is the one place the milestone deviates from its plan's literal wording, on evidence from the harness docs.

## What's In Scope

- **`abcd hook session-end`** — a hidden, operator-internal verb following the shipped `hook prompt-router` pattern. Reads the `SessionEnd` payload from stdin, takes `session_id` and `transcript_path`, and calls `history.Capture` (which already redacts).
- **A `SessionEnd` entry in `hooks/hooks.json`** — without this the verb is dead code and the corpus still does not accrue.
- **Fail-closed, always exit 0, never blocks the host.** A malformed payload, an absent transcript, a hostile session id, or a repo with no root commit each capture nothing, log out of band, and exit 0.
- **Idempotent re-capture** — a SessionEnd hook can fire more than once per session; the corpus must not grow a duplicate each time.

## What's Out of Scope

- **Retrieval, search, or synthesis over the corpus.** This intent starts the clock; reading the corpus is later work (Pass B, itd-36).
- **Importing historical sessions.** There is nothing to import — that is exactly the loss this intent stops compounding.
- **Changing the redaction engine.** `history.Capture` already redacts two-stage and fail-closed; this intent calls it, it does not touch it.
- **Any user-facing surface.** `hook` is operator-internal wiring, not a `/abcd:` command.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a repo where abcd is installed, **when** a session ends and the `SessionEnd` hook fires, **then** the transcript is redacted and stored, and `abcd history list` shows the record — with no flag passed and no command typed by the user.
- **Given** a session that ran for many turns (so its transcript grew), **when** it ends, **then** the store holds **exactly one** record for that session — capture is wired to `SessionEnd` (once per session), not `Stop` (once per turn), so a long session does not leave a pile of growing supersets.
- **Given** the same transcript is offered twice (a SessionEnd hook can fire more than once for one session), **when** the second capture runs, **then** the store holds **one** record, not two.
- **Given** a malformed payload, an absent `transcript_path`, a `transcript_path` that is not a regular file, a session id that is not `[A-Za-z0-9._-]+`, or a cwd that is not a git repo with commits, **when** the hook runs, **then** it **writes nothing, exits 0, and reports the reason on stderr** — a SessionEnd hook that errors or hangs wedges the user's session, which is strictly worse than a missed transcript.
- **Given** a `transcript_path` naming a FIFO or a device node, **when** the hook opens it, **then** the open does **not block** — the hook must never hang the session it is ending.
- **Given** any capture, **when** the hook runs, **then** it writes **nothing to stdout** — a SessionEnd hook's stdout is not a channel to the model; diagnostics go to stderr, out of band, as the prompt-router already does.
- **Given** a transcript containing a secret or an absolute home path, **when** it is captured, **then** the stored record is redacted (the existing two-stage, fail-closed pass), and if a hard-fail span survives redaction **no file is written at all**.

## Prior Art

- **[adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)** decided the native local redacted transcript corpus. `internal/core/history` implements it. This intent is the missing *trigger*: the store has existed, correct and unused, with no caller.
- **itd-3 (rules loader, shipped)** established the pattern this follows exactly: a hidden Go subcommand invoked from `hooks/hooks.json`, fail-closed, non-blocking, exiting 0 whatever happens, with diagnostics out of band on stderr. `hook session-end` is a third entrypoint in the same `hook` sub-tree — no new mechanism is invented here.
- **[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md) / itd-88** name the cost this intent stops: Pass B has no corpus and cannot get one retroactively, so it ships as a declared exemption until this lands. This intent is sequenced *first* in that plan for exactly that reason.
- **itd-59 (autonomous-worker transcript capture, draft)** is the adjacent case — capturing transcripts of unattended workers. It is not this: this intent covers the ordinary interactive session, which is the corpus that is being lost today.
- The `--pages-json` seam on `memory.Distiller` and `intent review ingest --verdict-json` are the shipped precedents for host-supplied data crossing into the core; the SessionEnd payload is the same shape of trust boundary and is validated at the same boundary.

## Open Questions

- The 64 MiB transcript cap: generous for a JSONL session log, chosen so a pathological file cannot stall the redaction pass inside a SessionEnd hook. If a real session ever exceeds it, the capture is refused — and the refusal is logged, so we would see it. Is refusing right, or should an over-cap transcript be captured in part? (Capturing in part breaks the sha256 idempotency, so refusal is the conservative default.)
- Should the hook also fire on `SubagentStop`? Subagent transcripts are a different corpus with a different signal density; deliberately not decided here. (`SubagentStop`, like `Stop`, fires per subagent-turn, so it would carry the same growing-superset problem `SessionEnd` sidesteps.)
- `SessionEnd` does **not** fire on a hard crash or `SIGKILL` (only on orderly termination — `/exit`, `/clear`, logout, EOF). A session killed uncleanly is therefore not captured. The alternative — wiring `Stop` for per-turn crash-resilience — trades that for N growing records per session and would need `Capture` to dedup on `session_id` rather than raw sha256 (a change to shipped core with its own tests). Deferred as a deliberate trade-off; the common case (orderly exit) is covered, and the rarer crash case is recorded rather than silently assumed away.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-791c91982a80 -->
Fidelity review — receipt rcp-791c91982a80 (verifier intent-fidelity-reviewer claude-fable-5).

Provenance: intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86 · prompt_hash sha256:82804d99f5a1de14cee029c6d45847ab447907d09bdd9b1610791dfd28d15143
Input attestations: diff:c3aeab7b08d0e8b9154bf47d70411f281ac5c72e@-;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: hooks.json wires SessionEnd to `abcd hook session-end`, which calls history.Capture (redact-on-write) with no user flag; `abcd history list` renders history.List; TestHookSessionEndCapturesTranscript proves one stored, listable record (suite passes: `go test -run TestHookSessionEnd ./internal/surface/cli/` ok)
  evidence: hooks/hooks.json:16 — "{\"type\": \"command\", \"command\": \"\\\"$CLAUDE_PLUGIN_ROOT/abcd\\\" hook session-end\"}"
  evidence: internal/surface/cli/cli.go:890 — "res, err := history.Capture(captureRoot(cwd), det.RootSHA, in.SessionID, raw, \"native\")"
  evidence: internal/surface/cli/cli.go:2111 — "Use:   \"list\""
  evidence: internal/surface/cli/hook_session_end_test.go:93 — "if len(recs) != 1 {"
- ac-2 — MET: hooks.json contains a SessionEnd entry and no Stop entry anywhere, and TestHookSessionEndOneRecordPerSession asserts a multi-turn grown transcript leaves exactly one record
  evidence: hooks/hooks.json:15 — "\"SessionEnd\": ["
  evidence: internal/surface/cli/hook_session_end_test.go:148 — "if len(recs) != 1 {"
  evidence: internal/surface/cli/cli.go:840 — "Wired to SessionEnd, NOT Stop."
- ac-3 — MET: Capture's sha256 dedup returns Wrote=false on an identical re-offer, and TestHookSessionEndIsIdempotent runs the same payload twice through the hook and asserts one record
  evidence: internal/core/history/history.go:121 — "if r.SourceSHA256 == sourceSHA && r.SessionID == sessionID && r.SourceKind == kind {"
  evidence: internal/surface/cli/hook_session_end_test.go:119 — "if len(recs) != 1 {"
- ac-4 — MET: every guard degrades via warn() which prints to stderr and returns nil (exit 0); TestHookSessionEndNeverBlocksTheHost table covers malformed json, empty payload, missing transcript_path, absent file, directory (non-regular), hostile session id ../../escape (rejected by history's ^[A-Za-z0-9._-]+$), and non-repo cwd (RootSHA=="" guard), asserting non-empty stderr and zero records with runHook failing on any non-zero exit
  evidence: internal/surface/cli/cli.go:866 — "return nil // never non-zero: a Stop hook must not wedge the session"
  evidence: internal/surface/cli/cli.go:883 — "if err != nil || det.RootSHA == \"\" {"
  evidence: internal/core/history/store.go:25 — "var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)"
  evidence: internal/surface/cli/hook_session_end_test.go:195 — "if strings.TrimSpace(errlog) == \"\" {"
  evidence: internal/surface/cli/cli_test.go:107 — "if err := cmd.Execute(); err != nil {"
- ac-5 — MET: readTranscript opens O_RDONLY|O_NONBLOCK then rejects non-regular files, and TestHookSessionEndDoesNotBlockOnIrregularFiles exercises a writer-less FIFO and /dev/null under a 10-second watchdog, asserting no hang, exit 0, stderr reason, zero records
  evidence: internal/surface/cli/cli.go:967 — "f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)"
  evidence: internal/surface/cli/cli.go:977 — "if !st.Mode().IsRegular() {"
  evidence: internal/surface/cli/hook_session_end_test.go:360 — "case <-time.After(10 * time.Second):"
- ac-6 — MET: every diagnostic in the session-end RunE goes to cmd.ErrOrStderr() and nothing writes to stdout; TestHookSessionEndWritesNothingToStdout asserts stdout == "" on a successful capture
  evidence: internal/surface/cli/cli.go:865 — "fmt.Fprintf(cmd.ErrOrStderr(), \"abcd history: \"+format+\"\\n\", a...)"
  evidence: internal/surface/cli/hook_session_end_test.go:315 — "if stdout != \"\" {"
- ac-7 — MET: the hook path triggers the existing two-stage redaction (TestHookSessionEndRedactsOnThisPath asserts a ghp_ token and a home path are masked in the stored record), and a hard_fail span surviving stage-two refuses the write entirely (TestHookSessionEndRefusesResidualHardFail: "refusing to write" on stderr, zero records)
  evidence: internal/core/history/history.go:68 — "history: redaction left %d hard_fail span(s) unresolved [%s]; refusing to write"
  evidence: internal/surface/cli/hook_session_end_test.go:248 — "if strings.Contains(string(stored), token) {"
  evidence: internal/surface/cli/hook_session_end_test.go:433 — "if !strings.Contains(errlog, \"refusing to write\") {"

Gap audit:
- honoured:
  - SessionEnd hook wired so the corpus accrues with no flag and no user command
    evidence: hooks/hooks.json:16 — "\\\"$CLAUDE_PLUGIN_ROOT/abcd\\\" hook session-end"
  - fail-closed, always exit 0, never blocks the host — every failure path logs to stderr and returns nil
    evidence: internal/surface/cli/cli.go:866 — "return nil // never non-zero"
  - a session that ends twice is stored once
    evidence: internal/core/history/history.go:122 — "return CaptureResult{Record: r, Wrote: false}, nil"
  - secrets and home paths stripped on write, refusal on residual hard-fail
    evidence: internal/core/history/history.go:156 — "Stage two — verify. Re-scan the redacted text"
  - 64 MiB cap refuses an over-cap transcript whole rather than truncating (open question resolved to refusal)
    evidence: internal/surface/cli/cli.go:958 — "const maxTranscriptBytes = 64 << 20 // 64 MiB"
    evidence: internal/surface/cli/hook_session_end_test.go:396 — "if !strings.Contains(errlog, \"cap\") {"
- diverged:
  - "no flag to remember, no command to run" — capture still requires a one-time store bootstrap (`ahoy install` creates the transcripts dir); an uninstalled repo captures nothing, mitigated by a delivered SessionStart warning (iss-95) rather than by auto-creation
    evidence: internal/surface/cli/hook_session_end_test.go:53 — "Capture requires the transcripts dir to exist already (abcd install creates it)."
    evidence: internal/surface/cli/cli.go:944 — "Run `/abcd:ahoy install` (or `abcd ahoy install`) to start recording."
- missing:
  - "stores your session transcripts as they happen" — a session ended by hard crash or SIGKILL fires no SessionEnd and is never captured; declared as a deliberate trade-off in the intent, not closed by the delivery
    evidence: .abcd/development/intents/shipped/itd-89-start-the-transcript-clock.md:71 — "`SessionEnd` does **not** fire on a hard crash or `SIGKILL`"

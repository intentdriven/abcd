---
id: itd-59
slug: autonomous-worker-transcript-capture
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-27, adr-29]
routed_from: []
prd_path: null
severity: minor
builds_on: [itd-2609201916151817]
---

# Every Autonomous Run Pass Leaves the Same Durable, Queryable Transcript an Interactive Session Does

## Press Release

> **abcd captures the instruction-set and reasoning of every autonomous-run pass into the same native transcript corpus it already keeps for interactive sessions — so the work done while no human is watching is exactly as auditable as the work done with one.** There is an asymmetry to close: an operator's interactive session is redirected into the native transcript corpus at `~/.abcd.noindex/history/<root_sha>/` (a durable, queryable store). But the autonomous worker on the pluggable run seam — the thing that runs unattended for hours, plans, implements, and self-reviews — emits its instructions and reasoning only to an ephemeral per-pass log that run-dir hygiene prunes. The agent doing the most consequential, least-supervised work leaves the least durable record. This intent closes that gap: every autonomous run pass's instructions + reasoning is captured to the same corpus, with the same queryability, as an interactive session.
>
> "abcd's whole promise is that you can trust what the agent did while you were away," said Kira, framework author. "But when I went to audit an autonomous run after the fact, the interactive sessions were all there in the transcript corpus and the autonomous worker — the one I most needed to check — had left only an ephemeral log that the run-dir cleanup had already eaten. The unattended work is exactly the work that most needs a durable trail."

## Why This Matters

abcd is an autonomous-development framework whose differentiator is **auditable, provenance-bearing automation** — the operator can leave an autonomous run going and later reconstruct what it decided and why. That promise has a hole precisely where it matters most:

- **Interactive sessions ARE captured**: operator sessions are redirected into the native transcript corpus at `~/.abcd.noindex/history/<root_sha>/` and stay durably queryable.
- **Autonomous-run passes are NOT**: the worker on the pluggable run seam emits its instructions and reasoning only to an ephemeral per-pass log under the run directory, which run-dir hygiene prunes — neither committed nor durably stored.

The consequence: the worker's full chain of reasoning (the instructions it received, the autonomous-run rules, every tool call and decision across a pass) exists only transiently. After a run is pruned, the autonomous reasoning is gone, while every interactive session that touched the repo is permanently queryable. For a framework that markets unattended trustworthiness, the unattended path is the one missing its black box.

This is also a *symmetry* failure, not just a missing feature: the native transcript corpus already exists and already works for one caller class. The autonomous caller was simply never routed through it (or through an equivalent durable sink).

## What's In Scope

- **The capture gap itself:** every autonomous-run pass's instructions + reasoning lands in the native transcript corpus under `~/.abcd.noindex/history/<root_sha>/` — parity with interactive-session capture.
- **All autonomous entry points:** every phase the run seam drives (plan / work / completion-review), and any other unattended model invocation the seam makes, are covered — not just the work phase.
- **Queryability + provenance:** captured transcripts are associated with the run id, spec/task id, run pass, and phase, so a later audit can answer "what did the worker reason for spc-NN.M, run pass K?".
- **Honor abcd's boundaries:** whatever the mechanism, it feeds the native transcript corpus through the run seam's own capture point rather than forking anything, and stays intact across the pluggable seam's implementations (Workflows / the companion harness / native loop).

## What's Out of Scope

- **Re-architecting the transcript corpus.** `~/.abcd.noindex/history/<root_sha>/` and its single-owner provisioning (spc-15, predecessor store) stay as-is; this intent FEEDS that store, it does not redesign it.
- **Capturing the operator's interactive sessions** — already solved (native transcript-corpus redirect). This is only the autonomous-worker half.
- **The live run-console view** — that is orthogonal to durable capture; unchanged.
- **Retroactive recovery of already-pruned runs** — this is forward-looking capture, not archaeology of lost logs.

## Approach (deliberately OPEN — for plan/grill to decide)

The mechanism is intentionally left undecided; the planning/grill stage chooses between (at least) these candidates, weighing the feed-don't-fork rule, survival across seam implementations, and the autonomous worker's non-interactive mode:

- **(a) Capture at the seam** — route each pass's model invocation through the native transcript corpus's capture point, so autonomous transcripts land in the SAME store and shape as interactive ones (maximal symmetry; verify the corpus captures a non-interactive run, and that capture holds across seam implementations).
- **(b) Ingest the per-pass log** — keep the current spawn but durably ingest each pass's log into the corpus at pass end (less invasive to the spawn path; but must capture the instructions too, not just stdout).
- **(c) A structured stream sink** — capture the worker's structured stream output into a per-pass record in the corpus.

Open questions for grill: does capturing a headless (non-interactive) run need a different hook than an interactive one? Does capture interfere with the live run-console view or the timeout/cancel controls? Should the capture be best-effort (never block a run pass) or gated? What is the retention/size policy for autonomous transcripts vs interactive?

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [autonomous-loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).
The store, the redaction door and the hook as capture path were settled in the
2026-07-30 session-recording note; this pass covers what the host's
non-interactive mode does to that path.

- **A headless run is stored like a session.** A `claude -p` session is written
  to the same transcript store and is resumable by id, though left out of the
  picker; sub-agent transcripts persist under the session's `subagents/` folder
  and are deleted after `cleanupPeriodDays` (30 days by default);
  `--no-session-persistence` suppresses a run's transcript (Claude Code
  sessions and sub-agents docs).
- **The hooks run unless the lane is started bare, and abcd's runner starts
  lanes bare.** Without `--bare` a `-p` session runs the project's hooks and
  plugins; `--bare` skips them, is the host's recommendation for scripts, and
  "will become the default for `-p` in a future release" (Claude Code
  headless docs). abcd's shipped claude runner launches every lane with
  `--print --bare` on a paid key (RN2, dcn-2610020000000016) and with
  `--no-session-persistence`, so a runner lane fires none of abcd's
  `SessionEnd` and `SubagentStop` capture and leaves no transcript in the
  host's store; the guard reaches it as the run's permission-prompt answerer
  instead (dcn-2610161624025493).
- **Termination matters.** On SIGTERM the host runs `SessionEnd` and no other
  hook, records no result for the turn, and exits 143; SIGINT ends the turn
  first; the Stop hook's `transcript_path` may lag the conversation (Claude
  Code docs). `SubagentStop` carries `agent_transcript_path`, which abcd
  already captures; `--forward-subagent-text` on `stream-json` is the
  alternative reconstruction from the stream.
- **The format is declared unstable.** The transcript "entry format is internal
  to Claude Code and changes between versions" (Claude Code sessions docs), so
  the raw file is stored and the index derived.

**Path 2, with the capture source left to a ruling.** Under the runner as
shipped, the one record a lane leaves is the `stream-json` stream the runner
already reads, so approach (c), a sink on that stream, is the one that needs no
reversal. Approach (a), capture at the seam through the same hooks, needs a
lane started without `--bare` (or with the plugin and hooks passed
explicitly), which reverses RN2 and dcn-2610161624025493; it is a proposal for
the technical facilitator to rule on, not the decided shape. An ingest by
session id from the result is dropped as a fallback, because runner lanes
persist no session. Whichever is ruled keys the capture by run id, lane, pass
and phase set in the lane's environment and goes through `history.Capture` and
its redaction door, best-effort as the criteria require. No Go dependency. The independent fit-challenge runs at plan
time.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** an autonomous-run pass (plan, work, or completion-review phase), **when** it completes (or is terminated), **then** its instructions + reasoning is present in the native transcript corpus under `~/.abcd.noindex/history/<root_sha>/`, queryable after the run dir is pruned, and associated with run id + spec/task id + run-pass + phase.
- **Given** the existing interactive-session capture, **when** the autonomous capture lands, **then** both reach the same transcript corpus (parity), and a single audit query can enumerate autonomous AND interactive activity for the repo.
- **Given** abcd's boundaries, **when** the mechanism is implemented, **then** it feeds the native transcript corpus (no fork) and remains intact across the pluggable run seam's implementations.
- **Given** the live controls, **when** capture is active, **then** it does not break the live run-console view, the worker timeout, or the cancel/budget controls, and a capture failure never blocks or fails a run pass (best-effort, fail-open on the capture path only).

## References

- The native transcript corpus at `~/.abcd.noindex/history/<root_sha>/` and its single-owner provisioning (spc-15, predecessor store) — the store this feeds
- `.abcd/development/research/notes/ahoy-history-store-manual-scaffolding.md` (the transcript-store design this feeds)
- ADR-29 (native transcript corpus), ADR-27 (pluggable autonomous seam)
- Related: the abcd transparency/provenance promise in the brief (`01-product/`), itd-58 (a sibling autonomous-run-provenance closure)

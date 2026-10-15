# Forge-mirror pilot — experiment record (2026-08-19)

A hand-run of the forge-mirror protocol decided on 2026-08-19 (the
ledger-canonical entry [dcn-2608190000000001](../../../work/decisions/dcn-2608190000000001-the-abcd-work-issues-ledger-is-the.md), grounded in
[`2026-08-19-issue-ledger-forge-sync-sota.md`](2026-08-19-issue-ledger-forge-sync-sota.md)),
executed per the script-first rule: the smallest documented protocol is run by
hand and graded before any automation is filed. This note is the grading; the
automation intent cites it.

## Hypothesis

The decided topology — mirror-out one way, forge id written back into the
canonical record, human-gated import, self-healing reopen — is runnable by
hand with nothing but `gh` and a text editor, and a bounded pilot will
surface where the protocol creaks before a script hard-codes the creaks.

## Protocol as run

Scope: the three records from the PR #294 review intake (iss-285, iss-286,
iss-287) — a deliberate pilot bound, not the whole open ledger.

1. **Mirror-out**: one forge issue per record, titled `[iss-N] <title>`, body
   carrying a one-paragraph summary, the canonical record path, severity, and
   a footer stating the topology (status truth lives in the ledger; comments
   are triaged, not acted on). Result: iss-285→#372, iss-286→#373,
   iss-287→#374.
2. **Write-back**: record the mirror pointer in each canonical record.
3. **Self-healing test**: close #373 on the forge, deliberately without any
   ledger import, then run a reconciliation pass (for each open record with a
   mirror pointer, read the mirror's state; reopen a closed mirror whose
   record is still open, with a comment explaining why).
4. **Resolution direction**: resolve a record in the ledger, close its mirror
   citing the handle.

## Findings

- **F-A (the substantive one): the issue schema rejects the write-back.**
  `forge_id:` in frontmatter is `malformed frontmatter: unknown property` —
  and the failure mode is that the record is **skipped by the capture
  surface entirely** (`abcd capture list` drops it with a per-record notice).
  The parse is loud but the consequence is severe: one unknown field makes a
  record invisible to its own tooling. The pilot fell back to a body line
  (`Forge mirror: #N`), which parses everywhere and greps cleanly but is
  weakly typed prose. The automation's first acceptance criterion is
  therefore a schema'd `forge_id` field (shape-checked, lint-visible), not a
  convention.
- **F-B: mirror-out is fully mechanical.** Title/summary/path/footer is a
  template; no judgment was needed beyond the one-paragraph summary. Good
  candidate for the script rung as-is.
- **F-C: self-healing works and needed no state of its own.** The pass found
  #373 CLOSED against an open canonical record and reopened it with the
  explanatory comment; the mirror pointer plus the two visible states are
  sufficient. Cost note: the hand pass spends one `gh issue view` per
  mirrored record — a script should batch (`gh issue list --json
  number,state`) before the open ledger's ~140 records make that a rate
  problem.
- **F-D: the resolution direction was NOT exercised.** iss-287's resolution
  is blocked on PR #294 merging, and manufacturing a fake resolution to test
  the close path would have put a false record in the ledger. Protocol step 4
  remains documented-but-unproven; the first real resolution of a mirrored
  record should be watched as the completion of this pilot.
- **F-E: minting under concurrency held up incidentally.** During the same
  day's work, main minted iss-284 in parallel with this branch's iss-285/287
  and there was no collision — the allocator's cross-ref floor did its job.
  Recorded here because the routines now launching rely on exactly this.

## Adjacent observation (not this experiment's scope)

`.abcd/work/DECISIONS.md` hit its predicted append-contention conflict today
(PR #366 vs #364), and the newly launched hourly hunt routines each append a
round line to it per PR. The file's own header names the remedy (graduate to
per-file `decisions/<date>--<slug>.md` when contention bites). Watch whether
the routines make it bite; capture as an issue if so.

## Verdict

The topology survives contact: mirror-out, write-back, and self-healing all
ran by hand without ambiguity, and the one real obstacle (F-A) is a schema
gap with an obvious shape, not a design flaw. Proceed to the automation
intent with F-A/F-C/F-D as acceptance criteria; the ADR graduating the
topology decision can now cite a run, not just a survey.

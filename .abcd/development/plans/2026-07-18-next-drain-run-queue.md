# Next drain-run queue

**Status:** the backlog for the *next* autonomous drain run, consumed by the
generic protocol at
[`2026-07-12-abcd-run-protocol.md`](2026-07-12-abcd-run-protocol.md). The
2026-07-16 drain ([`2026-07-16-planned-intent-drain-run.md`](2026-07-16-planned-intent-drain-run.md))
shipped M1–M5 and is closed; this file carries what to pick up next.

Same run contract as the 2026-07-16 plan unless a queued item states otherwise:
`make preflight` is the sole gate; ledger is capture-only; correctness +
security reviews before each PR; one PR per item; never merge, never commit to
main.

## Queued

### itd-93 — `abcd scaffolds a hardened release gate for managed repos` (feature, L)

The forward-looking generalisation of abcd-cli's fixed release gate (adr-37,
PR #99) — scaffold `release.yml`/`auto-release.yml` (armed against the reviewed
content commit), the runbook, and the sha-keyed-receipt/RD001 interop into a
managed repo, so its first public release cannot hit the self-reference abcd-cli
paid to discover. Backing intent:
[`../intents/planned/itd-93-abcd-scaffolds-a-hardened-changelog-driven-release-gate-into.md`](../intents/shipped/itd-93-abcd-scaffolds-a-hardened-changelog-driven-release-gate-into.md).

**NOT yet run-ready — readiness gates (a drain burst must check these first and
SKIP-with-reason if unmet, per the protocol's skip filter):**

1. **Still a draft.** It sits in `intents/drafts/`; it must be planned
   (`abcd intent plan itd-93` → mints its spec, moves to `planned/`) before a
   run can implement against a spec.
2. **AC unconfirmed — maintainer decision.** The press release and Acceptance
   Criteria are *facilitator-seeded* (from the abcd-cli fix), explicitly left
   for the product thinker to confirm/refine. Under the run contract, an item
   whose AC needs a new interpretation is a **maintainer decision → SKIP** until
   the product thinker signs off. Do not autonomously invent the AC bar.
3. **Open design questions need answers** (in the intent's *Open Questions*):
   which surface scaffolds it (an explicit verb vs. `ahoy install`);
   template-vs-verbatim-copy (with a lockstep test against abcd-cli's own
   workflows); whether the gate is exercisable while private so the flaw
   surfaces before the public flip; the itd-73 derived-version seam. These are
   premise/design choices, not self-contained cuts.

**Shape note:** itd-93 is a **feature** (a template/scaffold engine + a new
surface + CI generation), not the record-catch-up / small-additive-cut class
the 2026-07-16 drain targeted. Once its gates clear, it likely warrants its own
**focused run** (or decomposition into smaller planned specs) rather than being
folded into a general drain burst. Flag to the maintainer at pick-up time.

## Also open (not yet queued — recorded so the next run sees them)

- **Ledger follow-ups** (capture-only; the bug-hunt loop's finds): iss-104
  (intent typo-guard symmetry), iss-105 (intent has no plugin-markdown surface —
  itd-93-adjacent), iss-106 (GL002 inline-code masking false-positive edge).
  Small, self-contained — genuine drain candidates once triaged.
- **itd-28** (native reviews subsystem) — still NEEDS-MAINTAINER (new gitleaks
  dependency sign-off); unblocks spc-8 (itd-43) AC3. Not queued until approved.

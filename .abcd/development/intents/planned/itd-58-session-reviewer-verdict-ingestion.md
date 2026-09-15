---
id: itd-58
slug: session-reviewer-verdict-ingestion
spec_id: null
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: ["adr-8", "adr-27"]
prd_path: null
severity: major
---

# A Real Reviewer's SHIP Verdict Reaches The Session Gate Through A Channel The Worker Cannot Forge

## Press Release

> **abcd's autonomous-run enforcement learns to ingest a live reviewer's verdict through a trusted production path — so a genuine SHIP advances the work, while a worker's self-written SHIP still cannot.** The pluggable autonomous seam closes the A4 *exploit*: a worker that writes its own `"verdict":"SHIP"` receipt is denied at the advance gate, because no trusted verdict was ever recorded. But that is only the negative half. The positive half — a real reviewer (any oracle adapter) returning SHIP, parsed into a `TrustedVerdict`, and recorded so the legitimate advance proceeds — needs a production path: recording a trusted verdict is a seam-owned operation, and something in the live run must parse reviewer output and invoke it. This intent builds that ingestion path and proves it end-to-end, so the trusted-verdict channel is a full round-trip, not a one-sided denial.

## Why This Matters

The A4 finding in the security SSOT is "the worker self-writes its own SHIP review receipt." The pluggable autonomous seam closes the dangerous direction of that: on an autonomous-seam run, a forged receipt no longer advances the task, because the advance gate only honors a verdict that was *recorded* through the seam-owned record operation, and a worker cannot reach it. That is the security win, and it is real.

What the seam's enforcement deliberately does **not** do on its own is wire the *legitimate* side: when a real reviewer returns SHIP, some production component must (a) capture the reviewer's output from the run, (b) parse it into a `TrustedVerdict` (the trusted-verdict parser exists for this), and (c) record it so the genuine advance is permitted. Without it, the only way to make a trusted SHIP "advance" in a test is to record the verdict directly — which is a test-only shortcut, not proof that the production channel works. A plan-review caught exactly this: requiring the A4 *positive* proof inside the enforcement-wiring spec would force inventing this ingestion path under a spec scoped to "wiring only," so it was split out here.

Until this lands, the seam enforces A4 as a **denial without a counterpart**: nothing forged gets through, but the framework has not demonstrated that a real reviewer SHIP flows through the same trusted channel rather than some side door. Closing the loop is what marks A4 fully closed for the autonomous-run path, not just "self-attestation exploit closed."

## What's In Scope

- A production component on the pluggable autonomous seam that captures the live reviewer's output, parses it into a `TrustedVerdict`, and records it through the seam-owned record operation (gating reviewer gates; advisory mined for findings, per ADR-8).
- An end-to-end A4 *positive* proof entering through the seam's run entry point: a trusted reviewer SHIP injected through the production ingestion endpoint advances the task, and the test performs no direct run-state writes and no direct verdict-record call.
- Pairing with the seam's A4 negative proof so the channel is shown as a full round-trip (forged → denied; trusted → advances).
- The A4 annotation upgraded from "self-attestation exploit closed" to "closed" for the autonomous-run path once the positive proof is green.

## What's Out of Scope

- The seam's enforcement wiring (readiness composition, supervisor hook composition, fail-closed launch, A1/A2/A3/B2 conversions, A4 negative) — this intent depends on it, it does not redo it.
- Non-seam runners: this ingestion is scoped to the pluggable autonomous seam; any runner driven outside the seam is out of scope.
- Any change to how the advance gate DENIES (that is the enforcement wiring + the existing gate); this intent only adds the trusted INGESTION half.

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** an autonomous-seam run where a real reviewer returns SHIP, **when** the run reaches the advance point, **then** the reviewer output is parsed into a `TrustedVerdict` and recorded through the production path (no direct verdict-record from outside the seam), and the legitimate advance proceeds.
- **Given** the same run but a worker-forged SHIP receipt with no trusted reviewer verdict, **when** the worker attempts the advance, **then** it is denied (the seam's negative behavior is preserved, now as one half of a demonstrated round-trip).
- **Given** the end-to-end positive test, **when** it runs, **then** it enters via the seam's run entry point, injects through the production ingestion endpoint, and asserts the advance with zero direct run-state writes by the test.

## References

- A local code-review working note, §A4 (the finding) + §N1
- The seam-owned verdict-record operation (the gap), the trusted-verdict parser, and the advance gate on the pluggable autonomous seam
- The seam's enforcement (dependency — wiring + A4 negative), ADR-8 (trusted-verdict policy resolution), ADR-27 (pluggable autonomous seam)
- Surfaced by: a plan-review of the seam's enforcement wiring

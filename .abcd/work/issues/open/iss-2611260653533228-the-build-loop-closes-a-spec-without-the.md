---
schema_version: 1
id: "iss-2611260653533228"
slug: "the-build-loop-closes-a-spec-without-the"
severity: "major"
category: "inconsistency"
source: "agent-observation"
found_during: "an ideate record grill of the build loop, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Route the loop's close through the same gate the CLI's spec close uses (move the doc-fidelity check into the core close path both callers share), so a close that ships an intent refuses until a docs review is saved for HEAD and the loop hands the run back naming it; prove it with a loop test whose close is refused without a review and passes with one."
deferred_after: v0.17.1
deferral_reason: "The release cut already refuses without a docs review saved for HEAD, so no unreviewed intent reaches a release; gating the loop's close as well would hand back every unattended build at its last lane until the loop has a docs-review role of its own, which is design work for the build spec's no-host drive step (spc-2609202134338445). Fix it with that role, then route the loop's close through the gate."
---

The build loop closes a delivered spec by calling intent.Reconcile directly (internal/core/implement/loop/land.go), which ships the intent without the doc-fidelity gate the CLI's spec close runs before an intent moves to shipped/, so an intent delivered by abcd build reaches shipped/ with no docs review saved for its head, and the review binds only later, at the release cut.

## Deferral 2026-11-26

Deferred past v0.17.1: The release cut already refuses without a docs review saved for HEAD, so no unreviewed intent reaches a release; gating the loop's close as well would hand back every unattended build at its last lane until the loop has a docs-review role of its own, which is design work for the build spec's no-host drive step (spc-2609202134338445). Fix it with that role, then route the loop's close through the gate.

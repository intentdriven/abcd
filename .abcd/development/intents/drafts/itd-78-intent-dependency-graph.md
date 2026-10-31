---
id: itd-78
slug: intent-dependency-graph
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
blocked_by: []
builds_on: []
---

# abcd Knows What to Build First — Even When It's Something Small

## Press Release

> **Severity says how much a thing matters; the dependency graph says when to build it — abcd keeps the two honest and computes the order.** A roadmap stalls quietly when a small piece of plumbing blocks a big feature and nobody notices, because everything is ranked by how important it *looks*. abcd separates the two axes. Each intent **declares** two kinds of fact known at capture time: its **severity** (`nitpick | minor | major | critical` — the same enum the issue ledger already uses) and its **edges** — `blocked_by` (cannot ship before) and `builds_on` (cheaper or better if the other exists first). From those declarations abcd **derives** what it never stores: effective priority, by *priority inheritance* — an intent's effective priority is the maximum of its own severity and the severity of everything it transitively blocks. A minor intent holding up a major one jumps the queue while staying honestly minor; the classic priority inversion is resolved the way schedulers have always resolved it. The graph is linted, not trusted: cycles, dangling ids, a phase that scopes an intent before its blocker's phase, and the silent stall — a major intent scheduled while its minor blocker sits unscheduled — all fail deterministically.
>
> "I kept re-discovering, one grilling at a time, that the unglamorous banlist work was the thing to do first," said Kira, a maintainer. "Now the graph tells me: it's minor, and it computes to major, because everything I actually care about sits behind it." Their collaborator Bob stopped arguing rank entirely: "We only debate two things now — how severe, and what blocks what. The order falls out."

## Why This Matters

Severity and priority get conflated because most trackers offer one field for both, and the conflation has a failure mode in each direction: rank by severity and small blockers starve everything behind them; hand-rank priority and the number silently goes stale as the graph changes. The resolution is the same one this record already applies elsewhere — declare inputs, derive outputs, lint the consistency. Declared severity and declared edges are capture-time human judgments (legitimate frontmatter, like `kind`); computed priority is a cached mirror if written down, so it never is — the same doctrine that removed `status:` fields (directory is state) keeps effective priority out of frontmatter. Sequencing authority does not move: phase docs' `## Scope` sections remain the single source of truth for what ships when (adr-9); the graph's job is to make a *contradictory* schedule un-lintable, not to generate the schedule.

## What It Looks Like

- **Two declared fields, one shared enum.** Intent frontmatter gains `severity:` (the capture ledger's `nitpick|minor|major|critical`) and the edge lists `blocked_by:` / `builds_on:` (itd-N references). Deliberate non-dependencies stay in prose — the schema records edges, the press release records why an edge is absent (as itd-76 does for itd-16).
- **`abcd intent graph`** computes and reports, never persists: effective priority via inheritance (max of own severity and the severity of all transitively blocked intents), tie-broken by transitive unblock count, then capture order. Output is a ranked build-next list with the inheritance chain shown ("itd-74: minor, effective major ← blocks itd-76").
- **Lint, deterministic.** Record lint gains graph checks: unknown itd references, cycles, an intent phase-scoped earlier than its blocker's phase, priority inversion the schedule ignores (a scoped intent whose transitive blocker is unscheduled), and **hand-authored reverse fields** (`blocks:`, `blocked:`, any stored mirror of a derived view) — the reverse direction exists only as computed output, matching how every text-based traceability tool stores one direction and derives the other.
- **Suspect links (fingerprints).** An edge may carry a short content fingerprint of its target (`blocked_by: [itd-74@<hash-prefix>]`); when the target intent's content changes, lint flags the edge *suspect* — the dependant re-confirms the relationship still holds and refreshes the hash. This is what keeps edges honest over time: a dependency declared against last month's itd-74 is not automatically a dependency on today's.
- **Dogfooded corpus-wide.** A one-shot sweep (2026-07-08) landed severity and evidence-backed edges on all 64 live intents; the derived order reproduced hand-made grilling conclusions and computed two priority inversions (itd-74 → itd-76, itd-20 → itd-33). Method and assessor lessons: [`research/notes/2026-07-08-intent-dependency-sweep.md`](../../research/notes/2026-07-08-intent-dependency-sweep.md). Settled by the sweep: severity is set at capture going forward (the backfill is done); edges stay mutable for the intent's life (they appeared *during* assessment, which decides the bind-time question); edges point only at intents — spec/ADR/substrate prerequisites stay in prose.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [traceability-governance SOTA note](../../research/notes/2026-10-23-traceability-governance-sota.md). The sweep of 2026-07-08 remains the method record.

- **Priority inheritance is computed, never stored.** The Linux rt-mutex design keeps a tree of top waiters per lock owner, "compare[s] the top pi waiter to its own normal/deadline priority and take[s] the higher one", and recomputes when a waiter leaves so the boost drops; chains "may converge, but a chain would never diverge", because "a process can't be blocked on more than one mutex at a time". The protocol is Sha, Rajkumar and Lehoczky (1990). This is the draft's "derive, never persist" with a precedent for compute-on-read and the drop when a waiter leaves, not for the chain shape: an intent may have several blockers, so the inheritance here is a max over a DAG rather than a single chain.
- **Forges store the edge once, render both directions, and derive no order.** GitHub issue dependencies went generally available on 2025-08-21 ("blocked by" or "blocking", up to 50 per relationship type, `is:blocked` and `blocked-by:` filters, API and webhooks, `gh issue edit --add-blocked-by`); the docs say nothing about ordering, priority or cycles.
- **Every score punts dependencies to a separate step.** Cost of delay divided by duration reorders a worked example from $69,000 to $27,000 of delay cost; SAFe's relative variant is rejected by the method's own popularisers ("You simply can't make trade-off decisions with relative value estimates"); RICE's authors concede "One project may be a dependency for another project, so it needs to happen first". A derived order from edges is what none computes.
- **One direction stored, the other derived** is how Doorstop (links point at parents) and sphinx-needs (one link type with incoming and outgoing labels) work; the draft's refusal of hand-authored reverse fields matches.
- **Local state since the draft:** phases and milestones are retired (adr-2609212115255771: "Sequencing is dependencies plus the shelves"), so the two phase lints have no object; the shipped pick (itd-2609211116005482) filters on `blocked_by` and reads a superseded blocker as its replacement but inherits no severity; the revisit mark (itd-2610070549060530) and the product thinker's order (itd-2610070550516046) are further inputs to the same order.
- **Path 2**: native, `blocked_by` stored one way and any mirror refused by lint, effective priority derived at read time over hard edges only (`builds_on` reported, unweighted, as the draft's conservative start), the inheritance chain shown on the board, edges fingerprinted with the same primitive as section links. The seam is one ordering function the pick consumes, into which severity inheritance, the record score, the revisit mark and the product thinker's order enter with a stated precedence; this intent supplies the inheritance term only.

The independent fit-challenge runs at plan time.

## Open Questions

- Whether `builds_on` (soft edges) participates in priority inheritance at reduced weight or only in reporting — hard edges only is the conservative start.
- How supersession re-points inbound `blocked_by` edges (the sweep found three aimed at a dead intent) — manual at supersede time, or a lint-guided sweep.
- (SOTA 2026-10-23) The precedence among the four inputs to "what next" (the product thinker's order, the revisit mark, severity inheritance, the record score), stated once in one ordering function rather than in four specs.
- (SOTA 2026-10-23) Whether the two phase-consistency lints are dropped as moot now that phases are retired, or re-aimed at the rendered Now/Next/Later (a Now item whose blocker sits in Later).

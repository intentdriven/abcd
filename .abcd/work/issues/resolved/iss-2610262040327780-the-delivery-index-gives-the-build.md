---
schema_version: 1
id: "iss-2610262040327780"
slug: "the-delivery-index-gives-the-build"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 938 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/README.md:10"
remedy: "In the index row for 01-build-sequence.md describe a dependency DAG of build steps (scaffold, install and launch, history capture and memory, intent and review, spec engine, run seam, lifeboat) instead of 'MVP, then the companion harness, then a named host'."
finding_ref: cfk-d92229149573d195
resolution: "The delivery index row for 01-build-sequence.md now describes the dependency DAG of build steps (scaffold, install and launch, history capture and memory, intent and review, spec engine, run seam, lifeboat) instead of the MVP, companion harness, named-host order; a lint test pins the row."
impact: internal
resolved_by:
  commit: "333216ac78adf8cd9257e2b6a2fc17a93a50ff36"
---

The delivery index gives the build sequence an order of MVP, then companion harness, then a named host; the build sequence is a DAG of numbered Go build milestones.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/README.md:10` — "The delivery order (MVP → the companion harness → Claude Code) and what ships in each step"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:12` — "detail: what each milestone stands up in the Go core"

The chapter index says 01-build-sequence gives a delivery order from MVP to the companion harness to a named host. The page itself is a dependency DAG of build milestones (Go scaffold, install and launch, history, intent and review, spec engine, run seam, lifeboat). It mentions the companion harness only as a design-target backend inside one milestone, so the order the index describes is not on the page.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 938.

## Grounds

- pursued: the index row matches the page's milestones 0-6; TestBriefDeliveryIndexDescribesTheBuildSequenceAsADAG would show it wrong by failing if the row regains the linear order or drops a step.

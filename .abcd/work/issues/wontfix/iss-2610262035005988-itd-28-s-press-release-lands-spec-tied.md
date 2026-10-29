---
schema_version: 1
id: "iss-2610262035005988"
slug: "itd-28-s-press-release-lands-spec-tied"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 540 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-28]
finding_ref: cfk-0e05abead341e990
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The record says in its banner that the press release and scope are read through the re-scope; internal/core/reviews reads the dated folders and receipt directories, not a per-spec store."
---

itd-28's press release lands spec-tied reviews in a store under .abcd/reviews/< spec-id>/; the board reads reviews as dated folders under .abcd/work/reviews/.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:33` — "in a per-review directory at `.abcd/reviews/<spec-id>/<NNNN>-<slug>-<ref>/`"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:290` — "Every folder under `.abcd/work/reviews/` is a row: a dated"

itd-28's press release and scope say every spec-tied review lands as review.json and review.md in a per-review directory at .abcd/reviews/< spec-id>/< NNNN>-< slug>-< ref>/, with an index command and a CI check over that tree. 08-abcd.md says every folder under .abcd/work/reviews/ is a review row, a spec's review carrying its id in a dated folder name. The intent's own decision 1 drops that store, but its press release and scope still state it, so the shipped record describes a review location the surface does not read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 540.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The record says in its banner that the press release and scope are read through the re-scope; internal/core/reviews reads the dated folders and receipt directories, not a per-spec store.

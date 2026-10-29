---
schema_version: 1
id: "iss-2610262037334466"
slug: "itd-44-files-adrs-as-adr-next-free-slug"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 732 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:121"
remedy: "none (filed automatically)"
related_intents: [itd-44]
finding_ref: cfk-bdbba32cbf114cfd
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034191519. Same draft edit as iss-2610262034191519, set against a different statement of the same current state."
duplicates: [iss-2610262034191519]
---

itd-44 files ADRs as adr-< next-free>-< slug>.md; the drain brief cites the store's ADRs as < timestamp id>-< slug>.md

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:121` — "capture writes `adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:46` — "decisions/adrs/2609291342092738-a-drain-takes-an-issue-alone-only-when.md"

itd-44 sends a decision capture into `.abcd/development/decisions/adrs/` as `adr-<next-free>-<slug>.md`, which is a prefixed, sequentially numbered name. The drain chapter cites the record its rule is read from as a file in that store named with the bare timestamp id and slug, with no `adr-` prefix. An ADR filed the way itd-44 describes would be named in a form the store's own records do not use.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 732.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034191519. Same draft edit as iss-2610262034191519, set against a different statement of the same current state.

---
schema_version: 1
id: "iss-2610262035091535"
slug: "itd-46-has-promotion-pass-the-issue-text"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 551 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:58"
remedy: "none (filed automatically)"
related_intents: [itd-46]
finding_ref: cfk-7346d5cfdab63cb9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both records were true when written; internal/core/capture/promote.go seeds a pointer plus the first body line, never the body, per itd-119's first criterion."
---

itd-46 has promotion pass the issue text to the quoted-text create; the capture chapter has promotion mint a draft whose body is a by-id pointer, not a copy

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:58` — "change it to call `/abcd:intent` directly with the issue text as the quoted-text argument."
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:185` — "One invocation mints the draft under `intents/drafts/` with the slug reused and the body a by-id pointer rather than a copy"

itd-46 changes the promote path to call /abcd:intent directly with the issue text as the quoted-text argument, which would copy the text into the press release. The capture chapter says one invocation mints the draft with the body as a by-id pointer rather than a copy, and records its origin as extracted-from-record rather than through the create path. The promoted draft cannot be both a copy of the issue text and a pointer to it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 551.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both records were true when written; internal/core/capture/promote.go seeds a pointer plus the first body line, never the body, per itd-119's first criterion.

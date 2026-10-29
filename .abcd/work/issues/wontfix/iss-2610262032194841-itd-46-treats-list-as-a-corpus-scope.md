---
schema_version: 1
id: "iss-2610262032194841"
slug: "itd-46-treats-list-as-a-corpus-scope"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 253 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:78"
remedy: "none (filed automatically)"
related_intents: [itd-46]
finding_ref: cfk-edd55cce9e6d51d9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent command registers consistency, prepass, audit and the rest but no list or shape sub-verb; itd-46's own first shape is bare means status, so list had no separate job."
---

itd-46 treats `list` as a corpus-scope intent scan verb that already exists; the context page says there is no plain `list` sub-verb

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:78` — "scan verbs (`list`, `consistency`, `shape`) continue not to require an ID."
- End B: `.abcd/development/brief/01-product/02-context.md:11` — "There is no plain `list` sub-verb — it is folded into the bare render per SD001."

itd-46 describes `list`, alongside `consistency` and `shape`, as a scan verb that continues not to require an ID, an already-correct shape it leaves alone. 02-context says there is no plain `list` sub-verb because it is folded into the bare render per SD001. An intent sub-verb cannot both carry on unchanged and not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 253.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent command registers consistency, prepass, audit and the rest but no list or shape sub-verb; itd-46's own first shape is bare means status, so list had no separate job.

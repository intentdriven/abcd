---
schema_version: 1
id: "iss-2610262045235833"
slug: "itd-46-says-sub-verbs-act-on-existing"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1271 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-46, itd-2609020625400194]
finding_ref: cfk-bb69e2f64535cdb4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-46 fixes how quoted text and bare calls map for issues and intents; capture admit and capture surprise are explicit sub-verbs with an id argument, as commands/capture.md documents."
---

itd-46 says sub-verbs act on existing records and quoted text creates; itd-2609020625400194 adds a capture sub-verb that creates a new record from quoted text

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18` — "The three invocation shapes are crisp: bare → status, quoted text → create, sub-verb [+ID] → act on existing."
- End B: `.abcd/development/intents/shipped/itd-2609020625400194-an-admission-and-a-surprise-are-written.md:21` — "`abcd capture surprise --occasioned-by <id> "<what>"` writes one surprise entry as its own record."

itd-46 sets the invocation shapes: bare is status, quoted text creates, and a sub-verb with an id acts on an existing record. itd-2609020625400194's `abcd capture surprise --occasioned-by <id>` with quoted text creates a new surprise record through a sub-verb, and its id names the occasion, not an existing record to act on. `capture admit` likewise mints a new admission record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1271.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-46 fixes how quoted text and bare calls map for issues and intents; capture admit and capture surprise are explicit sub-verbs with an id argument, as commands/capture.md documents.

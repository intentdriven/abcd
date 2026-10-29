---
schema_version: 1
id: "iss-2610262036330589"
slug: "itd-102-calls-the-positioning-check-part"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 656 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-102-your-repo-says-the-same-thing-about.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-102]
finding_ref: cfk-c4dcdde0a6079b9b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-124's press release makes lint the form check and audit the promise check, with a clean break and dated records keeping the old name."
---

itd-102 calls the positioning check part of every audit; the identity chapter runs it as a rule of abcd lint

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-102-your-repo-says-the-same-thing-about.md:17` — "a deterministic positioning check compares each surface against it on every audit"
- End B: `.abcd/development/brief/04-surfaces/19-identity.md:122` — "The drift check runs as the `identity-positioning` rule on every `abcd lint`"

itd-102 says the positioning check compares each surface on every audit. 19-identity runs the same drift check as the identity-positioning rule on every abcd lint, and the lint chapter keeps the word audit for the hash-chain fidelity surface. The same check is named by two verbs whose vocabulary the record keeps apart.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 656.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-124's press release makes lint the form check and audit the promise check, with a clean break and dated records keeping the old name.

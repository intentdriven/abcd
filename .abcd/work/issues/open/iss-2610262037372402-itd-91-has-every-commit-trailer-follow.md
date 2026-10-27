---
schema_version: 1
id: "iss-2610262037372402"
slug: "itd-91-has-every-commit-trailer-follow"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 737 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-91]
finding_ref: cfk-8d755d4ed7c45e1e
---

itd-91 has every commit trailer follow the project's chosen AI-credit convention; the announce brief requires an announcement's Assisted-by to be abcd's own label and nothing else

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:17` — "Your answer is recorded once, in the repo, and from then on every commit trailer, pull-request body, and generated record uses exactly that convention."
- End B: `.abcd/development/brief/04-surfaces/38-announce.md:84` — "its `Assisted-by` abcd's own `abcd:<version>` label and no other value"

itd-91 records a per-project choice of convention, from none to `Co-Authored-By:` to custom wording, and says every commit trailer, pull-request body and generated record then uses exactly that convention. The announce chapter checks every announcement against a validator that requires its `Assisted-by` trailer to be abcd's own `abcd:<version>` label and no other value. In a project that chose a different convention, announcement commits either fail the validator or ignore the project's choice.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 737.

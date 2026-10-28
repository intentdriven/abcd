---
schema_version: 1
id: "iss-2610262033022683"
slug: "itd-91-adds-a-setup-question-that"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 351 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-91]
finding_ref: cfk-32f923352a02ae56
---

itd-91 adds a setup question that chooses an AI-acknowledgement convention; the ahoy page already has the install's attribution opt-in, a hook that asks each commit to declare assistance.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-91-ai-attribution-preference.md:31` — "in the onboarding surface (`/abcd:init` / `prepare-this-repo`) offering a selection of acknowledgement conventions"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:317` — "that run installs the committed `prepare-commit-msg` prompt asking every commit to declare whether a tool assisted it, and the choice is recorded"

itd-91 puts one setup question in the onboarding surface (/abcd:init or prepare-this-repo). It offers none, Assisted-by, Co-Authored-By, Generated-by or a custom wording, and applies the answer to every trailer. The ahoy page covers the same ground in the install. An opt-in attribution run installs the committed prepare-commit-msg prompt, which asks every commit to declare whether a tool assisted it, and the choice is recorded. It runs only where the user says the repository requires AI disclosure. The two describe the same setup choice in incompatible shapes: a yes/no hook against a choice among conventions. itd-91 also names a page the ahoy page says is retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 351.

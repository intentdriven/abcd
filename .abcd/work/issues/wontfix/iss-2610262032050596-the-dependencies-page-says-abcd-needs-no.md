---
schema_version: 1
id: "iss-2610262032050596"
slug: "the-dependencies-page-says-abcd-needs-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 220 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:3"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-e8d7cfbec0ecd3bf
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The hooks directory holds no Python, so the brief is right today; the shipped intent describes the mechanism it had when it landed."
---

The dependencies page says abcd needs no external tool and ships as one Go binary; itd-3 makes the prompt router a Python hook.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:3` — "abcd has no required external tools. It ships as a single Go binary"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "**`hooks/prompt_router_hook.py`** — `UserPromptSubmit` hook"

The brief says abcd has no required external tools and ships as a single Go binary. itd-3 names `hooks/prompt_router_hook.py` as the UserPromptSubmit hook, which needs a Python interpreter on every machine. A router built from itd-3 would add a required runtime the brief excludes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 220.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The hooks directory holds no Python, so the brief is right today; the shipped intent describes the mechanism it had when it landed.

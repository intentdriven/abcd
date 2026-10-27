---
schema_version: 1
id: "iss-2610262045091104"
slug: "itd-39-extends-a-python-hook-hooks"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1255 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-39, itd-66]
finding_ref: cfk-77e82780e71dec8e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031560983. Same End A (itd-39 line 33); itd-66's payload ships shell under hooks/ and scripts/, no Python."
duplicates: [iss-2610262031560983]
---

itd-39 extends a Python hook, hooks/prompt_router_hook.py; itd-66 says the payload has no Python and hooks/ ships shell

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "`hooks/prompt_router_hook.py` (itd-3) is extended from rules-only to rules + memory"
- End B: `.abcd/development/intents/shipped/itd-66-launch-payload-render-parity.md:85` — "and no other imported entrypoint (`hooks/` and `scripts/` ship shell)"

itd-39 plans memory recall as an extension of `hooks/prompt_router_hook.py`. itd-66 records that the payload's include roots carry no Python and that `hooks/` and `scripts/` ship shell. The file itd-39 would extend does not exist in the shipped layout itd-66 describes, so whoever builds itd-39 is pointed at a component that is not there.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1255.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031560983. Same End A (itd-39 line 33); itd-66's payload ships shell under hooks/ and scripts/, no Python.

---
schema_version: 1
id: "iss-2610262033133842"
slug: "itd-3-rejects-a-machine-wide-rules-json"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 377 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:490"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-6444b154845a2906
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-3's audit notes already say a machine-wide rules.json layer was added after the intent shipped (itd-117), so the later record states which holds."
---

itd-3 rejects a machine-wide rules.json outright, per-repo only; the ahoy chapter's home inventory has one as the user layer of the rules

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:490` — "the machine's rule conventions, the user layer"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "explicitly rejected. Keeping a personal cross-repo rules file recreates exactly the"

The rules-loader intent rules a personal cross-repo rules file out of scope, calls it explicitly rejected and says the loader is per-repo only. The ahoy chapter's `~/.abcd.noindex/` inventory lists a machine `rules.json` as the user layer between the bundled domains and each repo's `.abcd/rules.json`. The loader either reads a machine-wide layer or it does not. The shipped intent records no reversal of its rejection.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 377.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-3's audit notes already say a machine-wide rules.json layer was added after the intent shipped (itd-117), so the later record states which holds.

---
schema_version: 1
id: "iss-2609100508566033"
slug: "abcd-does-not-name-its-own-adjacent"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface (lint, docs lint, intent status, spec close)"
resolution: "At BASE two of the three instances were already answered: bare abcd lint runs the docs-lint engine as its docs-currency rule and names abcd lint docs in its fix, and abcd <itd-N> names abcd intent ready --grounds for a record missing grounds. The third is fixed here: the ready-intent and open-spec moves say the close ships the intent when no open spec still names it. The broader 'next verb on the board' want is carried by iss-2609201954342967."
impact: additive
resolved_by:
  commit: "401094487"
---

abcd does not name its own adjacent capabilities, so a verb that exactly answers the operator's need is found by accident or not at all. Three instances in one run, from two independent sessions.

`abcd docs lint` exists and is exactly the gate the documentation convention needs, and no documentation page and no AGENTS line in the managed repository names it. The session ran it only after noticing a peer's commit message mention it, having until then been enforcing the documentation convention by reading. Their own suggested remedy: have `abcd lint` say "run docs lint too", or fold it in.

That `abcd spec close` is what ships the intent — that the intent's move from planned to shipped is a hook on the spec's close, not a separate act — was discovered mid-run by reading a skill page. It is invisible from the bare status output, which reports the intent's bucket and the spec's state without saying that one drives the other.

That `abcd intent ready --grounds` is the write path for grounds was discovered the same way, on the same page.

The common shape: the capability exists, is correct, and is reachable only by someone who already knows it is there. Status output names states without naming the verb that changes them; `lint` does not mention the sibling lint; the skill pages hold knowledge the CLI surface does not. An autonomous run is the worst case for this, because a session that does not know a verb exists does not go looking for it — it does the work by hand, or skips the gate.

Wanted, cheapest first: have each status render name the verb that advances the state it is reporting, and have `abcd lint` name the sibling lints it does not itself run. Then, more broadly, treat "which verb do I reach for next" as something the surface owes the operator rather than something the skill pages happen to record.

Distinct from the sibling finding about required flags learned from a refusal: that one is a verb the operator has found and cannot call.

## Grounds

- pursued: a session reading abcd <itd-N> or abcd <spc-N> learns the close is what ships the intent without a skill page; shown wrong if either move names spec close without saying what it does to the intent

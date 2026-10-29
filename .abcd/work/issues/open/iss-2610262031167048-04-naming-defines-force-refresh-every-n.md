---
schema_version: 1
id: "iss-2610262031167048"
slug: "04-naming-defines-force-refresh-every-n"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 107 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:208"
remedy: "Correct the Rules-loader dedup row (02-verification-matrix.md line 73): say the primary refresh is event-driven and a fixed-N backstop (default 15) also clears the ledger; delete 'Nothing counts prompts'."
finding_ref: cfk-0bbb2e8f1fd7c735
---

04-naming defines force_refresh_every_n as a full re-inject forced every N prompts; the verification matrix says the rules-loader refresh is event-driven and nothing counts prompts.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:208` — "Every N prompts, the prompt-router hook forces a full re-inject regardless of dedup signature match"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:73` — "Nothing counts prompts"

04-naming's reserved-vocabulary row says 'Every N prompts, the prompt-router hook forces a full re-inject regardless of dedup signature match', with a default of 15. The matrix's Rules-loader dedup row says the refresh is event-driven rather than turn-counted, SessionStart and PreCompact being the two events that clear the ledger, and 'Nothing counts prompts'. A prompt counter that forces a re-inject and the absence of any prompt counter cannot both describe the loader.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 107.

---
schema_version: 1
id: "iss-2609240227447110"
slug: "itd-7-depends-on-a-dev-sync-verb-and-an"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md"
resolution: "itd-7's waiting note now records both missing prerequisites beside itd-6: no dev-sync verb exists (the command tree carries none; only the draft itd-13 describes it), and embark writes only the four record families in embarkFamilies (internal/core/lifeboat/embark_types.go), so .abcd/rp/workspace.json has no route from a lifeboat into a target repository as acceptance criteria 6 and 7 require. Both facts re-checked at 8322cdf65. No gate reads a planned intent's prose for its prerequisites, so none would have caught this; the correction is to the record, and building either prerequisite stays with itd-13 and a future embark change."
impact: internal
resolved_by:
  commit: "efe9ad162"
---

itd-7 (planned, spec_id null) hangs its workspace pull on abcd dev-sync, which has no verb and no code: the CLI's command list carries no dev-sync, grep over internal/ and cmd/ finds none, and the only record of it is the draft itd-13 (scheduled dev-sync). Its lifeboat route does not exist either: embark writes only four record families, adrs, issues, intents and specs (embarkFamilies in internal/core/lifeboat/embark_types.go), and names every other file as one it does not write, so .abcd/rp/workspace.json has no path from a lifeboat into a target repository as acceptance criteria 6 and 7 require. itd-6 Decision 4 already has itd-7 waiting; the record does not say that its own two prerequisites are missing as well.

## Grounds

- pursued: a reader of itd-7 learns every missing prerequisite from the record itself; a dev-sync verb or an embark route for workspace.json landing without the note being updated would show it stale

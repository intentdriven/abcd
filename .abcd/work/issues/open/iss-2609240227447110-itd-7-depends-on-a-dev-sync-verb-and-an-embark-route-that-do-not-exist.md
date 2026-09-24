---
schema_version: 1
id: "iss-2609240227447110"
slug: "itd-7-depends-on-a-dev-sync-verb-and-an-embark-route-that-do-not-exist"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A, planning briefs"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md"
---

itd-7 (planned, spec_id null) hangs its workspace pull on abcd dev-sync, which has no verb and no code: the CLI's command list carries no dev-sync, grep over internal/ and cmd/ finds none, and the only record of it is the draft itd-13 (scheduled dev-sync). Its lifeboat route does not exist either: embark writes only four record families, adrs, issues, intents and specs (embarkFamilies in internal/core/lifeboat/embark_types.go), and names every other file as one it does not write, so .abcd/rp/workspace.json has no path from a lifeboat into a target repository as acceptance criteria 6 and 7 require. itd-6 Decision 4 already has itd-7 waiting; the record does not say that its own two prerequisites are missing as well.

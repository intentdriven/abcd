---
schema_version: 1
id: "iss-2611040554433915"
slug: "04-launch-md-300-says-each-gate-row"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md"
remedy: "State which rows carry a tier as the preview reports them, or stamp hard-fail on the rows the chapter calls hard-fail."
refines: [iss-2608231226342272]
---

04-launch.md:300 says each gate row carries a tier, 04-launch.md:385 says two rows report without one (citation baseline, semantic receipts), and lines 305-368 list the secret scan, merge hold and installability smoke as hard-fail gates. abcd launch --dry-run --json gives no tier on five rows: secret+pii-scan, installability-smoke, merge-hold, citation-baseline and semantic-receipts (internal/core/launch/dryrun.go:153,165; internal/core/launch/mergehold.go:112).

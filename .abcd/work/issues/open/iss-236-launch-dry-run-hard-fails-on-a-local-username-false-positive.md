---
schema_version: 1
id: "iss-236"
slug: "launch-dry-run-hard-fails-on-a-local-username-false-positive"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: "internal/core/launch"
---

launch --dry-run hard-fails on a local_username false positive: the scan matches the maintainer's short username against the documented '--dev' CLI flag text, so the release preview reports 5 hard fails on a clean tree. The matcher needs a word-boundary/flag-context exemption.

## Evidence

- 2026-09-23, autonomous run A: the same false positive fires in CI on the CI machine's account name. The forge's hosted Linux and macOS machines run as the account `runner`, so the payload scan in the `check` job read that word as the local username wherever it stood as an ordinary noun: #684 failed its secret and PII scan on both operating systems (hard fail 2) because commands/implement.md and the `implement` help text used it. Reproduced locally with HOME set to a directory of that name (red on the old tree, green once the text said "in CI" instead). Every lane of the run then carried a rule never to write the bare word in commands/, agents/ or CLI help. The exemption this record asks for would retire that rule.

---
schema_version: 1
id: "iss-236"
slug: "launch-dry-run-hard-fails-on-a-local-username-false-positive"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: "internal/core/launch"
resolution: "local_username no longer reports a generic account name (a built-in list of CI, image and role account names, or a one- or two-rune name) as a bare word; it still hard-fails the name where an account stands: after a home root, a tilde, or in an address or login local part. launch --dry-run reports 0 scan hardfails on a pristine tree under this machine's HOME, where it reported 10."
impact: fix
resolved_by:
  commit: "f05623e7bf66faa787f1c1e81380c9d00265accc"
---

launch --dry-run hard-fails on a local_username false positive: the scan matches the maintainer's short username against the documented '--dev' CLI flag text, so the release preview reports 5 hard fails on a clean tree. The matcher needs a word-boundary/flag-context exemption.

## Evidence

- 2026-09-23, autonomous run A: the same false positive fires in CI on the CI machine's account name. The forge's hosted Linux and macOS machines run as the account `runner`, so the payload scan in the `check` job read that word as the local username wherever it stood as an ordinary noun: #684 failed its secret and PII scan on both operating systems (hard fail 2) because commands/implement.md and the `implement` help text used it. Reproduced locally with HOME set to a directory of that name (red on the old tree, green once the text said "in CI" instead). Every lane of the run then carried a rule never to write the bare word in commands/, agents/ or CLI help. The exemption this record asks for would retire that rule.

## Grounds

- pursued: a generic account name identifies no person, so the payload and the redactors stop reporting its bare word while every home path and login shape is still caught; a leaked home path or login of a generic name that the scan no longer reports would show it wrong

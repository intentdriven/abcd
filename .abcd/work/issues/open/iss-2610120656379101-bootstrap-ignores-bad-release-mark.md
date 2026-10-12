---
schema_version: 1
id: "iss-2610120656379101"
slug: "bootstrap-ignores-bad-release-mark"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "Fable review of security lanes B and B2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/bootstrap.sh"
remedy: "Print one line on stderr when a mark exists but fails the ownership or mode check, naming the fix (chmod 600 or remove it), as the bootstrap does for other refused declarations; prove with a bootstrap test"
---

The bootstrap ignores a newest-release mark that fails its ownership or mode check (find ... ! -perm -0020) silently, so a machine whose mark is unreadable looks the same as one with no mark (fix for iss-2609151847117977). The attestation behaves the same way.

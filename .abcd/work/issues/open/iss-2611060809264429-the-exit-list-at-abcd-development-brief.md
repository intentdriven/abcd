---
schema_version: 1
id: "iss-2611060809264429"
slug: "the-exit-list-at-abcd-development-brief"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-reserve.md"
remedy: "State in the exit list that a release refused over a holder elsewhere exits 2, or class those refusals as back-offs."
---

The exit list at .abcd/development/brief/04-surfaces/37-reserve.md:961-964 gives exit 3 when another person's reservation is live or the same person's other copy holds it and no take-over was asked for. That holds for a reservation, not for `--release`: internal/core/claim/remote.go:682-690 refuses a release of another person's or another copy's reservation as `ErrRefused`, exit 2. A run from a sibling worktree of the holder gave exit 3 for `abcd reserve <id>` and exit 2 for `abcd reserve <id> --release`.

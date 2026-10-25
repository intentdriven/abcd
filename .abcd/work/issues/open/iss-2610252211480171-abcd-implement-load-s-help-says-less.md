---
schema_version: 1
id: "iss-2610252211480171"
slug: "abcd-implement-load-s-help-says-less"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D6"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement_load.go"
remedy: "Add 'and reservation renewals' to the sentence ('Check the load before abcd's tests start, or wait for capacity: Writes a run-log event, the wait's hold and reservation renewals; refuses an unknown --site.' is 156 characters); write 'Without --wait or --race-gate it never waits'; name the home-inside-a-working-tree refusal beside the symlink one; and say a release reservation is not renewed. Then regenerate the CLI reference."
---

abcd implement load's help says less than the verb does, in three places. (1) Its sentence (internal/core/surface/sentences.go:195-196; page line 2286) says 'Writes a run-log event or the wait's hold', but --wait also renews the caller's reservations, each renewal a commit pushed to origin (internal/surface/cli/implement_load.go:283,334-392; internal/core/claim/renewdue.go:45-64; claim/remote.go:155-159), added in 6931d4ac4. (2) The Long help says 'Without --wait it never refuses, never waits and never stops anything' (implement_load.go:65-66; page line 2298), yet --race-gate without --wait waits (implement_load.go:221-232), and an unknown --site or a stray --timeout is refused at exit 2 (implement_load.go:234-240). (3) The Long help names a symlinked ~/.abcd.noindex as a refusal (implement_load.go:186-187) but not the refusal under a HOME inside a git working tree that is not the account's own (internal/core/implement/load_wait.go:473-478, cd798880f), which commands/implement.md states; and it says the waiter renews 'each of your live reservations' (implement_load.go:189-191) while release reservations are never renewed (renewdue.go:53). The --wait flag line is already held by iss-2610141706070227.
